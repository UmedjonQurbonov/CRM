package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/UmedjonQurbonov/CRM/internal/delivery/http/response"
	"github.com/UmedjonQurbonov/CRM/internal/middleware"
	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/domain"
	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/usecase"
)

// ExpenseCategoryHandler handles expense category HTTP requests.
type ExpenseCategoryHandler struct {
	categoryUsecase *usecase.ExpenseCategoryUsecase
}

// NewExpenseCategoryHandler initializes the expense category handler.
func NewExpenseCategoryHandler(categoryUsecase *usecase.ExpenseCategoryUsecase) *ExpenseCategoryHandler {
	return &ExpenseCategoryHandler{categoryUsecase: categoryUsecase}
}

// CreateCategoryRequest defines the JSON body for creating a category.
type CreateCategoryRequest struct {
	Name string `json:"name" example:"Оборудование"`
}

// ListCategories godoc
// @Summary List Expense Categories
// @Description Retrieve all user-managed expense categories. Owner only.
// @Tags Expense Categories
// @Security BearerAuth
// @Produce json
// @Success 200 {array} usecase.ExpenseCategoryResponseDTO
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/expense-categories [get]
func (h *ExpenseCategoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := middleware.GetUserRole(r.Context())

	categories, err := h.categoryUsecase.ListCategories(r.Context(), callerRole)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list categories", nil)
		return
	}

	response.JSON(w, http.StatusOK, categories)
}

// CreateCategory godoc
// @Summary Create Expense Category
// @Description Create a new user-managed expense category. Owner only.
// @Tags Expense Categories
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateCategoryRequest true "Category payload"
// @Success 201 {object} usecase.ExpenseCategoryResponseDTO
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/expense-categories [post]
func (h *ExpenseCategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := middleware.GetUserRole(r.Context())

	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON payload", nil)
		return
	}

	cat, err := h.categoryUsecase.CreateCategory(r.Context(), callerRole, req.Name)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrEmptyCategory) {
			response.Error(w, http.StatusBadRequest, "EMPTY_CATEGORY", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrDuplicateCategory) {
			response.Error(w, http.StatusConflict, "DUPLICATE_CATEGORY", err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create category", nil)
		return
	}

	response.JSON(w, http.StatusCreated, cat)
}

// DeleteCategory godoc
// @Summary Delete Expense Category
// @Description Remove an expense category by UUID. Owner only.
// @Tags Expense Categories
// @Security BearerAuth
// @Param id path string true "Category UUID"
// @Success 204 "No Content"
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/expense-categories/{id} [delete]
func (h *ExpenseCategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := middleware.GetUserRole(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid category ID format", nil)
		return
	}

	if err := h.categoryUsecase.DeleteCategory(r.Context(), callerRole, id); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrCategoryNotFound) {
			response.Error(w, http.StatusNotFound, "CATEGORY_NOT_FOUND", "Category not found", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete category", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
