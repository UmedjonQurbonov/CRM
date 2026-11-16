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
	categoryHandler ...*ExpenseCategoryHandler,
) {
	r.Route("/expenses", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(ownerOnlyMiddleware)

		r.Post("/", handler.CreateExpense)
		r.Get("/", handler.ListExpenses)
		r.Delete("/{id}", handler.DeleteExpense)
	})

	if len(categoryHandler) > 0 && categoryHandler[0] != nil {
		r.Route("/expense-categories", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(ownerOnlyMiddleware)

			r.Get("/", categoryHandler[0].ListCategories)
			r.Post("/", categoryHandler[0].CreateCategory)
			r.Delete("/{id}", categoryHandler[0].DeleteCategory)
		})
	}
}
