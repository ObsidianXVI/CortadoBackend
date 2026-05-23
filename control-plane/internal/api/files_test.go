package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	agentpb "github.com/your-org/cortado/agent/gen/agent/v1"
	"github.com/your-org/cortado/control-plane/internal/workspace"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestFileRoutesListReadWriteMakeDirRenameAndDelete(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	now := time.Date(2026, time.May, 23, 20, 0, 0, 0, time.UTC)
	workspaces := &workspaceServiceStub{
		getResult: workspace.Workspace{
			ID:        "ws-123",
			TenantID:  "dev-tenant",
			Status:    workspace.StatusRunning,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	files := &workspaceFileServiceStub{
		listEntries: []*agentpb.DirectoryEntry{
			{
				Name:        "main.go",
				Size:        123,
				Permissions: 0o644,
				ModTime:     timestamppb.New(now),
			},
		},
		readContent: []byte("package main\n"),
		writeResponse: &agentpb.WriteFileResponse{
			BytesWritten: 12,
			Checksum:     []byte("checksum"),
		},
	}

	router := NewRouter(RouterConfig{
		WorkspaceSvc:     workspaces,
		WorkspaceFileSvc: files,
	})

	listReq := httptest.NewRequest(http.MethodGet, "/v1/workspaces/ws-123/files?path=src", nil)
	listReq.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("unexpected list status: got %d want %d", listRec.Code, http.StatusOK)
	}
	if files.listPath != "src" {
		t.Fatalf("unexpected list path: %q", files.listPath)
	}

	readReq := httptest.NewRequest(http.MethodGet, "/v1/workspaces/ws-123/files/content?path=src/main.go", nil)
	readReq.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	readRec := httptest.NewRecorder()
	router.ServeHTTP(readRec, readReq)

	if readRec.Code != http.StatusOK {
		t.Fatalf("unexpected read status: got %d want %d", readRec.Code, http.StatusOK)
	}
	if readRec.Body.String() != "package main\n" {
		t.Fatalf("unexpected read body: %q", readRec.Body.String())
	}

	writeReq := httptest.NewRequest(http.MethodPut, "/v1/workspaces/ws-123/files/content?path=src/main.go", bytes.NewBufferString("updated body"))
	writeReq.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	writeRec := httptest.NewRecorder()
	router.ServeHTTP(writeRec, writeReq)

	if writeRec.Code != http.StatusOK {
		t.Fatalf("unexpected write status: got %d want %d", writeRec.Code, http.StatusOK)
	}
	if files.writePath != "src/main.go" || string(files.writeContent) != "updated body" {
		t.Fatalf("unexpected write capture: path=%q body=%q", files.writePath, files.writeContent)
	}

	mkdirReq := httptest.NewRequest(http.MethodPost, "/v1/workspaces/ws-123/files/directory?path=src/newdir", nil)
	mkdirReq.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	mkdirRec := httptest.NewRecorder()
	router.ServeHTTP(mkdirRec, mkdirReq)

	if mkdirRec.Code != http.StatusCreated {
		t.Fatalf("unexpected mkdir status: got %d want %d", mkdirRec.Code, http.StatusCreated)
	}
	if files.mkdirPath != "src/newdir" {
		t.Fatalf("unexpected mkdir path: %q", files.mkdirPath)
	}

	renameReq := httptest.NewRequest(http.MethodPost, "/v1/workspaces/ws-123/files/rename?path=src/main.go&newPath=src/app.go", nil)
	renameReq.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	renameRec := httptest.NewRecorder()
	router.ServeHTTP(renameRec, renameReq)

	if renameRec.Code != http.StatusNoContent {
		t.Fatalf("unexpected rename status: got %d want %d", renameRec.Code, http.StatusNoContent)
	}
	if files.renameOldPath != "src/main.go" || files.renameNewPath != "src/app.go" {
		t.Fatalf("unexpected rename paths: old=%q new=%q", files.renameOldPath, files.renameNewPath)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/v1/workspaces/ws-123/files?path=src/main.go", nil)
	deleteReq.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("unexpected delete status: got %d want %d", deleteRec.Code, http.StatusNoContent)
	}
	if files.deletePath != "src/main.go" {
		t.Fatalf("unexpected delete path: %q", files.deletePath)
	}
}

func TestFileRoutesMakeDirConflictReturnsConflict(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	now := time.Date(2026, time.May, 23, 20, 0, 0, 0, time.UTC)
	router := NewRouter(RouterConfig{
		WorkspaceSvc: &workspaceServiceStub{
			getResult: workspace.Workspace{
				ID:        "ws-123",
				TenantID:  "dev-tenant",
				Status:    workspace.StatusRunning,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		WorkspaceFileSvc: &workspaceFileServiceStub{
			mkdirErr: workspace.ErrAlreadyExists,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/workspaces/ws-123/files/directory?path=src/newdir", nil)
	req.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("unexpected mkdir conflict status: got %d want %d", rec.Code, http.StatusConflict)
	}
}

type workspaceFileServiceStub struct {
	deleteErr     error
	deletePath    string
	listEntries   []*agentpb.DirectoryEntry
	listErr       error
	listPath      string
	mkdirErr      error
	mkdirPath     string
	readContent   []byte
	readErr       error
	readPath      string
	renameErr     error
	renameNewPath string
	renameOldPath string
	writeContent  []byte
	writeErr      error
	writePath     string
	writeResponse *agentpb.WriteFileResponse
}

func (s *workspaceFileServiceStub) DeletePath(_ context.Context, _, path string) error {
	s.deletePath = path
	return s.deleteErr
}

func (s *workspaceFileServiceStub) ListDir(_ context.Context, _, path string) ([]*agentpb.DirectoryEntry, error) {
	s.listPath = path
	return s.listEntries, s.listErr
}

func (s *workspaceFileServiceStub) MakeDir(_ context.Context, _, path string) error {
	s.mkdirPath = path
	return s.mkdirErr
}

func (s *workspaceFileServiceStub) ReadFile(_ context.Context, _, path string, writer io.Writer) error {
	s.readPath = path
	if s.readErr != nil {
		return s.readErr
	}
	_, err := writer.Write(s.readContent)
	return err
}

func (s *workspaceFileServiceStub) WriteFile(_ context.Context, _, path string, reader io.Reader) (*agentpb.WriteFileResponse, error) {
	s.writePath = path
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	s.writeContent = body
	return s.writeResponse, s.writeErr
}

func (s *workspaceFileServiceStub) RenamePath(_ context.Context, _, oldPath, newPath string) error {
	s.renameOldPath = oldPath
	s.renameNewPath = newPath
	return s.renameErr
}
