package api

import (
	"errors"
	"net/http"

	"github.com/your-org/cortado/control-plane/internal/tenant"
)

type tenantAuthProviderHandler struct {
	service TenantAuthProviderService
}

type putTenantAuthProviderRequest struct {
	AllowedAudiences         []string                  `json:"allowedAudiences"`
	AllowedSigningAlgorithms []string                  `json:"allowedSigningAlgorithms"`
	ClaimRequirements        []tenant.ClaimRequirement `json:"claimRequirements,omitempty"`
	DiscoveryURL             string                    `json:"discoveryUrl,omitempty"`
	Issuer                   string                    `json:"issuer,omitempty"`
	JWKSURI                  string                    `json:"jwksUri,omitempty"`
	UserIDClaim              string                    `json:"userIdClaim,omitempty"`
}

type tenantAuthProviderEnvelope struct {
	AuthProvider tenant.AuthProviderConfig `json:"authProvider"`
}

func newTenantAuthProviderHandler(service TenantAuthProviderService) *tenantAuthProviderHandler {
	return &tenantAuthProviderHandler{service: service}
}

func (h *tenantAuthProviderHandler) get(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requestUserActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	config, err := h.service.GetAuthProvider(r.Context(), tenantID)
	if err != nil {
		writeTenantAuthProviderError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tenantAuthProviderEnvelope{AuthProvider: config})
}

func (h *tenantAuthProviderHandler) put(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requestUserActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	var request putTenantAuthProviderRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config, created, err := h.service.PutAuthProvider(r.Context(), tenantID, tenant.UpsertAuthProviderInput{
		AllowedAudiences:         request.AllowedAudiences,
		AllowedSigningAlgorithms: request.AllowedSigningAlgorithms,
		ClaimRequirements:        request.ClaimRequirements,
		DiscoveryURL:             request.DiscoveryURL,
		Issuer:                   request.Issuer,
		JWKSURI:                  request.JWKSURI,
		UserIDClaim:              request.UserIDClaim,
	})
	if err != nil {
		writeTenantAuthProviderError(w, err)
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, tenantAuthProviderEnvelope{AuthProvider: config})
}

func (h *tenantAuthProviderHandler) delete(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requestUserActor(r)
	if !ok {
		http.Error(w, "missing tenant context", http.StatusUnauthorized)
		return
	}

	if err := h.service.DeleteAuthProvider(r.Context(), tenantID); err != nil {
		writeTenantAuthProviderError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeTenantAuthProviderError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, tenant.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, tenant.ErrInvalidRequest):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
