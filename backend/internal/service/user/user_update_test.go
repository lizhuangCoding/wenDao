package user

import (
	"testing"

	"wenDao/internal/model"
)

func TestUserServiceUpdateUsername_Success(t *testing.T) {
	user := &model.User{
		ID:       1,
		Username: "old-username",
		Email:    "test@example.com",
		Role:     "user",
		Status:   "active",
	}
	repo := newStubUserRepository(user)
	svc := newTestUserService(repo, &stubGitHubOAuthService{})

	err := svc.UpdateUsername(1, "new-username")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	updatedUser, _ := repo.GetByID(1)
	if updatedUser.Username != "new-username" {
		t.Fatalf("expected username 'new-username', got '%s'", updatedUser.Username)
	}
}

func TestUserServiceUpdateUsername_Duplicate(t *testing.T) {
	user1 := &model.User{
		ID:       1,
		Username: "user1",
		Email:    "user1@example.com",
	}
	user2 := &model.User{
		ID:       2,
		Username: "user2",
		Email:    "user2@example.com",
	}
	repo := newStubUserRepository(user1, user2)
	svc := newTestUserService(repo, &stubGitHubOAuthService{})

	err := svc.UpdateUsername(1, "user2")
	if err == nil {
		t.Fatal("expected error for duplicate username, got nil")
	}
	if err.Error() != "username already exists" {
		t.Fatalf("expected error 'username already exists', got '%v'", err)
	}
}

func TestUserServiceUpdateUsername_NotFound(t *testing.T) {
	repo := newStubUserRepository()
	svc := newTestUserService(repo, &stubGitHubOAuthService{})

	err := svc.UpdateUsername(999, "new-username")
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}
}
