package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/your-org/cortado/control-plane/internal/ai"
	"github.com/your-org/cortado/control-plane/internal/workspace"
)

func TestAICompletionRouteStreamsSSE(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	now := time.Date(2026, time.May, 24, 0, 0, 0, 0, time.UTC)
	workspaces := &workspaceServiceStub{
		getResult: workspace.Workspace{
			ID:        "ws-123",
			TenantID:  "dev-tenant",
			Status:    workspace.StatusRunning,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	completions := &completionServiceStub{tokens: []string{"func ", "main() {}"}}
	router := NewRouter(RouterConfig{
		AICompletionSvc: completions,
		WorkspaceSvc:    workspaces,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/workspaces/ws-123/ai/complete", bytes.NewBufferString(`{"path":"lib/main.dart","prefix":"void ","suffix":"{}"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("unexpected content type: %q", got)
	}
	if completions.params.WorkspaceID != "ws-123" || completions.params.Path != "lib/main.dart" {
		t.Fatalf("unexpected completion params: %+v", completions.params)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"token":"func "`) || !strings.Contains(body, `"token":"main() {}"`) {
		t.Fatalf("unexpected sse body: %q", body)
	}
}

func TestAICompletionRouteMapsInvalidRequestToBadRequest(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	now := time.Date(2026, time.May, 24, 0, 0, 0, 0, time.UTC)
	router := NewRouter(RouterConfig{
		AICompletionSvc: &completionServiceStub{
			err: fmt.Errorf("wrap: %w", ai.ErrInvalidRequest),
		},
		WorkspaceSvc: &workspaceServiceStub{
			getResult: workspace.Workspace{
				ID:        "ws-123",
				TenantID:  "dev-tenant",
				Status:    workspace.StatusRunning,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/workspaces/ws-123/ai/complete", bytes.NewBufferString(`{"prefix":"void ","suffix":"{}"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusBadRequest)
	}
}

type completionServiceStub struct {
	err    error
	params ai.CompletionParams
	tokens []string
}

func (s *completionServiceStub) StreamCompletion(_ context.Context, params ai.CompletionParams, emit func(string) error) error {
	s.params = params
	if s.err != nil {
		return s.err
	}
	for _, token := range s.tokens {
		if err := emit(token); err != nil {
			return err
		}
	}
	return nil
}
