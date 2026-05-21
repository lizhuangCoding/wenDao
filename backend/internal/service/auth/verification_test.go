package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"wenDao/config"
)

type memoryVerificationStore struct {
	records   map[string]verificationRecord
	cooldowns map[string]bool
}

func newMemoryVerificationStore() *memoryVerificationStore {
	return &memoryVerificationStore{
		records:   make(map[string]verificationRecord),
		cooldowns: make(map[string]bool),
	}
}

func (s *memoryVerificationStore) SetCode(_ context.Context, key string, record verificationRecord, _ time.Duration) error {
	s.records[key] = record
	return nil
}

func (s *memoryVerificationStore) GetCode(_ context.Context, key string) (verificationRecord, error) {
	record, ok := s.records[key]
	if !ok {
		return verificationRecord{}, ErrVerificationCodeInvalid
	}
	return record, nil
}

func (s *memoryVerificationStore) Delete(_ context.Context, key string) error {
	delete(s.records, key)
	return nil
}

func (s *memoryVerificationStore) ReserveCooldown(_ context.Context, key string, _ time.Duration) (bool, error) {
	if s.cooldowns[key] {
		return false, nil
	}
	s.cooldowns[key] = true
	return true, nil
}

type recordingVerificationSender struct {
	email   string
	purpose VerificationPurpose
	code    string
	ttl     time.Duration
	err     error
}

func (s *recordingVerificationSender) SendVerificationCode(_ context.Context, email string, purpose VerificationPurpose, code string, ttl time.Duration) error {
	if s.err != nil {
		return s.err
	}
	s.email = email
	s.purpose = purpose
	s.code = code
	s.ttl = ttl
	return nil
}

func TestVerificationServiceSendAndVerifyCodeConsumesRecord(t *testing.T) {
	store := newMemoryVerificationStore()
	sender := &recordingVerificationSender{}
	svc := newVerificationServiceWithDeps(config.VerificationConfig{
		CodeTTLMinutes:          10,
		ResendCooldownSeconds:   60,
		MaxVerificationAttempts: 5,
	}, "test-secret", store, sender, func() (string, error) {
		return "123456", nil
	})

	err := svc.SendCode(context.Background(), " User@Example.COM ", PurposeRegister)
	if err != nil {
		t.Fatalf("expected send to succeed, got %v", err)
	}
	if sender.email != "user@example.com" {
		t.Fatalf("expected normalized email, got %q", sender.email)
	}
	if sender.purpose != PurposeRegister || sender.code != "123456" {
		t.Fatalf("expected register code to be sent, got purpose=%q code=%q", sender.purpose, sender.code)
	}

	if len(store.records) != 1 {
		t.Fatalf("expected one stored verification record, got %d", len(store.records))
	}
	for _, record := range store.records {
		if record.CodeHash == "123456" {
			t.Fatalf("expected stored code to be hashed")
		}
	}

	if err := svc.VerifyCode(context.Background(), "user@example.com", PurposeRegister, "123456"); err != nil {
		t.Fatalf("expected verification to succeed, got %v", err)
	}
	if len(store.records) != 0 {
		t.Fatalf("expected verification record to be consumed")
	}
}

func TestVerificationServiceRejectsTooFrequentSend(t *testing.T) {
	store := newMemoryVerificationStore()
	sender := &recordingVerificationSender{}
	svc := newVerificationServiceWithDeps(config.VerificationConfig{
		CodeTTLMinutes:          10,
		ResendCooldownSeconds:   60,
		MaxVerificationAttempts: 5,
	}, "test-secret", store, sender, func() (string, error) {
		return "123456", nil
	})

	if err := svc.SendCode(context.Background(), "user@example.com", PurposePasswordReset); err != nil {
		t.Fatalf("expected first send to succeed, got %v", err)
	}

	err := svc.SendCode(context.Background(), "user@example.com", PurposePasswordReset)
	if !errors.Is(err, ErrVerificationCodeTooFrequent) {
		t.Fatalf("expected cooldown error, got %v", err)
	}
}
