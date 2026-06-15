package handlerutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestMustGetInt64Param_Success(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test/42", nil)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	val, ok := MustGetInt64Param(c, "id")
	if !ok {
		t.Fatal("expected ok==true")
	}
	if val != 42 {
		t.Fatalf("expected 42, got %d", val)
	}
}

func TestMustGetInt64Param_Invalid(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test/abc", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	_, ok := MustGetInt64Param(c, "id")
	if ok {
		t.Fatal("expected ok==false for invalid param")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestMustGetInt64Param_Missing(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	_, ok := MustGetInt64Param(c, "id")
	if ok {
		t.Fatal("expected ok==false for missing param")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestMustGetInt64Param_Zero(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test/0", nil)
	c.Params = gin.Params{{Key: "id", Value: "0"}}

	val, ok := MustGetInt64Param(c, "id")
	if !ok {
		t.Fatal("expected ok==true for zero value")
	}
	if val != 0 {
		t.Fatalf("expected 0, got %d", val)
	}
}

func TestMustGetInt64Param_Negative(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test/-1", nil)
	c.Params = gin.Params{{Key: "id", Value: "-1"}}

	val, ok := MustGetInt64Param(c, "id")
	if !ok {
		t.Fatal("expected ok==true for negative value")
	}
	if val != -1 {
		t.Fatalf("expected -1, got %d", val)
	}
}

func TestMustGetInt64Param_LargeValue(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test/9223372036854775807", nil)
	c.Params = gin.Params{{Key: "id", Value: "9223372036854775807"}}

	val, ok := MustGetInt64Param(c, "id")
	if !ok {
		t.Fatal("expected ok==true for max int64")
	}
	if val != 9223372036854775807 {
		t.Fatalf("expected max int64, got %d", val)
	}
}

func TestMustGetUserID_Success(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Set("user_id", int64(1))

	val, ok := MustGetUserID(c)
	if !ok {
		t.Fatal("expected ok==true")
	}
	if val != 1 {
		t.Fatalf("expected 1, got %d", val)
	}
}

func TestMustGetUserID_Missing(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	_, ok := MustGetUserID(c)
	if ok {
		t.Fatal("expected ok==false for missing user_id")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestMustGetUserID_WrongType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Set("user_id", "not_an_int64")

	_, ok := MustGetUserID(c)
	if ok {
		t.Fatal("expected ok==false for wrong type")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for wrong type, got %d", w.Code)
	}
}

func TestMustGetUserID_ZeroValue(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Set("user_id", int64(0))

	val, ok := MustGetUserID(c)
	if !ok {
		t.Fatal("expected ok==true for zero user_id")
	}
	if val != 0 {
		t.Fatalf("expected 0, got %d", val)
	}
}

func TestMustGetOptionalQueryInt_HasValue(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test?page=3", nil)

	val := MustGetOptionalQueryInt(c, "page", 1)
	if val != 3 {
		t.Fatalf("expected 3, got %d", val)
	}
}

func TestMustGetOptionalQueryInt_Missing(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	val := MustGetOptionalQueryInt(c, "page", 1)
	if val != 1 {
		t.Fatalf("expected default 1, got %d", val)
	}
}

func TestMustGetOptionalQueryInt_Invalid(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test?page=abc", nil)

	val := MustGetOptionalQueryInt(c, "page", 10)
	if val != 10 {
		t.Fatalf("expected default 10 for invalid input, got %d", val)
	}
}

func TestMustGetOptionalQueryInt_Zero(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test?limit=0", nil)

	val := MustGetOptionalQueryInt(c, "limit", 10)
	if val != 0 {
		t.Fatalf("expected 0, got %d", val)
	}
}

func TestMustGetOptionalQueryInt_Negative(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test?offset=-5", nil)

	val := MustGetOptionalQueryInt(c, "offset", 0)
	if val != -5 {
		t.Fatalf("expected -5, got %d", val)
	}
}

func TestMustGetOptionalQueryInt_EmptyString(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test?page=", nil)

	val := MustGetOptionalQueryInt(c, "page", 1)
	if val != 1 {
		t.Fatalf("expected default 1 for empty string, got %d", val)
	}
}

func TestMustGetOptionalQueryInt_Float(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test?page=3.14", nil)

	val := MustGetOptionalQueryInt(c, "page", 1)
	if val != 1 {
		t.Fatalf("expected default 1 for float input, got %d", val)
	}
}
