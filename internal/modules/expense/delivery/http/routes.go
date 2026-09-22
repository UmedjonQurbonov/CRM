package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all expense endpoints onto the Chi router.
func RegisterRoutes(
	r chi.Router,
	handler *ExpenseHandler,
	authMiddleware func(http.Handler) http.Handler,
	ownerOnlyMiddleware func(http.Handler) http.Handler,
) {
	r.Route("/expenses", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(ownerOnlyMiddleware)

		r.Post("/", handler.CreateExpense)
		r.Get("/", handler.ListExpenses)
		r.Delete("/{id}", handler.DeleteExpense)
	})
}
