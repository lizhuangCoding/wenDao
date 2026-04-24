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
