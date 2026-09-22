package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/UmedjonQurbonov/CRM/internal/delivery/http/response"
	"github.com/UmedjonQurbonov/CRM/internal/middleware"
	"github.com/UmedjonQurbonov/CRM/internal/modules/analytics/domain"
	"github.com/UmedjonQurbonov/CRM/internal/modules/analytics/usecase"
)

// AnalyticsHandler handles HTTP endpoints for reporting and seller earnings.
type AnalyticsHandler struct {
	analyticsUsecase *usecase.AnalyticsUsecase
}

// NewAnalyticsHandler creates a new AnalyticsHandler.
func NewAnalyticsHandler(analyticsUsecase *usecase.AnalyticsUsecase) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsUsecase: analyticsUsecase}
}

// GetSummary godoc
// @Summary Financial P&L Statement
// @Description Calculates Revenue, Cost of Goods Sold, Gross Profit, Expenses, Commissions, and Net Profit for a period. Accessible exclusively to Owner.
// @Tags Analytics
// @Security BearerAuth
// @Produce json
// @Param from query string false "Filter start date (YYYY-MM-DD)"
// @Param to query string false "Filter end date (YYYY-MM-DD)"
// @Success 200 {object} usecase.AnalyticsSummaryDTO
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/analytics/summary [get]
func (h *AnalyticsHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := middleware.GetUserRole(r.Context())

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	summary, err := h.analyticsUsecase.GetSummary(r.Context(), callerRole, from, to)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrInvalidDateRange) {
			response.Error(w, http.StatusBadRequest, "INVALID_DATE_RANGE", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrValidation) {
			response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid date format. Expected YYYY-MM-DD", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to calculate P&L summary", nil)
		return
	}

	response.JSON(w, http.StatusOK, summary)
}

// GetSellersRanking godoc
// @Summary Sellers Ranking
// @Description Ranks sellers by completed sales volume, commission earned, and revenue share percentage for a period. Accessible exclusively to Owner.
// @Tags Analytics
// @Security BearerAuth
// @Produce json
// @Param from query string false "Filter start date (YYYY-MM-DD)"
// @Param to query string false "Filter end date (YYYY-MM-DD)"
// @Param limit query int false "Top N sellers" default(10)
// @Success 200 {array} usecase.SellerRankingDTO
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/analytics/sellers-ranking [get]
func (h *AnalyticsHandler) GetSellersRanking(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := middleware.GetUserRole(r.Context())

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	rankings, err := h.analyticsUsecase.GetSellersRanking(r.Context(), callerRole, from, to, limit)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrInvalidDateRange) {
			response.Error(w, http.StatusBadRequest, "INVALID_DATE_RANGE", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrValidation) {
			response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid date format. Expected YYYY-MM-DD", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve sellers ranking", nil)
		return
	}

	response.JSON(w, http.StatusOK, rankings)
}

// GetTopProducts godoc
// @Summary Top Products by Revenue
// @Description Ranks catalog products by sales volume and total revenue generated for a period. Accessible exclusively to Owner.
// @Tags Analytics
// @Security BearerAuth
// @Produce json
// @Param from query string false "Filter start date (YYYY-MM-DD)"
// @Param to query string false "Filter end date (YYYY-MM-DD)"
// @Param limit query int false "Top N products" default(10)
// @Success 200 {array} usecase.TopProductDTO
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/analytics/top-products [get]
func (h *AnalyticsHandler) GetTopProducts(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := middleware.GetUserRole(r.Context())

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	topProds, err := h.analyticsUsecase.GetTopProducts(r.Context(), callerRole, from, to, limit)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrInvalidDateRange) {
			response.Error(w, http.StatusBadRequest, "INVALID_DATE_RANGE", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrValidation) {
			response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid date format. Expected YYYY-MM-DD", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve top products", nil)
		return
	}

	response.JSON(w, http.StatusOK, topProds)
}

// GetMyEarnings godoc
// @Summary Seller Personal Earnings
// @Description Returns the calling seller's total sales volume, commission rate, and accumulated commissions for a period. Accessible to Sellers and Owners.
// @Tags Sellers
// @Security BearerAuth
// @Produce json
// @Param from query string false "Filter start date (YYYY-MM-DD)"
// @Param to query string false "Filter end date (YYYY-MM-DD)"
// @Success 200 {object} usecase.SellerEarningsDTO
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/sellers/my-earnings [get]
func (h *AnalyticsHandler) GetMyEarnings(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "User identity missing", nil)
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	earnings, err := h.analyticsUsecase.GetMyEarnings(r.Context(), callerID, from, to)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidDateRange) {
			response.Error(w, http.StatusBadRequest, "INVALID_DATE_RANGE", err.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrValidation) {
			response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid date format. Expected YYYY-MM-DD", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve personal earnings", nil)
		return
	}

	response.JSON(w, http.StatusOK, earnings)
}
