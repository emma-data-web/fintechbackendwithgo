package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
	CREATE TABLE IF NOT EXISTS ecom_users (
		id VARCHAR(36) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL DEFAULT 'customer',
		address TEXT DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_ecom_users_email ON ecom_users(email);

	CREATE TABLE IF NOT EXISTS wallets (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL UNIQUE REFERENCES ecom_users(id) ON DELETE CASCADE,
		balance BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
		currency VARCHAR(3) NOT NULL DEFAULT 'USD',
		created_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id);

	CREATE TABLE IF NOT EXISTS transactions (
		id VARCHAR(36) PRIMARY KEY,
		wallet_id VARCHAR(36) NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
		amount BIGINT NOT NULL CHECK (amount > 0),
		type VARCHAR(20) NOT NULL,
		status VARCHAR(20) NOT NULL,
		reference VARCHAR(255) UNIQUE,
		description TEXT DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_transactions_wallet_id ON transactions(wallet_id);
	CREATE INDEX IF NOT EXISTS idx_transactions_reference ON transactions(reference);
	`

	_, err := pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
