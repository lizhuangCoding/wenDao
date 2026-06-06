package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

const placeholderJWTSecret = "your-secret-key-change-in-production"

// Config 应用配置
type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Site         SiteConfig         `mapstructure:"site"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Redis        RedisConfig        `mapstructure:"redis"`
	RedisVector  RedisConfig        `mapstructure:"redis_vector"`
	JWT          JWTConfig          `mapstructure:"jwt"`
	AI           AIConfig           `mapstructure:"ai"`
	OAuth        OAuthConfig        `mapstructure:"oauth"`
	Upload       UploadConfig       `mapstructure:"upload"`
	RateLimit    RateLimitConfig    `mapstructure:"ratelimit"`
	Email        EmailConfig        `mapstructure:"email"`
	Verification VerificationConfig `mapstructure:"verification"`
	Log          LogConfig          `mapstructure:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// SiteConfig 网站配置
type SiteConfig struct {
	Slogan string `mapstructure:"slogan"`
	URL    string `mapstructure:"url"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         string `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret            string `mapstructure:"secret"`
	AccessExpireHours int    `mapstructure:"access_expire_hours"`
	RefreshExpireDays int    `mapstructure:"refresh_expire_days"`
}

// OAuthConfig OAuth 配置
type OAuthConfig struct {
	GitHub GitHubOAuthConfig `mapstructure:"github"`
}

// GitHubOAuthConfig GitHub OAuth 配置
type GitHubOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	CallbackURL  string `mapstructure:"callback_url"`
}

// UploadConfig 上传配置
type UploadConfig struct {
	MaxSize                int64    `mapstructure:"max_size"`
	AllowedTypes           []string `mapstructure:"allowed_types"`
	StoragePath            string   `mapstructure:"storage_path"`
	EnableImageCompression bool     `mapstructure:"enable_image_compression"`
	ImageQuality           int      `mapstructure:"image_quality"`
	MaxImageWidth          int      `mapstructure:"max_image_width"`
	MaxImageHeight         int      `mapstructure:"max_image_height"`
	CleanupEnabled         bool     `mapstructure:"cleanup_enabled"`
	CleanupRetentionDays   int      `mapstructure:"cleanup_retention_days"`
	CleanupIntervalHours   int      `mapstructure:"cleanup_interval_hours"`
	CleanupBatchSize       int      `mapstructure:"cleanup_batch_size"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	MaxSizeMB  int    `mapstructure:"max_size_mb"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAgeDays int    `mapstructure:"max_age_days"`
	Compress   bool   `mapstructure:"compress"`
}

// ModelConfig 单个模型配置
type ModelConfig struct {
	Provider    string `mapstructure:"provider"`
	ModelName   string `mapstructure:"model_name"`
	DisplayName string `mapstructure:"display_name"`
	APIKey      string `mapstructure:"api_key"`
	BaseURL     string `mapstructure:"base_url"`
}

// AIConfig AI 配置
type AIConfig struct {
	Provider               string  `mapstructure:"provider"`
	APIKey                 string  `mapstructure:"api_key"`
	Endpoint               string  `mapstructure:"endpoint"`
	EmbeddingModel         string  `mapstructure:"embedding_model"`
	LLMModel               string  `mapstructure:"llm_model"`
	Temperature            float32 `mapstructure:"temperature"`
	MaxTokens              int     `mapstructure:"max_tokens"`
	TopK                   int     `mapstructure:"top_k"`
	RAGMinScore            float32 `mapstructure:"rag_min_score"`
	ResearchEndpoint       string  `mapstructure:"research_endpoint"`
	ResearchAPIKey         string  `mapstructure:"research_api_key"`
	ResearchMaxResults     int     `mapstructure:"research_max_results"`
	ResearchTimeoutSeconds int           `mapstructure:"research_timeout_seconds"`
	Models                 []ModelConfig `mapstructure:"models"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Global           int `mapstructure:"global"`
	Register         int `mapstructure:"register"`
	Login            int `mapstructure:"login"`
	VerificationCode int `mapstructure:"verification_code"`
	PasswordReset    int `mapstructure:"password_reset"`
	Refresh          int `mapstructure:"refresh"`
	AIChat           int `mapstructure:"ai_chat"`
}

// EmailConfig 邮件发送配置
type EmailConfig struct {
	SMTPHost    string `mapstructure:"smtp_host"`
	SMTPPort    int    `mapstructure:"smtp_port"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	FromAddress string `mapstructure:"from_address"`
	FromName    string `mapstructure:"from_name"`
}

// VerificationConfig 邮箱验证码配置
type VerificationConfig struct {
	CodeTTLMinutes          int `mapstructure:"code_ttl_minutes"`
	ResendCooldownSeconds   int `mapstructure:"resend_cooldown_seconds"`
	MaxVerificationAttempts int `mapstructure:"max_verification_attempts"`
}

// LoadConfig 加载配置文件
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("../config")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.AutomaticEnv()
	viper.SetDefault("log.max_size_mb", 100)
	viper.SetDefault("log.max_backups", 7)
	viper.SetDefault("log.max_age_days", 28)
	viper.SetDefault("log.compress", true)
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "3306")
	viper.SetDefault("database.dbname", "wendao")
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis_vector.host", "localhost")
	viper.SetDefault("redis_vector.port", "6379")
	viper.SetDefault("redis_vector.pool_size", 5)
	viper.SetDefault("jwt.access_expire_hours", 1)
	viper.SetDefault("jwt.refresh_expire_days", 2)
	viper.SetDefault("upload.max_size", 10485760)
	viper.SetDefault("upload.allowed_types", []string{"image/jpeg", "image/png", "image/gif", "image/webp"})
	viper.SetDefault("upload.storage_path", "./uploads")
	viper.SetDefault("ratelimit.global", 100)
	viper.SetDefault("ratelimit.register", 5)
	viper.SetDefault("ratelimit.login", 10)
	viper.SetDefault("ratelimit.ai_chat", 10)
	viper.SetDefault("email.smtp_port", 587)
	viper.SetDefault("email.from_name", "WenDao Blog")
	viper.SetDefault("verification.code_ttl_minutes", 10)
	viper.SetDefault("verification.resend_cooldown_seconds", 60)
	viper.SetDefault("verification.max_verification_attempts", 5)
	viper.SetDefault("ratelimit.verification_code", 3)
	viper.SetDefault("ratelimit.password_reset", 5)
	viper.SetDefault("ratelimit.refresh", 30)
	viper.SetDefault("upload.cleanup_enabled", true)
	viper.SetDefault("upload.cleanup_retention_days", 2)
	viper.SetDefault("upload.cleanup_interval_hours", 24)
	viper.SetDefault("upload.cleanup_batch_size", 200)
	viper.SetDefault("ai.provider", "doubao")

	_ = viper.BindEnv("database.host", "DB_HOST")
	_ = viper.BindEnv("database.port", "DB_PORT")
	_ = viper.BindEnv("database.user", "DB_USER")
	_ = viper.BindEnv("database.password", "DB_PASSWORD")
	_ = viper.BindEnv("database.dbname", "DB_NAME")

	_ = viper.BindEnv("redis.host", "REDIS_HOST")
	_ = viper.BindEnv("redis.port", "REDIS_PORT")
	_ = viper.BindEnv("redis.password", "REDIS_PASSWORD")

	_ = viper.BindEnv("redis_vector.host", "REDIS_VECTOR_HOST")
	_ = viper.BindEnv("redis_vector.port", "REDIS_VECTOR_PORT")
	_ = viper.BindEnv("redis_vector.password", "REDIS_VECTOR_PASSWORD")

	_ = viper.BindEnv("jwt.secret", "JWT_SECRET")
	_ = viper.BindEnv("jwt.access_expire_hours", "JWT_ACCESS_EXPIRE_HOURS")
	_ = viper.BindEnv("jwt.refresh_expire_days", "JWT_REFRESH_EXPIRE_DAYS")
	_ = viper.BindEnv("ai.provider", "AI_PROVIDER", "LLM_PROVIDER")
	_ = viper.BindEnv("ai.api_key", "AI_API_KEY", "DOUBAO_API_KEY")
	_ = viper.BindEnv("ai.endpoint", "AI_ENDPOINT", "DOUBAO_ENDPOINT")
	_ = viper.BindEnv("ai.llm_model", "AI_CHAT_MODEL", "DOUBAO_CHAT_MODEL")
	_ = viper.BindEnv("ai.embedding_model", "AI_EMBEDDING_MODEL", "DOUBAO_EMBEDDING_MODEL")
	_ = viper.BindEnv("ai.research_endpoint", "RESEARCH_ENDPOINT")
	_ = viper.BindEnv("ai.research_api_key", "RESEARCH_API_KEY")
	_ = viper.BindEnv("upload.max_size", "UPLOAD_MAX_SIZE")
	_ = viper.BindEnv("upload.storage_path", "UPLOAD_PATH")
	_ = viper.BindEnv("upload.cleanup_enabled", "UPLOAD_CLEANUP_ENABLED")
	_ = viper.BindEnv("upload.cleanup_retention_days", "UPLOAD_CLEANUP_RETENTION_DAYS")
	_ = viper.BindEnv("upload.cleanup_interval_hours", "UPLOAD_CLEANUP_INTERVAL_HOURS")
	_ = viper.BindEnv("upload.cleanup_batch_size", "UPLOAD_CLEANUP_BATCH_SIZE")
	_ = viper.BindEnv("ratelimit.global", "RATELIMIT_GLOBAL")
	_ = viper.BindEnv("ratelimit.register", "RATELIMIT_REGISTER")
	_ = viper.BindEnv("ratelimit.login", "RATELIMIT_LOGIN")
	_ = viper.BindEnv("ratelimit.ai_chat", "RATELIMIT_AI_CHAT")
	_ = viper.BindEnv("site.url", "SITE_URL")
	_ = viper.BindEnv("log.level", "LOG_LEVEL")
	_ = viper.BindEnv("log.format", "LOG_FORMAT")
	_ = viper.BindEnv("log.output", "LOG_OUTPUT")
	_ = viper.BindEnv("log.max_size_mb", "LOG_MAX_SIZE_MB")
	_ = viper.BindEnv("log.max_backups", "LOG_MAX_BACKUPS")
	_ = viper.BindEnv("log.max_age_days", "LOG_MAX_AGE_DAYS")
	_ = viper.BindEnv("log.compress", "LOG_COMPRESS")

	_ = viper.BindEnv("oauth.github.client_id", "GITHUB_CLIENT_ID")
	_ = viper.BindEnv("oauth.github.client_secret", "GITHUB_CLIENT_SECRET")
	_ = viper.BindEnv("oauth.github.callback_url", "GITHUB_CALLBACK_URL")
	_ = viper.BindEnv("email.smtp_host", "EMAIL_SMTP_HOST")
	_ = viper.BindEnv("email.smtp_port", "EMAIL_SMTP_PORT")
	_ = viper.BindEnv("email.username", "EMAIL_USERNAME")
	_ = viper.BindEnv("email.password", "EMAIL_PASSWORD")
	_ = viper.BindEnv("email.from_address", "EMAIL_FROM_ADDRESS")
	_ = viper.BindEnv("email.from_name", "EMAIL_FROM_NAME")
	_ = viper.BindEnv("verification.code_ttl_minutes", "VERIFICATION_CODE_TTL_MINUTES")
	_ = viper.BindEnv("verification.resend_cooldown_seconds", "VERIFICATION_RESEND_COOLDOWN_SECONDS")
	_ = viper.BindEnv("verification.max_verification_attempts", "VERIFICATION_MAX_ATTEMPTS")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.AI.RAGMinScore <= 0 {
		cfg.AI.RAGMinScore = 0.30
	}
	if strings.TrimSpace(cfg.AI.Provider) == "" {
		cfg.AI.Provider = "doubao"
	}
	if cfg.AI.ResearchMaxResults <= 0 {
		cfg.AI.ResearchMaxResults = 5
	}
	if cfg.AI.ResearchTimeoutSeconds <= 0 {
		cfg.AI.ResearchTimeoutSeconds = 15
	}
	if cfg.Upload.ImageQuality <= 0 || cfg.Upload.ImageQuality > 100 {
		cfg.Upload.ImageQuality = 80
	}
	if cfg.Upload.MaxImageWidth <= 0 {
		cfg.Upload.MaxImageWidth = 2560
	}
	if cfg.Upload.MaxImageHeight <= 0 {
		cfg.Upload.MaxImageHeight = 2560
	}
	if cfg.Upload.CleanupRetentionDays <= 0 {
		cfg.Upload.CleanupRetentionDays = 2
	}
	if cfg.Upload.CleanupIntervalHours <= 0 {
		cfg.Upload.CleanupIntervalHours = 24
	}
	if cfg.Upload.CleanupBatchSize <= 0 {
		cfg.Upload.CleanupBatchSize = 200
	}
	if cfg.Log.MaxSizeMB <= 0 {
		cfg.Log.MaxSizeMB = 100
	}
	if cfg.Log.MaxBackups <= 0 {
		cfg.Log.MaxBackups = 7
	}
	if cfg.Log.MaxAgeDays <= 0 {
		cfg.Log.MaxAgeDays = 28
	}
	if cfg.Email.SMTPPort <= 0 {
		cfg.Email.SMTPPort = 587
	}
	if cfg.Email.FromName == "" {
		cfg.Email.FromName = "WenDao Blog"
	}
	if cfg.Verification.CodeTTLMinutes <= 0 {
		cfg.Verification.CodeTTLMinutes = 10
	}
	if cfg.Verification.ResendCooldownSeconds <= 0 {
		cfg.Verification.ResendCooldownSeconds = 60
	}
	if cfg.Verification.MaxVerificationAttempts <= 0 {
		cfg.Verification.MaxVerificationAttempts = 5
	}
	if cfg.RateLimit.VerificationCode <= 0 {
		cfg.RateLimit.VerificationCode = 3
	}
	if cfg.RateLimit.PasswordReset <= 0 {
		cfg.RateLimit.PasswordReset = 5
	}
	if cfg.RateLimit.Refresh <= 0 {
		cfg.RateLimit.Refresh = 30
	}
	if cfg.JWT.Secret == placeholderJWTSecret {
		return nil, fmt.Errorf("invalid placeholder JWT secret: configure a non-placeholder JWT secret before startup")
	}
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}
	if strings.TrimSpace(cfg.Database.Host) == "" {
		return fmt.Errorf("invalid database.host: must not be empty")
	}
	if strings.TrimSpace(cfg.Database.Port) == "" {
		return fmt.Errorf("invalid database.port: must not be empty")
	}
	if strings.TrimSpace(cfg.Redis.Host) == "" {
		return fmt.Errorf("invalid redis.host: must not be empty")
	}
	if strings.TrimSpace(cfg.Redis.Port) == "" {
		return fmt.Errorf("invalid redis.port: must not be empty")
	}
	if cfg.JWT.AccessExpireHours <= 0 {
		return fmt.Errorf("invalid jwt.access_expire_hours: must be greater than 0")
	}
	if cfg.Upload.MaxSize <= 0 {
		return fmt.Errorf("invalid upload.max_size: must be greater than 0")
	}
	if strings.TrimSpace(cfg.Upload.StoragePath) == "" {
		return fmt.Errorf("invalid upload.storage_path: must not be empty")
	}
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
