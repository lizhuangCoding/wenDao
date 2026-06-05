package main

import "testing"

func TestProviderUsesArkEmbeddingOnlyForArkCompatibleProviders(t *testing.T) {
	tests := []struct {
		provider string
		want     bool
	}{
		{provider: "", want: true},
		{provider: "doubao", want: true},
		{provider: "ark", want: true},
		{provider: " Doubao ", want: true},
		{provider: "deepseek", want: false},
		{provider: "openai", want: false},
		{provider: "openai-compatible", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if got := providerUsesArkEmbedding(tt.provider); got != tt.want {
				t.Fatalf("providerUsesArkEmbedding(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}
