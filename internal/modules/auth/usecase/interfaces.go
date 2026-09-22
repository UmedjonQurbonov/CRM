package usecase

import (
	"context"

	"github.com/UmedjonQurbonov/CRM/internal/modules/auth/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// UserRepository specifies data access operations for user entities.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByPhone(ctx context.Context, phone string) (*domain.User, error)
	ListSellers(ctx context.Context) ([]*domain.User, error)
	UpdateCommissionRate(ctx context.Context, id uuid.UUID, rate decimal.Decimal) error
	Count(ctx context.Context) (int64, error)
}

// SessionRepository specifies data access operations for refresh token sessions.
type SessionRepository interface {
	CreateSession(ctx context.Context, session *domain.RefreshSession) error
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshSession, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	DeleteUserSessions(ctx context.Context, userID uuid.UUID) error
}
