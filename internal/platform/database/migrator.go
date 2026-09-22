package database

import (
	"errors"
	"fmt"

	"github.com/UmedjonQurbonov/CRM/migrations"
	"github.com/golang-migrate/migrate/v4"
	migpgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// RunMigrations applies all pending migrations from the embedded filesystem against the provided pgxpool.
func RunMigrations(pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("pgxpool is nil")
	}

	// Create *sql.DB backed by the existing pgxpool
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	// Initialize iofs source driver with embedded migration files
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source driver: %w", err)
	}

	// Initialize pgx/v5 database driver
	dbDriver, err := migpgx.WithInstance(sqlDB, &migpgx.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver for migrations: %w", err)
	}

	// Initialize migrator instance
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "pgx", dbDriver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Apply migrations
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to execute migrations: %w", err)
	}

	return nil
}
