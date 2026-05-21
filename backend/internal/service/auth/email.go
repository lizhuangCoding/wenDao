package auth

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"wenDao/config"
)

var ErrVerificationEmailNotConfigured = errors.New("verification email sender is not configured")

type smtpVerificationEmailSender struct {
	cfg config.EmailConfig
}

func NewSMTPVerificationEmailSender(cfg config.EmailConfig) VerificationEmailSender {
	return &smtpVerificationEmailSender{cfg: cfg}
}

func (s *smtpVerificationEmailSender) SendVerificationCode(ctx context.Context, email string, purpose VerificationPurpose, code string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.cfg.SMTPHost == "" || s.cfg.FromAddress == "" {
		return ErrVerificationEmailNotConfigured
	}

	from := mail.Address{Name: s.cfg.FromName, Address: s.cfg.FromAddress}
	to := mail.Address{Address: email}
	subject := verificationEmailSubject(purpose)
	body := verificationEmailBody(purpose, code, ttl)
	message := strings.Join([]string{
		"From: " + from.String(),
		"To: " + to.String(),
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.smtpPort())
	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.SMTPHost)
	}
	if err := smtp.SendMail(addr, auth, s.cfg.FromAddress, []string{email}, []byte(message)); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}
	return nil
}

func (s *smtpVerificationEmailSender) smtpPort() int {
	if s.cfg.SMTPPort > 0 {
		return s.cfg.SMTPPort
	}
	return 587
}

func verificationEmailSubject(purpose VerificationPurpose) string {
	switch purpose {
	case PurposePasswordReset:
		return "wenDao 密码重置验证码"
	default:
		return "wenDao 注册验证码"
	}
}

func verificationEmailBody(purpose VerificationPurpose, code string, ttl time.Duration) string {
	action := "完成注册"
	if purpose == PurposePasswordReset {
		action = "重置密码"
	}
	minutes := int(ttl.Minutes())
	if minutes <= 0 {
		minutes = 1
	}
	return fmt.Sprintf("你好，\n\n你的 wenDao 验证码是：%s\n\n请在 %d 分钟内使用它%s。如果不是你本人操作，可以忽略这封邮件。\n", code, minutes, action)
}
