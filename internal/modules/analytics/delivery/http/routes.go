package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all analytics endpoints onto the Chi router.
func RegisterRoutes(
	r chi.Router,
	handler *AnalyticsHandler,
	authMiddleware func(http.Handler) http.Handler,
	ownerOnlyMiddleware func(http.Handler) http.Handler,
) {
	r.Route("/analytics", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(ownerOnlyMiddleware)

		r.Get("/summary", handler.GetSummary)
		r.Get("/sellers-ranking", handler.GetSellersRanking)
		r.Get("/top-products", handler.GetTopProducts)
	})
}
