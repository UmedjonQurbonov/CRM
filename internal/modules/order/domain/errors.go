package domain

import "errors"

var (
	ErrOrderNotFound        = errors.New("order not found")
	ErrInsufficientStock    = errors.New("insufficient product stock for item")
	ErrInvalidPaymentMethod = errors.New("invalid payment method, expected 'cash', 'card', or 'transfer'")
	ErrOrderAlreadyRefunded = errors.New("order has already been refunded")
	ErrEmptyCart            = errors.New("cart cannot be empty")
	ErrForbidden            = errors.New("access to order is forbidden")
	ErrValidation           = errors.New("order validation failed")
)
