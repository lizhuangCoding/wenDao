# Username Update and Auto-Save Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow users to change their username and implement a 30-second auto-save feature for the article editor.

**Architecture:** 
1. **Username Update**: Backend `PUT /api/users/me/username` -> `UserService` -> `UserRepo`. Frontend `Profile` page UI update.
2. **Auto-Save**: Backend `PUT /api/admin/articles/:id/autosave` (lightweight, no RAG re-indexing). Frontend `ArticleEditor` with `setInterval` (30s) and dirty check.

**Tech Stack:** Go (Gin, GORM), React (TypeScript, Zustand, Axios)

---

### Task 1: Backend Username Update

**Files:**
- Modify: `backend/internal/service/user.go`
- Modify: `backend/internal/handler/user.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Update `UserService` interface and implementation in `backend/internal/service/user.go`**

```go
// Add to interface
UpdateUsername(userID int64, username string) error

// Add to implementation
func (s *userService) UpdateUsername(userID int64, username string) error {
    user, err := s.userRepo.GetByID(userID)
    if err != nil {
        return err
    }
    // Check if username already exists
    existing, err := s.userRepo.GetByUsername(username)
    if err == nil && existing.ID != userID {
        return errors.New("username already exists")
    }
    user.Username = username
    return s.userRepo.Update(user)
}
```

- [ ] **Step 2: Add `UpdateUsername` handler in `backend/internal/handler/user.go`**

```go
type UpdateUsernameRequest struct {
    Username string `json:"username" binding:"required,min=2,max=50"`
}

func (h *UserHandler) UpdateUsername(c *gin.Context) {
    var req UpdateUsernameRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.InvalidParams(c, err.Error())
        return
    }
    userID, _ := c.Get("user_id")
    if err := h.userService.UpdateUsername(userID.(int64), req.Username); err != nil {
        if err.Error() == "username already exists" {
            response.Error(c, response.CodeInvalidParams, "Username already exists")
            return
        }
        response.InternalError(c, "Failed to update username")
        return
    }
    response.Success(c, nil)
}
```

- [ ] **Step 3: Register route in `backend/cmd/server/main.go`**

```go
api.PUT("/users/me/username", userHandler.UpdateUsername)
```

---

### Task 2: Frontend Username Update UI

**Files:**
- Modify: `frontend/src/api/auth.ts`
- Modify: `frontend/src/pages/Profile.tsx`

- [ ] **Step 1: Add API method in `frontend/src/api/auth.ts`**

```typescript
updateUsername: (username: string) => client.put('/users/me/username', { username }),
```

- [ ] **Step 2: Update UI in `frontend/src/pages/Profile.tsx` to support editing username**

---

### Task 3: Backend Article Auto-Save API

**Files:**
- Modify: `backend/internal/service/article.go`
- Modify: `backend/internal/handler/article.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add `AutoSave` to `ArticleService` in `backend/internal/service/article.go`**

```go
// Add to interface
AutoSave(id int64, title, content string) error

// Add to implementation
func (s *articleService) AutoSave(id int64, title, content string) error {
    article, err := s.articleRepo.GetByID(id)
    if err != nil {
        return err
    }
    article.Title = title
    article.Content = content
    // Lightweight update: no cache deletion or re-indexing for autosave
    return s.articleRepo.Update(article)
}
```

- [ ] **Step 2: Add `AutoSave` handler in `backend/internal/handler/article.go`**

```go
type AutoSaveRequest struct {
    Title   string `json:"title" binding:"required"`
    Content string `json:"content" binding:"required"`
}

func (h *ArticleHandler) AutoSave(c *gin.Context) {
    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
    var req AutoSaveRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.InvalidParams(c, err.Error())
        return
    }
    if err := h.articleService.AutoSave(id, req.Title, req.Content); err != nil {
        response.InternalError(c, "Auto-save failed")
        return
    }
    response.Success(c, nil)
}
```

- [ ] **Step 3: Register route in `backend/cmd/server/main.go`**

```go
admin.PUT("/articles/:id/autosave", articleHandler.AutoSave)
```

---

### Task 4: Frontend Article Auto-Save Timer

**Files:**
- Modify: `frontend/src/api/article.ts`
- Modify: `frontend/src/views/admin/articles/ArticleEditor.tsx`

- [ ] **Step 1: Add `autoSave` to `frontend/src/api/article.ts`**
- [ ] **Step 2: Implement auto-save logic in `ArticleEditor.tsx`**
    - Use `useRef` to track `lastSavedContent` and `lastSavedTitle`.
    - Use `useEffect` with `setInterval` (30s).
    - Compare current state with refs.
    - If dirty, call API and update refs.
    - Show "Draft auto-saved at HH:mm:ss" in the footer.
