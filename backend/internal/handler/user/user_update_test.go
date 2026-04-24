package user

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"wenDao/config"
)

func TestUserHandlerUpdateUsername_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userService := &stubUserService{}
	h := NewUserHandler(userService, &stubUploadService{}, &stubOAuthService{}, &config.Config{})

	reqBody, _ := json.Marshal(UpdateUsernameRequest{
		Username: "new-username",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/users/me/username", bytes.NewBuffer(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", int64(1))

	h.UpdateUsername(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if userService.updateUsernameID != 1 {
		t.Fatalf("expected update user id 1, got %d", userService.updateUsernameID)
	}

	if userService.updateUsername != "new-username" {
		t.Fatalf("expected username 'new-username', got '%s'", userService.updateUsername)
	}
}

func TestUserHandlerUpdateUsername_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewUserHandler(&stubUserService{}, &stubUploadService{}, &stubOAuthService{}, &config.Config{})

	tests := []struct {
		name     string
		username string
	}{
		{name: "empty", username: ""},
		{name: "too short", username: "a"},
		{name: "too long", username: "verylongusernamethatexceedsfiftycharacterslimitfortestingpurposes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, _ := json.Marshal(UpdateUsernameRequest{
				Username: tt.username,
			})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPut, "/api/users/me/username", bytes.NewBuffer(reqBody))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("user_id", int64(1))

			h.UpdateUsername(c)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d for %s, got %d", http.StatusBadRequest, tt.name, w.Code)
			}
		})
	}
}

func TestUserHandlerUpdateUsername_Duplicate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userService := &stubUserService{
		updateUsernameErr: errors.New("username already exists"),
	}
	h := NewUserHandler(userService, &stubUploadService{}, &stubOAuthService{}, &config.Config{})

	reqBody, _ := json.Marshal(UpdateUsernameRequest{
		Username: "user2",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/users/me/username", bytes.NewBuffer(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", int64(1))

	h.UpdateUsername(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
