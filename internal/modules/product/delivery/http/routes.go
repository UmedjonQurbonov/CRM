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
	r.Group(func(pr chi.Router) {
		pr.Use(authMiddleware)

		// Accessible to both Owner and Seller
		pr.Get("/products", handler.ListProducts)
		pr.Get("/products/", handler.ListProducts)
		pr.Get("/products/by-qr/{code}", handler.GetByQRCode)

		// Accessible exclusively to Owner
		pr.Group(func(ownerRouter chi.Router) {
			ownerRouter.Use(ownerOnlyMiddleware)
			ownerRouter.Post("/products", handler.CreateProduct)
			ownerRouter.Post("/products/", handler.CreateProduct)
			ownerRouter.Post("/products/upload-image", handler.UploadImage)
			ownerRouter.Put("/products/{id}", handler.UpdateProduct)
			ownerRouter.Post("/products/{id}/restock", handler.RestockProduct)
			ownerRouter.Post("/products/{id}/image", handler.UploadProductImage)
			ownerRouter.Get("/products/{id}/qr-image", handler.GenerateQRCodePNG)
		})
	})
}
