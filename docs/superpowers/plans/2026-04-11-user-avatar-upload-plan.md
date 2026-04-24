# User Avatar Upload Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a personal-center avatar upload flow for logged-in users while reusing the existing image upload pipeline and keeping avatar state consistent after page refresh.

**Architecture:** Keep the current admin upload flow unchanged and add a user-scoped avatar endpoint that calls the existing `UploadService.UploadImage(...)`, then writes the returned file URL back to `users.avatar_url`. On the frontend, add a protected `/profile` page, expose an `uploadAvatar` API, and make `/api/auth/me` return a full user payload so the auth store can recover avatar state correctly on reload.

**Tech Stack:** Go, Gin, GORM, existing upload service/repository pipeline, React, TypeScript, Zustand, Axios, React Router

---

**Git note:** The user explicitly asked not to perform git operations for this work. Do not add commit steps while executing this plan.

## File Structure

**Create:**
- `/Users/lizhuang/go/src/wenDao/backend/internal/handler/user_avatar_test.go` — backend handler contract tests for `/api/auth/me` avatar payload and `/api/users/me/avatar`
- `/Users/lizhuang/go/src/wenDao/frontend/src/pages/Profile.tsx` — personal center page with avatar preview and upload control

**Modify:**
- `/Users/lizhuang/go/src/wenDao/backend/cmd/server/main.go` — fix `UserHandler` dependency injection and register the new avatar route
- `/Users/lizhuang/go/src/wenDao/backend/cmd/server/routes_test.go` — lock route registration for the new avatar endpoint
- `/Users/lizhuang/go/src/wenDao/backend/internal/handler/auth.go` — return full user payload plus `expires_in` from `/api/auth/me`
- `/Users/lizhuang/go/src/wenDao/backend/internal/handler/user.go` — add `UploadAvatar` handler that reuses the existing upload service
- `/Users/lizhuang/go/src/wenDao/frontend/src/router.tsx` — register protected `/profile`
- `/Users/lizhuang/go/src/wenDao/frontend/src/types/index.ts` — add `CurrentUserResponse`
- `/Users/lizhuang/go/src/wenDao/frontend/src/api/auth.ts` — type `/auth/me` correctly
- `/Users/lizhuang/go/src/wenDao/frontend/src/api/upload.ts` — add `uploadAvatar(file)`
- `/Users/lizhuang/go/src/wenDao/frontend/src/store/authStore.ts` — add `setUser(...)` and consume the new `/auth/me` response shape
- `/Users/lizhuang/go/src/wenDao/frontend/src/hooks/useAuth.ts` — expose `setUser`
- `/Users/lizhuang/go/src/wenDao/frontend/src/components/common/Header.tsx` — add entry to the personal center

**Unchanged on purpose:**
- `/Users/lizhuang/go/src/wenDao/backend/internal/service/upload.go` — avatar upload must reuse this existing pipeline as-is
- `/Users/lizhuang/go/src/wenDao/backend/internal/service/user.go` — `UpdateAvatar(...)` already exists and should be reused
- `/Users/lizhuang/go/src/wenDao/backend/internal/repository/upload.go` — existing upload record persistence remains valid
- `/Users/lizhuang/go/src/wenDao/backend/internal/model/user.go` — `AvatarURL` field already exists
- `/Users/lizhuang/go/src/wenDao/backend/internal/model/upload.go` — existing upload metadata model remains valid

---

### Task 1: Lock backend route coverage before implementation

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/backend/cmd/server/routes_test.go`
- Modify: `/Users/lizhuang/go/src/wenDao/backend/cmd/server/main.go`

- [ ] **Step 1: Add the new route expectation to the route-registration test**

In `/Users/lizhuang/go/src/wenDao/backend/cmd/server/routes_test.go`, extend `required := []string{...}` to include the current-user route and the new avatar route:

```go
	required := []string{
		"GET /api/articles",
		"GET /api/articles/:id",
		"GET /api/articles/slug/:slug",
		"GET /api/categories/:id/articles",
		"GET /api/comments/article/:id",
		"POST /api/auth/refresh",
		"GET /api/auth/me",
		"POST /api/users/me/avatar",
		"GET /api/admin/articles/:id",
		"GET /api/admin/comments",
		"POST /api/admin/comments/:id/restore",
	}
```

- [ ] **Step 2: Run the route test and verify it fails before the route exists**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go test ./cmd/server -run TestRegisterRoutes_RegistersRequiredRoutes -v
```

Expected: FAIL with a message like `expected route POST /api/users/me/avatar to be registered`.

- [ ] **Step 3: Fix `UserHandler` injection and register the user avatar route**

In `/Users/lizhuang/go/src/wenDao/backend/cmd/server/main.go`, update the handler construction and route registration:

```go
	userHandler := handler.NewUserHandler(userService, uploadService, oauthService, cfg)
```

```go
	authRequired := api.Group("")
	authRequired.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb))
	{
		authRequired.POST("/auth/logout", authHandler.Logout)
		authRequired.GET("/auth/me", authHandler.GetUserInfo)
		authRequired.POST("/users/me/avatar", userHandler.UploadAvatar)
		authRequired.POST("/comments", commentHandler.Create)
		authRequired.DELETE("/comments/:id", commentHandler.Delete)
	}
```

- [ ] **Step 4: Re-run the route test and verify it passes**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go test ./cmd/server -run TestRegisterRoutes_RegistersRequiredRoutes -v
```

Expected: PASS.

---

### Task 2: Implement backend handler contracts for avatar upload and `/auth/me`

**Files:**
- Create: `/Users/lizhuang/go/src/wenDao/backend/internal/handler/user_avatar_test.go`
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/handler/auth.go`
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/handler/user.go`

- [ ] **Step 1: Write failing handler tests for the new backend behavior**

Create `/Users/lizhuang/go/src/wenDao/backend/internal/handler/user_avatar_test.go` with this content:

```go
package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"wenDao/config"
	"wenDao/internal/model"
)

type stubUserService struct {
	currentUser   *model.User
	lastUserID    int64
	lastAvatarURL string
	updateErr     error
	getErr        error
}

func (s *stubUserService) Register(email, password, username string) (*model.User, error) {
	return nil, nil
}

func (s *stubUserService) Login(email, password string) (string, *model.User, error) {
	return "", nil, nil
}

func (s *stubUserService) GitHubOAuthLogin(code string) (string, *model.User, error) {
	return "", nil, nil
}

func (s *stubUserService) Logout(token string) error {
	return nil
}

func (s *stubUserService) GetCurrentUser(userID int64) (*model.User, error) {
	s.lastUserID = userID
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.currentUser, nil
}

func (s *stubUserService) GenerateRefreshToken(userID int64, role string) (string, error) {
	return "", nil
}

func (s *stubUserService) UpdateAvatar(userID int64, avatarURL string) error {
	s.lastUserID = userID
	s.lastAvatarURL = avatarURL
	if s.updateErr != nil {
		return s.updateErr
	}
	if s.currentUser != nil {
		s.currentUser.AvatarURL = &avatarURL
	}
	return nil
}

type stubUploadService struct {
	upload     *model.Upload
	err        error
	lastUserID int64
}

func (s *stubUploadService) UploadImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	s.lastUserID = userID
	if s.err != nil {
		return nil, s.err
	}
	return s.upload, nil
}

func TestAuthHandlerGetUserInfo_IncludesAvatarURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	avatarURL := "/uploads/2026/04/avatar.png"
	userService := &stubUserService{
		currentUser: &model.User{
			ID:        7,
			Email:     "avatar@example.com",
			Username:  "avatar-user",
			Role:      "user",
			AvatarURL: &avatarURL,
		},
	}

	handler := NewAuthHandler(userService, &config.Config{JWT: config.JWTConfig{AccessExpireHours: 2}}, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	ctx.Set("user_id", int64(7))

	handler.GetUserInfo(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"avatar_url":"/uploads/2026/04/avatar.png"`)) {
		t.Fatalf("expected response body to include avatar_url, got %s", recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"expires_in":7200`)) {
		t.Fatalf("expected response body to include expires_in, got %s", recorder.Body.String())
	}
}

func TestUserHandlerUploadAvatar_UpdatesAvatarAndReturnsUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userService := &stubUserService{
		currentUser: &model.User{
			ID:       11,
			Email:    "profile@example.com",
			Username: "profile-user",
			Role:     "user",
		},
	}
	uploadService := &stubUploadService{
		upload: &model.Upload{FilePath: "/uploads/2026/04/new-avatar.png"},
	}
	handler := &UserHandler{userService: userService, uploadService: uploadService, cfg: &config.Config{}}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatalf("failed to create multipart file: %v", err)
	}
	if _, err := part.Write([]byte("fake-image-content")); err != nil {
		t.Fatalf("failed to write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/users/me/avatar", &body)
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
	ctx.Set("user_id", int64(11))

	handler.UploadAvatar(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if uploadService.lastUserID != 11 {
		t.Fatalf("expected upload service to receive user id 11, got %d", uploadService.lastUserID)
	}
	if userService.lastAvatarURL != "/uploads/2026/04/new-avatar.png" {
		t.Fatalf("expected avatar url to be updated, got %q", userService.lastAvatarURL)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"avatar_url":"/uploads/2026/04/new-avatar.png"`)) {
		t.Fatalf("expected response body to include updated avatar_url, got %s", recorder.Body.String())
	}
}
```

- [ ] **Step 2: Run the handler tests and verify they fail before implementation**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go test ./internal/handler -run 'Test(AuthHandlerGetUserInfo_IncludesAvatarURL|UserHandlerUploadAvatar_UpdatesAvatarAndReturnsUser)' -v
```

Expected: FAIL because `UserHandler.UploadAvatar` does not exist yet and `/api/auth/me` still omits `avatar_url`.

- [ ] **Step 3: Implement `/api/auth/me` and `UploadAvatar` with the existing services**

In `/Users/lizhuang/go/src/wenDao/backend/internal/handler/auth.go`, change `GetUserInfo` to return the full user object plus `expires_in`:

```go
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	user, err := h.userService.GetCurrentUser(userID.(int64))
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	user.PasswordHash = nil
	response.Success(c, gin.H{
		"user":       user,
		"expires_in": h.cfg.JWT.AccessExpireHours * 3600,
	})
}
```

In `/Users/lizhuang/go/src/wenDao/backend/internal/handler/user.go`, add `strings` to the imports and append this handler below `GetCurrentUser`:

```go
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.InvalidParams(c, "Missing file parameter")
		return
	}
	defer file.Close()

	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Missing user ID")
		return
	}

	upload, err := h.uploadService.UploadImage(file, header, userID.(int64))
	if err != nil {
		switch {
		case err.Error() == "file type not allowed":
			response.InvalidParams(c, "File type not allowed. Only jpg, png, gif, webp are supported.")
		case strings.HasPrefix(err.Error(), "file size exceeds limit"):
			response.InvalidParams(c, err.Error())
		default:
			response.InternalError(c, "Failed to upload avatar")
		}
		return
	}

	if err := h.userService.UpdateAvatar(userID.(int64), upload.FilePath); err != nil {
		response.InternalError(c, "Failed to update avatar")
		return
	}

	user, err := h.userService.GetCurrentUser(userID.(int64))
	if err != nil {
		response.InternalError(c, "Failed to get user")
		return
	}

	user.PasswordHash = nil
	response.Success(c, user)
}
```

Also update the import block in `/Users/lizhuang/go/src/wenDao/backend/internal/handler/user.go` to:

```go
import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"

	"wenDao/config"
	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
)
```

- [ ] **Step 4: Re-run the handler tests and verify they pass**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go test ./internal/handler -run 'Test(AuthHandlerGetUserInfo_IncludesAvatarURL|UserHandlerUploadAvatar_UpdatesAvatarAndReturnsUser)' -v
```

Expected: PASS.

- [ ] **Step 5: Run the backend-focused regression suite**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go test ./cmd/server ./internal/handler -v
```

Expected: PASS.

---

### Task 3: Add the personal-center page and make the frontend data layer compile against it

**Files:**
- Create: `/Users/lizhuang/go/src/wenDao/frontend/src/pages/Profile.tsx`
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/router.tsx`
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/types/index.ts`
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/api/auth.ts`
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/api/upload.ts`
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/store/authStore.ts`
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/hooks/useAuth.ts`

- [ ] **Step 1: Write the consumer-first profile page and route so the build fails on missing API/store contracts**

Create `/Users/lizhuang/go/src/wenDao/frontend/src/pages/Profile.tsx` with this content:

```tsx
import { ChangeEvent, useRef, useState } from 'react';
import { Layout, Loading } from '@/components/common';
import { uploadApi } from '@/api';
import { useAuth } from '@/hooks';
import { useUIStore } from '@/store';

export const Profile = () => {
  const { user, setUser } = useAuth();
  const { showToast } = useUIStore();
  const [isUploading, setIsUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const handleFileChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }

    setIsUploading(true);
    try {
      const updatedUser = await uploadApi.uploadAvatar(file);
      setUser(updatedUser);
      showToast('头像更新成功', 'success');
    } catch (error: any) {
      showToast(error.message || '头像上传失败，请重试', 'error');
    } finally {
      event.target.value = '';
      setIsUploading(false);
    }
  };

  if (!user) {
    return (
      <Layout>
        <div className="max-w-3xl mx-auto px-6 sm:px-10 lg:px-12 py-24 flex justify-center">
          <Loading />
        </div>
      </Layout>
    );
  }

  const avatarSrc = user.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${user.username}`;

  return (
    <Layout>
      <div className="max-w-3xl mx-auto px-6 sm:px-10 lg:px-12 py-24">
        <div className="rounded-[32px] border border-neutral-200/70 dark:border-neutral-800 bg-white/90 dark:bg-neutral-900/90 shadow-soft p-8 md:p-10 space-y-8">
          <div>
            <h1 className="text-3xl font-serif font-black text-neutral-900 dark:text-neutral-100">个人中心</h1>
            <p className="mt-2 text-sm text-neutral-500 dark:text-neutral-400">
              初始头像会自动生成，你也可以在这里上传自己的头像。
            </p>
          </div>

          <div className="flex flex-col md:flex-row md:items-center gap-8">
            <div className="w-32 h-32 rounded-full overflow-hidden border-4 border-primary-100 dark:border-primary-900 shadow-soft">
              <img src={avatarSrc} alt={user.username} className="w-full h-full object-cover" />
            </div>

            <div className="space-y-4">
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                disabled={isUploading}
                className="btn btn-primary disabled:opacity-60 disabled:cursor-not-allowed"
              >
                {isUploading ? '上传中...' : '更换头像'}
              </button>
              <p className="text-xs text-neutral-500 dark:text-neutral-400">
                支持 jpg、png、gif、webp，大小限制沿用现有图片上传配置。
              </p>
              <input
                ref={fileInputRef}
                type="file"
                accept="image/jpeg,image/png,image/gif,image/webp"
                onChange={handleFileChange}
                className="hidden"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="rounded-2xl border border-neutral-200/70 dark:border-neutral-800 px-5 py-4">
              <p className="text-xs font-bold tracking-[0.2em] text-neutral-400 dark:text-neutral-500 uppercase">用户名</p>
              <p className="mt-2 text-lg font-semibold text-neutral-900 dark:text-neutral-100">{user.username}</p>
            </div>
            <div className="rounded-2xl border border-neutral-200/70 dark:border-neutral-800 px-5 py-4">
              <p className="text-xs font-bold tracking-[0.2em] text-neutral-400 dark:text-neutral-500 uppercase">邮箱</p>
              <p className="mt-2 text-lg font-semibold text-neutral-900 dark:text-neutral-100">{user.email}</p>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
};
```

In `/Users/lizhuang/go/src/wenDao/frontend/src/router.tsx`, add the import and protected route:

```tsx
import { Profile } from './pages/Profile';
```

```tsx
  {
    path: '/profile',
    element: (
      <ProtectedRoute>
        <Profile />
      </ProtectedRoute>
    ),
  },
```

- [ ] **Step 2: Run the frontend build and verify it fails on the missing contracts**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run build
```

Expected: FAIL with TypeScript errors about `setUser` not existing on `useAuth()` and `uploadAvatar` not existing on `uploadApi`.

- [ ] **Step 3: Implement the missing type, API, and auth-store contracts**

In `/Users/lizhuang/go/src/wenDao/frontend/src/types/index.ts`, add this interface below `AuthResponse`:

```ts
export interface CurrentUserResponse {
  user: User;
  expires_in: number;
}
```

In `/Users/lizhuang/go/src/wenDao/frontend/src/api/auth.ts`, update the imports and `/auth/me` typing:

```ts
import type {
  LoginRequest,
  RegisterRequest,
  AuthResponse,
  CurrentUserResponse,
} from '@/types';
```

```ts
  getCurrentUser: () => {
    return request.get<CurrentUserResponse>('/auth/me');
  },
```

In `/Users/lizhuang/go/src/wenDao/frontend/src/api/upload.ts`, add the user import and the new avatar method:

```ts
import type { User } from '@/types';
```

```ts
export const uploadApi = {
  uploadImage: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return request.post<UploadResponse>('/admin/upload/image', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  },

  uploadAvatar: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return request.post<User>('/users/me/avatar', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  },
};
```

In `/Users/lizhuang/go/src/wenDao/frontend/src/store/authStore.ts`, extend the state and consume the new response shape:

```ts
interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isAdmin: boolean;

  setAuth: (user: User, token: string) => void;
  setUser: (user: User) => void;
  clearAuth: () => void;
  login: (email: string, password: string) => Promise<void>;
  register: (username: string, email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  fetchCurrentUser: () => Promise<void>;
}
```

```ts
      setAuth: (user, token) => {
        localStorage.setItem('access_token', token);
        set({
          user,
          token,
          isAuthenticated: true,
          isAdmin: user.role === 'admin',
        });
      },

      setUser: (user) => {
        set((state) => ({
          ...state,
          user,
          isAuthenticated: true,
          isAdmin: user.role === 'admin',
        }));
      },
```

```ts
      fetchCurrentUser: async () => {
        try {
          const response = await authApi.getCurrentUser();
          get().setUser(response.user);
        } catch (error) {
          get().clearAuth();
        }
      },
```

In `/Users/lizhuang/go/src/wenDao/frontend/src/hooks/useAuth.ts`, expose the new store action:

```ts
export const useAuth = () => {
  const { user, isAuthenticated, isAdmin, login, register, logout, fetchCurrentUser, setUser } =
    useAuthStore();

  return {
    user,
    isAuthenticated,
    isAdmin,
    login,
    register,
    logout,
    fetchCurrentUser,
    setUser,
  };
};
```

- [ ] **Step 4: Re-run the frontend build and verify the profile page compiles**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run build
```

Expected: PASS.

---

### Task 4: Add the personal-center entry point and run full verification

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/components/common/Header.tsx`
- Verify: `/Users/lizhuang/go/src/wenDao/backend/cmd/server/main.go`
- Verify: `/Users/lizhuang/go/src/wenDao/backend/internal/handler/auth.go`
- Verify: `/Users/lizhuang/go/src/wenDao/backend/internal/handler/user.go`
- Verify: `/Users/lizhuang/go/src/wenDao/frontend/src/pages/Profile.tsx`

- [ ] **Step 1: Add a visible entry from the header to the personal center**

In `/Users/lizhuang/go/src/wenDao/frontend/src/components/common/Header.tsx`, wrap the current avatar/name block with a `Link` to `/profile`:

```tsx
                <Link to="/profile" className="flex items-center gap-3 group">
                  <div className="w-10 h-10 rounded-full overflow-hidden border-2 border-primary-100 dark:border-primary-800 shadow-soft cursor-pointer transition-transform group-hover:scale-105">
                    <img
                      src={user?.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${user?.username}`}
                      alt={user?.username}
                      className="w-full h-full object-cover"
                    />
                  </div>
                  <div className="flex flex-col">
                    <span className="text-sm font-bold text-neutral-800 dark:text-neutral-200 leading-tight group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors">
                      {user?.username}
                    </span>
                    <span className="text-[10px] text-neutral-400 font-bold uppercase tracking-widest">{user?.role}</span>
                  </div>
                </Link>
```

Replace only the existing avatar/name `div` block and keep the admin link plus logout button unchanged.

- [ ] **Step 2: Run the final automated verification commands**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go test ./cmd/server ./internal/handler -v
```

Expected: PASS.

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run build
```

Expected: PASS.

- [ ] **Step 3: Perform manual end-to-end verification**

Start the app with the project’s normal backend/frontend startup flow, then verify this exact checklist:

1. Log in as a normal user.
2. Confirm the header avatar renders.
3. Click the avatar/name block and confirm `/profile` loads.
4. Confirm the page shows the current avatar, username, and email.
5. Upload a valid image and confirm a success toast appears.
6. Refresh the browser and confirm the new avatar still appears in both `/profile` and the header.
7. Confirm the uploaded avatar URL is served from `/uploads/...`.
8. Confirm the admin article image upload flow still works unchanged.

Expected: All checks pass.

---

## Coverage Check

This plan covers every approved spec requirement:
- user gets a default avatar initially: preserved by leaving registration/OAuth avatar defaults untouched;
- user can upload a custom avatar from a personal center: implemented by `/profile` + `/api/users/me/avatar`;
- avatar upload stays compatible with the existing image module: backend reuses `UploadService.UploadImage(...)` and existing `uploads` records;
- `users.avatar_url` becomes the source of truth: updated by `UserService.UpdateAvatar(...)`;
- avatar survives refresh: fixed by changing `/api/auth/me` to return a full user payload including `avatar_url`.

No extra scope has been added: no avatar cropping, no file-deletion workflow, no profile editing, and no storage redesign.
