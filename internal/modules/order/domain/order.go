package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	PaymentMethodCash     = "cash"
	PaymentMethodCard     = "card"
	PaymentMethodTransfer = "transfer"

	StatusCompleted = "completed"
	StatusRefunded  = "refunded"
)

// Order represents an atomic retail sales transaction.
type Order struct {
	ID                     uuid.UUID
	SellerID               uuid.UUID
	TotalAmount            decimal.Decimal
	TotalCost              decimal.Decimal
	CommissionRateSnapshot decimal.Decimal
	CommissionEarned       decimal.Decimal
	PaymentMethod          string
	Status                 string
	Items                  []OrderItem
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// OrderItem represents a single product line item inside an order.
type OrderItem struct {
	ID            uuid.UUID
	OrderID       uuid.UUID
	ProductID     uuid.UUID
	ProductName   string
	Quantity      int
	UnitPrice     decimal.Decimal
	UnitCostPrice decimal.Decimal
	Subtotal      decimal.Decimal
}

// IsValidPaymentMethod checks whether the provided payment method is supported.
func IsValidPaymentMethod(method string) bool {
	return method == PaymentMethodCash || method == PaymentMethodCard || method == PaymentMethodTransfer
}

// CanRefund returns true if the order is eligible for refund.
func (o *Order) CanRefund() bool {
	return o.Status == StatusCompleted
}
