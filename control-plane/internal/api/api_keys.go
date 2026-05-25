package api

import (
	"errors"
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"github.com/your-org/cortado/control-plane/internal/auth"
)

type apiKeysHandler struct {
	service APIKeyService
}

type issueAPIKeyResponse struct {
	APIKey string      `json:"apiKey"`
	Record auth.APIKey `json:"record"`
}

type listAPIKeysResponse struct {
	APIKeys []auth.APIKey `json:"apiKeys"`
}

type revokeAPIKeyResponse struct {
	Record auth.APIKey `json:"record"`
}

func newAPIKeysHandler(service APIKeyService) *apiKeysHandler {
	return &apiKeysHandler{service: service}
}

func (h *apiKeysHandler) issue(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requestActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	issued, err := h.service.IssueAPIKey(r.Context(), tenantID, userID)
	if err != nil {
		writeAPIKeyError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, issueAPIKeyResponse{
		APIKey: issued.APIKey,
		Record: issued.Record,
	})
}

func (h *apiKeysHandler) list(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requestActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	apiKeys, err := h.service.ListAPIKeys(r.Context(), tenantID, userID)
	if err != nil {
		writeAPIKeyError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, listAPIKeysResponse{APIKeys: apiKeys})
}

func (h *apiKeysHandler) revoke(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requestActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	record, err := h.service.RevokeAPIKey(r.Context(), tenantID, userID, chi.URLParam(r, "id"))
	if err != nil {
		writeAPIKeyError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, revokeAPIKeyResponse{Record: record})
}

func writeAPIKeyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrAPIKeyNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, auth.ErrInvalidRequest):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
