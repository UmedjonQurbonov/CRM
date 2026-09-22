package domain

import "errors"

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrPhoneAlreadyExists    = errors.New("phone number already registered")
	ErrInvalidCredentials    = errors.New("invalid phone or password")
	ErrSessionExpired        = errors.New("session has expired")
	ErrSessionNotFound       = errors.New("session not found or revoked")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrForbidden             = errors.New("access forbidden")
	ErrValidation            = errors.New("validation failed")
	ErrInvalidCommissionRate = errors.New("commission rate must be between 0 and 100")
)
