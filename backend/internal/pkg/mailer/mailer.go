package mailer

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"strings"

	"wenDao/config"
)

var ErrNotConfigured = errors.New("email sender is not configured")

type SMTPEmailSender struct {
	cfg config.EmailConfig
}

func NewSMTPEmailSender(cfg config.EmailConfig) *SMTPEmailSender {
	return &SMTPEmailSender{cfg: cfg}
}

func (s *SMTPEmailSender) Send(ctx context.Context, toAddress string, subject string, textBody string, htmlBody string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.cfg.SMTPHost == "" || s.cfg.FromAddress == "" {
		return ErrNotConfigured
	}

	from := mail.Address{Name: s.cfg.FromName, Address: s.cfg.FromAddress}
	to := mail.Address{Address: toAddress}
	message := buildMessage(from, to, subject, textBody, htmlBody)

	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.smtpPort())
	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.SMTPHost)
	}
	if err := smtp.SendMail(addr, auth, s.cfg.FromAddress, []string{toAddress}, []byte(message)); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}

func (s *SMTPEmailSender) smtpPort() int {
	if s.cfg.SMTPPort > 0 {
		return s.cfg.SMTPPort
	}
	return 587
}

func buildMessage(from mail.Address, to mail.Address, subject string, textBody string, htmlBody string) string {
	headers := []string{
		"From: " + from.String(),
		"To: " + to.String(),
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"MIME-Version: 1.0",
	}

	if strings.TrimSpace(htmlBody) == "" {
		return strings.Join(append(headers,
			"Content-Type: text/plain; charset=UTF-8",
			"",
			textBody,
		), "\r\n")
	}

	const boundary = "wendao-mail-boundary"
	lines := append(headers,
		"Content-Type: multipart/alternative; boundary="+boundary,
		"",
		"--"+boundary,
		"Content-Type: text/plain; charset=UTF-8",
		"",
		textBody,
		"--"+boundary,
		"Content-Type: text/html; charset=UTF-8",
		"",
		htmlBody,
		"--"+boundary+"--",
	)
	return strings.Join(lines, "\r\n")
}
