package usecase

import (
	"time"

	"github.com/UmedjonQurbonov/CRM/internal/modules/order/domain"
)

// CartItemDTO defines product item in checkout request.
type CartItemDTO struct {
	ProductID string `json:"product_id" example:"6ba7b810-9dad-11d1-80b4-00c04fd430c8"`
	Quantity  int    `json:"quantity" example:"2"`
}

// CheckoutRequestDTO defines checkout payload.
type CheckoutRequestDTO struct {
	PaymentMethod string        `json:"payment_method" example:"cash"` // cash, card, transfer
	Items         []CartItemDTO `json:"items"`
}

// OrderItemResponseDTO represents an item in an order receipt.
type OrderItemResponseDTO struct {
	ID          string `json:"id" example:"7ca7b810-9dad-11d1-80b4-00c04fd430c9"`
	ProductID   string `json:"product_id" example:"6ba7b810-9dad-11d1-80b4-00c04fd430c8"`
	ProductName string `json:"product_name" example:"Sony WH-1000XM5"`
	Quantity    int    `json:"quantity" example:"2"`
	UnitPrice   string `json:"unit_price" example:"399.99"`
	Subtotal    string `json:"subtotal" example:"799.98"`
}

// OrderResponseDTO represents order output.
type OrderResponseDTO struct {
	ID                     string                 `json:"id" example:"5ba7b810-9dad-11d1-80b4-00c04fd430c7"`
	SellerID               string                 `json:"seller_id" example:"adfae494-4586-4992-b63a-8884bd0aac6c"`
	TotalAmount            string                 `json:"total_amount" example:"799.98"`
	CommissionRateSnapshot string                 `json:"commission_rate_snapshot" example:"5.00"`
	CommissionEarned       string                 `json:"commission_earned" example:"40.00"`
	PaymentMethod          string                 `json:"payment_method" example:"cash"`
	Status                 string                 `json:"status" example:"completed"`
	Items                  []OrderItemResponseDTO `json:"items"`
	CreatedAt              time.Time              `json:"created_at"`
	UpdatedAt              time.Time              `json:"updated_at"`
}

// ToOrderResponseDTO converts a domain.Order entity into OrderResponseDTO.
func ToOrderResponseDTO(o *domain.Order) *OrderResponseDTO {
	items := make([]OrderItemResponseDTO, 0, len(o.Items))
	for _, item := range o.Items {
		items = append(items, OrderItemResponseDTO{
			ID:          item.ID.String(),
			ProductID:   item.ProductID.String(),
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice.StringFixed(2),
			Subtotal:    item.Subtotal.StringFixed(2),
		})
	}

	return &OrderResponseDTO{
		ID:                     o.ID.String(),
		SellerID:               o.SellerID.String(),
		TotalAmount:            o.TotalAmount.StringFixed(2),
		CommissionRateSnapshot: o.CommissionRateSnapshot.StringFixed(2),
		CommissionEarned:       o.CommissionEarned.StringFixed(2),
		PaymentMethod:          o.PaymentMethod,
		Status:                 o.Status,
		Items:                  items,
		CreatedAt:              o.CreatedAt,
		UpdatedAt:              o.UpdatedAt,
	}
}

// ToOrderResponseDTOList converts a slice of domain.Order into OrderResponseDTOs.
func ToOrderResponseDTOList(orders []*domain.Order) []*OrderResponseDTO {
	result := make([]*OrderResponseDTO, 0, len(orders))
	for _, o := range orders {
		result = append(result, ToOrderResponseDTO(o))
	}
	return result
}
