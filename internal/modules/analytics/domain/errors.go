package domain

import "errors"

var (
	// ErrInvalidDateRange indicates 'from' date is after 'to' date.
	ErrInvalidDateRange = errors.New("invalid date range: 'from' date must be before or equal to 'to' date")

	// ErrForbidden indicates lack of permission for the analytics resource.
	ErrForbidden = errors.New("forbidden: analytics data requires owner privileges")

	// ErrValidation indicates a generic validation failure.
	ErrValidation = errors.New("validation failed")
)
