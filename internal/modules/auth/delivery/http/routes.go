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
	myEarningsHandler http.HandlerFunc,
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
		if myEarningsHandler != nil {
			r.Get("/my-earnings", myEarningsHandler)
		}
		r.With(ownerOnlyMiddleware).Get("/", sellerHandler.ListSellers)
		r.With(ownerOnlyMiddleware).Post("/", sellerHandler.CreateSeller)
		r.With(ownerOnlyMiddleware).Patch("/{id}/commission", sellerHandler.UpdateCommission)
	})
}
