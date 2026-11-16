package domain

import (
	"time"

	"github.com/google/uuid"
)

// ExpenseCategory represents a user-managed expense classification.
type ExpenseCategory struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
