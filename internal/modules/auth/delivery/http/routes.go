package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all auth and seller endpoints into the provided Chi router.
func RegisterRoutes(
	r chi.Router,
	authHandler *AuthHandler,
	sellerHandler *SellerHandler,
	authMiddleware func(http.Handler) http.Handler,
	ownerOnlyMiddleware func(http.Handler) http.Handler,
) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.With(authMiddleware).Post("/logout", authHandler.Logout)
	})

	r.Route("/sellers", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(ownerOnlyMiddleware)
		r.Get("/", sellerHandler.ListSellers)
		r.Post("/", sellerHandler.CreateSeller)
		r.Patch("/{id}/commission", sellerHandler.UpdateCommission)
	})
}
