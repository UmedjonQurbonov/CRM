package domain

import "errors"

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrDuplicateSKU      = errors.New("product with this SKU already exists")
	ErrDuplicateQRCode   = errors.New("product with this QR code already exists")
	ErrInsufficientStock = errors.New("insufficient product stock")
	ErrInvalidPrice      = errors.New("price cannot be negative")
	ErrInvalidQuantity   = errors.New("quantity cannot be negative")
	ErrValidation        = errors.New("product validation failed")
)
