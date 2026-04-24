package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

const placeholderJWTSecret = "your-secret-key-change-in-production"

// Config 应用配置
type Config struct {
	Server      ServerConfig    `mapstructure:"server"`
	Site        SiteConfig      `mapstructure:"site"`
	Database    DatabaseConfig  `mapstructure:"database"`
	Redis       RedisConfig     `mapstructure:"redis"`
	RedisVector RedisConfig     `mapstructure:"redis_vector"`
	JWT         JWTConfig       `mapstructure:"jwt"`
	AI          AIConfig        `mapstructure:"ai"`
	OAuth       OAuthConfig     `mapstructure:"oauth"`
	Upload      UploadConfig    `mapstructure:"upload"`
	RateLimit   RateLimitConfig `mapstructure:"ratelimit"`
	Log         LogConfig       `mapstructure:"log"`
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
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// AIConfig AI 配置
type AIConfig struct {
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
	ResearchTimeoutSeconds int     `mapstructure:"research_timeout_seconds"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Global   int `mapstructure:"global"`
	Register int `mapstructure:"register"`
	Login    int `mapstructure:"login"`
	AIChat   int `mapstructure:"ai_chat"`
}

// LoadConfig 加载配置文件
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("../config")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.AutomaticEnv()

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
	_ = viper.BindEnv("ai.api_key", "DOUBAO_API_KEY")
	_ = viper.BindEnv("ai.endpoint", "DOUBAO_ENDPOINT")
	_ = viper.BindEnv("ai.llm_model", "DOUBAO_CHAT_MODEL")
	_ = viper.BindEnv("ai.embedding_model", "DOUBAO_EMBEDDING_MODEL")
	_ = viper.BindEnv("ai.research_endpoint", "RESEARCH_ENDPOINT")
	_ = viper.BindEnv("ai.research_api_key", "RESEARCH_API_KEY")
	_ = viper.BindEnv("upload.storage_path", "UPLOAD_PATH")
	_ = viper.BindEnv("site.url", "SITE_URL")

	_ = viper.BindEnv("oauth.github.client_id", "GITHUB_CLIENT_ID")
	_ = viper.BindEnv("oauth.github.client_secret", "GITHUB_CLIENT_SECRET")
	_ = viper.BindEnv("oauth.github.callback_url", "GITHUB_CALLBACK_URL")

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
	if cfg.JWT.Secret == placeholderJWTSecret {
		return nil, fmt.Errorf("invalid placeholder JWT secret: configure a non-placeholder JWT secret before startup")
	}

	return &cfg, nil
}
