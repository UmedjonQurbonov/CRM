package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/UmedjonQurbonov/CRM/internal/modules/auth/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSessionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSessionRepository(pool *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{pool: pool}
}

func (r *PostgresSessionRepository) CreateSession(ctx context.Context, s *domain.RefreshSession) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO refresh_sessions (id, user_id, token_hash, device_info, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(ctx, query,
		s.ID,
		s.UserID,
		s.TokenHash,
		s.DeviceInfo,
		s.ExpiresAt,
		s.CreatedAt,
	).Scan(&s.ID, &s.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert refresh session: %w", err)
	}

	return nil
}

func (r *PostgresSessionRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshSession, error) {
	query := `
		SELECT id, user_id, token_hash, COALESCE(device_info, ''), expires_at, created_at
		FROM refresh_sessions
		WHERE token_hash = $1
	`

	var s domain.RefreshSession
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&s.ID,
		&s.UserID,
		&s.TokenHash,
		&s.DeviceInfo,
		&s.ExpiresAt,
		&s.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to query refresh session: %w", err)
	}

	return &s, nil
}

func (r *PostgresSessionRepository) DeleteSession(ctx context.Context, tokenHash string) error {
	query := `
		DELETE FROM refresh_sessions
		WHERE token_hash = $1
	`

	_, err := r.pool.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to delete refresh session: %w", err)
	}

	return nil
}

func (r *PostgresSessionRepository) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	query := `
		DELETE FROM refresh_sessions
		WHERE user_id = $1
	`

	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user refresh sessions: %w", err)
	}

	return nil
}
