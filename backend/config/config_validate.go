package config

import (
	"fmt"
	"strings"
)

func validateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	validators := []func(*Config) error{
		validateDatabaseConfig,
		validateMigrationConfig,
		validateRedisConfig,
		validateJWTConfig,
		validateUploadConfig,
		validateRateLimitConfig,
		validateLogConfig,
	}

	for _, validate := range validators {
		if err := validate(cfg); err != nil {
			return err
		}
	}
	return nil
}

func validateDatabaseConfig(cfg *Config) error {
	if strings.TrimSpace(cfg.Database.Host) == "" {
		return fmt.Errorf("invalid database.host: must not be empty")
	}
	if strings.TrimSpace(cfg.Database.Port) == "" {
		return fmt.Errorf("invalid database.port: must not be empty")
	}
	return nil
}

func validateMigrationConfig(cfg *Config) error {
	switch cfg.Migration.Mode {
	case "versioned", "auto", "disabled":
	default:
		return fmt.Errorf("invalid migration.mode: must be versioned, auto, or disabled")
	}
	if cfg.Migration.Mode == "versioned" && strings.TrimSpace(cfg.Migration.Path) == "" {
		return fmt.Errorf("invalid migration.path: must not be empty when migration.mode is versioned")
	}
	return nil
}

func validateRedisConfig(cfg *Config) error {
	if strings.TrimSpace(cfg.Redis.Host) == "" {
		return fmt.Errorf("invalid redis.host: must not be empty")
	}
	if strings.TrimSpace(cfg.Redis.Port) == "" {
		return fmt.Errorf("invalid redis.port: must not be empty")
	}
	return nil
}

func validateJWTConfig(cfg *Config) error {
	if cfg.JWT.AccessExpireHours <= 0 {
		return fmt.Errorf("invalid jwt.access_expire_hours: must be greater than 0")
	}
	return nil
}

func validateUploadConfig(cfg *Config) error {
	if cfg.Upload.MaxSize <= 0 {
		return fmt.Errorf("invalid upload.max_size: must be greater than 0")
	}
	if strings.TrimSpace(cfg.Upload.StoragePath) == "" {
		return fmt.Errorf("invalid upload.storage_path: must not be empty")
	}
	return nil
}

func validateRateLimitConfig(cfg *Config) error {
	if cfg.RateLimit.Global <= 0 {
		return fmt.Errorf("invalid ratelimit.global: must be greater than 0")
	}
	if cfg.RateLimit.Register <= 0 {
		return fmt.Errorf("invalid ratelimit.register: must be greater than 0")
	}
	if cfg.RateLimit.Login <= 0 {
		return fmt.Errorf("invalid ratelimit.login: must be greater than 0")
	}
	if cfg.RateLimit.AIChat <= 0 {
		return fmt.Errorf("invalid ratelimit.ai_chat: must be greater than 0")
	}
	return nil
}

func validateLogConfig(cfg *Config) error {
	switch cfg.Log.AccessLevel {
	case "warn", "info":
		return nil
	default:
		return fmt.Errorf("invalid log.access_level: must be warn or info")
	}
}
