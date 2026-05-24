package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultQdrantPort    = 6333
	defaultQdrantTimeout = 15 * time.Second
	qdrantCollectionPref = "ws-"
)

var ErrQdrantCollectionNotFound = errors.New("qdrant collection not found")

type WorkspaceResolver interface {
	GetServiceDNS(workspaceID string) string
}

type QdrantClientConfig struct {
	HTTPClient        *http.Client
	Port              int
	WorkspaceResolver WorkspaceResolver
}

type QdrantClient struct {
	httpClient        *http.Client
	port              int
	workspaceResolver WorkspaceResolver
}

func NewQdrantClient(cfg QdrantClientConfig) *QdrantClient {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: defaultQdrantTimeout}
	}
	if cfg.Port <= 0 {
		cfg.Port = defaultQdrantPort
	}

	return &QdrantClient{
		httpClient:        cfg.HTTPClient,
		port:              cfg.Port,
		workspaceResolver: cfg.WorkspaceResolver,
	}
}

func (c *QdrantClient) Search(ctx context.Context, workspaceID string, vector []float64, limit int) ([]RetrievedChunk, error) {
	if strings.TrimSpace(workspaceID) == "" {
		return nil, fmt.Errorf("search qdrant: %w: workspace id is required", ErrInvalidRequest)
	}
	if len(vector) == 0 || limit <= 0 {
		return nil, nil
	}
	if c.workspaceResolver == nil {
		return nil, fmt.Errorf("search qdrant: %w: workspace resolver is not configured", ErrInvalidRequest)
	}

	endpoint := fmt.Sprintf(
		"http://%s:%d/collections/%s/points/search",
		c.workspaceResolver.GetServiceDNS(workspaceID),
		c.port,
		url.PathEscape(collectionNameForWorkspace(workspaceID)),
	)
	payload := map[string]any{
		"limit":        limit,
		"vector":       vector,
		"with_payload": true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal qdrant search request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build qdrant search request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("execute qdrant search request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if response.StatusCode >= http.StatusBadRequest {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 16*1024))
		return nil, fmt.Errorf("qdrant search failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(detail)))
	}

	var decoded struct {
		Result []struct {
			Payload struct {
				Metadata struct {
					EndLine    int    `json:"end_line"`
					File       string `json:"file"`
					Language   string `json:"language"`
					Name       string `json:"name"`
					StartLine  int    `json:"start_line"`
					SymbolType string `json:"symbol_type"`
				} `json:"metadata"`
				Text string `json:"text"`
			} `json:"payload"`
			Score float64 `json:"score"`
		} `json:"result"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode qdrant search response: %w", err)
	}

	results := make([]RetrievedChunk, 0, len(decoded.Result))
	for _, point := range decoded.Result {
		if strings.TrimSpace(point.Payload.Text) == "" {
			continue
		}
		results = append(results, RetrievedChunk{
			EndLine:    point.Payload.Metadata.EndLine,
			FilePath:   point.Payload.Metadata.File,
			Language:   point.Payload.Metadata.Language,
			Name:       point.Payload.Metadata.Name,
			Score:      point.Score,
			StartLine:  point.Payload.Metadata.StartLine,
			SymbolType: point.Payload.Metadata.SymbolType,
			Text:       point.Payload.Text,
		})
	}

	return results, nil
}

func collectionNameForWorkspace(workspaceID string) string {
	return qdrantCollectionPref + strings.TrimSpace(workspaceID)
}
