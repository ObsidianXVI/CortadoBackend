package api

import (
	"errors"
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"github.com/your-org/cortado/control-plane/internal/auth"
)

type platformTenantsHandler struct {
	service PlatformTenantService
}

type createPlatformTenantRequest struct {
	DisplayName string `json:"displayName,omitempty"`
}

type platformTenantEnvelope struct {
	Tenant auth.PlatformTenant `json:"tenant"`
}

type platformTenantListEnvelope struct {
	Tenants []auth.PlatformTenant `json:"tenants"`
}

func newPlatformTenantsHandler(service PlatformTenantService) *platformTenantsHandler {
	return &platformTenantsHandler{service: service}
}

func (h *platformTenantsHandler) createTenant(w http.ResponseWriter, r *http.Request) {
	_, userID, ok := requestUserActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	var request createPlatformTenantRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tenant, err := h.service.CreatePlatformTenant(r.Context(), userID, request.DisplayName)
	if err != nil {
		writePlatformTenantError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, platformTenantEnvelope{Tenant: tenant})
}

func (h *platformTenantsHandler) listTenants(w http.ResponseWriter, r *http.Request) {
	_, userID, ok := requestUserActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	tenants, err := h.service.ListPlatformTenants(r.Context(), userID)
	if err != nil {
		writePlatformTenantError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, platformTenantListEnvelope{Tenants: tenants})
}

func (h *platformTenantsHandler) issueAPIKey(w http.ResponseWriter, r *http.Request) {
	_, userID, ok := requestUserActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	issued, err := h.service.IssuePlatformAPIKey(
		r.Context(),
		userID,
		chi.URLParam(r, "id"),
	)
	if err != nil {
		writePlatformTenantError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, issueAPIKeyResponse{
		APIKey: issued.APIKey,
		Record: issued.Record,
	})
}

func (h *platformTenantsHandler) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	_, userID, ok := requestUserActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	keys, err := h.service.ListPlatformAPIKeys(
		r.Context(),
		userID,
		chi.URLParam(r, "id"),
	)
	if err != nil {
		writePlatformTenantError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, listAPIKeysResponse{APIKeys: keys})
}

func (h *platformTenantsHandler) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	_, userID, ok := requestUserActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	record, err := h.service.RevokePlatformAPIKey(
		r.Context(),
		userID,
		chi.URLParam(r, "id"),
		chi.URLParam(r, "keyID"),
	)
	if err != nil {
		writePlatformTenantError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, revokeAPIKeyResponse{Record: record})
}

func writePlatformTenantError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrPlatformTenantNotFound), errors.Is(err, auth.ErrAPIKeyNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, auth.ErrInvalidRequest):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
