package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"fintech-system/internal/domain"
)

type PostgresWalletRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresWalletRepository(pool *pgxpool.Pool) *PostgresWalletRepository {
	return &PostgresWalletRepository{pool: pool}
}

func (r *PostgresWalletRepository) CreateWallet(ctx context.Context, wallet *domain.Wallet) error {
	query := `
		INSERT INTO wallets (id, user_id, balance, currency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		wallet.ID,
		wallet.UserID,
		wallet.Balance,
		wallet.Currency,
		wallet.CreatedAt,
		wallet.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert wallet: %w", err)
	}
	return nil
}

func (r *PostgresWalletRepository) GetByUserID(ctx context.Context, userID string) (*domain.Wallet, error) {
	query := `
		SELECT id, user_id, balance, currency, created_at, updated_at
		FROM wallets
		WHERE user_id = $1
	`
	var w domain.Wallet
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&w.ID,
		&w.UserID,
		&w.Balance,
		&w.Currency,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to query wallet by user_id: %w", err)
	}
	return &w, nil
}

func (r *PostgresWalletRepository) GetByID(ctx context.Context, walletID string) (*domain.Wallet, error) {
	query := `
		SELECT id, user_id, balance, currency, created_at, updated_at
		FROM wallets
		WHERE id = $1
	`
	var w domain.Wallet
	err := r.pool.QueryRow(ctx, query, walletID).Scan(
		&w.ID,
		&w.UserID,
		&w.Balance,
		&w.Currency,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to query wallet by id: %w", err)
	}
	return &w, nil
}

func (r *PostgresWalletRepository) FulfillDeposit(ctx context.Context, userID string, amount int64, reference, description string) (*domain.Transaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check for existing reference to guarantee idempotency
	var existingID string
	checkErr := tx.QueryRow(ctx, "SELECT id FROM transactions WHERE reference = $1", reference).Scan(&existingID)
	if checkErr == nil {
		return nil, domain.ErrDuplicateReference
	} else if !errors.Is(checkErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check transaction reference: %w", checkErr)
	}

	// Lock wallet row to prevent concurrent modifications
	var walletID string
	var currentBalance int64
	lockQuery := `
		SELECT id, balance 
		FROM wallets 
		WHERE user_id = $1 
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, lockQuery, userID).Scan(&walletID, &currentBalance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to lock wallet: %w", err)
	}

	newBalance := currentBalance + amount
	now := time.Now().UTC()
	updateQuery := `
		UPDATE wallets 
		SET balance = $1, updated_at = $2 
		WHERE id = $3
	`
	_, err = tx.Exec(ctx, updateQuery, newBalance, now, walletID)
	if err != nil {
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	txRecord := &domain.Transaction{
		ID:          uuid.New().String(),
		WalletID:    walletID,
		Amount:      amount,
		Type:        domain.TxTypeDeposit,
		Status:      domain.TxStatusCompleted,
		Reference:   reference,
		Description: description,
		CreatedAt:   now,
	}

	insertTxQuery := `
		INSERT INTO transactions (id, wallet_id, amount, type, status, reference, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = tx.Exec(ctx, insertTxQuery,
		txRecord.ID,
		txRecord.WalletID,
		txRecord.Amount,
		txRecord.Type,
		txRecord.Status,
		txRecord.Reference,
		txRecord.Description,
		txRecord.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return txRecord, nil
}

func (r *PostgresWalletRepository) Transfer(ctx context.Context, senderUserID, receiverUserID string, amount int64, description string) (*domain.Transaction, *domain.Transaction, error) {
	if senderUserID == receiverUserID {
		return nil, nil, domain.ErrSelfTransfer
	}
	if amount <= 0 {
		return nil, nil, domain.ErrInvalidAmount
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Order IDs to prevent deadlocks during concurrent transfers
	firstUserID, secondUserID := senderUserID, receiverUserID
	if firstUserID > secondUserID {
		firstUserID, secondUserID = secondUserID, firstUserID
	}

	var firstWalletID string
	var firstBalance int64
	err = tx.QueryRow(ctx, "SELECT id, balance FROM wallets WHERE user_id = $1 FOR UPDATE", firstUserID).Scan(&firstWalletID, &firstBalance)
	if err != nil {
		return nil, nil, fmt.Errorf("wallet not found for user %s: %w", firstUserID, err)
	}

	var secondWalletID string
	var secondBalance int64
	err = tx.QueryRow(ctx, "SELECT id, balance FROM wallets WHERE user_id = $1 FOR UPDATE", secondUserID).Scan(&secondWalletID, &secondBalance)
	if err != nil {
		return nil, nil, fmt.Errorf("wallet not found for user %s: %w", secondUserID, err)
	}

	var senderWalletID, receiverWalletID string
	var senderCurrentBalance, receiverCurrentBalance int64

	if senderUserID == firstUserID {
		senderWalletID, senderCurrentBalance = firstWalletID, firstBalance
		receiverWalletID, receiverCurrentBalance = secondWalletID, secondBalance
	} else {
		senderWalletID, senderCurrentBalance = secondWalletID, secondBalance
		receiverWalletID, receiverCurrentBalance = firstWalletID, firstBalance
	}

	if senderCurrentBalance < amount {
		return nil, nil, domain.ErrInsufficientBalance
	}

	now := time.Now().UTC()

	_, err = tx.Exec(ctx, "UPDATE wallets SET balance = $1, updated_at = $2 WHERE id = $3",
		senderCurrentBalance-amount, now, senderWalletID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deduct sender balance: %w", err)
	}

	_, err = tx.Exec(ctx, "UPDATE wallets SET balance = $1, updated_at = $2 WHERE id = $3",
		receiverCurrentBalance+amount, now, receiverWalletID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to credit receiver balance: %w", err)
	}

	transferRef := fmt.Sprintf("tr_%s_%d", uuid.New().String()[:8], time.Now().Unix())

	senderTx := &domain.Transaction{
		ID:          uuid.New().String(),
		WalletID:    senderWalletID,
		Amount:      amount,
		Type:        domain.TxTypeTransferOut,
		Status:      domain.TxStatusCompleted,
		Reference:   transferRef + "_out",
		Description: fmt.Sprintf("Transfer to %s: %s", receiverUserID, description),
		CreatedAt:   now,
	}

	insertTxQuery := `
		INSERT INTO transactions (id, wallet_id, amount, type, status, reference, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = tx.Exec(ctx, insertTxQuery,
		senderTx.ID, senderTx.WalletID, senderTx.Amount, senderTx.Type, senderTx.Status, senderTx.Reference, senderTx.Description, senderTx.CreatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to insert sender transaction: %w", err)
	}

	receiverTx := &domain.Transaction{
		ID:          uuid.New().String(),
		WalletID:    receiverWalletID,
		Amount:      amount,
		Type:        domain.TxTypeTransferIn,
		Status:      domain.TxStatusCompleted,
		Reference:   transferRef + "_in",
		Description: fmt.Sprintf("Transfer from %s: %s", senderUserID, description),
		CreatedAt:   now,
	}
	_, err = tx.Exec(ctx, insertTxQuery,
		receiverTx.ID, receiverTx.WalletID, receiverTx.Amount, receiverTx.Type, receiverTx.Status, receiverTx.Reference, receiverTx.Description, receiverTx.CreatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to insert receiver transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transfer: %w", err)
	}

	return senderTx, receiverTx, nil
}

func (r *PostgresWalletRepository) GetTransactions(ctx context.Context, walletID string, limit, offset int) ([]*domain.Transaction, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, wallet_id, amount, type, status, reference, description, created_at
		FROM transactions
		WHERE wallet_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, walletID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var list []*domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		if err := rows.Scan(
			&t.ID,
			&t.WalletID,
			&t.Amount,
			&t.Type,
			&t.Status,
			&t.Reference,
			&t.Description,
			&t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction row: %w", err)
		}
		list = append(list, &t)
	}

	return list, nil
}
