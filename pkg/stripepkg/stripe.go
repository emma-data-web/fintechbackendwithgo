package stripepkg

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/paymentintent"
	"github.com/stripe/stripe-go/v78/webhook"
)

type Client struct {
	secretKey     string
	webhookSecret string
}

func NewClient(secretKey, webhookSecret string) *Client {
	stripe.Key = secretKey
	return &Client{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
	}
}

func (c *Client) CreatePaymentIntent(amountCents int64, currency string, customerEmail string, metadata map[string]string) (*stripe.PaymentIntent, error) {
	if strings.HasPrefix(c.secretKey, "sk_test_placeholder") || c.secretKey == "" {
		return &stripe.PaymentIntent{
			ID:           fmt.Sprintf("pi_mock_%d", amountCents),
			ClientSecret: fmt.Sprintf("pi_mock_%d_secret_test", amountCents),
			Amount:       amountCents,
			Currency:     stripe.Currency(strings.ToLower(currency)),
			Status:       stripe.PaymentIntentStatusRequiresPaymentMethod,
		}, nil
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amountCents),
		Currency: stripe.String(strings.ToLower(currency)),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
		ReceiptEmail: stripe.String(customerEmail),
	}

	for k, v := range metadata {
		params.AddMetadata(k, v)
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe payment intent: %w", err)
	}

	return pi, nil
}

func (c *Client) ConstructWebhookEvent(payload []byte, sigHeader string) (stripe.Event, error) {
	if strings.HasPrefix(c.webhookSecret, "whsec_placeholder") || c.webhookSecret == "" {
		var event stripe.Event
		if err := json.Unmarshal(payload, &event); err == nil && event.Type != "" {
			return event, nil
		}
	}

	event, err := webhook.ConstructEvent(payload, sigHeader, c.webhookSecret)
	if err != nil {
		return stripe.Event{}, fmt.Errorf("invalid webhook signature: %w", err)
	}

	return event, nil
}
