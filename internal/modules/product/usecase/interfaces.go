package usecase

import (
	"context"

	"github.com/UmedjonQurbonov/CRM/internal/modules/product/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ProductFilter encapsulates query filters for catalog browsing.
type ProductFilter struct {
	Search   string
	Category string
	LowStock bool
	InStock  *bool
	Limit    int
	Offset   int
}

// ProductRepository specifies data access operations for product entities.
type ProductRepository interface {
	Create(ctx context.Context, p *domain.Product) (*domain.Product, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	GetByQRCode(ctx context.Context, qrCode string) (*domain.Product, error)
	List(ctx context.Context, filter ProductFilter) ([]*domain.Product, int, error)
	Update(ctx context.Context, p *domain.Product) (*domain.Product, error)
	Deactivate(ctx context.Context, id uuid.UUID) error
	Restock(ctx context.Context, id uuid.UUID, addedQty int, newCostPrice *decimal.Decimal) (*domain.Product, error)
	UpdateImage(ctx context.Context, id uuid.UUID, imageURL string) error
	GetTotalStockProfit(ctx context.Context) (decimal.Decimal, error)
}
