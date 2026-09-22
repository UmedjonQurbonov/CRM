
package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all order endpoints onto the Chi router.
func RegisterRoutes(
	r chi.Router,
	handler *OrderHandler,
	authMiddleware func(http.Handler) http.Handler,
	ownerOnlyMiddleware func(http.Handler) http.Handler,
) {
	r.Route("/orders", func(r chi.Router) {
		r.Use(authMiddleware)

		// Accessible to both Owner and Seller
		r.Post("/", handler.Checkout)
		r.Get("/", handler.ListOrders)
		r.Get("/{id}", handler.GetByID)

		// Accessible exclusively to Owner
		r.With(ownerOnlyMiddleware).Post("/{id}/refund", handler.Refund)
	})
}
