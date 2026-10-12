package service

import (
	"testing"

	"go.uber.org/zap"
)

func newTestLogger() *zap.SugaredLogger {
	return zap.NewNop().Sugar()
}

// ─── IsConfigured ───────────────────────────────────────────────────────

func TestEmailService_IsConfigured(t *testing.T) {
	tests := []struct {
		name   string
		config EmailConfig
		want   bool
	}{
		{"fully configured", EmailConfig{Host: "smtp.example.com", Port: 587}, true},
		{"missing host", EmailConfig{Port: 587}, false},
		{"missing port", EmailConfig{Host: "smtp.example.com"}, false},
		{"zero port", EmailConfig{Host: "smtp.example.com", Port: 0}, false},
		{"empty config", EmailConfig{}, false},
		{"negative port", EmailConfig{Host: "smtp.example.com", Port: -1}, false},
		{"host only", EmailConfig{Host: "smtp.example.com"}, false},
		{"port only", EmailConfig{Port: 587}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewEmailService(tt.config, newTestLogger())
			if got := svc.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ─── Send guards ────────────────────────────────────────────────────────

func TestEmailService_Send_NotConfigured(t *testing.T) {
	svc := NewEmailService(EmailConfig{}, newTestLogger())
	err := svc.Send("test subject", "body", []string{"admin@example.com"})
	if err != nil {
		t.Errorf("Send() when unconfigured should return nil, got: %v", err)
	}
}

func TestEmailService_Send_NoRecipients(t *testing.T) {
	svc := NewEmailService(EmailConfig{Host: "smtp.example.com", Port: 587}, newTestLogger())
	err := svc.Send("test subject", "body", []string{})
	if err == nil {
		t.Error("Send() with no recipients should return error")
	}
}

func TestEmailService_Send_NilRecipients(t *testing.T) {
	svc := NewEmailService(EmailConfig{Host: "smtp.example.com", Port: 587}, newTestLogger())
	err := svc.Send("test subject", "body", nil)
	if err == nil {
		t.Error("Send() with nil recipients should return error")
	}
}

func TestEmailService_Send_InvalidEmailInList(t *testing.T) {
	svc := NewEmailService(EmailConfig{Host: "smtp.example.com", Port: 587}, newTestLogger())
	// Mix of valid and invalid — should fail on first invalid.
	err := svc.Send("test", "body", []string{"good@example.com", "not-an-email"})
	if err == nil {
		t.Error("Send() with invalid email in list should return error")
	}
}

func TestEmailService_Send_AllInvalidEmails(t *testing.T) {
	svc := NewEmailService(EmailConfig{Host: "smtp.example.com", Port: 587}, newTestLogger())
	err := svc.Send("test", "body", []string{"bad", "@missing.com", "user@"})
	if err == nil {
		t.Error("Send() with all invalid emails should return error")
	}
}

// ─── validateEmailAddress ────────────────────────────────────────────────

func TestValidateEmailAddress(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"simple", "user@example.com", true},
		{"subdomain", "admin@sub.domain.co.id", true},
		{"plus addressing", "user+tag@example.com", true},
		{"empty", "", false},
		{"no at sign", "noatsign", false},
		{"at start", "@example.com", false},
		{"at end", "user@", false},
		{"no dot in domain", "user@nodot", false},
		{"double at passes basic check", "user@@example.com", true},
		{"space in local passes basic check", "us er@example.com", true},
		{"address exactly 254 chars", string(make([]byte, 242)) + "@example.com", true},
		{"address over 254 chars", string(make([]byte, 250)) + "@example.com", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEmailAddress(tc.input)
			if tc.valid && err != nil {
				t.Errorf("validateEmailAddress(%q) unexpected error: %v", tc.input, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("validateEmailAddress(%q) should return error", tc.input)
			}
		})
	}
}

// ─── NewEmailService ────────────────────────────────────────────────────

func TestNewEmailService_NonNil(t *testing.T) {
	svc := NewEmailService(EmailConfig{}, newTestLogger())
	if svc == nil {
		t.Fatal("NewEmailService returned nil")
	}
	if svc.config.Host != "" {
		t.Errorf("expected empty host, got %q", svc.config.Host)
	}
}
