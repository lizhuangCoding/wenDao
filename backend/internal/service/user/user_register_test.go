package user

import (
	"testing"

	"wenDao/internal/model"
)

func TestUserServiceRegister_AllowsDuplicateUsername(t *testing.T) {
	existingUser := &model.User{
		ID:       1,
		Username: "same-name",
		Email:    "existing@example.com",
		Role:     "user",
		Status:   "active",
	}
	repo := newStubUserRepository(existingUser)
	svc := newTestUserService(repo, &stubGitHubOAuthService{})

	user, err := svc.Register("new@example.com", "password123", "same-name")
	if err != nil {
		t.Fatalf("expected duplicate username registration to succeed, got %v", err)
	}
	if user.Username != "same-name" {
		t.Fatalf("expected username to be preserved, got %q", user.Username)
	}
	if user.Email != "new@example.com" {
		t.Fatalf("expected new user email, got %q", user.Email)
	}
}

func TestUserServiceRegister_ReturnsErrorWhenEmailExists(t *testing.T) {
	existingUser := &model.User{
		ID:       1,
		Username: "existing-name",
		Email:    "same@example.com",
		Role:     "user",
		Status:   "active",
	}
	repo := newStubUserRepository(existingUser)
	svc := newTestUserService(repo, &stubGitHubOAuthService{})

	_, err := svc.Register("same@example.com", "password123", "new-name")
	if err == nil {
		t.Fatal("expected duplicate email error, got nil")
	}
	if err.Error() != "email already exists" {
		t.Fatalf("expected error 'email already exists', got %v", err)
	}
}
