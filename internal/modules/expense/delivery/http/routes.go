package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all expense and category endpoints onto the Chi router.
func RegisterRoutes(
	r chi.Router,
	handler *ExpenseHandler,
	authMiddleware func(http.Handler) http.Handler,
	ownerOnlyMiddleware func(http.Handler) http.Handler,
	categoryHandler ...*ExpenseCategoryHandler,
) {
	r.Group(func(er chi.Router) {
		er.Use(authMiddleware)
		er.Use(ownerOnlyMiddleware)

		er.Get("/expenses", handler.ListExpenses)
		er.Get("/expenses/", handler.ListExpenses)
		er.Post("/expenses", handler.CreateExpense)
		er.Post("/expenses/", handler.CreateExpense)
		er.Delete("/expenses/{id}", handler.DeleteExpense)
	})

	if len(categoryHandler) > 0 && categoryHandler[0] != nil {
		catH := categoryHandler[0]
		r.Group(func(cr chi.Router) {
			cr.Use(authMiddleware)
			cr.Use(ownerOnlyMiddleware)

			cr.Get("/expense-categories", catH.ListCategories)
			cr.Get("/expense-categories/", catH.ListCategories)
			cr.Post("/expense-categories", catH.CreateCategory)
			cr.Post("/expense-categories/", catH.CreateCategory)
			cr.Delete("/expense-categories/{id}", catH.DeleteCategory)
		})
	}
}
