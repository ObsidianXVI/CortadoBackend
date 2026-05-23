package api

import (
	"errors"
	"net/http"

	"github.com/your-org/cortado/control-plane/internal/auth"
)

type sessionsHandler struct {
	service SessionService
}

type createSessionRequest struct {
	APIKey string `json:"api_key"`
	UserID string `json:"user_id"`
}

type createSessionResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type refreshSessionRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshSessionResponse struct {
	AccessToken string `json:"access_token"`
}

func newSessionsHandler(service SessionService) *sessionsHandler {
	return &sessionsHandler{service: service}
}

func (h *sessionsHandler) create(w http.ResponseWriter, r *http.Request) {
	var request createSessionRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tokens, err := h.service.CreateSession(r.Context(), request.APIKey, request.UserID)
	if err != nil {
		writeSessionError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, createSessionResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *sessionsHandler) refresh(w http.ResponseWriter, r *http.Request) {
	var request refreshSessionRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	accessToken, err := h.service.RefreshSession(r.Context(), request.RefreshToken)
	if err != nil {
		writeSessionError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, refreshSessionResponse{
		AccessToken: accessToken,
	})
}

func writeSessionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	case errors.Is(err, auth.ErrInvalidRefreshToken):
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	case errors.Is(err, auth.ErrInvalidRequest), errors.Is(err, auth.ErrInvalidRefreshInput):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
