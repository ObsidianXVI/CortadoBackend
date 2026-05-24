package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"github.com/your-org/cortado/control-plane/internal/ai"
)

type aiHandler struct {
	completionService AICompletionService
	workspaceService  WorkspaceService
}

type completeRequest struct {
	Path   string `json:"path"`
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

func newAIHandler(workspaces WorkspaceService, completions AICompletionService) *aiHandler {
	return &aiHandler{
		completionService: completions,
		workspaceService:  workspaces,
	}
}

func (h *aiHandler) complete(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	var request completeRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not supported", http.StatusInternalServerError)
		return
	}

	started := false
	emit := func(token string) error {
		if !started {
			startSSE(w)
			started = true
		}
		if err := writeSSEToken(w, token); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	err := h.completionService.StreamCompletion(r.Context(), ai.CompletionParams{
		Path:        request.Path,
		Prefix:      request.Prefix,
		Suffix:      request.Suffix,
		WorkspaceID: workspaceID,
	}, emit)
	if err == nil {
		return
	}
	if !started {
		status := http.StatusInternalServerError
		if isInvalidCompletionRequest(err) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}
	_ = writeSSEError(w, err)
	flusher.Flush()
}

func (h *aiHandler) authorizeRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	tenantID, _, actorOK := requestActor(r)
	if !actorOK {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return "", false
	}

	workspaceID := chi.URLParam(r, "id")
	if _, err := h.workspaceService.GetWorkspace(r.Context(), tenantID, workspaceID); err != nil {
		writeWorkspaceError(w, err)
		return "", false
	}

	return workspaceID, true
}

func isInvalidCompletionRequest(err error) bool {
	return errors.Is(err, ai.ErrInvalidRequest)
}

func startSSE(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
}

func writeSSEToken(w http.ResponseWriter, token string) error {
	return writeSSEData(w, map[string]string{"token": token})
}

func writeSSEError(w http.ResponseWriter, err error) error {
	_, writeErr := fmt.Fprintf(w, "event: error\ndata: ")
	if writeErr != nil {
		return writeErr
	}
	if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
		return err
	}
	_, writeErr = fmt.Fprint(w, "\n")
	return writeErr
}

func writeSSEData(w http.ResponseWriter, payload interface{}) error {
	if _, err := fmt.Fprint(w, "data: "); err != nil {
		return err
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return err
	}
	_, err := fmt.Fprint(w, "\n")
	return err
}
