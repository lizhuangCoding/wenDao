package user

import (
	"errors"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
	"wenDao/internal/svcerrors"
)

type VerificationCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type PasswordResetConfirmRequest struct {
	Email            string `json:"email" binding:"required,email"`
	Password         string `json:"password" binding:"required,min=6"`
	VerificationCode string `json:"verification_code" binding:"required,min=4,max=10"`
}

// RequestRegisterCode 发送注册邮箱验证码
func (h *UserHandler) RequestRegisterCode(c *gin.Context) {
	var req VerificationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	exists, err := h.userService.EmailExists(req.Email)
	if err != nil {
		response.InternalError(c, "Failed to check email")
		return
	}
	if exists {
		response.Error(c, response.CodeInvalidParams, "Email already exists")
		return
	}

	if h.verificationService == nil {
		response.ServiceUnavailable(c, "Verification service is unavailable")
		return
	}
	if err := h.verificationService.SendCode(c.Request.Context(), req.Email, service.PurposeRegister); err != nil {
		h.handleVerificationSendError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Verification code sent"})
}

// RequestPasswordResetCode 发送密码重置验证码
func (h *UserHandler) RequestPasswordResetCode(c *gin.Context) {
	var req VerificationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	exists, err := h.userService.EmailExists(req.Email)
	if err != nil {
		response.InternalError(c, "Failed to check email")
		return
	}
	if !exists {
		h.log().Info("Password reset verification email skipped for unknown account", userEmailLogFields(req.Email)...)
		response.Success(c, gin.H{"message": "If the email exists, a verification code has been sent"})
		return
	}

	if h.verificationService == nil {
		response.ServiceUnavailable(c, "Verification service is unavailable")
		return
	}
	if err := h.verificationService.SendCode(c.Request.Context(), req.Email, service.PurposePasswordReset); err != nil {
		h.handleVerificationSendError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "If the email exists, a verification code has been sent"})
}

// ConfirmPasswordReset 校验验证码并重置密码
func (h *UserHandler) ConfirmPasswordReset(c *gin.Context) {
	var req PasswordResetConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	if h.verificationService == nil {
		response.ServiceUnavailable(c, "Verification service is unavailable")
		return
	}
	if err := h.verificationService.VerifyCode(c.Request.Context(), req.Email, service.PurposePasswordReset, req.VerificationCode); err != nil {
		h.handleVerificationVerifyError(c, err)
		return
	}

	if err := h.userService.ResetPassword(req.Email, req.Password); err != nil {
		if errors.Is(err, svcerrors.ErrUserNotFound) {
			response.InvalidParams(c, "Invalid verification code or email")
			return
		}
		response.InternalError(c, "Failed to reset password")
		return
	}

	response.Success(c, gin.H{"message": "Password reset successfully"})
}

func (h *UserHandler) handleVerificationSendError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrVerificationCodeTooFrequent):
		response.TooManyRequests(c, "Verification code was sent recently, please wait before retrying")
	case errors.Is(err, service.ErrVerificationUnavailable), errors.Is(err, service.ErrVerificationEmailNotConfigured):
		response.ServiceUnavailable(c, "Verification email service is unavailable")
	default:
		response.InternalError(c, "Failed to send verification code")
	}
}

func (h *UserHandler) handleVerificationVerifyError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrVerificationCodeInvalid):
		response.InvalidParams(c, "Invalid verification code")
	case errors.Is(err, service.ErrVerificationUnavailable):
		response.ServiceUnavailable(c, "Verification service is unavailable")
	default:
		response.InternalError(c, "Failed to verify code")
	}
}
