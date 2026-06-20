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
		response.InternalError(c, "检查邮箱是否已注册失败，请稍后重试")
		return
	}
	if exists {
		response.Error(c, response.CodeInvalidParams, "该邮箱已注册，请直接登录或更换邮箱")
		return
	}

	if h.verificationService == nil {
		response.ServiceUnavailable(c, "验证码服务暂不可用，请稍后重试")
		return
	}
	if err := h.verificationService.SendCode(c.Request.Context(), req.Email, service.PurposeRegister); err != nil {
		h.handleVerificationSendError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "验证码已发送，请查收邮箱"})
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
		response.InternalError(c, "检查邮箱是否已注册失败，请稍后重试")
		return
	}
	if !exists {
		h.log().Info("Password reset verification email skipped for unknown account", userEmailLogFields(req.Email)...)
		response.Success(c, gin.H{"message": "如果邮箱已注册，验证码将发送到该邮箱"})
		return
	}

	if h.verificationService == nil {
		response.ServiceUnavailable(c, "验证码服务暂不可用，请稍后重试")
		return
	}
	if err := h.verificationService.SendCode(c.Request.Context(), req.Email, service.PurposePasswordReset); err != nil {
		h.handleVerificationSendError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "如果邮箱已注册，验证码将发送到该邮箱"})
}

// ConfirmPasswordReset 校验验证码并重置密码
func (h *UserHandler) ConfirmPasswordReset(c *gin.Context) {
	var req PasswordResetConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	if h.verificationService == nil {
		response.ServiceUnavailable(c, "验证码服务暂不可用，请稍后重试")
		return
	}
	if err := h.verificationService.VerifyCode(c.Request.Context(), req.Email, service.PurposePasswordReset, req.VerificationCode); err != nil {
		h.handleVerificationVerifyError(c, err)
		return
	}

	if err := h.userService.ResetPassword(req.Email, req.Password); err != nil {
		if errors.Is(err, svcerrors.ErrUserNotFound) {
			response.InvalidParams(c, "验证码或邮箱不正确，请检查后重试")
			return
		}
		response.InternalError(c, "重置密码失败，请稍后重试")
		return
	}

	response.Success(c, gin.H{"message": "密码重置成功，请使用新密码登录"})
}

func (h *UserHandler) handleVerificationSendError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrVerificationCodeTooFrequent):
		response.TooManyRequests(c, "验证码发送过于频繁，请稍后再试")
	case errors.Is(err, service.ErrVerificationUnavailable), errors.Is(err, service.ErrVerificationEmailNotConfigured):
		response.ServiceUnavailable(c, "邮箱验证码服务暂不可用，请稍后重试")
	default:
		response.InternalError(c, "发送验证码失败，请稍后重试")
	}
}

func (h *UserHandler) handleVerificationVerifyError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrVerificationCodeInvalid):
		response.InvalidParams(c, "验证码不正确或已过期，请重新获取")
	case errors.Is(err, service.ErrVerificationUnavailable):
		response.ServiceUnavailable(c, "验证码服务暂不可用，请稍后重试")
	default:
		response.InternalError(c, "校验验证码失败，请稍后重试")
	}
}
