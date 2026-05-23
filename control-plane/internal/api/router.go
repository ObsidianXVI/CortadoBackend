package api

import (
	"context"
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"github.com/your-org/cortado/control-plane/internal/auth"
	"github.com/your-org/cortado/control-plane/internal/gateway"
	cpmiddleware "github.com/your-org/cortado/control-plane/internal/middleware"
)

type RouterConfig struct {
	ConnectHandler http.Handler
	JWKSProvider   JWKSProvider
	SessionSvc     SessionService
	WorkspaceSvc   WorkspaceService
}

type SessionService interface {
	CreateSession(ctx context.Context, apiKey, userID string) (auth.SessionTokens, error)
}

type JWKSProvider interface {
	JWKS() []byte
}

func NewRouter(cfg RouterConfig) http.Handler {
	connectHandler := cfg.ConnectHandler
	if connectHandler == nil {
		connectHandler = gateway.NewConnectHandler(gateway.ConnectHandlerConfig{})
	}
	var jwksJSON []byte
	if cfg.JWKSProvider != nil {
		jwksJSON = cfg.JWKSProvider.JWKS()
	}

	router := chi.NewRouter()

	router.Get("/health", healthHandler)
	if cfg.JWKSProvider != nil {
		router.Get("/.well-known/jwks.json", newJWKSHandler(cfg.JWKSProvider).get)
	}

	router.Route("/v1", func(r chi.Router) {
		if cfg.SessionSvc != nil {
			r.Post("/sessions", newSessionsHandler(cfg.SessionSvc).create)
		}
		r.Group(func(protected chi.Router) {
			protected.Use(cpmiddleware.NewAuthMiddleware(cpmiddleware.AuthConfig{JWKSJSON: jwksJSON}))
			if cfg.WorkspaceSvc != nil {
				handler := newWorkspacesHandler(cfg.WorkspaceSvc)
				protected.Get("/workspaces", handler.list)
				protected.Post("/workspaces", handler.create)
				protected.Get("/workspaces/{id}", handler.get)
				protected.Post("/workspaces/{id}/start", handler.start)
				protected.Post("/workspaces/{id}/stop", handler.stop)
				protected.Delete("/workspaces/{id}", handler.delete)
			}
			protected.Method(http.MethodGet, "/workspaces/{id}/connect", connectHandler)
		})
	})

	return router
}
