package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	defaultMaxPrefixBytes     = 4 * 1024
	defaultMaxSuffixBytes     = 1 * 1024
	defaultMaxRetrievalResult = 3
	defaultQueryLineCount     = 5
)

var ErrInvalidRequest = errors.New("invalid completion request")

type CompletionParams struct {
	Path        string
	Prefix      string
	Suffix      string
	WorkspaceID string
}

type RetrievedChunk struct {
	EndLine    int
	FilePath   string
	Language   string
	Name       string
	Score      float64
	StartLine  int
	SymbolType string
	Text       string
}

type CompletionContext struct {
	Path           string
	Prefix         string
	Related        []RetrievedChunk
	RetrievalQuery string
	Suffix         string
	WorkspaceID    string
}

type QueryEmbedder interface {
	EmbedQuery(ctx context.Context, query string) ([]float64, error)
}

type Retriever interface {
	Search(ctx context.Context, workspaceID string, vector []float64, limit int) ([]RetrievedChunk, error)
}

type Provider interface {
	StreamCompletion(ctx context.Context, prompt CompletionContext, emit func(string) error) error
}

type ServiceConfig struct {
	MaxPrefixBytes int
	MaxSuffixBytes int
	MaxResults     int
	QueryLines     int
	QueryEmbedder  QueryEmbedder
	Provider       Provider
	Retriever      Retriever
}

type Service struct {
	maxPrefixBytes int
	maxSuffixBytes int
	maxResults     int
	provider       Provider
	queryEmbedder  QueryEmbedder
	queryLines     int
	retriever      Retriever
}

func NewService(cfg ServiceConfig) *Service {
	if cfg.MaxPrefixBytes <= 0 {
		cfg.MaxPrefixBytes = defaultMaxPrefixBytes
	}
	if cfg.MaxSuffixBytes <= 0 {
		cfg.MaxSuffixBytes = defaultMaxSuffixBytes
	}
	if cfg.MaxResults <= 0 {
		cfg.MaxResults = defaultMaxRetrievalResult
	}
	if cfg.QueryLines <= 0 {
		cfg.QueryLines = defaultQueryLineCount
	}

	return &Service{
		maxPrefixBytes: cfg.MaxPrefixBytes,
		maxSuffixBytes: cfg.MaxSuffixBytes,
		maxResults:     cfg.MaxResults,
		provider:       cfg.Provider,
		queryEmbedder:  cfg.QueryEmbedder,
		queryLines:     cfg.QueryLines,
		retriever:      cfg.Retriever,
	}
}

func (s *Service) StreamCompletion(ctx context.Context, params CompletionParams, emit func(string) error) error {
	if err := validateCompletionParams(params, emit); err != nil {
		return err
	}
	if s.provider == nil {
		return fmt.Errorf("stream completion: %w: provider is not configured", ErrInvalidRequest)
	}

	completionContext := CompletionContext{
		Path:        strings.TrimSpace(params.Path),
		Prefix:      tailUTF8(params.Prefix, s.maxPrefixBytes),
		Suffix:      headUTF8(params.Suffix, s.maxSuffixBytes),
		WorkspaceID: strings.TrimSpace(params.WorkspaceID),
	}
	completionContext.RetrievalQuery = lastNonEmptyLines(completionContext.Prefix, s.queryLines)

	if completionContext.RetrievalQuery != "" && s.queryEmbedder != nil && s.retriever != nil {
		vector, err := s.queryEmbedder.EmbedQuery(ctx, completionContext.RetrievalQuery)
		if err != nil {
			return fmt.Errorf("embed retrieval query: %w", err)
		}

		related, err := s.retriever.Search(ctx, completionContext.WorkspaceID, vector, s.maxResults)
		if err != nil {
			return fmt.Errorf("search related chunks: %w", err)
		}
		completionContext.Related = related
	}

	if err := s.provider.StreamCompletion(ctx, completionContext, emit); err != nil {
		return fmt.Errorf("stream provider completion: %w", err)
	}
	return nil
}

func validateCompletionParams(params CompletionParams, emit func(string) error) error {
	if emit == nil {
		return fmt.Errorf("%w: token emitter is required", ErrInvalidRequest)
	}
	if strings.TrimSpace(params.WorkspaceID) == "" {
		return fmt.Errorf("%w: workspace id is required", ErrInvalidRequest)
	}
	if params.Prefix == "" && params.Suffix == "" {
		return fmt.Errorf("%w: prefix or suffix is required", ErrInvalidRequest)
	}
	return nil
}
