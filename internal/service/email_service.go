package service

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"go.uber.org/zap"
)

// EmailConfig holds SMTP connection parameters loaded from environment.
type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
}

// EmailService sends transactional emails via SMTP.
// Uses net/smtp (stdlib) — zero external dependencies.
type EmailService struct {
	config EmailConfig
	log    *zap.SugaredLogger
}

func NewEmailService(config EmailConfig, log *zap.SugaredLogger) *EmailService {
	return &EmailService{
		config: config,
		log:    log,
	}
}

// IsConfigured returns true when host and port are set (non-zero).
func (s *EmailService) IsConfigured() bool {
	return s.config.Host != "" && s.config.Port > 0
}

// Send sends a plain-text email to the given recipients.
// Recipients must be valid email addresses; they are validated before sending.
func (s *EmailService) Send(subject string, body string, to []string) error {
	if !s.IsConfigured() {
		s.log.Warnw("email service not configured, skipping send", "subject", subject)
		return nil
	}

	if len(to) == 0 {
		return fmt.Errorf("email: no recipients")
	}

	for _, addr := range to {
		if err := validateEmailAddress(addr); err != nil {
			return fmt.Errorf("email: invalid recipient %q: %w", addr, err)
		}
	}

	from := s.config.From
	if from == "" {
		from = s.config.Username
	}
	fromName := s.config.FromName
	if fromName == "" {
		fromName = "SIMBA"
	}

	header := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n",
		fromName, from, strings.Join(to, ", "), subject)

	msg := []byte(header + body)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// Try STARTTLS-capable connection first, fall back to plain SMTP.
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)

	if s.config.Port == 465 {
		return s.sendWithTLS(addr, from, to, msg, auth)
	}

	return s.sendWithSTARTTLS(addr, from, to, msg, auth)
}

// sendWithSTARTTLS connects on port 25/587 and upgrades via STARTTLS.
func (s *EmailService) sendWithSTARTTLS(addr, from string, to []string, msg []byte, auth smtp.Auth) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("email: dial %s: %w", addr, err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: s.config.Host}
		if err = client.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("email: STARTTLS: %w", err)
		}
	}

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("email: auth: %w", err)
	}

	if err = client.Mail(from); err != nil {
		return fmt.Errorf("email: MAIL FROM: %w", err)
	}

	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("email: RCPT TO %s: %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("email: DATA: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("email: write body: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("email: close data: %w", err)
	}

	return client.Quit()
}

// sendWithTLS connects directly over TLS (port 465).
func (s *EmailService) sendWithTLS(addr, from string, to []string, msg []byte, auth smtp.Auth) error {
	tlsCfg := &tls.Config{ServerName: s.config.Host}

	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("email: TLS dial %s: %w", addr, err)
	}

	host, _, _ := net.SplitHostPort(addr)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("email: new client: %w", err)
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("email: auth: %w", err)
	}
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("email: MAIL FROM: %w", err)
	}
	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
			return fmt.Errorf("email: RCPT TO %s: %w", addr, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("email: DATA: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("email: write body: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("email: close data: %w", err)
	}
	return client.Quit()
}

// validateEmailAddress performs basic RFC 5322 address validation.
func validateEmailAddress(addr string) error {
	if addr == "" {
		return fmt.Errorf("empty address")
	}
	if len(addr) > 254 {
		return fmt.Errorf("address too long")
	}
	parts := strings.SplitN(addr, "@", 2)
	if len(parts) != 2 {
		return fmt.Errorf("missing @")
	}
	local, domain := parts[0], parts[1]
	if local == "" || domain == "" {
		return fmt.Errorf("empty local or domain part")
	}
	if !strings.Contains(domain, ".") {
		return fmt.Errorf("domain must contain a dot")
	}
	return nil
}
