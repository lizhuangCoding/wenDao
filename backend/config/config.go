package config

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
	Migration    MigrationConfig    `mapstructure:"migration"`
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

// MigrationConfig controls database schema migration behavior.
type MigrationConfig struct {
	Mode string `mapstructure:"mode"`
	Path string `mapstructure:"path"`
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
	Provider               string        `mapstructure:"provider"`
	APIKey                 string        `mapstructure:"api_key"`
	Endpoint               string        `mapstructure:"endpoint"`
	EmbeddingModel         string        `mapstructure:"embedding_model"`
	LLMModel               string        `mapstructure:"llm_model"`
	Temperature            float32       `mapstructure:"temperature"`
	MaxTokens              int           `mapstructure:"max_tokens"`
	TopK                   int           `mapstructure:"top_k"`
	RAGMinScore            float32       `mapstructure:"rag_min_score"`
	ResearchEndpoint       string        `mapstructure:"research_endpoint"`
	ResearchAPIKey         string        `mapstructure:"research_api_key"`
	ResearchMaxResults     int           `mapstructure:"research_max_results"`
	ResearchTimeoutSeconds int           `mapstructure:"research_timeout_seconds"`
	DailyTokenLimit        int64         `mapstructure:"daily_token_limit"`
	DailyRunLimit          int           `mapstructure:"daily_run_limit"`
	PromptPricePer1K       float64       `mapstructure:"prompt_price_per_1k"`
	CompletionPricePer1K   float64       `mapstructure:"completion_price_per_1k"`
	CostCurrency           string        `mapstructure:"cost_currency"`
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
	CommentCreate    int `mapstructure:"comment_create"`
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
