package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"fintech-system/internal/domain"
	"fintech-system/internal/service"
)

type WalletHandler struct {
	walletService *service.WalletService
}

func NewWalletHandler(walletService *service.WalletService) *WalletHandler {
	return &WalletHandler{walletService: walletService}
}

func (h *WalletHandler) GetWallet(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	wallet, err := h.walletService.GetWallet(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrWalletNotFound) {
			// Auto-provision if not existing
			newWallet, provErr := h.walletService.ProvisionWallet(c.Request.Context(), userID)
			if provErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to provision wallet"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"wallet": newWallet})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"wallet": wallet})
}

type DepositRequest struct {
	Amount   int64  `json:"amount" binding:"required,min=100"` // minimum $1.00 (100 cents)
	Currency string `json:"currency"`
}

func (h *WalletHandler) InitiateDeposit(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req DepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: amount (in cents, min 100) is required"})
		return
	}

	intent, err := h.walletService.CreateDepositIntent(c.Request.Context(), userID, req.Amount, req.Currency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Stripe PaymentIntent generated successfully",
		"data":    intent,
	})
}

type TransferRequest struct {
	RecipientEmail string `json:"recipient_email" binding:"required,email"`
	Amount         int64  `json:"amount" binding:"required,min=1"` // in cents
	Description    string `json:"description"`
}

func (h *WalletHandler) Transfer(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: recipient_email and positive amount required"})
		return
	}

	tx, err := h.walletService.Transfer(c.Request.Context(), userID, req.RecipientEmail, req.Amount, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInsufficientBalance):
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "insufficient wallet balance"})
		case errors.Is(err, domain.ErrSelfTransfer):
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot transfer funds to your own account"})
		case errors.Is(err, domain.ErrInvalidAmount):
			c.JSON(http.StatusBadRequest, gin.H{"error": "transfer amount must be greater than zero"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "funds transferred successfully",
		"transaction": tx,
	})
}

func (h *WalletHandler) GetTransactions(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	transactions, err := h.walletService.GetTransactions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":        len(transactions),
		"transactions": transactions,
	})
}
