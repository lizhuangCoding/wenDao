package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type configSection struct {
	name        string
	defaults    map[string]any
	envBindings map[string][]string
}

func registerConfigSections(v *viper.Viper) error {
	for _, section := range configSections() {
		for key, value := range section.defaults {
			v.SetDefault(key, value)
		}
		for key, envVars := range section.envBindings {
			if err := v.BindEnv(append([]string{key}, envVars...)...); err != nil {
				return fmt.Errorf("bind env for %s section key %s: %w", section.name, key, err)
			}
		}
	}
	return nil
}

func configSections() []configSection {
	return []configSection{
		serverConfigSection(),
		siteConfigSection(),
		databaseConfigSection(),
		migrationConfigSection(),
		redisConfigSection(),
		redisVectorConfigSection(),
		jwtConfigSection(),
		aiConfigSection(),
		oauthConfigSection(),
		uploadConfigSection(),
		rateLimitConfigSection(),
		emailConfigSection(),
		verificationConfigSection(),
		logConfigSection(),
	}
}

func serverConfigSection() configSection {
	return configSection{
		name: "server",
		envBindings: map[string][]string{
			"server.port": {"SERVER_PORT"},
			"server.mode": {"SERVER_MODE"},
		},
	}
}

func siteConfigSection() configSection {
	return configSection{
		name: "site",
		envBindings: map[string][]string{
			"site.url": {"SITE_URL"},
		},
	}
}

func databaseConfigSection() configSection {
	return configSection{
		name: "database",
		defaults: map[string]any{
			"database.host":   "localhost",
			"database.port":   "3306",
			"database.dbname": "wendao",
		},
		envBindings: map[string][]string{
			"database.host":     {"DB_HOST"},
			"database.port":     {"DB_PORT"},
			"database.user":     {"DB_USER"},
			"database.password": {"DB_PASSWORD"},
			"database.dbname":   {"DB_NAME"},
		},
	}
}

func migrationConfigSection() configSection {
	return configSection{
		name: "migration",
		defaults: map[string]any{
			"migration.mode": "versioned",
			"migration.path": "migrations",
		},
		envBindings: map[string][]string{
			"migration.mode": {"MIGRATION_MODE"},
			"migration.path": {"MIGRATION_PATH"},
		},
	}
}

func redisConfigSection() configSection {
	return configSection{
		name: "redis",
		defaults: map[string]any{
			"redis.host":      "localhost",
			"redis.port":      "6379",
			"redis.pool_size": 10,
		},
		envBindings: map[string][]string{
			"redis.host":     {"REDIS_HOST"},
			"redis.port":     {"REDIS_PORT"},
			"redis.password": {"REDIS_PASSWORD"},
		},
	}
}

func redisVectorConfigSection() configSection {
	return configSection{
		name: "redis_vector",
		defaults: map[string]any{
			"redis_vector.host":      "localhost",
			"redis_vector.port":      "6379",
			"redis_vector.pool_size": 5,
		},
		envBindings: map[string][]string{
			"redis_vector.host":     {"REDIS_VECTOR_HOST"},
			"redis_vector.port":     {"REDIS_VECTOR_PORT"},
			"redis_vector.password": {"REDIS_VECTOR_PASSWORD"},
		},
	}
}

func jwtConfigSection() configSection {
	return configSection{
		name: "jwt",
		defaults: map[string]any{
			"jwt.access_expire_hours": 1,
			"jwt.refresh_expire_days": 2,
		},
		envBindings: map[string][]string{
			"jwt.secret":              {"JWT_SECRET"},
			"jwt.access_expire_hours": {"JWT_ACCESS_EXPIRE_HOURS"},
			"jwt.refresh_expire_days": {"JWT_REFRESH_EXPIRE_DAYS"},
		},
	}
}

func aiConfigSection() configSection {
	return configSection{
		name: "ai",
		defaults: map[string]any{
			"ai.provider":                 "doubao",
			"ai.rag_min_score":            0.30,
			"ai.research_max_results":     5,
			"ai.research_timeout_seconds": 15,
			"ai.daily_token_limit":        200000,
			"ai.daily_run_limit":          100,
			"ai.cost_currency":            "USD",
		},
		envBindings: map[string][]string{
			"ai.provider":                {"AI_PROVIDER", "LLM_PROVIDER"},
			"ai.api_key":                 {"AI_API_KEY", "DOUBAO_API_KEY"},
			"ai.endpoint":                {"AI_ENDPOINT", "DOUBAO_ENDPOINT"},
			"ai.llm_model":               {"AI_CHAT_MODEL", "DOUBAO_CHAT_MODEL"},
			"ai.embedding_model":         {"AI_EMBEDDING_MODEL", "DOUBAO_EMBEDDING_MODEL"},
			"ai.research_endpoint":       {"RESEARCH_ENDPOINT"},
			"ai.research_api_key":        {"RESEARCH_API_KEY"},
			"ai.daily_token_limit":       {"AI_DAILY_TOKEN_LIMIT"},
			"ai.daily_run_limit":         {"AI_DAILY_RUN_LIMIT"},
			"ai.prompt_price_per_1k":     {"AI_PROMPT_PRICE_PER_1K"},
			"ai.completion_price_per_1k": {"AI_COMPLETION_PRICE_PER_1K"},
			"ai.cost_currency":           {"AI_COST_CURRENCY"},
		},
	}
}

func oauthConfigSection() configSection {
	return configSection{
		name: "oauth",
		envBindings: map[string][]string{
			"oauth.github.client_id":     {"GITHUB_CLIENT_ID"},
			"oauth.github.client_secret": {"GITHUB_CLIENT_SECRET"},
			"oauth.github.callback_url":  {"GITHUB_CALLBACK_URL"},
		},
	}
}

func uploadConfigSection() configSection {
	return configSection{
		name: "upload",
		defaults: map[string]any{
			"upload.max_size":               int64(10485760),
			"upload.allowed_types":          []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
			"upload.storage_path":           "./uploads",
			"upload.image_quality":          80,
			"upload.max_image_width":        2560,
			"upload.max_image_height":       2560,
			"upload.cleanup_enabled":        true,
			"upload.cleanup_retention_days": 2,
			"upload.cleanup_interval_hours": 24,
			"upload.cleanup_batch_size":     200,
		},
		envBindings: map[string][]string{
			"upload.max_size":               {"UPLOAD_MAX_SIZE"},
			"upload.storage_path":           {"UPLOAD_PATH"},
			"upload.cleanup_enabled":        {"UPLOAD_CLEANUP_ENABLED"},
			"upload.cleanup_retention_days": {"UPLOAD_CLEANUP_RETENTION_DAYS"},
			"upload.cleanup_interval_hours": {"UPLOAD_CLEANUP_INTERVAL_HOURS"},
			"upload.cleanup_batch_size":     {"UPLOAD_CLEANUP_BATCH_SIZE"},
		},
	}
}

func rateLimitConfigSection() configSection {
	return configSection{
		name: "ratelimit",
		defaults: map[string]any{
			"ratelimit.global":            100,
			"ratelimit.register":          5,
			"ratelimit.login":             10,
			"ratelimit.verification_code": 3,
			"ratelimit.password_reset":    5,
			"ratelimit.refresh":           30,
			"ratelimit.ai_chat":           10,
			"ratelimit.comment_create":    5,
		},
		envBindings: map[string][]string{
			"ratelimit.global":         {"RATELIMIT_GLOBAL"},
			"ratelimit.register":       {"RATELIMIT_REGISTER"},
			"ratelimit.login":          {"RATELIMIT_LOGIN"},
			"ratelimit.ai_chat":        {"RATELIMIT_AI_CHAT"},
			"ratelimit.comment_create": {"RATELIMIT_COMMENT_CREATE"},
		},
	}
}

func emailConfigSection() configSection {
	return configSection{
		name: "email",
		defaults: map[string]any{
			"email.smtp_port": 587,
			"email.from_name": "WenDao Blog",
		},
		envBindings: map[string][]string{
			"email.smtp_host":    {"EMAIL_SMTP_HOST"},
			"email.smtp_port":    {"EMAIL_SMTP_PORT"},
			"email.username":     {"EMAIL_USERNAME"},
			"email.password":     {"EMAIL_PASSWORD"},
			"email.from_address": {"EMAIL_FROM_ADDRESS"},
			"email.from_name":    {"EMAIL_FROM_NAME"},
		},
	}
}

func verificationConfigSection() configSection {
	return configSection{
		name: "verification",
		defaults: map[string]any{
			"verification.code_ttl_minutes":          10,
			"verification.resend_cooldown_seconds":   60,
			"verification.max_verification_attempts": 5,
		},
		envBindings: map[string][]string{
			"verification.code_ttl_minutes":          {"VERIFICATION_CODE_TTL_MINUTES"},
			"verification.resend_cooldown_seconds":   {"VERIFICATION_RESEND_COOLDOWN_SECONDS"},
			"verification.max_verification_attempts": {"VERIFICATION_MAX_ATTEMPTS"},
		},
	}
}

func logConfigSection() configSection {
	return configSection{
		name: "log",
		defaults: map[string]any{
			"log.max_size_mb":  100,
			"log.max_backups":  7,
			"log.max_age_days": 28,
			"log.compress":     true,
		},
		envBindings: map[string][]string{
			"log.level":        {"LOG_LEVEL"},
			"log.format":       {"LOG_FORMAT"},
			"log.output":       {"LOG_OUTPUT"},
			"log.max_size_mb":  {"LOG_MAX_SIZE_MB"},
			"log.max_backups":  {"LOG_MAX_BACKUPS"},
			"log.max_age_days": {"LOG_MAX_AGE_DAYS"},
			"log.compress":     {"LOG_COMPRESS"},
		},
	}
}
