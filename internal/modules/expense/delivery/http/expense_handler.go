package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/UmedjonQurbonov/CRM/internal/delivery/http/response"
	"github.com/UmedjonQurbonov/CRM/internal/middleware"
	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/domain"
	"github.com/UmedjonQurbonov/CRM/internal/modules/expense/usecase"
)

type ExpenseHandler struct {
	expenseUsecase *usecase.ExpenseUsecase
}

func NewExpenseHandler(expenseUsecase *usecase.ExpenseUsecase) *ExpenseHandler {
	return &ExpenseHandler{expenseUsecase: expenseUsecase}
}

// CreateExpenseRequest defines the JSON body for creating an expense.
type CreateExpenseRequest struct {
	Category    string  `json:"category" example:"utilities"`
	Amount      string  `json:"amount" example:"450.00"`
	Comment     *string `json:"comment,omitempty" example:"Monthly electricity bill for store"`
	ExpenseDate string  `json:"expense_date,omitempty" example:"2026-09-22"`
}

// ExpenseListResponse wraps paginated expense items.
type ExpenseListResponse struct {
	Items  []usecase.ExpenseResponseDTO `json:"items"`
	Total  int                          `json:"total" example:"25"`
	Limit  int                          `json:"limit" example:"20"`
	Offset int                          `json:"offset" example:"0"`
}

// CreateExpense godoc
// @Summary Create Operational Expense
// @Description Record a store operational expense (rent, utilities, salaries, marketing, etc.). Accessible exclusively to Owner.
// @Tags Expenses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateExpenseRequest true "Expense creation payload"
// @Success 201 {object} usecase.ExpenseResponseDTO
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/expenses [post]
func (h *ExpenseHandler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "User identity missing", nil)
		return
	}
	callerRole, _ := middleware.GetUserRole(r.Context())

	var req CreateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON payload", nil)
		return
	}

	dto := usecase.CreateExpenseDTO{
		Category:    req.Category,
		Amount:      req.Amount,
		Comment:     req.Comment,
		ExpenseDate: req.ExpenseDate,
	}

	created, err := h.expenseUsecase.CreateExpense(r.Context(), callerID, callerRole, dto)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrEmptyCategory) {
			response.Error(w, http.StatusBadRequest, "EMPTY_CATEGORY", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrInvalidExpenseAmount) {
			response.Error(w, http.StatusBadRequest, "INVALID_AMOUNT", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrValidation) {
			response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create expense", nil)
		return
	}

	response.JSON(w, http.StatusCreated, created)
}

// ListExpenses godoc
// @Summary List Operational Expenses
// @Description Retrieve operational expenses with filtering by category, date range, and pagination. Accessible exclusively to Owner.
// @Tags Expenses
// @Security BearerAuth
// @Produce json
// @Param category query string false "Filter by category substring"
// @Param from_date query string false "Filter start date (YYYY-MM-DD)"
// @Param to_date query string false "Filter end date (YYYY-MM-DD)"
// @Param limit query int false "Pagination limit (max 100)" default(20)
// @Param offset query int false "Pagination offset" default(0)
// @Success 200 {object} ExpenseListResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/expenses [get]
func (h *ExpenseHandler) ListExpenses(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := middleware.GetUserRole(r.Context())

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	filter := usecase.ExpenseFilter{
		Category: r.URL.Query().Get("category"),
		Limit:    limit,
		Offset:   offset,
	}

	if fromParam := r.URL.Query().Get("from_date"); fromParam != "" {
		if t, err := time.Parse("2006-01-02", fromParam); err == nil {
			filter.FromDate = &t
		}
	}

	if toParam := r.URL.Query().Get("to_date"); toParam != "" {
		if t, err := time.Parse("2006-01-02", toParam); err == nil {
			filter.ToDate = &t
		}
	}

	items, total, err := h.expenseUsecase.ListExpenses(r.Context(), callerRole, filter)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve expenses", nil)
		return
	}

	response.JSON(w, http.StatusOK, ExpenseListResponse{
		Items:  items,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
}

// DeleteExpense godoc
// @Summary Delete Operational Expense
// @Description Permanently delete an erroneously recorded expense by UUID. Accessible exclusively to Owner.
// @Tags Expenses
// @Security BearerAuth
// @Param id path string true "Expense UUID"
// @Success 204 "No Content"
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/expenses/{id} [delete]
func (h *ExpenseHandler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := middleware.GetUserRole(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid expense ID format", nil)
		return
	}

	if err := h.expenseUsecase.DeleteExpense(r.Context(), callerRole, id); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrExpenseNotFound) {
			response.Error(w, http.StatusNotFound, "EXPENSE_NOT_FOUND", "Expense record not found", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete expense", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
