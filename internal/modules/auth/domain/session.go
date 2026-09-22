package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefreshSession represents an active refresh token session for a user.
type RefreshSession struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  string
	DeviceInfo string
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

// IsExpired checks whether the refresh session has passed its expiration time.
func (s *RefreshSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
