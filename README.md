# Fintech System with Stripe Integration

A digital wallet and ledger service built with Go, PostgreSQL, Stripe, and Brevo. The service provides user accounts, wallet balance management, deposit processing via Stripe PaymentIntents, peer-to-peer fund transfers, and asynchronous email receipts.

## Architecture

The project follows standard Clean Architecture patterns:

- `cmd/api`: Application entrypoint and dependency wiring
- `internal/domain`: Core domain models, repository interfaces, and domain errors
- `internal/repository`: Data access implementations using PostgreSQL and pgx/v5
- `internal/service`: Core business logic, transaction handling, and third-party integrations
- `internal/http`: HTTP handlers, routing, and middleware (authentication, logging)
- `pkg`: Reusable packages for Stripe, Brevo email, password hashing, and JWT utilities
- `migrations`: SQL migration scripts

## Key Technical Decisions

- **Integer Currency:** All monetary values are represented as 64-bit integers (`int64`) in cents to prevent floating-point calculation errors.
- **Concurrency & Row Locking:** Fund transfers use database transactions with `SELECT ... FOR UPDATE` row locks. Wallet IDs are ordered before locking to prevent deadlocks during concurrent transfers between the same users.
- **Webhook Idempotency:** Stripe webhook events are processed idempotently by tracking Stripe payment intent IDs in the transaction ledger.
- **Asynchronous Notifications:** Email notifications (welcome emails, deposit receipts, transfer confirmations) are dispatched in background goroutines to keep API response times low.

## Requirements

- Go 1.22 or later
- PostgreSQL 14+
- Docker and Docker Compose (optional, for containerized run)

## Configuration

Copy `.env.example` to `.env` and fill in the required values:

```bash
cp .env.example .env
```

Environment variables:

| Variable | Description | Default |
| --- | --- | --- |
| `PORT` | HTTP server listening port | `8080` |
| `DB_URL` | PostgreSQL connection string | `postgres://postgres:mypassword@localhost:5433/clean_db?sslmode=disable` |
| `JWT_SECRET` | Secret key used for signing JWT tokens | - |
| `TOKEN_EXPIRY_HOURS` | JWT token expiration time in hours | `24` |
| `STRIPE_SECRET_KEY` | Stripe API secret key (`sk_test_...`) | - |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook endpoint secret (`whsec_...`) | - |
| `BREVO_API_KEY` | Brevo API key for transactional emails | - |
| `BREVO_SENDER_EMAIL` | Sender email address for notifications | `support@fintech.local` |
| `BREVO_SENDER_NAME` | Sender display name | `Fintech Wallet` |

## Getting Started

### Local Setup

1. Start PostgreSQL:
```bash
docker run -d --name fintech_postgres -p 5433:5432 -e POSTGRES_PASSWORD=mypassword -e POSTGRES_DB=clean_db postgres:16-alpine
```

2. Run database migrations and start the server:
```bash
go run cmd/api/main.go
```

The database schema is verified and migrated automatically on startup.

### Docker Compose

To start both PostgreSQL and the API service in containers:

```bash
docker compose up -d --build
```

## API Reference

### Authentication

- `POST /api/v1/auth/register` - Create a user account and initialize a wallet
- `POST /api/v1/auth/login` - Authenticate and receive a JWT Bearer token

### Wallet Operations (Requires Bearer Token)

- `GET /api/v1/wallet` - Retrieve current balance and currency
- `POST /api/v1/wallet/deposit` - Initiate a deposit via Stripe PaymentIntent
- `POST /api/v1/wallet/transfer` - Transfer funds to another user by email
- `GET /api/v1/wallet/transactions` - List paginated transaction history
- `GET /api/v1/users/me` - Retrieve current user profile

### Webhooks

- `POST /api/v1/webhooks/stripe` - Receive Stripe webhook events (verified with Stripe signature)

## Testing Examples

### Register
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name": "Emmanuel", "email": "emmanuel@example.com", "password": "Password123!"}'
```

### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "emmanuel@example.com", "password": "Password123!"}'
```

### Check Balance
```bash
curl -X GET http://localhost:8080/api/v1/wallet \
  -H "Authorization: Bearer <TOKEN>"
```

### Transfer Funds
```bash
curl -X POST http://localhost:8080/api/v1/wallet/transfer \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"recipient_email": "jane@example.com", "amount": 2500, "description": "Payment"}'
```
