package user

import (
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"

	"wenDao/config"
	"wenDao/internal/model"
	"wenDao/internal/repository"
)

type stubUserRepository struct {
	usersByID       map[int64]*model.User
	usersByEmail    map[string]*model.User
	usersByUsername map[string]*model.User
	usersByOAuth    map[string]*model.User
	nextID          int64
	updatedUsers    []*model.User
}

func newStubUserRepository(users ...*model.User) *stubUserRepository {
	repo := &stubUserRepository{
		usersByID:       make(map[int64]*model.User),
		usersByEmail:    make(map[string]*model.User),
		usersByUsername: make(map[string]*model.User),
		usersByOAuth:    make(map[string]*model.User),
		nextID:          1,
	}
	for _, user := range users {
		repo.store(user)
		if user.ID >= repo.nextID {
			repo.nextID = user.ID + 1
		}
	}
	return repo
}

func (r *stubUserRepository) Create(user *model.User) error {
	if user.ID == 0 {
		user.ID = r.nextID
		r.nextID++
	}
	r.store(user)
	return nil
}

func (r *stubUserRepository) GetByID(id int64) (*model.User, error) {
	user, ok := r.usersByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (r *stubUserRepository) GetByEmail(email string) (*model.User, error) {
	user, ok := r.usersByEmail[email]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (r *stubUserRepository) GetByUsername(username string) (*model.User, error) {
	user, ok := r.usersByUsername[username]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (r *stubUserRepository) GetByOAuth(provider string, oauthID string) (*model.User, error) {
	user, ok := r.usersByOAuth[provider+":"+oauthID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (r *stubUserRepository) Update(user *model.User) error {
	r.updatedUsers = append(r.updatedUsers, user)
	r.store(user)
	return nil
}

func (r *stubUserRepository) store(user *model.User) {
	r.usersByID[user.ID] = user
	if user.Email != "" {
		r.usersByEmail[user.Email] = user
	}
	if user.Username != "" {
		r.usersByUsername[user.Username] = user
	}
	if user.OAuthProvider != nil && user.OAuthID != nil {
		r.usersByOAuth[*user.OAuthProvider+":"+*user.OAuthID] = user
	}
}

var _ repository.UserRepository = (*stubUserRepository)(nil)

type stubGitHubOAuthService struct {
	githubUser *GitHubUserInfo
	err        error
}

func (s *stubGitHubOAuthService) GetGitHubAuthURL(state string) string {
	return ""
}

func (s *stubGitHubOAuthService) ExchangeGitHubCode(code string) (*GitHubUserInfo, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.githubUser, nil
}

func newTestUserService(userRepo repository.UserRepository, oauthService OAuthService) UserService {
	return NewUserService(userRepo, oauthService, &config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-jwt-secret",
			AccessExpireHours: 1,
		},
	}, nil)
}

func TestUserServiceRegister_SetsDefaultAvatarSource(t *testing.T) {
	repo := newStubUserRepository()
	svc := newTestUserService(repo, &stubGitHubOAuthService{})

	user, err := svc.Register("new@example.com", "password123", "new-user")
	if err != nil {
		t.Fatalf("expected register to succeed, got %v", err)
	}

	if user.AvatarURL == nil || *user.AvatarURL == "" {
		t.Fatalf("expected default avatar url to be set")
	}
	if user.AvatarSource != model.AvatarSourceDefault {
		t.Fatalf("expected avatar source %q, got %q", model.AvatarSourceDefault, user.AvatarSource)
	}
}

func TestUserServiceGitHubOAuthLogin_CreatesUserWithGitHubAvatar(t *testing.T) {
	repo := newStubUserRepository()
	svc := newTestUserService(repo, &stubGitHubOAuthService{githubUser: &GitHubUserInfo{
		ID:        1001,
		Login:     "octocat",
		Email:     "octocat@example.com",
		AvatarURL: "https://avatars.githubusercontent.com/u/1001?v=4",
	}})

	_, user, err := svc.GitHubOAuthLogin("oauth-code")
	if err != nil {
		t.Fatalf("expected github oauth login to succeed, got %v", err)
	}

	if user.AvatarURL == nil || *user.AvatarURL != "https://avatars.githubusercontent.com/u/1001?v=4" {
		t.Fatalf("expected github avatar url to be stored, got %#v", user.AvatarURL)
	}
	if user.AvatarSource != model.AvatarSourceGitHub {
		t.Fatalf("expected avatar source %q, got %q", model.AvatarSourceGitHub, user.AvatarSource)
	}
}

func TestUserServiceGitHubOAuthLogin_ReturnsErrorWhenEmailMissing(t *testing.T) {
	repo := newStubUserRepository()
	svc := newTestUserService(repo, &stubGitHubOAuthService{githubUser: &GitHubUserInfo{
		ID:        1002,
		Login:     "octocat",
		Email:     "",
		AvatarURL: "https://avatars.githubusercontent.com/u/1002?v=4",
	}})

	_, _, err := svc.GitHubOAuthLogin("oauth-code")
	if err == nil {
		t.Fatalf("expected missing email error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "email") {
		t.Fatalf("expected clear missing email error, got %v", err)
	}
}

func TestUserServiceGitHubOAuthLogin_ReturnsErrorWhenEmailBelongsToAnotherUser(t *testing.T) {
	existingUser := &model.User{
		ID:       20,
		Username: "manual-user",
		Email:    "octocat@example.com",
		Role:     "user",
		Status:   "active",
	}
	repo := newStubUserRepository(existingUser)
	svc := newTestUserService(repo, &stubGitHubOAuthService{githubUser: &GitHubUserInfo{
		ID:        1003,
		Login:     "octocat",
		Email:     "octocat@example.com",
		AvatarURL: "https://avatars.githubusercontent.com/u/1003?v=4",
	}})

	_, _, err := svc.GitHubOAuthLogin("oauth-code")
	if err == nil {
		t.Fatalf("expected email collision error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "email") {
		t.Fatalf("expected clear email collision error, got %v", err)
	}
}

func TestUserServiceGitHubOAuthLogin_GeneratesAlternateUsernameWhenLoginTaken(t *testing.T) {
	existingUser := &model.User{
		ID:       21,
		Username: "octocat",
		Email:    "existing@example.com",
		Role:     "user",
		Status:   "active",
	}
	repo := newStubUserRepository(existingUser)
	svc := newTestUserService(repo, &stubGitHubOAuthService{githubUser: &GitHubUserInfo{
		ID:        987654321,
		Login:     "octocat",
		Email:     "new-octocat@example.com",
		AvatarURL: "https://avatars.githubusercontent.com/u/987654321?v=4",
	}})

	_, user, err := svc.GitHubOAuthLogin("oauth-code")
	if err != nil {
		t.Fatalf("expected username fallback to succeed, got %v", err)
	}
	if user.Username == "octocat" {
		t.Fatalf("expected alternate username when login collides")
	}
	if !strings.HasPrefix(user.Username, "octocat-") {
		t.Fatalf("expected deterministic username fallback prefix, got %q", user.Username)
	}
	if len(user.Username) > 50 {
		t.Fatalf("expected username to remain within max length, got %d chars", len(user.Username))
	}
}

func TestUserServiceGitHubOAuthLogin_RefreshesAvatarForDefaultOrGitHubSources(t *testing.T) {
	provider := "github"
	oauthID := "2002"

	tests := []struct {
		name         string
		avatarSource string
	}{
		{name: "default avatar source", avatarSource: model.AvatarSourceDefault},
		{name: "github avatar source", avatarSource: model.AvatarSourceGitHub},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldAvatar := "https://example.com/old.png"
			user := &model.User{
				ID:            10,
				Username:      "octocat",
				Email:         "octocat@example.com",
				Role:          "user",
				Status:        "active",
				OAuthProvider: &provider,
				OAuthID:       &oauthID,
				AvatarURL:     &oldAvatar,
				AvatarSource:  tt.avatarSource,
			}
			repo := newStubUserRepository(user)
			svc := newTestUserService(repo, &stubGitHubOAuthService{githubUser: &GitHubUserInfo{
				ID:        2002,
				Login:     "octocat",
				Email:     "octocat@example.com",
				AvatarURL: "https://avatars.githubusercontent.com/u/2002?v=5",
			}})

			_, gotUser, err := svc.GitHubOAuthLogin("oauth-code")
			if err != nil {
				t.Fatalf("expected github oauth login to succeed, got %v", err)
			}

			if gotUser.AvatarURL == nil || *gotUser.AvatarURL != "https://avatars.githubusercontent.com/u/2002?v=5" {
				t.Fatalf("expected avatar url to refresh from github, got %#v", gotUser.AvatarURL)
			}
			if gotUser.AvatarSource != model.AvatarSourceGitHub {
				t.Fatalf("expected avatar source to become %q, got %q", model.AvatarSourceGitHub, gotUser.AvatarSource)
			}
			if len(repo.updatedUsers) != 1 {
				t.Fatalf("expected existing user to be updated once, got %d updates", len(repo.updatedUsers))
			}
		})
	}
}

func TestUserServiceGitHubOAuthLogin_DoesNotOverwriteCustomAvatar(t *testing.T) {
	provider := "github"
	oauthID := "3003"
	customAvatar := "/uploads/avatars/custom.png"
	user := &model.User{
		ID:            11,
		Username:      "octocat",
		Email:         "octocat@example.com",
		Role:          "user",
		Status:        "active",
		OAuthProvider: &provider,
		OAuthID:       &oauthID,
		AvatarURL:     &customAvatar,
		AvatarSource:  model.AvatarSourceCustom,
	}
	repo := newStubUserRepository(user)
	svc := newTestUserService(repo, &stubGitHubOAuthService{githubUser: &GitHubUserInfo{
		ID:        3003,
		Login:     "octocat",
		Email:     "octocat@example.com",
		AvatarURL: "https://avatars.githubusercontent.com/u/3003?v=7",
	}})

	_, gotUser, err := svc.GitHubOAuthLogin("oauth-code")
	if err != nil {
		t.Fatalf("expected github oauth login to succeed, got %v", err)
	}

	if gotUser.AvatarURL == nil || *gotUser.AvatarURL != customAvatar {
		t.Fatalf("expected custom avatar to remain unchanged, got %#v", gotUser.AvatarURL)
	}
	if gotUser.AvatarSource != model.AvatarSourceCustom {
		t.Fatalf("expected avatar source to remain %q, got %q", model.AvatarSourceCustom, gotUser.AvatarSource)
	}
	if len(repo.updatedUsers) != 0 {
		t.Fatalf("expected custom avatar user not to be updated, got %d updates", len(repo.updatedUsers))
	}
}

func TestUserServiceUpdateAvatar_MarksAvatarSourceCustom(t *testing.T) {
	avatarURL := "https://example.com/old.png"
	user := &model.User{
		ID:           12,
		Username:     "avatar-user",
		Email:        "avatar@example.com",
		Role:         "user",
		Status:       "active",
		AvatarURL:    &avatarURL,
		AvatarSource: model.AvatarSourceDefault,
	}
	repo := newStubUserRepository(user)
	svc := newTestUserService(repo, &stubGitHubOAuthService{})

	err := svc.UpdateAvatar(12, "/uploads/avatars/custom.png")
	if err != nil {
		t.Fatalf("expected avatar update to succeed, got %v", err)
	}

	updatedUser, err := repo.GetByID(12)
	if err != nil {
		t.Fatalf("expected updated user to exist, got %v", err)
	}
	if updatedUser.AvatarURL == nil || *updatedUser.AvatarURL != "/uploads/avatars/custom.png" {
		t.Fatalf("expected avatar url to be updated, got %#v", updatedUser.AvatarURL)
	}
	if updatedUser.AvatarSource != model.AvatarSourceCustom {
		t.Fatalf("expected avatar source %q, got %q", model.AvatarSourceCustom, updatedUser.AvatarSource)
	}
}

func TestUserServiceGitHubOAuthLogin_PropagatesExchangeErrors(t *testing.T) {
	repo := newStubUserRepository()
	svc := newTestUserService(repo, &stubGitHubOAuthService{err: errors.New("oauth unavailable")})

	_, _, err := svc.GitHubOAuthLogin("oauth-code")
	if err == nil {
		t.Fatalf("expected exchange error")
	}
}
