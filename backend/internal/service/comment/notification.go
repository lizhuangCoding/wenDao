package comment

import (
	"context"
	"fmt"
	"html"
	"strings"

	"wenDao/config"
	"wenDao/internal/pkg/mailer"
)

type CommentReplyNotification struct {
	RecipientEmail      string
	RecipientUsername   string
	ReplyAuthorUsername string
	ArticleTitle        string
	ArticleSlug         string
	CommentPreview      string
}

type CommentReplyNotificationSender interface {
	SendCommentReplyNotification(ctx context.Context, notification CommentReplyNotification) error
}

type smtpCommentReplyEmailSender struct {
	sender *mailer.SMTPEmailSender
}

func NewSMTPCommentReplyEmailSender(cfg config.EmailConfig, _ string) CommentReplyNotificationSender {
	return &smtpCommentReplyEmailSender{
		sender: mailer.NewSMTPEmailSender(cfg),
	}
}

func (s *smtpCommentReplyEmailSender) SendCommentReplyNotification(ctx context.Context, notification CommentReplyNotification) error {
	if s == nil || s.sender == nil {
		return mailer.ErrNotConfigured
	}
	subject := fmt.Sprintf("你在问道的评论有新回复：%s", notification.ArticleTitle)
	return s.sender.Send(ctx, notification.RecipientEmail, subject, commentReplyEmailTextBody(notification), commentReplyEmailHTMLBody(notification))
}

func commentReplyEmailTextBody(notification CommentReplyNotification) string {
	lines := []string{
		fmt.Sprintf("你好，%s：", displayName(notification.RecipientUsername)),
		"",
		fmt.Sprintf("%s 回复了你在《%s》下的评论。", displayName(notification.ReplyAuthorUsername), notification.ArticleTitle),
		"",
		"回复内容：",
		notification.CommentPreview,
	}
	lines = append(lines, "", "你可以在个人设置中关闭评论回复邮件提醒。")
	return strings.Join(lines, "\n")
}

func commentReplyEmailHTMLBody(notification CommentReplyNotification) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<body style="margin:0;background:#f8fafc;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;color:#172033;">
  <div style="max-width:620px;margin:0 auto;padding:32px 18px;">
    <div style="border-radius:22px;background:#ffffff;border:1px solid #e7ecf4;box-shadow:0 18px 50px rgba(31,41,55,.08);overflow:hidden;">
      <div style="padding:28px 30px;background:linear-gradient(135deg,#111827,#0f766e);color:#ffffff;">
        <div style="font-size:13px;font-weight:800;letter-spacing:.08em;text-transform:uppercase;color:#99f6e4;">Comment Reply</div>
        <h1 style="margin:12px 0 8px;font-size:26px;line-height:1.3;">你的评论有新回复</h1>
        <p style="margin:0;color:#d1fae5;line-height:1.8;">%s 回复了你在《%s》下的评论。</p>
      </div>
      <div style="padding:30px;">
        <div style="display:flex;align-items:center;gap:14px;margin-bottom:20px;">
          <div style="width:56px;height:56px;border-radius:18px;background:#ecfeff;text-align:center;line-height:56px;font-size:26px;font-weight:900;color:#0f766e;">@</div>
          <div>
            <div style="font-weight:800;color:#111827;">%s</div>
            <div style="font-size:13px;color:#64748b;">刚刚留下了回复</div>
          </div>
        </div>
        <div style="padding:18px 20px;border-radius:16px;background:#f8fafc;border-left:4px solid #14b8a6;color:#334155;line-height:1.8;">%s</div>
        <p style="margin:22px 0 0;font-size:13px;color:#8a95a5;line-height:1.7;">你可以在个人设置中关闭评论回复邮件提醒。</p>
      </div>
    </div>
  </div>
</body>
</html>`,
		html.EscapeString(displayName(notification.ReplyAuthorUsername)),
		html.EscapeString(notification.ArticleTitle),
		html.EscapeString(displayName(notification.ReplyAuthorUsername)),
		html.EscapeString(notification.CommentPreview),
	)
}

func displayName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "读者"
	}
	return value
}

func commentPreview(content string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
	if normalized == "" {
		return "对方回复了你的评论。"
	}
	runes := []rune(normalized)
	if len(runes) <= 120 {
		return normalized
	}
	return string(runes[:120]) + "..."
}
