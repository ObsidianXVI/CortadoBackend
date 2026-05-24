package portforward

import (
	"net/http"

	chi "github.com/go-chi/chi/v5"
	cpmiddleware "github.com/your-org/cortado/control-plane/internal/middleware"
)

type JWKSProvider interface {
	JWKS() []byte
}

type RouterConfig struct {
	Handler      http.Handler
	JWKSProvider JWKSProvider
}

func NewRouter(cfg RouterConfig) http.Handler {
	router := chi.NewRouter()
	router.Get("/health", healthHandler)

	protected := chi.NewRouter()
	protected.Use(cpmiddleware.NewAuthMiddleware(cpmiddleware.AuthConfig{
		JWKSJSON: jwksJSON(cfg.JWKSProvider),
	}))
	protected.Handle("/*", cfg.Handler)

	router.Mount("/", protected)
	return router
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func jwksJSON(provider JWKSProvider) []byte {
	if provider == nil {
		return nil
	}
	return provider.JWKS()
}
