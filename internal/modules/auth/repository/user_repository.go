package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/UmedjonQurbonov/CRM/internal/modules/auth/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (r *PostgresUserRepository) Create(ctx context.Context, u *domain.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}

	query := `
		INSERT INTO users (id, name, phone, password_hash, role, commission_rate, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		u.ID,
		u.Name,
		u.Phone,
		u.PasswordHash,
		u.Role,
		u.CommissionRate,
		u.CreatedAt,
		u.UpdatedAt,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return domain.ErrPhoneAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, name, phone, password_hash, role, commission_rate, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var u domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Name,
		&u.Phone,
		&u.PasswordHash,
		&u.Role,
		&u.CommissionRate,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &u, nil
}

func (r *PostgresUserRepository) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	query := `
		SELECT id, name, phone, password_hash, role, commission_rate, created_at, updated_at
		FROM users
		WHERE phone = $1
	`

	var u domain.User
	err := r.pool.QueryRow(ctx, query, phone).Scan(
		&u.ID,
		&u.Name,
		&u.Phone,
		&u.PasswordHash,
		&u.Role,
		&u.CommissionRate,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by phone: %w", err)
	}

	return &u, nil
}

func (r *PostgresUserRepository) ListSellers(ctx context.Context) ([]*domain.User, error) {
	query := `
		SELECT id, name, phone, password_hash, role, commission_rate, created_at, updated_at
		FROM users
		WHERE role = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, domain.RoleSeller)
	if err != nil {
		return nil, fmt.Errorf("failed to list sellers: %w", err)
	}
	defer rows.Close()

	var sellers []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Phone,
			&u.PasswordHash,
			&u.Role,
			&u.CommissionRate,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan seller row: %w", err)
		}
		sellers = append(sellers, &u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during sellers iteration: %w", err)
	}

	return sellers, nil
}

func (r *PostgresUserRepository) UpdateCommissionRate(ctx context.Context, id uuid.UUID, rate decimal.Decimal) error {
	query := `
		UPDATE users
		SET commission_rate = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND role = $3
	`

	cmdTag, err := r.pool.Exec(ctx, query, rate, id, domain.RoleSeller)
	if err != nil {
		return fmt.Errorf("failed to update commission rate: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

func (r *PostgresUserRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM users`
	var count int64
	if err := r.pool.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}
