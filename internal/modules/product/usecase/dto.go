package usecase

import (
	"time"

	"github.com/UmedjonQurbonov/CRM/internal/modules/product/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateProductDTO contains data needed to create a new product.
type CreateProductDTO struct {
	Name          string
	SKU           string
	QRCode        string
	Category      string
	ImageURL      string
	CostPrice     decimal.Decimal
	SellingPrice  decimal.Decimal
	StockQuantity int
	MinStockAlert int
}

// UpdateProductDTO contains data needed to update an existing product.
type UpdateProductDTO struct {
	ID            uuid.UUID
	Name          string
	SKU           string
	QRCode        string
	Category      string
	ImageURL      string
	CostPrice     decimal.Decimal
	SellingPrice  decimal.Decimal
	StockQuantity int
	MinStockAlert int
}

// RestockProductDTO contains data for replenishing stock of an existing product.
type RestockProductDTO struct {
	ID        uuid.UUID
	Quantity  int
	CostPrice *decimal.Decimal // Optional updated cost price
}

// ProductResponseDTO represents product output with role-based masking and potential profit metrics.
type ProductResponseDTO struct {
	ID                   string    `json:"id" example:"6ba7b810-9dad-11d1-80b4-00c04fd430c8"`
	Name                 string    `json:"name" example:"Smartphone Case Pro"`
	SKU                  string    `json:"sku" example:"CASE-001"`
	QRCode               string    `json:"qr_code" example:"QR-CASE-001"`
	Category             string    `json:"category" example:"Аксессуары"`
	ImageURL             string    `json:"image_url" example:"/uploads/products/case.png"`
	CostPrice            string    `json:"cost_price" example:"15.00"` // Masked to "0.00" for role 'seller'
	SellingPrice         string    `json:"selling_price" example:"25.00"`
	StockQuantity        int       `json:"stock_quantity" example:"50"`
	MinStockAlert        int       `json:"min_stock_alert" example:"5"`
	IsLowStock           bool      `json:"is_low_stock" example:"false"`
	IsActive             bool      `json:"is_active" example:"true"`
	UnitProfit           string    `json:"unit_profit" example:"10.00"`                     // Masked to "0.00" for 'seller'
	TotalPotentialProfit string    `json:"total_potential_profit" example:"500.00"`         // Masked to "0.00" for 'seller'
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// ToProductResponseDTO converts a domain.Product entity into ProductResponseDTO,
// strictly masking CostPrice and profit metrics to "0.00" if caller has role 'seller'.
func ToProductResponseDTO(p *domain.Product, callerRole string) *ProductResponseDTO {
	cost := p.CostPrice
	unitProfit := p.UnitProfit()
	totalProfit := p.TotalPotentialProfit()

	if callerRole == "seller" {
		cost = decimal.Zero
		unitProfit = decimal.Zero
		totalProfit = decimal.Zero
	}

	return &ProductResponseDTO{
		ID:                   p.ID.String(),
		Name:                 p.Name,
		SKU:                  p.SKU,
		QRCode:               p.QRCode,
		Category:             p.Category,
		ImageURL:             p.ImageURL,
		CostPrice:            cost.StringFixed(2),
		SellingPrice:         p.SellingPrice.StringFixed(2),
		StockQuantity:        p.StockQuantity,
		MinStockAlert:        p.MinStockAlert,
		IsLowStock:           p.IsLowStock(),
		IsActive:             p.IsActive,
		UnitProfit:           unitProfit.StringFixed(2),
		TotalPotentialProfit: totalProfit.StringFixed(2),
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}

// ToProductResponseDTOList converts a slice of domain.Product into ProductResponseDTOs.
func ToProductResponseDTOList(products []*domain.Product, callerRole string) []*ProductResponseDTO {
	result := make([]*ProductResponseDTO, 0, len(products))
	for _, p := range products {
		result = append(result, ToProductResponseDTO(p, callerRole))
	}
	return result
}
