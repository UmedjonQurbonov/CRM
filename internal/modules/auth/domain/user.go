package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	RoleOwner  = "owner"
	RoleSeller = "seller"
)

// User represents the pure domain entity for application accounts (owners & sellers).
type User struct {
	ID             uuid.UUID
	Name           string
	Phone          string
	PasswordHash   string
	Role           string
	CommissionRate decimal.Decimal
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Validate checks business invariants for User entity.
func (u *User) Validate() error {
	if strings.TrimSpace(u.Name) == "" {
		return ErrValidation
	}
	if strings.TrimSpace(u.Phone) == "" {
		return ErrValidation
	}
	if u.Role != RoleOwner && u.Role != RoleSeller {
		return ErrValidation
	}
	if u.CommissionRate.IsNegative() || u.CommissionRate.GreaterThan(decimal.NewFromInt(100)) {
		return ErrInvalidCommissionRate
	}
	return nil
}

// IsOwner returns true if user has owner role.
func (u *User) IsOwner() bool {
	return u.Role == RoleOwner
}

// IsSeller returns true if user has seller role.
func (u *User) IsSeller() bool {
	return u.Role == RoleSeller
}
