package service

import (
	"context"
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/brevo"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/turnstile"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

const emailConnectionTimeout = 15 * time.Second

// EmailStorageTester writes and deletes a test object in object storage.
type EmailStorageTester interface {
	Test(ctx context.Context) error
}

// EmailTurnstileVerifier verifies Turnstile tokens.
type EmailTurnstileVerifier interface {
	Verify(ctx context.Context, secret, token, ip string) (turnstile.Result, error)
}

// EmailConnectionResult is the outcome of a Settings "Test Connection" button.
type EmailConnectionResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// EmailIntegrationTester runs the Test Connection checks of the email, storage and security settings tabs (PRD 9).
type EmailIntegrationTester struct {
	delivery  EmailDelivery
	storage   EmailStorageTester
	turnstile EmailTurnstileVerifier
	settings  SettingsReader
}

// NewEmailIntegrationTester builds an EmailIntegrationTester.
func NewEmailIntegrationTester(delivery EmailDelivery, storage EmailStorageTester, verifier EmailTurnstileVerifier, settings SettingsReader) *EmailIntegrationTester {
	return &EmailIntegrationTester{delivery: delivery, storage: storage, turnstile: verifier, settings: settings}
}

// Test runs the check for group; the test email goes to the signed-in admin.
func (t *EmailIntegrationTester) Test(ctx context.Context, group string, admin model.Admin) (EmailConnectionResult, error) {
	ctx, cancel := context.WithTimeout(ctx, emailConnectionTimeout)
	defer cancel()
	switch group {
	case "email":
		return t.testEmail(ctx, admin)
	case "storage":
		if err := t.storage.Test(ctx); err != nil {
			return EmailConnectionResult{Message: "Storage test failed: " + err.Error()}, nil
		}
		return EmailConnectionResult{OK: true, Message: "A test object was written and deleted successfully."}, nil
	case "security":
		return t.testTurnstile(ctx)
	}
	return EmailConnectionResult{}, apperr.NotFound("Connection test for this settings group")
}

func (t *EmailIntegrationTester) testEmail(ctx context.Context, admin model.Admin) (EmailConnectionResult, error) {
	values, err := t.settings.Values(ctx, "email")
	if err != nil {
		return EmailConnectionResult{}, err
	}
	key, _ := values["brevo_api_key"].(string)
	sender, _ := values["sender_email"].(string)
	name, _ := values["sender_name"].(string)
	if key == "" || sender == "" {
		return EmailConnectionResult{Message: "Set the Brevo API key and the sender email first."}, nil
	}
	to := admin.Email
	if to == "" {
		if list := emailStrings(values["admin_notification_emails"]); len(list) > 0 {
			to = list[0]
		}
	}
	if to == "" {
		return EmailConnectionResult{Message: "No recipient is available for the test email."}, nil
	}
	_, err = t.delivery.Send(ctx, brevo.Message{
		APIKey:  key,
		Sender:  brevo.Address{Email: sender, Name: name},
		To:      []brevo.Address{{Email: to}},
		Subject: "Seafarers Crew test email",
		HTML:    "<p>This is a test email from the Seafarers Crew admin panel. Your Brevo settings work.</p>",
	})
	if err != nil {
		return EmailConnectionResult{Message: "Sending failed: " + err.Error()}, nil
	}
	return EmailConnectionResult{OK: true, Message: "A test email was sent to " + to + "."}, nil
}

func (t *EmailIntegrationTester) testTurnstile(ctx context.Context) (EmailConnectionResult, error) {
	values, err := t.settings.Values(ctx, "security")
	if err != nil {
		return EmailConnectionResult{}, err
	}
	secret, _ := values["turnstile_secret_key"].(string)
	site, _ := values["turnstile_site_key"].(string)
	if secret == "" || site == "" {
		return EmailConnectionResult{Message: "Set the Turnstile site key and secret key first."}, nil
	}
	res, err := t.turnstile.Verify(ctx, secret, "connection-test.invalid-token", "")
	if err != nil {
		return EmailConnectionResult{Message: "Turnstile could not be reached: " + err.Error()}, nil
	}
	if res.HasError("invalid-input-secret") || res.HasError("missing-input-secret") {
		return EmailConnectionResult{Message: "Cloudflare rejected the Turnstile secret key."}, nil
	}
	return EmailConnectionResult{OK: true, Message: "Cloudflare accepted the Turnstile secret key."}, nil
}
