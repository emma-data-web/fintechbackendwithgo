package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v78"
	"fintech-system/internal/domain"
	"fintech-system/internal/repository"
	"fintech-system/pkg/email"
	"fintech-system/pkg/stripepkg"
)

type WalletService struct {
	walletRepo domain.WalletRepository
	userRepo   repository.UserRepository
	stripeCli  *stripepkg.Client
	brevo      *email.BrevoService
}

func NewWalletService(
	walletRepo domain.WalletRepository,
	userRepo repository.UserRepository,
	stripeCli *stripepkg.Client,
	brevo *email.BrevoService,
) *WalletService {
	return &WalletService{
		walletRepo: walletRepo,
		userRepo:   userRepo,
		stripeCli:  stripeCli,
		brevo:      brevo,
	}
}

func (s *WalletService) ProvisionWallet(ctx context.Context, userID string) (*domain.Wallet, error) {
	wallet := &domain.Wallet{
		ID:        uuid.New().String(),
		UserID:    userID,
		Balance:   0,
		Currency:  "USD",
		CreatedAt: domain.NowUTC(),
		UpdatedAt: domain.NowUTC(),
	}
	if err := s.walletRepo.CreateWallet(ctx, wallet); err != nil {
		return nil, err
	}
	return wallet, nil
}

func (s *WalletService) GetWallet(ctx context.Context, userID string) (*domain.Wallet, error) {
	return s.walletRepo.GetByUserID(ctx, userID)
}

type DepositIntentResponse struct {
	PaymentIntentID string `json:"payment_intent_id"`
	ClientSecret    string `json:"client_secret"`
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
}

func (s *WalletService) CreateDepositIntent(ctx context.Context, userID string, amount int64, currency string) (*DepositIntentResponse, error) {
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if currency == "" {
		currency = "USD"
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	metadata := map[string]string{
		"user_id":    userID,
		"user_email": user.Email,
		"user_name":  user.Name,
		"action":     "wallet_deposit",
	}

	pi, err := s.stripeCli.CreatePaymentIntent(amount, currency, user.Email, metadata)
	if err != nil {
		return nil, fmt.Errorf("stripe payment intent failed: %w", err)
	}

	return &DepositIntentResponse{
		PaymentIntentID: pi.ID,
		ClientSecret:    pi.ClientSecret,
		Amount:          amount,
		Currency:        currency,
	}, nil
}

func (s *WalletService) HandleStripeWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	event, err := s.stripeCli.ConstructWebhookEvent(payload, sigHeader)
	if err != nil {
		return fmt.Errorf("stripe webhook validation failed: %w", err)
	}

	switch event.Type {
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return fmt.Errorf("failed to unmarshal payment intent data: %w", err)
		}

		userID := pi.Metadata["user_id"]
		if userID == "" {
			log.Printf("payment intent %s skipped: missing user_id metadata", pi.ID)
			return nil
		}

		amount := pi.Amount
		currency := string(pi.Currency)
		reference := pi.ID

		tx, err := s.walletRepo.FulfillDeposit(ctx, userID, amount, reference, "Stripe Deposit")
		if err != nil {
			if errors.Is(err, domain.ErrDuplicateReference) {
				log.Printf("payment intent %s already processed", pi.ID)
				return nil
			}
			return fmt.Errorf("failed to fulfill deposit: %w", err)
		}

		log.Printf("wallet deposit completed for user %s: %d cents (ref: %s)", userID, amount, tx.Reference)

		// Send receipt email in background
		go func(targetUserID string, depositAmount int64, depCurrency, txRef string) {
			bgCtx := context.Background()
			u, err := s.userRepo.GetByID(bgCtx, targetUserID)
			if err != nil {
				return
			}
			_ = s.brevo.SendDepositReceipt(u.Email, u.Name, depositAmount, depCurrency, txRef)
		}(userID, amount, currency, reference)

	default:
		log.Printf("unhandled webhook event type: %s", event.Type)
	}

	return nil
}

func (s *WalletService) Transfer(ctx context.Context, senderUserID, recipientEmail string, amount int64, description string) (*domain.Transaction, error) {
	recipient, err := s.userRepo.GetByEmail(ctx, recipientEmail)
	if err != nil {
		return nil, fmt.Errorf("recipient user not found with email %s", recipientEmail)
	}

	sender, err := s.userRepo.GetByID(ctx, senderUserID)
	if err != nil {
		return nil, fmt.Errorf("sender not found: %w", err)
	}

	senderTx, _, err := s.walletRepo.Transfer(ctx, senderUserID, recipient.ID, amount, description)
	if err != nil {
		return nil, err
	}

	// Send receipts in background
	go func(sEmail, sName, rEmail, rName string, amt int64) {
		_ = s.brevo.SendTransferReceipt(sEmail, sName, amt, "USD", true, rEmail)
		_ = s.brevo.SendTransferReceipt(rEmail, rName, amt, "USD", false, sEmail)
	}(sender.Email, sender.Name, recipient.Email, recipient.Name, amount)

	return senderTx, nil
}

func (s *WalletService) GetTransactions(ctx context.Context, userID string, limit, offset int) ([]*domain.Transaction, error) {
	wallet, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.walletRepo.GetTransactions(ctx, wallet.ID, limit, offset)
}
