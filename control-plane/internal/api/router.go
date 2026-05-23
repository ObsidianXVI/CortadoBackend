package api

import (
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"github.com/your-org/cortado/control-plane/internal/gateway"
	cpmiddleware "github.com/your-org/cortado/control-plane/internal/middleware"
)

type RouterConfig struct {
	ConnectHandler http.Handler
	WorkspaceSvc   WorkspaceService
}

func NewRouter(cfg RouterConfig) http.Handler {
	connectHandler := cfg.ConnectHandler
	if connectHandler == nil {
		connectHandler = gateway.NewConnectHandler(gateway.ConnectHandlerConfig{})
	}

	router := chi.NewRouter()

	router.Get("/health", healthHandler)

	router.Route("/v1", func(r chi.Router) {
		r.Use(cpmiddleware.DevBypassAuth)
		if cfg.WorkspaceSvc != nil {
			handler := newWorkspacesHandler(cfg.WorkspaceSvc)
			r.Get("/workspaces", handler.list)
			r.Post("/workspaces", handler.create)
			r.Get("/workspaces/{id}", handler.get)
			r.Post("/workspaces/{id}/start", handler.start)
			r.Post("/workspaces/{id}/stop", handler.stop)
			r.Delete("/workspaces/{id}", handler.delete)
		}
		r.Method(http.MethodGet, "/workspaces/{id}/connect", connectHandler)
	})

	return router
}
