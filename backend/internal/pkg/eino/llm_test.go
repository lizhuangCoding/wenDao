package eino

import (
	"testing"

	"wenDao/config"
)

func TestNewLLMClient_ProviderRouting(t *testing.T) {
	t.Run("defaults to doubao", func(t *testing.T) {
		client, err := NewLLMClient(&config.AIConfig{
			APIKey:   "test-key",
			Endpoint: "https://ark.example.com/api/v3",
			LLMModel: "doubao-test-model",
		})
		if err != nil {
			t.Fatalf("expected doubao client to initialize, got %v", err)
		}
		if client == nil {
			t.Fatal("expected doubao client, got nil")
		}
	})

	t.Run("supports deepseek", func(t *testing.T) {
		client, err := NewLLMClient(&config.AIConfig{
			Provider: "deepseek",
			APIKey:   "test-key",
			Endpoint: "https://api.deepseek.example.com",
			LLMModel: "deepseek-chat",
		})
		if err != nil {
			t.Fatalf("expected deepseek client to initialize, got %v", err)
		}
		if client == nil {
			t.Fatal("expected deepseek client, got nil")
		}
	})

	t.Run("rejects unsupported provider", func(t *testing.T) {
		client, err := NewLLMClient(&config.AIConfig{
			Provider: "unknown-provider",
			APIKey:   "test-key",
			LLMModel: "test-model",
		})
		if err == nil {
			t.Fatalf("expected unsupported provider error, got client %#v", client)
		}
	})
}
