package user

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"

	"wenDao/internal/model"
)

func TestUserModelOAuthFields_MapToGormOAuthColumnNames(t *testing.T) {
	s, err := schema.Parse(&model.User{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse user schema: %v", err)
	}

	oauthProviderField := s.LookUpField("OAuthProvider")
	if oauthProviderField == nil {
		t.Fatalf("expected OAuthProvider field to exist in schema")
	}
	if oauthProviderField.DBName != "o_auth_provider" {
		t.Fatalf("expected OAuthProvider DB column to be o_auth_provider, got %q", oauthProviderField.DBName)
	}

	oauthIDField := s.LookUpField("OAuthID")
	if oauthIDField == nil {
		t.Fatalf("expected OAuthID field to exist in schema")
	}
	if oauthIDField.DBName != "o_auth_id" {
		t.Fatalf("expected OAuthID DB column to be o_auth_id, got %q", oauthIDField.DBName)
	}
}
