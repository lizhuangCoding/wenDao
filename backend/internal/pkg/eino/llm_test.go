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

	t.Run("supports openai-compatible with endpoint", func(t *testing.T) {
		client, err := NewLLMClient(&config.AIConfig{
			Provider: "openai-compatible",
			APIKey:   "test-key",
			Endpoint: "https://llm.example.com/v1",
			LLMModel: "compatible-chat",
		})
		if err != nil {
			t.Fatalf("expected openai-compatible client to initialize, got %v", err)
		}
		if client == nil {
			t.Fatal("expected openai-compatible client, got nil")
		}
	})

	t.Run("supports openai without endpoint", func(t *testing.T) {
		client, err := NewLLMClient(&config.AIConfig{
			Provider: "openai",
			APIKey:   "test-key",
			LLMModel: "gpt-test",
		})
		if err != nil {
			t.Fatalf("expected openai client to initialize, got %v", err)
		}
		if client == nil {
			t.Fatal("expected openai client, got nil")
		}
	})

	t.Run("rejects nil config", func(t *testing.T) {
		client, err := NewLLMClient(nil)
		if err == nil {
			t.Fatalf("expected nil config error, got client %#v", client)
		}
	})

	t.Run("rejects empty API key", func(t *testing.T) {
		client, err := NewLLMClient(&config.AIConfig{
			Provider: "deepseek",
			LLMModel: "deepseek-chat",
		})
		if err == nil {
			t.Fatalf("expected empty API key error, got client %#v", client)
		}
	})
}
