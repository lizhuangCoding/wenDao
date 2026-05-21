package user

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"wenDao/config"
	"wenDao/internal/service"
)

func TestUserHandlerRequestRegisterCodeSendsVerificationCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userService := &stubUserService{}
	verificationService := &stubVerificationService{}
	h := NewUserHandler(userService, &stubUploadService{}, &stubOAuthService{}, verificationService, &config.Config{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/register/code", strings.NewReader(`{"email":"new@example.com"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.RequestRegisterCode(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body %s", w.Code, w.Body.String())
	}
	if verificationService.sendEmail != "new@example.com" || verificationService.sendPurpose != service.PurposeRegister {
		t.Fatalf("expected register code send, got email=%q purpose=%q", verificationService.sendEmail, verificationService.sendPurpose)
	}
}

func TestUserHandlerRequestRegisterCodeRejectsExistingEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userService := &stubUserService{emailExists: true}
	verificationService := &stubVerificationService{}
	h := NewUserHandler(userService, &stubUploadService{}, &stubOAuthService{}, verificationService, &config.Config{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/register/code", strings.NewReader(`{"email":"used@example.com"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.RequestRegisterCode(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body %s", w.Code, w.Body.String())
	}
	if verificationService.sendEmail != "" {
		t.Fatalf("expected no verification email for existing account, got %q", verificationService.sendEmail)
	}
}

func TestUserHandlerConfirmPasswordResetVerifiesCodeBeforeReset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userService := &stubUserService{}
	verificationService := &stubVerificationService{}
	h := NewUserHandler(userService, &stubUploadService{}, &stubOAuthService{}, verificationService, &config.Config{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/confirm", strings.NewReader(`{"email":"reset@example.com","password":"new-password","verification_code":"654321"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ConfirmPasswordReset(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body %s", w.Code, w.Body.String())
	}
	if verificationService.verifyEmail != "reset@example.com" || verificationService.verifyPurpose != service.PurposePasswordReset || verificationService.verifyCode != "654321" {
		t.Fatalf("expected reset verification, got email=%q purpose=%q code=%q", verificationService.verifyEmail, verificationService.verifyPurpose, verificationService.verifyCode)
	}
	if userService.resetPasswordEmail != "reset@example.com" || userService.resetPasswordValue != "new-password" {
		t.Fatalf("expected reset password call, got email=%q password=%q", userService.resetPasswordEmail, userService.resetPasswordValue)
	}
}

func TestUserHandlerConfirmPasswordResetRejectsInvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userService := &stubUserService{}
	verificationService := &stubVerificationService{verifyErr: service.ErrVerificationCodeInvalid}
	h := NewUserHandler(userService, &stubUploadService{}, &stubOAuthService{}, verificationService, &config.Config{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/confirm", strings.NewReader(`{"email":"reset@example.com","password":"new-password","verification_code":"000000"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ConfirmPasswordReset(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body %s", w.Code, w.Body.String())
	}
	if userService.resetPasswordEmail != "" {
		t.Fatalf("expected password not to be reset when code is invalid")
	}
}

func TestUserHandlerRequestPasswordResetCodeHidesMissingEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userService := &stubUserService{emailExists: false}
	verificationService := &stubVerificationService{sendErr: errors.New("should not send")}
	h := NewUserHandler(userService, &stubUploadService{}, &stubOAuthService{}, verificationService, &config.Config{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/code", strings.NewReader(`{"email":"missing@example.com"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.RequestPasswordResetCode(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected generic success, got %d body %s", w.Code, w.Body.String())
	}
	if verificationService.sendEmail != "" {
		t.Fatalf("expected no email send for missing account")
	}
}
