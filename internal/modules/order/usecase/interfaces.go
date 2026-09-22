package usecase

import (
	"context"
	"time"

	"github.com/UmedjonQurbonov/CRM/internal/modules/order/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CartItemInput represents product selection for atomic checkout.
type CartItemInput struct {
	ProductID uuid.UUID
	Quantity  int
}

// OrderFilter specifies search criteria for sales orders.
type OrderFilter struct {
	SellerID  *uuid.UUID
	Status    string
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
	Offset    int
}

// OrderRepository encapsulates atomic sales transactions and data access for orders.
type OrderRepository interface {
	CreateOrderTx(
		ctx context.Context,
		sellerID uuid.UUID,
		paymentMethod string,
		commissionRate decimal.Decimal,
		items []CartItemInput,
	) (*domain.Order, error)

	RefundOrderTx(ctx context.Context, orderID uuid.UUID) (*domain.Order, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	List(ctx context.Context, filter OrderFilter) ([]*domain.Order, int, error)
}
