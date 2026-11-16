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
	r.Group(func(or chi.Router) {
		or.Use(authMiddleware)

		// Accessible to both Owner and Seller
		or.Post("/orders", handler.Checkout)
		or.Post("/orders/", handler.Checkout)
		or.Get("/orders", handler.ListOrders)
		or.Get("/orders/", handler.ListOrders)
		or.Get("/orders/{id}", handler.GetByID)

		// Accessible exclusively to Owner
		or.With(ownerOnlyMiddleware).Post("/orders/{id}/refund", handler.Refund)
	})
}
