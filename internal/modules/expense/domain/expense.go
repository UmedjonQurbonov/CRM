package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Expense represents an operational expense record of the retail store.
type Expense struct {
	ID          uuid.UUID
	CreatedBy   uuid.UUID
	Category    string
	Amount      decimal.Decimal
	Comment     *string
	ExpenseDate time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate verifies domain invariants for an expense record.
func (e *Expense) Validate() error {
	if e.CreatedBy == uuid.Nil {
		return ErrValidation
	}
	if strings.TrimSpace(e.Category) == "" {
		return ErrEmptyCategory
	}
	if !e.Amount.GreaterThan(decimal.Zero) {
		return ErrInvalidExpenseAmount
	}
	if e.ExpenseDate.IsZero() {
		return ErrValidation
	}
	return nil
}
