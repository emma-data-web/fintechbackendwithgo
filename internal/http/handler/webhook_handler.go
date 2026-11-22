package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"fintech-system/internal/service"
)

type WebhookHandler struct {
	walletService *service.WalletService
}

func NewWebhookHandler(walletService *service.WalletService) *WebhookHandler {
	return &WebhookHandler{walletService: walletService}
}

func (h *WebhookHandler) HandleStripe(c *gin.Context) {
	const maxBodyBytes = int64(65536)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error reading webhook payload"})
		return
	}

	sigHeader := c.GetHeader("Stripe-Signature")
	if err := h.walletService.HandleStripeWebhook(c.Request.Context(), payload, sigHeader); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}
