package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/UmedjonQurbonov/CRM/internal/delivery/http/response"
	"github.com/UmedjonQurbonov/CRM/internal/middleware"
	"github.com/UmedjonQurbonov/CRM/internal/modules/order/domain"
	"github.com/UmedjonQurbonov/CRM/internal/modules/order/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type OrderHandler struct {
	orderUsecase *usecase.OrderUsecase
}

func NewOrderHandler(orderUsecase *usecase.OrderUsecase) *OrderHandler {
	return &OrderHandler{orderUsecase: orderUsecase}
}

// OrderListResponse wraps a paginated order list.
type OrderListResponse struct {
	Items  []*usecase.OrderResponseDTO `json:"items"`
	Total  int                         `json:"total" example:"50"`
	Limit  int                         `json:"limit" example:"20"`
	Offset int                         `json:"offset" example:"0"`
}

// Checkout godoc
// @Summary Process Order Checkout
// @Description Creates sales receipt with atomic stock decrement and commission snapshot.
// @Tags Orders
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body usecase.CheckoutRequestDTO true "Checkout details"
// @Success 201 {object} usecase.OrderResponseDTO
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/orders [post]
func (h *OrderHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "User identity missing", nil)
		return
	}
	userRole, _ := middleware.GetUserRole(r.Context())

	var req usecase.CheckoutRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	order, err := h.orderUsecase.Checkout(r.Context(), userID, userRole, req)
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientStock) {
			response.Error(w, http.StatusConflict, "INSUFFICIENT_STOCK", "One or more products have insufficient stock", nil)
			return
		}
		if errors.Is(err, domain.ErrEmptyCart) || errors.Is(err, domain.ErrInvalidPaymentMethod) || errors.Is(err, domain.ErrValidation) {
			response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Checkout transaction failed", nil)
		return
	}

	response.JSON(w, http.StatusCreated, order)
}

// ListOrders godoc
// @Summary List Sales Orders
// @Description Browse sales orders with optional filters. Sellers only see their own receipts.
// @Tags Orders
// @Security BearerAuth
// @Produce json
// @Param seller_id query string false "Filter by seller UUID (Owner only)"
// @Param status query string false "Filter by status: completed or refunded"
// @Param start_date query string false "Filter from date (YYYY-MM-DD)"
// @Param end_date query string false "Filter to date (YYYY-MM-DD)"
// @Param limit query int false "Pagination limit (max 100)" default(20)
// @Param offset query int false "Pagination offset" default(0)
// @Success 200 {object} OrderListResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/orders [get]
func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "User identity missing", nil)
		return
	}
	userRole, _ := middleware.GetUserRole(r.Context())

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	filter := usecase.OrderFilter{
		Status: r.URL.Query().Get("status"),
		Limit:  limit,
		Offset: offset,
	}

	if sellerParam := r.URL.Query().Get("seller_id"); sellerParam != "" {
		if sid, err := uuid.Parse(sellerParam); err == nil {
			filter.SellerID = &sid
		}
	}

	if startParam := r.URL.Query().Get("start_date"); startParam != "" {
		if t, err := time.Parse("2006-01-02", startParam); err == nil {
			filter.StartDate = &t
		}
	}

	if endParam := r.URL.Query().Get("end_date"); endParam != "" {
		if t, err := time.Parse("2006-01-02", endParam); err == nil {
			endOfDay := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			filter.EndDate = &endOfDay
		}
	}

	items, total, err := h.orderUsecase.ListOrders(r.Context(), filter, userID, userRole)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve orders", nil)
		return
	}

	response.JSON(w, http.StatusOK, OrderListResponse{
		Items:  items,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
}

// GetByID godoc
// @Summary Get Order by ID
// @Description Retrieve receipt details with items. Sellers can only view their own receipts.
// @Tags Orders
// @Security BearerAuth
// @Produce json
// @Param id path string true "Order UUID"
// @Success 200 {object} usecase.OrderResponseDTO
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/orders/{id} [get]
func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "User identity missing", nil)
		return
	}
	userRole, _ := middleware.GetUserRole(r.Context())

	idParam := chi.URLParam(r, "id")
	orderID, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid order UUID format", nil)
		return
	}

	order, err := h.orderUsecase.GetOrderByID(r.Context(), orderID, userID, userRole)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "Order not found", nil)
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "You do not have access to view this order", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get order", nil)
		return
	}

	response.JSON(w, http.StatusOK, order)
}

// Refund godoc
// @Summary Refund Order
// @Description Issues a full refund, restocks inventory, and sets order status to refunded (Owner only).
// @Tags Orders
// @Security BearerAuth
// @Produce json
// @Param id path string true "Order UUID"
// @Success 200 {object} usecase.OrderResponseDTO
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/orders/{id}/refund [post]
func (h *OrderHandler) Refund(w http.ResponseWriter, r *http.Request) {
	userRole, _ := middleware.GetUserRole(r.Context())

	idParam := chi.URLParam(r, "id")
	orderID, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid order UUID format", nil)
		return
	}

	order, err := h.orderUsecase.RefundOrder(r.Context(), orderID, userRole)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "Order not found", nil)
			return
		}
		if errors.Is(err, domain.ErrOrderAlreadyRefunded) {
			response.Error(w, http.StatusConflict, "ALREADY_REFUNDED", "Order has already been refunded", nil)
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Only the owner can refund orders", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to refund order", nil)
		return
	}

	response.JSON(w, http.StatusOK, order)
}
