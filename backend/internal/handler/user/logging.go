package user

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"go.uber.org/zap"
)

func (h *UserHandler) log() *zap.Logger {
	if h == nil || h.logger == nil {
		return zap.NewNop()
	}
	return h.logger
}

func userEmailLogFields(email string) []zap.Field {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	fields := []zap.Field{
		zap.String("email_hash", shortUserHash(normalizedEmail)),
	}
	if _, domain, ok := strings.Cut(normalizedEmail, "@"); ok {
		fields = append(fields, zap.String("email_domain", domain))
	}
	return fields
}

func shortUserHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	hash := hex.EncodeToString(sum[:])
	if len(hash) < 12 {
		return hash
	}
	return hash[:12]
}
