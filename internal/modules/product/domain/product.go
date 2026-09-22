package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Product represents the inventory entity in the catalog.
type Product struct {
	ID             uuid.UUID
	Name           string
	SKU            string
	QRCode         string
	CostPrice      decimal.Decimal
	SellingPrice   decimal.Decimal
	StockQuantity  int
	MinStockAlert  int
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Validate checks domain invariants for Product.
func (p *Product) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return ErrValidation
	}
	if strings.TrimSpace(p.SKU) == "" {
		return ErrValidation
	}
	if strings.TrimSpace(p.QRCode) == "" {
		return ErrValidation
	}
	if p.CostPrice.IsNegative() || p.SellingPrice.IsNegative() {
		return ErrInvalidPrice
	}
	if p.StockQuantity < 0 || p.MinStockAlert < 0 {
		return ErrInvalidQuantity
	}
	return nil
}

// IsLowStock returns true if current stock is at or below the minimum alert threshold.
func (p *Product) IsLowStock() bool {
	return p.StockQuantity <= p.MinStockAlert
}
