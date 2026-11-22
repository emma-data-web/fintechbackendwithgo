package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrInsufficientBalance = errors.New("insufficient wallet balance")
	ErrSelfTransfer        = errors.New("cannot transfer funds to yourself")
	ErrInvalidAmount       = errors.New("amount must be greater than zero")
	ErrDuplicateReference  = errors.New("transaction reference already processed")
)

type TransactionType string

const (
	TxTypeDeposit     TransactionType = "deposit"
	TxTypeWithdrawal  TransactionType = "withdrawal"
	TxTypeTransferIn  TransactionType = "transfer_in"
	TxTypeTransferOut TransactionType = "transfer_out"
)

type TransactionStatus string

const (
	TxStatusPending   TransactionStatus = "pending"
	TxStatusCompleted TransactionStatus = "completed"
	TxStatusFailed    TransactionStatus = "failed"
)

type Wallet struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Balance   int64     `json:"balance"` // balance in cents
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Transaction struct {
	ID          string            `json:"id"`
	WalletID    string            `json:"wallet_id"`
	Amount      int64             `json:"amount"` // amount in cents
	Type        TransactionType   `json:"type"`
	Status      TransactionStatus `json:"status"`
	Reference   string            `json:"reference"`
	Description string            `json:"description,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

type WalletRepository interface {
	CreateWallet(ctx context.Context, wallet *Wallet) error
	GetByUserID(ctx context.Context, userID string) (*Wallet, error)
	GetByID(ctx context.Context, walletID string) (*Wallet, error)
	FulfillDeposit(ctx context.Context, userID string, amount int64, reference, description string) (*Transaction, error)
	Transfer(ctx context.Context, senderUserID, receiverUserID string, amount int64, description string) (*Transaction, *Transaction, error)
	GetTransactions(ctx context.Context, walletID string, limit, offset int) ([]*Transaction, error)
}
