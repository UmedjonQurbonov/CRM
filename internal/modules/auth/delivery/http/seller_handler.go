package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/UmedjonQurbonov/CRM/internal/delivery/http/response"
	"github.com/UmedjonQurbonov/CRM/internal/modules/auth/domain"
	"github.com/UmedjonQurbonov/CRM/internal/modules/auth/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SellerHandler struct {
	sellerUsecase *usecase.SellerUsecase
}

func NewSellerHandler(sellerUsecase *usecase.SellerUsecase) *SellerHandler {
	return &SellerHandler{sellerUsecase: sellerUsecase}
}

// CreateSellerRequest defines input to create a new seller account.
type CreateSellerRequest struct {
	Name           string          `json:"name" example:"Umedjon Qurbonov"`
	Phone          string          `json:"phone" example:"+992901111111"`
	Password       string          `json:"password" example:"SellerPass123!"`
	CommissionRate decimal.Decimal `json:"commission_rate" example:"5.00"`
}

// UpdateCommissionRequest defines input for altering commission rate.
type UpdateCommissionRequest struct {
	CommissionRate decimal.Decimal `json:"commission_rate" example:"7.50"`
}

// ListSellers godoc
// @Summary List Sellers
// @Description Returns a list of all seller accounts and their commission rates (Owner only).
// @Tags Sellers
// @Security BearerAuth
// @Produce json
// @Success 200 {array} UserDTO
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/sellers [get]
func (h *SellerHandler) ListSellers(w http.ResponseWriter, r *http.Request) {
	sellers, err := h.sellerUsecase.ListSellers(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve sellers", nil)
		return
	}

	result := make([]UserDTO, 0, len(sellers))
	for _, s := range sellers {
		result = append(result, UserDTO{
			ID:             s.ID.String(),
			Name:           s.Name,
			Phone:          s.Phone,
			Role:           s.Role,
			CommissionRate: s.CommissionRate.StringFixed(2),
		})
	}

	response.JSON(w, http.StatusOK, result)
}

// CreateSeller godoc
// @Summary Create Seller Account
// @Description Creates a new seller user with custom commission percentage (Owner only).
// @Tags Sellers
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateSellerRequest true "Seller registration details"
// @Success 201 {object} UserDTO
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/sellers [post]
func (h *SellerHandler) CreateSeller(w http.ResponseWriter, r *http.Request) {
	var req CreateSellerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	seller, err := h.sellerUsecase.CreateSeller(r.Context(), usecase.CreateSellerDTO{
		Name:           req.Name,
		Phone:          req.Phone,
		Password:       req.Password,
		CommissionRate: req.CommissionRate,
	})
	if err != nil {
		if errors.Is(err, domain.ErrPhoneAlreadyExists) {
			response.Error(w, http.StatusConflict, "PHONE_ALREADY_EXISTS", "A user with this phone number already exists", nil)
			return
		}
		if errors.Is(err, domain.ErrValidation) || errors.Is(err, domain.ErrInvalidCommissionRate) {
			response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create seller", nil)
		return
	}

	response.JSON(w, http.StatusCreated, UserDTO{
		ID:             seller.ID.String(),
		Name:           seller.Name,
		Phone:          seller.Phone,
		Role:           seller.Role,
		CommissionRate: seller.CommissionRate.StringFixed(2),
	})
}

// UpdateCommission godoc
// @Summary Update Seller Commission
// @Description Updates the commission rate percentage for a specific seller (Owner only).
// @Tags Sellers
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Seller UUID"
// @Param request body UpdateCommissionRequest true "Updated commission percentage"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/sellers/{id}/commission [patch]
func (h *SellerHandler) UpdateCommission(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	sellerID, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid seller UUID format", nil)
		return
	}

	var req UpdateCommissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	if err := h.sellerUsecase.UpdateCommission(r.Context(), sellerID, req.CommissionRate); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "Seller not found", nil)
			return
		}
		if errors.Is(err, domain.ErrInvalidCommissionRate) || errors.Is(err, domain.ErrValidation) {
			response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update commission rate", nil)
		return
	}

	response.JSON(w, http.StatusOK, MessageResponse{Message: "Commission rate updated successfully"})
}
