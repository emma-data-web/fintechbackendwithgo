package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type BrevoService struct {
	apiKey      string
	senderEmail string
	senderName  string
	httpClient  *http.Client
}

func NewBrevoService(apiKey, senderEmail, senderName string) *BrevoService {
	return &BrevoService{
		apiKey:      apiKey,
		senderEmail: senderEmail,
		senderName:  senderName,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

type brevoRecipient struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type brevoSender struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type brevoPayload struct {
	Sender      brevoSender      `json:"sender"`
	To          []brevoRecipient `json:"to"`
	Subject     string           `json:"subject"`
	HTMLContent string           `json:"htmlContent"`
}

func (s *BrevoService) sendEmail(toEmail, toName, subject, htmlContent string) error {
	if s.apiKey == "" || strings.HasPrefix(s.apiKey, "xkeysib-placeholder") {
		log.Printf("email disabled or missing api key: to=%s, subject=%s", toEmail, subject)
		return nil
	}

	payload := brevoPayload{
		Sender: brevoSender{
			Email: s.senderEmail,
			Name:  s.senderName,
		},
		To: []brevoRecipient{
			{Email: toEmail, Name: toName},
		},
		Subject:     subject,
		HTMLContent: htmlContent,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.brevo.com/v3/smtp/email", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create email request: %w", err)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("api-key", s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Brevo API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("brevo API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	log.Printf("email sent to %s", toEmail)
	return nil
}

func (s *BrevoService) SendWelcomeEmail(toEmail, toName string) error {
	subject := "Welcome to Fintech Wallet"
	html := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
			<h2>Welcome, %s</h2>
			<p>Your wallet account has been created and initialized.</p>
			<p>You can now deposit funds using Stripe and send transfers directly to other accounts.</p>
		</div>
	`, toName)

	return s.sendEmail(toEmail, toName, subject, html)
}

func (s *BrevoService) SendDepositReceipt(toEmail, toName string, amountCents int64, currency, reference string) error {
	formattedAmount := fmt.Sprintf("%.2f %s", float64(amountCents)/100.0, strings.ToUpper(currency))
	subject := fmt.Sprintf("Deposit Confirmed: %s", formattedAmount)
	html := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
			<h2>Deposit Confirmed</h2>
			<p>Hi %s,</p>
			<p>Your deposit of <strong>%s</strong> has been completed.</p>
			<p>Reference: <code>%s</code></p>
			<p>Date: %s</p>
		</div>
	`, toName, formattedAmount, reference, time.Now().UTC().Format(time.RFC1123))

	return s.sendEmail(toEmail, toName, subject, html)
}

func (s *BrevoService) SendTransferReceipt(toEmail, toName string, amountCents int64, currency string, isSender bool, counterparty string) error {
	formattedAmount := fmt.Sprintf("%.2f %s", float64(amountCents)/100.0, strings.ToUpper(currency))
	var subject, message string

	if isSender {
		subject = fmt.Sprintf("Transfer Sent: %s", formattedAmount)
		message = fmt.Sprintf("You transferred <strong>%s</strong> to <strong>%s</strong>.", formattedAmount, counterparty)
	} else {
		subject = fmt.Sprintf("Transfer Received: %s", formattedAmount)
		message = fmt.Sprintf("You received <strong>%s</strong> from <strong>%s</strong>.", formattedAmount, counterparty)
	}

	html := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
			<h2>Transfer Receipt</h2>
			<p>Hi %s,</p>
			<p>%s</p>
			<p>Date: %s</p>
		</div>
	`, toName, message, time.Now().UTC().Format(time.RFC1123))

	return s.sendEmail(toEmail, toName, subject, html)
}
