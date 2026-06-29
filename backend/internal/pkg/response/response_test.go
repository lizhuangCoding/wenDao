package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccessEncodesTypedData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	Success(ctx, map[string]string{"status": "ok"})

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var resp Response[map[string]string]
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != CodeSuccess {
		t.Fatalf("expected code %d, got %d", CodeSuccess, resp.Code)
	}
	if resp.Message != "success" {
		t.Fatalf("expected message %q, got %q", "success", resp.Message)
	}
	if resp.Data == nil || (*resp.Data)["status"] != "ok" {
		t.Fatalf("expected typed data payload, got %#v", resp.Data)
	}
}
