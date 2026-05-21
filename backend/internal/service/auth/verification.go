package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"wenDao/config"
)

type VerificationPurpose string

const (
	PurposeRegister      VerificationPurpose = "register"
	PurposePasswordReset VerificationPurpose = "password_reset"
)

var (
	ErrVerificationUnavailable     = errors.New("verification service unavailable")
	ErrVerificationCodeInvalid     = errors.New("invalid verification code")
	ErrVerificationCodeTooFrequent = errors.New("verification code requested too frequently")
)

type VerificationService interface {
	SendCode(ctx context.Context, email string, purpose VerificationPurpose) error
	VerifyCode(ctx context.Context, email string, purpose VerificationPurpose, code string) error
}

type VerificationEmailSender interface {
	SendVerificationCode(ctx context.Context, email string, purpose VerificationPurpose, code string, ttl time.Duration) error
}

type verificationStore interface {
	SetCode(ctx context.Context, key string, record verificationRecord, ttl time.Duration) error
	GetCode(ctx context.Context, key string) (verificationRecord, error)
	Delete(ctx context.Context, key string) error
	ReserveCooldown(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

type verificationRecord struct {
	CodeHash  string    `json:"code_hash"`
	Attempts  int       `json:"attempts"`
	ExpiresAt time.Time `json:"expires_at"`
}

type verificationService struct {
	cfg           config.VerificationConfig
	secret        string
	store         verificationStore
	sender        VerificationEmailSender
	codeGenerator func() (string, error)
	logger        *zap.Logger
}

func NewVerificationService(cfg *config.Config, rdb *redis.Client, sender VerificationEmailSender) VerificationService {
	if cfg == nil {
		cfg = &config.Config{}
	}
	if sender == nil {
		sender = NewSMTPVerificationEmailSender(cfg.Email)
	}

	var store verificationStore
	if rdb != nil {
		store = &redisVerificationStore{rdb: rdb}
	}

	return newVerificationServiceWithDepsAndLogger(cfg.Verification, cfg.JWT.Secret, store, sender, generateNumericCode, zap.L())
}

func newVerificationServiceWithDeps(
	cfg config.VerificationConfig,
	secret string,
	store verificationStore,
	sender VerificationEmailSender,
	codeGenerator func() (string, error),
) *verificationService {
	return newVerificationServiceWithDepsAndLogger(cfg, secret, store, sender, codeGenerator, zap.NewNop())
}

func newVerificationServiceWithDepsAndLogger(
	cfg config.VerificationConfig,
	secret string,
	store verificationStore,
	sender VerificationEmailSender,
	codeGenerator func() (string, error),
	logger *zap.Logger,
) *verificationService {
	cfg = normalizeVerificationConfig(cfg)
	if logger == nil {
		logger = zap.NewNop()
	}
	return &verificationService{
		cfg:           cfg,
		secret:        secret,
		store:         store,
		sender:        sender,
		codeGenerator: codeGenerator,
		logger:        logger,
	}
}

func (s *verificationService) SendCode(ctx context.Context, email string, purpose VerificationPurpose) error {
	if s == nil || s.store == nil || s.sender == nil || s.secret == "" {
		if s != nil {
			s.log().Warn("Verification service unavailable",
				zap.Bool("store_configured", s.store != nil),
				zap.Bool("sender_configured", s.sender != nil),
				zap.Bool("secret_configured", s.secret != ""),
			)
		}
		return ErrVerificationUnavailable
	}
	normalizedEmail := normalizeVerificationEmail(email)
	if normalizedEmail == "" || !isKnownVerificationPurpose(purpose) {
		return ErrVerificationCodeInvalid
	}
	fields := verificationLogFields(normalizedEmail, purpose)

	cooldownKey := s.cooldownKey(normalizedEmail, purpose)
	cooldownReserved, err := s.store.ReserveCooldown(ctx, cooldownKey, s.cooldown())
	if err != nil {
		s.log().Warn("Verification cooldown reservation failed", append(fields, zap.Error(err))...)
		return fmt.Errorf("%w: %v", ErrVerificationUnavailable, err)
	}
	if !cooldownReserved {
		s.log().Info("Verification code send throttled", fields...)
		return ErrVerificationCodeTooFrequent
	}

	code, err := s.codeGenerator()
	if err != nil {
		_ = s.store.Delete(ctx, cooldownKey)
		s.log().Error("Verification code generation failed", append(fields, zap.Error(err))...)
		return fmt.Errorf("failed to generate verification code: %w", err)
	}

	ttl := s.ttl()
	record := verificationRecord{
		CodeHash:  s.hashCode(normalizedEmail, purpose, code),
		ExpiresAt: time.Now().Add(ttl),
	}
	if err := s.store.SetCode(ctx, s.codeKey(normalizedEmail, purpose), record, ttl); err != nil {
		_ = s.store.Delete(ctx, cooldownKey)
		s.log().Warn("Verification code store failed", append(fields, zap.Error(err))...)
		return fmt.Errorf("%w: %v", ErrVerificationUnavailable, err)
	}

	if err := s.sender.SendVerificationCode(ctx, normalizedEmail, purpose, code, ttl); err != nil {
		_ = s.store.Delete(ctx, s.codeKey(normalizedEmail, purpose))
		_ = s.store.Delete(ctx, cooldownKey)
		s.log().Error("Verification email send failed", append(fields, zap.Error(err))...)
		return err
	}
	s.log().Info("Verification email sent", append(fields, zap.Duration("ttl", ttl))...)
	return nil
}

func (s *verificationService) VerifyCode(ctx context.Context, email string, purpose VerificationPurpose, code string) error {
	if s == nil || s.store == nil || s.secret == "" {
		return ErrVerificationUnavailable
	}
	normalizedEmail := normalizeVerificationEmail(email)
	normalizedCode := strings.TrimSpace(code)
	if normalizedEmail == "" || normalizedCode == "" || !isKnownVerificationPurpose(purpose) {
		return ErrVerificationCodeInvalid
	}

	key := s.codeKey(normalizedEmail, purpose)
	record, err := s.store.GetCode(ctx, key)
	if err != nil {
		if errors.Is(err, ErrVerificationCodeInvalid) {
			return ErrVerificationCodeInvalid
		}
		return fmt.Errorf("%w: %v", ErrVerificationUnavailable, err)
	}

	if time.Now().After(record.ExpiresAt) || record.Attempts >= s.maxAttempts() {
		_ = s.store.Delete(ctx, key)
		return ErrVerificationCodeInvalid
	}

	expectedHash := s.hashCode(normalizedEmail, purpose, normalizedCode)
	if subtle.ConstantTimeCompare([]byte(record.CodeHash), []byte(expectedHash)) != 1 {
		record.Attempts++
		if record.Attempts >= s.maxAttempts() {
			_ = s.store.Delete(ctx, key)
		} else {
			_ = s.store.SetCode(ctx, key, record, time.Until(record.ExpiresAt))
		}
		return ErrVerificationCodeInvalid
	}

	return s.store.Delete(ctx, key)
}

func (s *verificationService) codeKey(email string, purpose VerificationPurpose) string {
	return "auth:verification:" + string(purpose) + ":" + hashKey(email)
}

func (s *verificationService) cooldownKey(email string, purpose VerificationPurpose) string {
	return "auth:verification-cooldown:" + string(purpose) + ":" + hashKey(email)
}

func (s *verificationService) hashCode(email string, purpose VerificationPurpose, code string) string {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(string(purpose)))
	mac.Write([]byte{0})
	mac.Write([]byte(email))
	mac.Write([]byte{0})
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *verificationService) ttl() time.Duration {
	return time.Duration(s.cfg.CodeTTLMinutes) * time.Minute
}

func (s *verificationService) cooldown() time.Duration {
	return time.Duration(s.cfg.ResendCooldownSeconds) * time.Second
}

func (s *verificationService) maxAttempts() int {
	return s.cfg.MaxVerificationAttempts
}

func (s *verificationService) log() *zap.Logger {
	if s == nil || s.logger == nil {
		return zap.NewNop()
	}
	return s.logger
}

func normalizeVerificationConfig(cfg config.VerificationConfig) config.VerificationConfig {
	if cfg.CodeTTLMinutes <= 0 {
		cfg.CodeTTLMinutes = 10
	}
	if cfg.ResendCooldownSeconds <= 0 {
		cfg.ResendCooldownSeconds = 60
	}
	if cfg.MaxVerificationAttempts <= 0 {
		cfg.MaxVerificationAttempts = 5
	}
	return cfg
}

func normalizeVerificationEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isKnownVerificationPurpose(purpose VerificationPurpose) bool {
	return purpose == PurposeRegister || purpose == PurposePasswordReset
}

func hashKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func verificationLogFields(email string, purpose VerificationPurpose) []zap.Field {
	fields := []zap.Field{
		zap.String("purpose", string(purpose)),
	}
	if email != "" {
		fields = append(fields,
			zap.String("email_hash", shortVerificationHash(email)),
			zap.String("email_domain", verificationEmailDomain(email)),
		)
	}
	return fields
}

func shortVerificationHash(value string) string {
	hash := hashKey(normalizeVerificationEmail(value))
	if len(hash) < 12 {
		return hash
	}
	return hash[:12]
}

func verificationEmailDomain(email string) string {
	_, domain, ok := strings.Cut(normalizeVerificationEmail(email), "@")
	if !ok {
		return ""
	}
	return domain
}

func generateNumericCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

type redisVerificationStore struct {
	rdb *redis.Client
}

func (s *redisVerificationStore) SetCode(ctx context.Context, key string, record verificationRecord, ttl time.Duration) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, key, data, ttl).Err()
}

func (s *redisVerificationStore) GetCode(ctx context.Context, key string) (verificationRecord, error) {
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return verificationRecord{}, ErrVerificationCodeInvalid
		}
		return verificationRecord{}, err
	}

	var record verificationRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return verificationRecord{}, err
	}
	return record, nil
}

func (s *redisVerificationStore) Delete(ctx context.Context, key string) error {
	return s.rdb.Del(ctx, key).Err()
}

func (s *redisVerificationStore) ReserveCooldown(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return s.rdb.SetNX(ctx, key, "1", ttl).Result()
}
