package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadConfig_RejectsPlaceholderJWTSecret(t *testing.T) {
	viper.Reset()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	defer viper.Reset()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `server:
  port: "8089"
  mode: "debug"
site:
  slogan: "test"
  url: "http://localhost:3000"
jwt:
  secret: "your-secret-key-change-in-production"
  access_expire_hours: 1
  refresh_expire_days: 7
upload:
  max_size: 10485760
  allowed_types:
    - "image/jpeg"
  storage_path: "./uploads"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	cfg, err := LoadConfig()
	if err == nil {
		t.Fatalf("expected placeholder JWT secret to be rejected, got config %+v", cfg)
	}
	if !strings.Contains(err.Error(), "placeholder JWT secret") {
		t.Fatalf("expected placeholder JWT secret error, got %v", err)
	}
}

func TestLoadConfig_BindsSiteURLFromEnv(t *testing.T) {
	viper.Reset()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	defer viper.Reset()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `server:
  port: "8089"
  mode: "debug"
site:
  slogan: "test"
  url: "http://localhost:3000"
jwt:
  secret: "real-test-secret"
  access_expire_hours: 1
  refresh_expire_days: 7
upload:
  max_size: 10485760
  allowed_types:
    - "image/jpeg"
  storage_path: "./uploads"
oauth:
  github:
    client_id: ""
    client_secret: ""
    callback_url: ""
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	oldSiteURL, hadSiteURL := os.LookupEnv("SITE_URL")
	_ = os.Setenv("SITE_URL", "https://frontend.example.com")
	defer func() {
		if hadSiteURL {
			_ = os.Setenv("SITE_URL", oldSiteURL)
			return
		}
		_ = os.Unsetenv("SITE_URL")
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected config to load, got %v", err)
	}
	if cfg.Site.URL != "https://frontend.example.com" {
		t.Fatalf("expected site.url from env binding, got %q", cfg.Site.URL)
	}
}

func TestLoadConfig_BindsServerAndLogSettingsFromEnv(t *testing.T) {
	viper.Reset()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	defer viper.Reset()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `server:
  port: "8089"
  mode: "debug"
site:
  slogan: "test"
  url: "http://localhost:3000"
jwt:
  secret: "real-test-secret"
  access_expire_hours: 1
  refresh_expire_days: 7
upload:
  max_size: 10485760
  allowed_types:
    - "image/jpeg"
  storage_path: "./uploads"
log:
  level: "info"
  format: "console"
  output: "log/"
  max_size_mb: 100
  max_backups: 7
  max_age_days: 28
  compress: true
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("SERVER_MODE", "release")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("LOG_OUTPUT", "stdout")
	t.Setenv("LOG_MAX_SIZE_MB", "64")
	t.Setenv("LOG_MAX_BACKUPS", "5")
	t.Setenv("LOG_MAX_AGE_DAYS", "14")
	t.Setenv("LOG_COMPRESS", "false")

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected config to load, got %v", err)
	}
	if cfg.Server.Port != "9090" {
		t.Fatalf("expected server port from env binding, got %q", cfg.Server.Port)
	}
	if cfg.Server.Mode != "release" {
		t.Fatalf("expected server mode from env binding, got %q", cfg.Server.Mode)
	}
	if cfg.Log.Format != "json" {
		t.Fatalf("expected log format from env binding, got %q", cfg.Log.Format)
	}
	if cfg.Log.Output != "stdout" {
		t.Fatalf("expected log output from env binding, got %q", cfg.Log.Output)
	}
	if cfg.Log.MaxSizeMB != 64 {
		t.Fatalf("expected log max size from env binding, got %d", cfg.Log.MaxSizeMB)
	}
	if cfg.Log.MaxBackups != 5 {
		t.Fatalf("expected log max backups from env binding, got %d", cfg.Log.MaxBackups)
	}
	if cfg.Log.MaxAgeDays != 14 {
		t.Fatalf("expected log max age from env binding, got %d", cfg.Log.MaxAgeDays)
	}
	if cfg.Log.Compress {
		t.Fatalf("expected log compression from env binding to be false")
	}
}

func TestLoadConfig_BindsResearchSettingsFromEnv(t *testing.T) {
	viper.Reset()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	defer viper.Reset()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `server:
  port: "8089"
  mode: "debug"
site:
  slogan: "test"
  url: "http://localhost:3000"
jwt:
  secret: "real-test-secret"
  access_expire_hours: 1
  refresh_expire_days: 7
ai:
  api_key: "x"
  endpoint: "https://ark.example.com"
  embedding_model: "embed-model"
  llm_model: "chat-model"
  temperature: 0.7
  max_tokens: 500
  top_k: 3
  rag_min_score: 0.30
upload:
  max_size: 10485760
  allowed_types:
    - "image/jpeg"
  storage_path: "./uploads"
oauth:
  github:
    client_id: ""
    client_secret: ""
    callback_url: ""
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	oldResearchEndpoint, hadResearchEndpoint := os.LookupEnv("RESEARCH_ENDPOINT")
	oldResearchAPIKey, hadResearchAPIKey := os.LookupEnv("RESEARCH_API_KEY")
	_ = os.Setenv("RESEARCH_ENDPOINT", "https://search.example.com")
	_ = os.Setenv("RESEARCH_API_KEY", "research-secret")
	defer func() {
		if hadResearchEndpoint {
			_ = os.Setenv("RESEARCH_ENDPOINT", oldResearchEndpoint)
		} else {
			_ = os.Unsetenv("RESEARCH_ENDPOINT")
		}
		if hadResearchAPIKey {
			_ = os.Setenv("RESEARCH_API_KEY", oldResearchAPIKey)
		} else {
			_ = os.Unsetenv("RESEARCH_API_KEY")
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected config to load, got %v", err)
	}
	if cfg.AI.ResearchEndpoint != "https://search.example.com" {
		t.Fatalf("expected research endpoint from env binding, got %q", cfg.AI.ResearchEndpoint)
	}
	if cfg.AI.ResearchAPIKey != "research-secret" {
		t.Fatalf("expected research api key from env binding, got %q", cfg.AI.ResearchAPIKey)
	}
	if cfg.AI.ResearchMaxResults != 5 {
		t.Fatalf("expected default research max results 5, got %d", cfg.AI.ResearchMaxResults)
	}
	if cfg.AI.ResearchTimeoutSeconds != 15 {
		t.Fatalf("expected default research timeout seconds 15, got %d", cfg.AI.ResearchTimeoutSeconds)
	}
}

func TestLoadConfig_BindsEmailVerificationSettingsFromEnv(t *testing.T) {
	viper.Reset()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	defer viper.Reset()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `server:
  port: "8089"
  mode: "debug"
site:
  slogan: "test"
  url: "http://localhost:3000"
jwt:
  secret: "real-test-secret"
  access_expire_hours: 1
  refresh_expire_days: 7
upload:
  max_size: 10485760
  allowed_types:
    - "image/jpeg"
  storage_path: "./uploads"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	t.Setenv("EMAIL_SMTP_HOST", "smtp.example.com")
	t.Setenv("EMAIL_SMTP_PORT", "465")
	t.Setenv("EMAIL_USERNAME", "mailer")
	t.Setenv("EMAIL_PASSWORD", "mail-secret")
	t.Setenv("EMAIL_FROM_ADDRESS", "noreply@example.com")
	t.Setenv("EMAIL_FROM_NAME", "WenDao Mail")
	t.Setenv("VERIFICATION_CODE_TTL_MINUTES", "15")
	t.Setenv("VERIFICATION_RESEND_COOLDOWN_SECONDS", "90")
	t.Setenv("VERIFICATION_MAX_ATTEMPTS", "4")

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected config to load, got %v", err)
	}
	if cfg.Email.SMTPHost != "smtp.example.com" {
		t.Fatalf("expected smtp host from env binding, got %q", cfg.Email.SMTPHost)
	}
	if cfg.Email.SMTPPort != 465 {
		t.Fatalf("expected smtp port from env binding, got %d", cfg.Email.SMTPPort)
	}
	if cfg.Email.Username != "mailer" || cfg.Email.Password != "mail-secret" {
		t.Fatalf("expected smtp credentials from env binding")
	}
	if cfg.Email.FromAddress != "noreply@example.com" || cfg.Email.FromName != "WenDao Mail" {
		t.Fatalf("expected sender fields from env binding, got address=%q name=%q", cfg.Email.FromAddress, cfg.Email.FromName)
	}
	if cfg.Verification.CodeTTLMinutes != 15 {
		t.Fatalf("expected verification ttl 15, got %d", cfg.Verification.CodeTTLMinutes)
	}
	if cfg.Verification.ResendCooldownSeconds != 90 {
		t.Fatalf("expected verification cooldown 90, got %d", cfg.Verification.ResendCooldownSeconds)
	}
	if cfg.Verification.MaxVerificationAttempts != 4 {
		t.Fatalf("expected max attempts 4, got %d", cfg.Verification.MaxVerificationAttempts)
	}
}

func TestLoadConfig_BindsUploadCleanupSettingsFromEnvAndDefaults(t *testing.T) {
	viper.Reset()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	defer viper.Reset()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `server:
  port: "8089"
  mode: "debug"
site:
  slogan: "test"
  url: "http://localhost:3000"
jwt:
  secret: "real-test-secret"
  access_expire_hours: 1
  refresh_expire_days: 7
upload:
  max_size: 10485760
  allowed_types:
    - "image/jpeg"
  storage_path: "./uploads"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	t.Setenv("UPLOAD_CLEANUP_ENABLED", "false")
	t.Setenv("UPLOAD_CLEANUP_INTERVAL_HOURS", "6")
	t.Setenv("UPLOAD_CLEANUP_BATCH_SIZE", "50")

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected config to load, got %v", err)
	}
	if cfg.Upload.CleanupEnabled {
		t.Fatalf("expected upload cleanup enabled to bind from env as false")
	}
	if cfg.Upload.CleanupRetentionDays != 2 {
		t.Fatalf("expected default cleanup retention 2 days, got %d", cfg.Upload.CleanupRetentionDays)
	}
	if cfg.Upload.CleanupIntervalHours != 6 {
		t.Fatalf("expected cleanup interval from env, got %d", cfg.Upload.CleanupIntervalHours)
	}
	if cfg.Upload.CleanupBatchSize != 50 {
		t.Fatalf("expected cleanup batch size from env, got %d", cfg.Upload.CleanupBatchSize)
	}
}

func TestLoadConfig_BindsAIAndOAuthEndpointsFromEnv(t *testing.T) {
	viper.Reset()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	defer viper.Reset()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `server:
  port: "8089"
  mode: "debug"
site:
  slogan: "test"
  url: ""
jwt:
  secret: "real-test-secret"
  access_expire_hours: 1
  refresh_expire_days: 7
ai:
  api_key: ""
  endpoint: ""
  embedding_model: ""
  llm_model: ""
  temperature: 0.7
  max_tokens: 500
  top_k: 3
  rag_min_score: 0.30
upload:
  max_size: 10485760
  allowed_types:
    - "image/jpeg"
  storage_path: "./uploads"
oauth:
  github:
    client_id: ""
    client_secret: ""
    callback_url: ""
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	oldDoubaoEndpoint, hadDoubaoEndpoint := os.LookupEnv("DOUBAO_ENDPOINT")
	oldDoubaoChatModel, hadDoubaoChatModel := os.LookupEnv("DOUBAO_CHAT_MODEL")
	oldDoubaoEmbeddingModel, hadDoubaoEmbeddingModel := os.LookupEnv("DOUBAO_EMBEDDING_MODEL")
	oldGitHubCallbackURL, hadGitHubCallbackURL := os.LookupEnv("GITHUB_CALLBACK_URL")
	_ = os.Setenv("DOUBAO_ENDPOINT", "https://ark.example.com/api/v3")
	_ = os.Setenv("DOUBAO_CHAT_MODEL", "chat-model-from-env")
	_ = os.Setenv("DOUBAO_EMBEDDING_MODEL", "embedding-model-from-env")
	_ = os.Setenv("GITHUB_CALLBACK_URL", "https://backend.example.com/api/auth/github/callback")
	defer func() {
		if hadDoubaoEndpoint {
			_ = os.Setenv("DOUBAO_ENDPOINT", oldDoubaoEndpoint)
		} else {
			_ = os.Unsetenv("DOUBAO_ENDPOINT")
		}
		if hadDoubaoChatModel {
			_ = os.Setenv("DOUBAO_CHAT_MODEL", oldDoubaoChatModel)
		} else {
			_ = os.Unsetenv("DOUBAO_CHAT_MODEL")
		}
		if hadDoubaoEmbeddingModel {
			_ = os.Setenv("DOUBAO_EMBEDDING_MODEL", oldDoubaoEmbeddingModel)
		} else {
			_ = os.Unsetenv("DOUBAO_EMBEDDING_MODEL")
		}
		if hadGitHubCallbackURL {
			_ = os.Setenv("GITHUB_CALLBACK_URL", oldGitHubCallbackURL)
		} else {
			_ = os.Unsetenv("GITHUB_CALLBACK_URL")
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected config to load, got %v", err)
	}
	if cfg.AI.Endpoint != "https://ark.example.com/api/v3" {
		t.Fatalf("expected ai endpoint from env binding, got %q", cfg.AI.Endpoint)
	}
	if cfg.AI.LLMModel != "chat-model-from-env" {
		t.Fatalf("expected llm model from env binding, got %q", cfg.AI.LLMModel)
	}
	if cfg.AI.EmbeddingModel != "embedding-model-from-env" {
		t.Fatalf("expected embedding model from env binding, got %q", cfg.AI.EmbeddingModel)
	}
	if cfg.OAuth.GitHub.CallbackURL != "https://backend.example.com/api/auth/github/callback" {
		t.Fatalf("expected github callback url from env binding, got %q", cfg.OAuth.GitHub.CallbackURL)
	}
}
