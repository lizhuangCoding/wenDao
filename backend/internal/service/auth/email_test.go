package auth

import (
	"strings"
	"testing"
	"time"
)

func TestVerificationEmailTemplateUsesRichHTMLAndPlainTextFallback(t *testing.T) {
	htmlBody := verificationEmailHTMLBody(PurposeRegister, "123456", 10*time.Minute)
	textBody := verificationEmailTextBody(PurposeRegister, "123456", 10*time.Minute)

	for _, want := range []string{
		"<html",
		"开启你的问道账户",
		"123456",
		"10 分钟",
		"如果不是你本人操作",
	} {
		if !strings.Contains(htmlBody, want) {
			t.Fatalf("expected html email to contain %q, got:\n%s", want, htmlBody)
		}
	}
	if strings.Contains(textBody, "<html") {
		t.Fatalf("expected plain text fallback without html markup, got:\n%s", textBody)
	}
	if !strings.Contains(textBody, "你的 wenDao 验证码是：123456") {
		t.Fatalf("expected plain text fallback to include code, got:\n%s", textBody)
	}
}

func TestVerificationEmailTemplateSeparatesPasswordResetTone(t *testing.T) {
	htmlBody := verificationEmailHTMLBody(PurposePasswordReset, "654321", 5*time.Minute)

	for _, want := range []string{
		"安全重置密码",
		"654321",
		"这封邮件只用于本次密码重置",
	} {
		if !strings.Contains(htmlBody, want) {
			t.Fatalf("expected password reset html email to contain %q, got:\n%s", want, htmlBody)
		}
	}
}
