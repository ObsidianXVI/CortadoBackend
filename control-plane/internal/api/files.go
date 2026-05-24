package api

import (
	"net/http"
	"strconv"
	"time"

	chi "github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type filesHandler struct {
	fileService      WorkspaceFileService
	workspaceService WorkspaceService
}

type fileEntryResponse struct {
	IsDir       bool      `json:"isDir"`
	ModTime     time.Time `json:"modTime"`
	Name        string    `json:"name"`
	Permissions uint32    `json:"permissions"`
	Size        int64     `json:"size"`
}

type fileListEnvelope struct {
	Entries []fileEntryResponse `json:"entries"`
}

type writeFileEnvelope struct {
	BytesWritten int64  `json:"bytesWritten"`
	Checksum     []byte `json:"checksum"`
}

func newFilesHandler(workspaces WorkspaceService, files WorkspaceFileService) *filesHandler {
	return &filesHandler{
		fileService:      files,
		workspaceService: workspaces,
	}
}

func (h *filesHandler) list(w http.ResponseWriter, r *http.Request) {
	workspaceID, path, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	entries, err := h.fileService.ListDir(r.Context(), workspaceID, path)
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}

	response := fileListEnvelope{Entries: make([]fileEntryResponse, 0, len(entries))}
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		response.Entries = append(response.Entries, fileEntryResponse{
			Name:        entry.GetName(),
			Size:        entry.GetSize(),
			IsDir:       entry.GetIsDir(),
			Permissions: entry.GetPermissions(),
			ModTime:     timestampAsTime(entry.GetModTime()),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *filesHandler) readContent(w http.ResponseWriter, r *http.Request) {
	workspaceID, path, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	if err := h.fileService.ReadFile(r.Context(), workspaceID, path, w); err != nil {
		writeWorkspaceError(w, err)
		return
	}
}

func (h *filesHandler) writeContent(w http.ResponseWriter, r *http.Request) {
	workspaceID, path, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	createMissingDirs, err := parseCreateMissingDirs(r)
	if err != nil {
		http.Error(w, "invalid createMissingDirs query parameter", http.StatusBadRequest)
		return
	}

	response, err := h.fileService.WriteFile(r.Context(), workspaceID, path, createMissingDirs, r.Body)
	if err != nil {
		writeWorkspaceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, writeFileEnvelope{
		BytesWritten: response.GetBytesWritten(),
		Checksum:     response.GetChecksum(),
	})
}

func (h *filesHandler) makeDir(w http.ResponseWriter, r *http.Request) {
	workspaceID, path, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	if err := h.fileService.MakeDir(r.Context(), workspaceID, path); err != nil {
		writeWorkspaceError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *filesHandler) rename(w http.ResponseWriter, r *http.Request) {
	workspaceID, oldPath, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	newPath := r.URL.Query().Get("newPath")
	if err := h.fileService.RenamePath(r.Context(), workspaceID, oldPath, newPath); err != nil {
		writeWorkspaceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *filesHandler) delete(w http.ResponseWriter, r *http.Request) {
	workspaceID, path, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	if err := h.fileService.DeletePath(r.Context(), workspaceID, path); err != nil {
		writeWorkspaceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *filesHandler) authorizeRequest(w http.ResponseWriter, r *http.Request) (workspaceID string, path string, ok bool) {
	tenantID, _, actorOK := requestActor(r)
	if !actorOK {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return "", "", false
	}

	workspaceID = chi.URLParam(r, "id")
	if _, err := h.workspaceService.GetWorkspace(r.Context(), tenantID, workspaceID); err != nil {
		writeWorkspaceError(w, err)
		return "", "", false
	}

	return workspaceID, r.URL.Query().Get("path"), true
}

func timestampAsTime(value *timestamppb.Timestamp) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.AsTime().UTC()
}

func parseCreateMissingDirs(r *http.Request) (bool, error) {
	value := r.URL.Query().Get("createMissingDirs")
	if value == "" {
		return true, nil
	}
	return strconv.ParseBool(value)
}
