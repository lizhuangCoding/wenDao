package comment

import (
	"strings"
	"testing"
)

func TestCommentReplyEmailDoesNotRenderReplyLinkButton(t *testing.T) {
	body := commentReplyEmailHTMLBody(CommentReplyNotification{
		RecipientUsername:   "reader",
		ReplyAuthorUsername: "author",
		ArticleTitle:        "文章标题",
		CommentPreview:      "回复内容",
	})

	for _, unwanted := range []string{"href=", "查看回复"} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("expected reply email html not to contain %q, got:\n%s", unwanted, body)
		}
	}
}
