package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/UmedjonQurbonov/CRM/internal/delivery/http/response"
	"github.com/UmedjonQurbonov/CRM/internal/middleware"
	"github.com/UmedjonQurbonov/CRM/internal/modules/product/domain"
	"github.com/UmedjonQurbonov/CRM/internal/modules/product/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ProductHandler struct {
	productUsecase *usecase.ProductUsecase
}

func NewProductHandler(productUsecase *usecase.ProductUsecase) *ProductHandler {
	return &ProductHandler{productUsecase: productUsecase}
}

// CreateProductRequest defines JSON payload for product creation.
type CreateProductRequest struct {
	Name          string          `json:"name" example:"Smartphone Case Pro"`
	SKU           string          `json:"sku" example:"CASE-001"`
	QRCode        string          `json:"qr_code" example:"QR-CASE-001"`
	CostPrice     decimal.Decimal `json:"cost_price" example:"15.00"`
	SellingPrice  decimal.Decimal `json:"selling_price" example:"25.00"`
	StockQuantity int             `json:"stock_quantity" example:"50"`
	MinStockAlert int             `json:"min_stock_alert" example:"5"`
}

// UpdateProductRequest defines JSON payload for product update.
type UpdateProductRequest struct {
	Name          string          `json:"name" example:"Smartphone Case Pro (Black)"`
	SKU           string          `json:"sku" example:"CASE-001"`
	QRCode        string          `json:"qr_code" example:"QR-CASE-001"`
	CostPrice     decimal.Decimal `json:"cost_price" example:"16.00"`
	SellingPrice  decimal.Decimal `json:"selling_price" example:"27.00"`
	StockQuantity int             `json:"stock_quantity" example:"60"`
	MinStockAlert int             `json:"min_stock_alert" example:"5"`
}

// ProductListResponse wraps paginated product items.
type ProductListResponse struct {
	Items  []*usecase.ProductResponseDTO `json:"items"`
	Total  int                           `json:"total" example:"100"`
	Limit  int                           `json:"limit" example:"20"`
	Offset int                           `json:"offset" example:"0"`
}

// ListProducts godoc
// @Summary List Products
// @Description Browse catalog with optional search, low stock filter, and pagination. Cost price is masked to 0.00 for sellers.
// @Tags Products
// @Security BearerAuth
// @Produce json
// @Param search query string false "Filter by name or SKU substring"
// @Param low_stock query bool false "Filter products where stock <= min_stock_alert"
// @Param limit query int false "Pagination limit (max 100)" default(20)
// @Param offset query int false "Pagination offset" default(0)
// @Success 200 {object} ProductListResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/products [get]
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := middleware.GetUserRole(r.Context())

	search := r.URL.Query().Get("search")
	lowStock, _ := strconv.ParseBool(r.URL.Query().Get("low_stock"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	filter := usecase.ProductFilter{
		Search:   search,
		LowStock: lowStock,
		Limit:    limit,
		Offset:   offset,
	}

	items, total, err := h.productUsecase.ListProducts(r.Context(), filter, callerRole)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve products", nil)
		return
	}

	response.JSON(w, http.StatusOK, ProductListResponse{
		Items:  items,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
}

// GetByQRCode godoc
// @Summary Get Product by QR Code
// @Description Find a product by its scanned QR code barcode. Cost price is masked to 0.00 for sellers.
// @Tags Products
// @Security BearerAuth
// @Produce json
// @Param code path string true "Scanned QR code string"
// @Success 200 {object} usecase.ProductResponseDTO
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/products/by-qr/{code} [get]
func (h *ProductHandler) GetByQRCode(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := middleware.GetUserRole(r.Context())
	qrCode := chi.URLParam(r, "code")

	if qrCode == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PARAM", "QR code parameter is required", nil)
		return
	}

	product, err := h.productUsecase.GetByQRCode(r.Context(), qrCode, callerRole)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "Product not found", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get product by QR code", nil)
		return
	}

	response.JSON(w, http.StatusOK, product)
}

// CreateProduct godoc
// @Summary Create Product
// @Description Register a new product in the catalog (Owner only).
// @Tags Products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateProductRequest true "Product details"
// @Success 201 {object} usecase.ProductResponseDTO
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/products [post]
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	dto := usecase.CreateProductDTO{
		Name:          req.Name,
		SKU:           req.SKU,
		QRCode:        req.QRCode,
		CostPrice:     req.CostPrice,
		SellingPrice:  req.SellingPrice,
		StockQuantity: req.StockQuantity,
		MinStockAlert: req.MinStockAlert,
	}

	product, err := h.productUsecase.CreateProduct(r.Context(), dto)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateSKU) {
			response.Error(w, http.StatusConflict, "DUPLICATE_SKU", "A product with this SKU already exists", nil)
			return
		}
		if errors.Is(err, domain.ErrDuplicateQRCode) {
			response.Error(w, http.StatusConflict, "DUPLICATE_QR_CODE", "A product with this QR code already exists", nil)
			return
		}
		if errors.Is(err, domain.ErrValidation) || errors.Is(err, domain.ErrInvalidPrice) || errors.Is(err, domain.ErrInvalidQuantity) {
			response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create product", nil)
		return
	}

	response.JSON(w, http.StatusCreated, product)
}

// UpdateProduct godoc
// @Summary Update Product
// @Description Modify properties of an existing product in the catalog (Owner only).
// @Tags Products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Product UUID"
// @Param request body UpdateProductRequest true "Updated product details"
// @Success 200 {object} usecase.ProductResponseDTO
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/products/{id} [put]
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	prodID, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid product UUID format", nil)
		return
	}

	var req UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	dto := usecase.UpdateProductDTO{
		ID:            prodID,
		Name:          req.Name,
		SKU:           req.SKU,
		QRCode:        req.QRCode,
		CostPrice:     req.CostPrice,
		SellingPrice:  req.SellingPrice,
		StockQuantity: req.StockQuantity,
		MinStockAlert: req.MinStockAlert,
	}

	product, err := h.productUsecase.UpdateProduct(r.Context(), dto)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "Product not found", nil)
			return
		}
		if errors.Is(err, domain.ErrDuplicateSKU) {
			response.Error(w, http.StatusConflict, "DUPLICATE_SKU", "A product with this SKU already exists", nil)
			return
		}
		if errors.Is(err, domain.ErrDuplicateQRCode) {
			response.Error(w, http.StatusConflict, "DUPLICATE_QR_CODE", "A product with this QR code already exists", nil)
			return
		}
		if errors.Is(err, domain.ErrValidation) || errors.Is(err, domain.ErrInvalidPrice) || errors.Is(err, domain.ErrInvalidQuantity) {
			response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update product", nil)
		return
	}

	response.JSON(w, http.StatusOK, product)
}

// GenerateQRCodePNG godoc
// @Summary Get Product QR Code Image
// @Description Generates and streams PNG image barcode encoding product's QR code (Owner only).
// @Tags Products
// @Security BearerAuth
// @Produce image/png
// @Param id path string true "Product UUID"
// @Success 200 {file} binary "PNG image"
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/products/{id}/qr-image [get]
func (h *ProductHandler) GenerateQRCodePNG(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	prodID, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid product UUID format", nil)
		return
	}

	pngBytes, err := h.productUsecase.GenerateQRCodePNG(r.Context(), prodID)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "Product not found", nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate QR code", nil)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "inline; filename=\"qr-"+prodID.String()+".png\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pngBytes)
}
