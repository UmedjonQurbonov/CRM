package domain

import "errors"

var (
	// ErrExpenseNotFound indicates the requested expense record does not exist.
	ErrExpenseNotFound = errors.New("expense not found")

	// ErrInvalidExpenseAmount indicates an amount <= 0 or unparseable.
	ErrInvalidExpenseAmount = errors.New("expense amount must be greater than zero")

	// ErrEmptyCategory indicates an empty or whitespace-only category name.
	ErrEmptyCategory = errors.New("expense category cannot be empty")

	// ErrForbidden indicates the caller lacks permission to perform the action.
	ErrForbidden = errors.New("forbidden: action requires owner privileges")

	// ErrValidation indicates a generic validation failure.
	ErrValidation = errors.New("expense validation failed")
)
