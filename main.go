package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fintech-system/internal/config"
	"fintech-system/internal/database"
	appHTTP "fintech-system/internal/http"
	"fintech-system/internal/http/handler"
	"fintech-system/internal/repository"
	"fintech-system/internal/service"
	"fintech-system/pkg/email"
	"fintech-system/pkg/stripepkg"
)

func main() {
	cfg := config.Load()
	log.Println("Starting Fintech System with Stripe Integration...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to connect to database pool: %v", err)
	}
	defer pool.Close()

	migrationCtx, migrationCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer migrationCancel()

	if err := database.RunMigrations(migrationCtx, pool); err != nil {
		log.Fatalf("database migrations failed: %v", err)
	}

	stripeCli := stripepkg.NewClient(cfg.StripeSecretKey, cfg.StripeWebhookSecret)
	brevoService := email.NewBrevoService(cfg.BrevoAPIKey, cfg.BrevoSenderEmail, cfg.BrevoSenderName)

	userRepo := repository.NewPostgresUserRepository(pool)
	walletRepo := repository.NewPostgresWalletRepository(pool)

	authService := service.NewAuthService(userRepo, cfg)
	userService := service.NewUserService(userRepo)
	walletService := service.NewWalletService(walletRepo, userRepo, stripeCli, brevoService)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	walletHandler := handler.NewWalletHandler(walletService)
	webhookHandler := handler.NewWebhookHandler(walletService)

	router := appHTTP.NewRouter(cfg, authHandler, userHandler, walletHandler, webhookHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server listening on port :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server stopped gracefully")
}
