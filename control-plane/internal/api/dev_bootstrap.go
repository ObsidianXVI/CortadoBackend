package api

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/your-org/cortado/control-plane/internal/auth"
	cpmiddleware "github.com/your-org/cortado/control-plane/internal/middleware"
)

type DevFirebaseBootstrapService interface {
	AssignTenantClaim(
		ctx context.Context,
		token *auth.VerifiedFirebaseToken,
		tenantID string,
	) (auth.DevTenantClaimAssignment, error)
}

type devBootstrapHandler struct {
	service DevFirebaseBootstrapService
}

type assignTenantClaimRequest struct {
	TenantID string `json:"tenantId"`
}

type assignTenantClaimResponse struct {
	Assignment auth.DevTenantClaimAssignment `json:"assignment"`
}

func newDevBootstrapHandler(
	service DevFirebaseBootstrapService,
) *devBootstrapHandler {
	return &devBootstrapHandler{service: service}
}

func (h *devBootstrapHandler) assignTenantClaim(
	w http.ResponseWriter,
	r *http.Request,
) {
	token, ok := cpmiddleware.FirebaseToken(r.Context())
	if !ok {
		http.Error(w, "missing firebase token context", http.StatusUnauthorized)
		return
	}

	var request assignTenantClaimRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := decodeJSON(r, &request); err != nil && !errors.Is(err, io.EOF) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	assignment, err := h.service.AssignTenantClaim(
		r.Context(),
		token,
		request.TenantID,
	)
	if err != nil {
		writeDevBootstrapError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, assignTenantClaimResponse{
		Assignment: assignment,
	})
}

func writeDevBootstrapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrDevBootstrapDisabled):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, auth.ErrInvalidRequest), errors.Is(err, auth.ErrFirebaseTokenInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
