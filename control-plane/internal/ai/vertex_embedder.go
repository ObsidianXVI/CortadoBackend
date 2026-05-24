package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	defaultVertexDimensions = 768
	defaultVertexLocation   = "us-central1"
	defaultVertexModel      = "text-embedding-004"
	defaultVertexTaskType   = "RETRIEVAL_QUERY"
	defaultVertexTimeout    = 15 * time.Second
	vertexScope             = "https://www.googleapis.com/auth/cloud-platform"
)

type VertexEmbedderConfig struct {
	Dimensions  int
	HTTPClient  *http.Client
	Location    string
	Model       string
	ProjectID   string
	TaskType    string
	TokenSource oauth2.TokenSource
}

type VertexEmbedder struct {
	dimensions  int
	httpClient  *http.Client
	location    string
	model       string
	projectID   string
	taskType    string
	tokenSource oauth2.TokenSource
}

func NewVertexEmbedder(cfg VertexEmbedderConfig) (*VertexEmbedder, error) {
	if strings.TrimSpace(cfg.ProjectID) == "" {
		return nil, fmt.Errorf("create vertex embedder: %w: project id is required", ErrInvalidRequest)
	}
	if cfg.Dimensions <= 0 {
		cfg.Dimensions = defaultVertexDimensions
	}
	if strings.TrimSpace(cfg.Location) == "" {
		cfg.Location = defaultVertexLocation
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = defaultVertexModel
	}
	if strings.TrimSpace(cfg.TaskType) == "" {
		cfg.TaskType = defaultVertexTaskType
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: defaultVertexTimeout}
	}
	if cfg.TokenSource == nil {
		tokenSource, err := google.DefaultTokenSource(context.Background(), vertexScope)
		if err != nil {
			return nil, fmt.Errorf("create vertex token source: %w", err)
		}
		cfg.TokenSource = tokenSource
	}

	return &VertexEmbedder{
		dimensions:  cfg.Dimensions,
		httpClient:  cfg.HTTPClient,
		location:    cfg.Location,
		model:       cfg.Model,
		projectID:   cfg.ProjectID,
		taskType:    cfg.TaskType,
		tokenSource: cfg.TokenSource,
	}, nil
}

func (e *VertexEmbedder) EmbedQuery(ctx context.Context, query string) ([]float64, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}

	token, err := e.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("fetch vertex access token: %w", err)
	}

	payload := map[string]any{
		"instances": []map[string]any{
			{
				"content":   query,
				"task_type": e.taskType,
			},
		},
		"parameters": map[string]any{
			"autoTruncate":         true,
			"outputDimensionality": e.dimensions,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal vertex embed request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build vertex embed request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	response, err := e.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("execute vertex embed request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 16*1024))
		return nil, fmt.Errorf("vertex embed request failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(detail)))
	}

	var decoded struct {
		Predictions []struct {
			Embeddings struct {
				Values []float64 `json:"values"`
			} `json:"embeddings"`
		} `json:"predictions"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode vertex embed response: %w", err)
	}
	if len(decoded.Predictions) != 1 {
		return nil, fmt.Errorf("decode vertex embed response: expected 1 prediction, got %d", len(decoded.Predictions))
	}

	return decoded.Predictions[0].Embeddings.Values, nil
}

func (e *VertexEmbedder) endpoint() string {
	return fmt.Sprintf(
		"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:predict",
		e.location,
		e.projectID,
		e.location,
		e.model,
	)
}
