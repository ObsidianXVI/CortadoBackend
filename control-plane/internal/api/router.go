package api

import (
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"github.com/your-org/cortado/control-plane/internal/gateway"
	cpmiddleware "github.com/your-org/cortado/control-plane/internal/middleware"
)

type RouterConfig struct {
	ConnectHandler http.Handler
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
		r.Method(http.MethodGet, "/workspaces/{id}/connect", connectHandler)
	})

	return router
}
