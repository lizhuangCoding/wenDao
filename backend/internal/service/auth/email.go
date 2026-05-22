package auth

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"wenDao/config"
	"wenDao/internal/pkg/mailer"
)

var ErrVerificationEmailNotConfigured = mailer.ErrNotConfigured

type smtpVerificationEmailSender struct {
	sender *mailer.SMTPEmailSender
}

func NewSMTPVerificationEmailSender(cfg config.EmailConfig) VerificationEmailSender {
	return &smtpVerificationEmailSender{sender: mailer.NewSMTPEmailSender(cfg)}
}

func (s *smtpVerificationEmailSender) SendVerificationCode(ctx context.Context, email string, purpose VerificationPurpose, code string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.sender == nil {
		return ErrVerificationEmailNotConfigured
	}

	subject := verificationEmailSubject(purpose)
	if err := s.sender.Send(ctx, email, subject, verificationEmailTextBody(purpose, code, ttl), verificationEmailHTMLBody(purpose, code, ttl)); err != nil {
		if errors.Is(err, mailer.ErrNotConfigured) {
			return ErrVerificationEmailNotConfigured
		}
		return fmt.Errorf("failed to send verification email: %w", err)
	}
	return nil
}

func verificationEmailSubject(purpose VerificationPurpose) string {
	switch purpose {
	case PurposePasswordReset:
		return "wenDao 密码重置验证码"
	default:
		return "wenDao 注册验证码"
	}
}

func verificationEmailTextBody(purpose VerificationPurpose, code string, ttl time.Duration) string {
	action := "完成注册"
	if purpose == PurposePasswordReset {
		action = "重置密码"
	}
	minutes := int(ttl.Minutes())
	if minutes <= 0 {
		minutes = 1
	}
	return fmt.Sprintf("你好，\n\n你的 wenDao 验证码是：%s\n\n请在 %d 分钟内使用它%s。如果不是你本人操作，可以忽略这封邮件。\n", code, minutes, action)
}

func verificationEmailHTMLBody(purpose VerificationPurpose, code string, ttl time.Duration) string {
	headline := "开启你的问道账户"
	description := "欢迎来到问道。输入下面的验证码，就可以完成注册，开始阅读、评论和与 AI 助手交流。"
	badge := "注册验证码"
	footnote := "这封邮件只用于本次注册验证。"
	if purpose == PurposePasswordReset {
		headline = "安全重置密码"
		description = "我们收到了你的密码重置请求。输入下面的验证码后，即可设置新的登录密码。"
		badge = "密码重置"
		footnote = "这封邮件只用于本次密码重置。"
	}

	minutes := int(ttl.Minutes())
	if minutes <= 0 {
		minutes = 1
	}
	escapedCode := html.EscapeString(strings.TrimSpace(code))

	return fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<body style="margin:0;background:#f5f7fb;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;color:#172033;">
  <div style="max-width:620px;margin:0 auto;padding:32px 18px;">
    <div style="overflow:hidden;border-radius:22px;background:#ffffff;border:1px solid #e7ecf4;box-shadow:0 18px 50px rgba(31,41,55,.08);">
      <div style="background:linear-gradient(135deg,#0f766e,#2563eb);padding:28px 30px;color:#ffffff;">
        <div style="display:inline-block;padding:6px 12px;border-radius:999px;background:rgba(255,255,255,.18);font-size:13px;font-weight:700;">%s</div>
        <h1 style="margin:18px 0 8px;font-size:28px;line-height:1.25;">%s</h1>
        <p style="margin:0;line-height:1.8;color:#dff7ff;">%s</p>
      </div>
      <div style="padding:30px;">
        <div style="margin:0 auto 24px;width:160px;height:108px;border-radius:28px;background:linear-gradient(135deg,#ecfeff,#eef2ff);position:relative;text-align:center;">
          <div style="padding-top:24px;font-size:36px;line-height:1;">W</div>
          <div style="margin:6px auto 0;width:78px;height:8px;border-radius:999px;background:#99f6e4;"></div>
        </div>
        <p style="margin:0 0 12px;color:#526070;line-height:1.8;">你的 wenDao 验证码是：</p>
        <div style="letter-spacing:8px;font-size:34px;font-weight:800;text-align:center;color:#111827;background:#f8fafc;border:1px dashed #94a3b8;border-radius:16px;padding:18px 12px;">%s</div>
        <p style="margin:18px 0 0;color:#526070;line-height:1.8;">请在 <strong>%d 分钟</strong> 内使用。%s</p>
        <p style="margin:10px 0 0;color:#8a95a5;font-size:13px;line-height:1.7;">如果不是你本人操作，可以忽略这封邮件，你的账户不会因此发生变化。</p>
      </div>
    </div>
  </div>
</body>
</html>`, badge, headline, description, escapedCode, minutes, footnote)
}
