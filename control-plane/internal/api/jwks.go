package api

import "net/http"

type jwksHandler struct {
	provider JWKSProvider
}

func newJWKSHandler(provider JWKSProvider) *jwksHandler {
	return &jwksHandler{provider: provider}
}

func (h *jwksHandler) get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(h.provider.JWKS())
}
