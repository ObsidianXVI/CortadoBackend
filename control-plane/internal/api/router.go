package api

import (
	"net/http"

	chi "github.com/go-chi/chi/v5"
	cpmiddleware "github.com/your-org/cortado/control-plane/internal/middleware"
)

func NewRouter() http.Handler {
	router := chi.NewRouter()

	router.Get("/health", healthHandler)

	router.Route("/v1", func(r chi.Router) {
		r.Use(cpmiddleware.DevBypassAuth)
	})

	return router
}
