package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"github.com/your-org/cortado/control-plane/internal/auth"
	cpmiddleware "github.com/your-org/cortado/control-plane/internal/middleware"
	"github.com/your-org/cortado/control-plane/internal/workspace"
)

type WorkspaceService interface {
	CreateWorkspace(ctx context.Context, params workspace.CreateParams) (workspace.Workspace, error)
	DeleteWorkspace(ctx context.Context, tenantID, workspaceID string) (workspace.Workspace, error)
	GetWorkspace(ctx context.Context, tenantID, workspaceID string) (workspace.Workspace, error)
	ListWorkspaces(ctx context.Context, tenantID string) ([]workspace.Workspace, error)
	StartWorkspace(ctx context.Context, tenantID, workspaceID string) (workspace.Workspace, error)
	StopWorkspace(ctx context.Context, tenantID, workspaceID string) (workspace.Workspace, error)
}

type workspacesHandler struct {
	service WorkspaceService
}

type createWorkspaceRequest struct {
	Image     string                    `json:"image"`
	Resources *createWorkspaceResources `json:"resources,omitempty"`
}

type createWorkspaceResources struct {
	CPU       float64 `json:"cpu"`
	MemoryGB  float64 `json:"memoryGb"`
	StorageGB float64 `json:"storageGb"`
}

type workspaceEnvelope struct {
	Workspace workspace.Workspace `json:"workspace"`
}

type workspaceListEnvelope struct {
	Workspaces []workspace.Workspace `json:"workspaces"`
}

func newWorkspacesHandler(service WorkspaceService) *workspacesHandler {
	return &workspacesHandler{service: service}
}

func (h *workspacesHandler) create(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requestActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	var request createWorkspaceRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	params := workspace.CreateParams{
		Image:    request.Image,
		TenantID: tenantID,
		UserID:   userID,
	}
	if request.Resources != nil {
		params.Resources = workspace.Resources{
			CPU:       request.Resources.CPU,
			MemoryGB:  request.Resources.MemoryGB,
			StorageGB: request.Resources.StorageGB,
		}
	}

	created, err := h.service.CreateWorkspace(r.Context(), params)
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, workspaceEnvelope{Workspace: created})
}

func (h *workspacesHandler) get(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requestActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	ws, err := h.service.GetWorkspace(r.Context(), tenantID, chi.URLParam(r, "id"))
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, workspaceEnvelope{Workspace: ws})
}

func (h *workspacesHandler) list(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requestActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	workspaces, err := h.service.ListWorkspaces(r.Context(), tenantID)
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, workspaceListEnvelope{Workspaces: workspaces})
}

func (h *workspacesHandler) start(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.StartWorkspace)
}

func (h *workspacesHandler) stop(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.StopWorkspace)
}

func (h *workspacesHandler) delete(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.DeleteWorkspace)
}

func (h *workspacesHandler) transition(
	w http.ResponseWriter,
	r *http.Request,
	fn func(ctx context.Context, tenantID, workspaceID string) (workspace.Workspace, error),
) {
	tenantID, _, ok := requestActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	ws, err := fn(r.Context(), tenantID, chi.URLParam(r, "id"))
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, workspaceEnvelope{Workspace: ws})
}

func requestActor(r *http.Request) (tenantID string, userID string, ok bool) {
	tenantID, ok = cpmiddleware.TenantID(r.Context())
	if !ok || tenantID == "" {
		return "", "", false
	}

	userID, _ = cpmiddleware.UserID(r.Context())
	return tenantID, userID, true
}

func requestUserActor(r *http.Request) (tenantID string, userID string, ok bool) {
	tenantID, userID, ok = requestActor(r)
	if !ok || tenantID == "" || userID == "" {
		return "", "", false
	}

	actorType, found := cpmiddleware.ActorType(r.Context())
	if !found || actorType == "" {
		actorType = auth.ActorTypeUser
	}
	if actorType != auth.ActorTypeUser {
		return "", "", false
	}

	return tenantID, userID, true
}

func decodeJSON(r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func writeWorkspaceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, workspace.ErrNotFound):
		http.Error(w, "workspace not found", http.StatusNotFound)
	case errors.Is(err, workspace.ErrPathNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, workspace.ErrAlreadyExists):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, workspace.ErrConflict):
		http.Error(w, "workspace conflict", http.StatusConflict)
	case errors.Is(err, workspace.ErrInvalid), errors.Is(err, workspace.ErrTenantID), errors.Is(err, workspace.ErrWorkspace):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
