package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                string
	DatabaseURL         string
	JWTSecret           string
	TokenExpiryHours    int
	DBMaxConns          int32
	DBMinConns          int32
	StripeSecretKey     string
	StripeWebhookSecret string
	BrevoAPIKey         string
	BrevoSenderEmail    string
	BrevoSenderName     string
	Environment         string
}

func Load() *Config {
	expiryHours, err := strconv.Atoi(getEnv("TOKEN_EXPIRY_HOURS", "24"))
	if err != nil {
		expiryHours = 24
	}

	return &Config{
		Port:                getEnv("PORT", "8080"),
		DatabaseURL:         getEnv("DB_URL", getEnv("DATABASE_URL", "postgres://postgres:mypassword@localhost:5433/clean_db?sslmode=disable")),
		JWTSecret:           getEnv("JWT_SECRET", "default-dev-secret-change-in-production"),
		TokenExpiryHours:    expiryHours,
		DBMaxConns:          25,
		DBMinConns:          5,
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", "sk_test_placeholder"),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", "whsec_placeholder"),
		BrevoAPIKey:         getEnv("BREVO_API_KEY", ""),
		BrevoSenderEmail:    getEnv("BREVO_SENDER_EMAIL", "support@fintech.local"),
		BrevoSenderName:     getEnv("BREVO_SENDER_NAME", "Fintech Wallet"),
		Environment:         getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return defaultVal
}
