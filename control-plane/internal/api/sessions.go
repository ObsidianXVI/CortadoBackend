package api

import (
	"errors"
	"net/http"
	"strings"

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

type exchangeFirebaseSessionRequest struct {
	FirebaseIDToken string `json:"firebase_id_token"`
	IDToken         string `json:"id_token,omitempty"`
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

func (h *sessionsHandler) exchangeFirebase(w http.ResponseWriter, r *http.Request) {
	var request exchangeFirebaseSessionRequest
	if err := decodeJSON(r, &request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tokens, err := h.service.ExchangeFirebaseSession(r.Context(), request.token())
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

func (r exchangeFirebaseSessionRequest) token() string {
	if token := strings.TrimSpace(r.FirebaseIDToken); token != "" {
		return token
	}
	return strings.TrimSpace(r.IDToken)
}

func writeSessionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	case errors.Is(err, auth.ErrFirebaseTokenInvalid):
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	case errors.Is(err, auth.ErrInvalidRefreshToken):
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	case errors.Is(err, auth.ErrFirebaseTokenMissing):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, auth.ErrPlatformUserID), errors.Is(err, auth.ErrUserIDRequired):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, auth.ErrInvalidRequest), errors.Is(err, auth.ErrInvalidRefreshInput):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
