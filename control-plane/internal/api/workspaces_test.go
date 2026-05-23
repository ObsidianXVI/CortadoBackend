package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/your-org/cortado/control-plane/internal/auth"
	"github.com/your-org/cortado/control-plane/internal/workspace"
	"golang.org/x/crypto/bcrypt"
)

func TestWorkspaceRoutesCreateListAndGet(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	now := time.Date(2026, time.May, 23, 19, 0, 0, 0, time.UTC)
	service := &workspaceServiceStub{
		createResult: workspace.Workspace{
			ID:       "ws-123",
			TenantID: "dev-tenant",
			UserID:   "dev-user",
			Image:    "example.com/cortado/workspace:test",
			Resources: workspace.Resources{
				CPU:      2,
				MemoryGB: 4,
			},
			Status:    workspace.StatusCreating,
			CreatedAt: now,
			UpdatedAt: now,
		},
		getResult: workspace.Workspace{
			ID:        "ws-123",
			TenantID:  "dev-tenant",
			Status:    workspace.StatusRunning,
			CreatedAt: now,
			UpdatedAt: now,
		},
		listResult: []workspace.Workspace{
			{
				ID:        "ws-123",
				TenantID:  "dev-tenant",
				Status:    workspace.StatusRunning,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	router := NewRouter(RouterConfig{WorkspaceSvc: service})

	createBody := bytes.NewBufferString(`{"image":"example.com/cortado/workspace:test","resources":{"cpu":2,"memoryGb":4}}`)
	createReq := httptest.NewRequest(http.MethodPost, "/v1/workspaces", createBody)
	createReq.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	createRec := httptest.NewRecorder()

	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusAccepted {
		t.Fatalf("unexpected create status: got %d want %d", createRec.Code, http.StatusAccepted)
	}
	if service.createParams.TenantID != "dev-tenant" || service.createParams.UserID != "dev-user" {
		t.Fatalf("unexpected create actor: %+v", service.createParams)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/workspaces", nil)
	listReq.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("unexpected list status: got %d want %d", listRec.Code, http.StatusOK)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/workspaces/ws-123", nil)
	getReq.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("unexpected get status: got %d want %d", getRec.Code, http.StatusOK)
	}

	var createResponse workspaceEnvelope
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResponse); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResponse.Workspace.ID != "ws-123" || createResponse.Workspace.Status != workspace.StatusCreating {
		t.Fatalf("unexpected create response: %+v", createResponse.Workspace)
	}
}

func TestWorkspaceRoutesStartStopAndDelete(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	service := &workspaceServiceStub{
		startResult:  workspace.Workspace{ID: "ws-123", Status: workspace.StatusStarting},
		stopResult:   workspace.Workspace{ID: "ws-123", Status: workspace.StatusStopping},
		deleteResult: workspace.Workspace{ID: "ws-123", Status: workspace.StatusDeleted},
	}
	router := NewRouter(RouterConfig{WorkspaceSvc: service})

	for _, testCase := range []struct {
		method string
		path   string
		status workspace.Status
	}{
		{method: http.MethodPost, path: "/v1/workspaces/ws-123/start", status: workspace.StatusStarting},
		{method: http.MethodPost, path: "/v1/workspaces/ws-123/stop", status: workspace.StatusStopping},
		{method: http.MethodDelete, path: "/v1/workspaces/ws-123", status: workspace.StatusDeleted},
	} {
		req := httptest.NewRequest(testCase.method, testCase.path, nil)
		req.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Fatalf("%s %s unexpected status: got %d want %d", testCase.method, testCase.path, rec.Code, http.StatusAccepted)
		}

		var response workspaceEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode transition response for %s %s: %v", testCase.method, testCase.path, err)
		}
		if response.Workspace.Status != testCase.status {
			t.Fatalf("%s %s unexpected workspace status: got %q want %q", testCase.method, testCase.path, response.Workspace.Status, testCase.status)
		}
	}
}

func TestWorkspaceRoutesMapNotFoundErrors(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	router := NewRouter(RouterConfig{
		WorkspaceSvc: &workspaceServiceStub{getErr: workspace.ErrNotFound},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/workspaces/ws-404", nil)
	req.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("unexpected get status: got %d want %d", rec.Code, http.StatusNotFound)
	}
}

func TestWorkspaceRoutesCreateWithJWTAuth(t *testing.T) {
	t.Setenv("CORTADO_ENV", "production")

	now := time.Date(2026, time.May, 23, 19, 0, 0, 0, time.UTC)
	authService, accessToken := mustIssueAccessToken(t, "tenant-1", "user-1")
	service := &workspaceServiceStub{
		createResult: workspace.Workspace{
			ID:        "ws-123",
			TenantID:  "tenant-1",
			UserID:    "user-1",
			Status:    workspace.StatusCreating,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	router := NewRouter(RouterConfig{
		JWKSProvider: authService,
		WorkspaceSvc: service,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/workspaces", bytes.NewBufferString(`{"image":"example.com/cortado/workspace:test"}`))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected create status: got %d want %d", rec.Code, http.StatusAccepted)
	}
	if service.createParams.TenantID != "tenant-1" || service.createParams.UserID != "user-1" {
		t.Fatalf("unexpected create actor: %+v", service.createParams)
	}
}

type workspaceServiceStub struct {
	createErr    error
	deleteErr    error
	getErr       error
	listErr      error
	startErr     error
	stopErr      error
	createParams workspace.CreateParams
	createResult workspace.Workspace
	deleteResult workspace.Workspace
	getResult    workspace.Workspace
	listResult   []workspace.Workspace
	startResult  workspace.Workspace
	stopResult   workspace.Workspace
}

func (s *workspaceServiceStub) CreateWorkspace(_ context.Context, params workspace.CreateParams) (workspace.Workspace, error) {
	s.createParams = params
	return s.createResult, s.createErr
}

func (s *workspaceServiceStub) DeleteWorkspace(_ context.Context, _, _ string) (workspace.Workspace, error) {
	return s.deleteResult, s.deleteErr
}

func (s *workspaceServiceStub) GetWorkspace(_ context.Context, _, _ string) (workspace.Workspace, error) {
	return s.getResult, s.getErr
}

func (s *workspaceServiceStub) ListWorkspaces(_ context.Context, _ string) ([]workspace.Workspace, error) {
	return s.listResult, s.listErr
}

func (s *workspaceServiceStub) StartWorkspace(_ context.Context, _, _ string) (workspace.Workspace, error) {
	return s.startResult, s.startErr
}

func (s *workspaceServiceStub) StopWorkspace(_ context.Context, _, _ string) (workspace.Workspace, error) {
	return s.stopResult, s.stopErr
}

var _ WorkspaceService = (*workspaceServiceStub)(nil)

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/workspaces", bytes.NewBufferString(`{"image":"ok","unexpected":true}`))

	var payload createWorkspaceRequest
	err := decodeJSON(request, &payload)
	if err == nil || errors.Is(err, context.Canceled) {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func mustIssueAccessToken(t *testing.T, tenantID, userID string) (*auth.Service, string) {
	t.Helper()

	privateKeyPEM, err := auth.GenerateRSAKeyPEM()
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("secret-api-key"), 12)
	if err != nil {
		t.Fatalf("hash api key: %v", err)
	}

	service, err := auth.NewService(auth.ServiceConfig{
		PrivateKeyPEM: privateKeyPEM,
		Repository: &workspaceAuthRepositoryStub{
			apiKeys: []auth.APIKeyRecord{{TenantID: tenantID, Hash: string(hash)}},
		},
	})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	tokens, err := service.CreateSession(context.Background(), "secret-api-key", userID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	return service, tokens.AccessToken
}

type workspaceAuthRepositoryStub struct {
	apiKeys []auth.APIKeyRecord
}

func (r *workspaceAuthRepositoryStub) ListAPIKeys(_ context.Context) ([]auth.APIKeyRecord, error) {
	return append([]auth.APIKeyRecord(nil), r.apiKeys...), nil
}

func (r *workspaceAuthRepositoryStub) SaveRefreshToken(_ context.Context, token auth.RefreshTokenRecord) error {
	return nil
}
