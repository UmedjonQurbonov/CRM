package usecase

import (
	"context"
	"strings"

	authUsecase "github.com/UmedjonQurbonov/CRM/internal/modules/auth/usecase"
	"github.com/UmedjonQurbonov/CRM/internal/modules/order/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrderUsecase encapsulates checkout transactions, receipts, and returns logic.
type OrderUsecase struct {
	orderRepo OrderRepository
	userRepo  authUsecase.UserRepository
}

func NewOrderUsecase(orderRepo OrderRepository, userRepo authUsecase.UserRepository) *OrderUsecase {
	return &OrderUsecase{
		orderRepo: orderRepo,
		userRepo:  userRepo,
	}
}

// Checkout executes atomic sales transaction with stock decrement and commission snapshot.
func (u *OrderUsecase) Checkout(
	ctx context.Context,
	callerID uuid.UUID,
	callerRole string,
	dto CheckoutRequestDTO,
) (*OrderResponseDTO, error) {
	method := strings.ToLower(strings.TrimSpace(dto.PaymentMethod))
	if !domain.IsValidPaymentMethod(method) {
		return nil, domain.ErrInvalidPaymentMethod
	}

	if len(dto.Items) == 0 {
		return nil, domain.ErrEmptyCart
	}

	cartInputs := make([]CartItemInput, 0, len(dto.Items))
	for _, item := range dto.Items {
		prodID, err := uuid.Parse(item.ProductID)
		if err != nil || prodID == uuid.Nil {
			return nil, domain.ErrValidation
		}
		if item.Quantity <= 0 {
			return nil, domain.ErrValidation
		}
		cartInputs = append(cartInputs, CartItemInput{
			ProductID: prodID,
			Quantity:  item.Quantity,
		})
	}

	// Calculate commission rate snapshot
	commissionRate := decimal.Zero
	if callerRole == "seller" {
		user, err := u.userRepo.GetByID(ctx, callerID)
		if err != nil {
			return nil, err
		}
		commissionRate = user.CommissionRate
	}

	order, err := u.orderRepo.CreateOrderTx(ctx, callerID, method, commissionRate, cartInputs)
	if err != nil {
		return nil, err
	}

	return ToOrderResponseDTO(order), nil
}

// RefundOrder issues a refund and restocks inventory (Owner only).
func (u *OrderUsecase) RefundOrder(ctx context.Context, orderID uuid.UUID, callerRole string) (*OrderResponseDTO, error) {
	if callerRole != "owner" {
		return nil, domain.ErrForbidden
	}

	if orderID == uuid.Nil {
		return nil, domain.ErrValidation
	}

	refundedOrder, err := u.orderRepo.RefundOrderTx(ctx, orderID)
	if err != nil {
		return nil, err
	}

	return ToOrderResponseDTO(refundedOrder), nil
}

// GetOrderByID retrieves order details with role-based visibility restrictions.
func (u *OrderUsecase) GetOrderByID(
	ctx context.Context,
	orderID uuid.UUID,
	callerID uuid.UUID,
	callerRole string,
) (*OrderResponseDTO, error) {
	if orderID == uuid.Nil {
		return nil, domain.ErrValidation
	}

	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Sellers can only view their own receipts
	if callerRole == "seller" && order.SellerID != callerID {
		return nil, domain.ErrForbidden
	}

	return ToOrderResponseDTO(order), nil
}

// ListOrders retrieves sales orders with mandatory filtering for sellers.
func (u *OrderUsecase) ListOrders(
	ctx context.Context,
	filter OrderFilter,
	callerID uuid.UUID,
	callerRole string,
) ([]*OrderResponseDTO, int, error) {
	// Sellers can strictly view only their own orders
	if callerRole == "seller" {
		filter.SellerID = &callerID
	}

	orders, total, err := u.orderRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return ToOrderResponseDTOList(orders), total, nil
}
