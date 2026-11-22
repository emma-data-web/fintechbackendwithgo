package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"fintech-system/internal/config"
	"fintech-system/internal/domain"
	"fintech-system/internal/http/handler"
	"fintech-system/internal/http/middleware"
)

func NewRouter(
	cfg *config.Config,
	authH *handler.AuthHandler,
	userH *handler.UserHandler,
	walletH *handler.WalletHandler,
	webhookH *handler.WebhookHandler,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(middleware.StructuredLogger())
	r.Use(gin.Recovery())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "fintech-system-with-stripe-integration",
			"version": "1.0.0",
		})
	})

	api := r.Group("/api/v1")
	{
		// Public Auth routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", authH.Register)
			auth.POST("/login", authH.Login)
		}

		// Public Webhook routes (cryptographically verified by Stripe signature)
		webhooks := api.Group("/webhooks")
		{
			webhooks.POST("/stripe", webhookH.HandleStripe)
		}

		// Protected routes under JWT Bearer Middleware
		protected := api.Group("")
		protected.Use(middleware.RequireAuth(cfg.JWTSecret))
		{
			// User Profile
			protected.GET("/users/me", userH.GetMe)

			// Digital Wallet & Fintech Engine
			wallet := protected.Group("/wallet")
			{
				wallet.GET("", walletH.GetWallet)
				wallet.POST("/deposit", walletH.InitiateDeposit)
				wallet.POST("/transfer", walletH.Transfer)
				wallet.GET("/transactions", walletH.GetTransactions)
			}

			// Admin-only management
			admin := protected.Group("/admin")
			admin.Use(middleware.RequireRole(domain.RoleAdmin))
			{
				admin.GET("/users", userH.ListUsers)
			}
		}
	}

	return r
}
