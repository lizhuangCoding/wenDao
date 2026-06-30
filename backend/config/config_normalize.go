package config

import "strings"

func normalizeConfig(cfg *Config) {
	if cfg == nil {
		return
	}

	normalizeAIConfig(&cfg.AI)
	normalizeMigrationConfig(&cfg.Migration)
	normalizeUploadConfig(&cfg.Upload)
	normalizeLogConfig(&cfg.Log)
	normalizeEmailConfig(&cfg.Email)
	normalizeVerificationConfig(&cfg.Verification)
	normalizeRateLimitConfig(&cfg.RateLimit)
}

func normalizeAIConfig(cfg *AIConfig) {
	if cfg == nil {
		return
	}
	if cfg.RAGMinScore <= 0 {
		cfg.RAGMinScore = 0.30
	}
	if strings.TrimSpace(cfg.Provider) == "" {
		cfg.Provider = "doubao"
	}
	if cfg.ResearchMaxResults <= 0 {
		cfg.ResearchMaxResults = 5
	}
	if cfg.ResearchTimeoutSeconds <= 0 {
		cfg.ResearchTimeoutSeconds = 15
	}
	if cfg.DailyTokenLimit <= 0 {
		cfg.DailyTokenLimit = 200000
	}
	if cfg.DailyRunLimit <= 0 {
		cfg.DailyRunLimit = 100
	}
	if strings.TrimSpace(cfg.CostCurrency) == "" {
		cfg.CostCurrency = "USD"
	}
}

func normalizeMigrationConfig(cfg *MigrationConfig) {
	if cfg == nil {
		return
	}
	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	if cfg.Mode == "" {
		cfg.Mode = "versioned"
	}
	if strings.TrimSpace(cfg.Path) == "" {
		cfg.Path = "migrations"
	}
}

func normalizeUploadConfig(cfg *UploadConfig) {
	if cfg == nil {
		return
	}
	if cfg.ImageQuality <= 0 || cfg.ImageQuality > 100 {
		cfg.ImageQuality = 80
	}
	if cfg.MaxImageWidth <= 0 {
		cfg.MaxImageWidth = 2560
	}
	if cfg.MaxImageHeight <= 0 {
		cfg.MaxImageHeight = 2560
	}
	if cfg.CleanupRetentionDays <= 0 {
		cfg.CleanupRetentionDays = 2
	}
	if cfg.CleanupIntervalHours <= 0 {
		cfg.CleanupIntervalHours = 24
	}
	if cfg.CleanupBatchSize <= 0 {
		cfg.CleanupBatchSize = 200
	}
}

func normalizeLogConfig(cfg *LogConfig) {
	if cfg == nil {
		return
	}
	if cfg.MaxSizeMB <= 0 {
		cfg.MaxSizeMB = 100
	}
	if cfg.MaxBackups <= 0 {
		cfg.MaxBackups = 7
	}
	if cfg.MaxAgeDays <= 0 {
		cfg.MaxAgeDays = 28
	}
}

func normalizeEmailConfig(cfg *EmailConfig) {
	if cfg == nil {
		return
	}
	if cfg.SMTPPort <= 0 {
		cfg.SMTPPort = 587
	}
	if strings.TrimSpace(cfg.FromName) == "" {
		cfg.FromName = "WenDao Blog"
	}
}

func normalizeVerificationConfig(cfg *VerificationConfig) {
	if cfg == nil {
		return
	}
	if cfg.CodeTTLMinutes <= 0 {
		cfg.CodeTTLMinutes = 10
	}
	if cfg.ResendCooldownSeconds <= 0 {
		cfg.ResendCooldownSeconds = 60
	}
	if cfg.MaxVerificationAttempts <= 0 {
		cfg.MaxVerificationAttempts = 5
	}
}

func normalizeRateLimitConfig(cfg *RateLimitConfig) {
	if cfg == nil {
		return
	}
	if cfg.VerificationCode <= 0 {
		cfg.VerificationCode = 3
	}
	if cfg.PasswordReset <= 0 {
		cfg.PasswordReset = 5
	}
	if cfg.Refresh <= 0 {
		cfg.Refresh = 30
	}
}
