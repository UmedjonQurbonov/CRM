package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers all product endpoints under the specified Chi router.
func RegisterRoutes(
	r chi.Router,
	handler *ProductHandler,
	authMiddleware func(http.Handler) http.Handler,
	ownerOnlyMiddleware func(http.Handler) http.Handler,
) {
	r.Route("/products", func(r chi.Router) {
		r.Use(authMiddleware)

		// Accessible to both Owner and Seller
		r.Get("/", handler.ListProducts)
		r.Get("/by-qr/{code}", handler.GetByQRCode)

		// Accessible exclusively to Owner
		r.Group(func(ownerRouter chi.Router) {
			ownerRouter.Use(ownerOnlyMiddleware)
			ownerRouter.Post("/", handler.CreateProduct)
			ownerRouter.Put("/{id}", handler.UpdateProduct)
			ownerRouter.Get("/{id}/qr-image", handler.GenerateQRCodePNG)
		})
	})
}
