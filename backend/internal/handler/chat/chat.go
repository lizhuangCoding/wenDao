package chat

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/config"
	"wenDao/internal/model"
	"wenDao/internal/pkg/response"
	"wenDao/internal/repository"
)

// ChatHandler 对话处理器
type ChatHandler struct {
	cfg         *config.Config
	convRepo    repository.ConversationRepository
	msgRepo     repository.ChatMessageRepository
	runRepo     repository.ConversationRunRepository
	runStepRepo repository.ConversationRunStepRepository
	memoryRepo  repository.ConversationMemoryRepository
}

// NewChatHandler 创建对话处理器
func NewChatHandler(
	cfg *config.Config,
	convRepo repository.ConversationRepository,
	msgRepo repository.ChatMessageRepository,
	runRepo repository.ConversationRunRepository,
	runStepRepo repository.ConversationRunStepRepository,
	memoryRepo repository.ConversationMemoryRepository,
) *ChatHandler {
	return &ChatHandler{
		cfg:         cfg,
		convRepo:    convRepo,
		msgRepo:     msgRepo,
		runRepo:     runRepo,
		runStepRepo: runStepRepo,
		memoryRepo:  memoryRepo,
	}
}

// CreateConversationRequest 创建对话请求
type CreateConversationRequest struct {
	Title string `json:"title" binding:"required,min=1,max=255"`
}

// UpdateConversationRequest 更新对话请求
type UpdateConversationRequest struct {
	Title string `json:"title" binding:"required,min=1,max=255"`
}

// ConversationResponse 对话响应
type ConversationResponse struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	Title      string `json:"title"`
	ShareToken string `json:"share_token,omitempty"`
	IsShared   bool   `json:"is_shared"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// MessageResponse 消息响应
type MessageResponse struct {
	ID             int64          `json:"id"`
	ConversationID int64          `json:"conversation_id"`
	RunID          *int64         `json:"run_id,omitempty"`
	Role           string         `json:"role"`
	Content        string         `json:"content"`
	CreatedAt      string         `json:"created_at"`
	ProcessSteps   []StepResponse `json:"process_steps,omitempty"`
}

// ConversationDetailResponse 对话详情响应
type ConversationDetailResponse struct {
	Conversation ConversationResponse `json:"conversation"`
	Messages     []MessageResponse    `json:"messages"`
	Steps        []StepResponse       `json:"steps,omitempty"`
	ActiveRun    *ActiveRunResponse   `json:"active_run,omitempty"`
	ActiveSteps  []StepResponse       `json:"active_steps,omitempty"`
}

type ActiveRunResponse struct {
	ID              int64   `json:"id"`
	Status          string  `json:"status"`
	CurrentStage    string  `json:"current_stage"`
	PendingQuestion *string `json:"pending_question,omitempty"`
	LastAnswer      string  `json:"last_answer"`
	HeartbeatAt     string  `json:"heartbeat_at,omitempty"`
	CanResume       bool    `json:"can_resume"`
}

// StepResponse 步骤响应
type StepResponse struct {
	ID        int64  `json:"id"`
	RunID     int64  `json:"run_id"`
	AgentName string `json:"agent_name"`
	Type      string `json:"type"`
	Summary   string `json:"summary"`
	Detail    string `json:"detail"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func buildStepResponse(step model.ConversationRunStep) StepResponse {
	return StepResponse{
		ID:        step.ID,
		RunID:     step.RunID,
		AgentName: step.AgentName,
		Type:      step.Type,
		Summary:   step.Summary,
		Detail:    step.Detail,
		Status:    step.Status,
		CreatedAt: step.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func parseConversationID(c *gin.Context) (int64, bool) {
	convID := c.Param("id")
	var convIDInt int64
	if _, err := fmt.Sscanf(convID, "%d", &convIDInt); err != nil {
		response.InvalidParams(c, "会话 ID 无效，请刷新页面后重试")
		return 0, false
	}
	return convIDInt, true
}

func buildConversationResponse(conv *model.Conversation) ConversationResponse {
	return ConversationResponse{
		ID:         conv.ID,
		UserID:     conv.UserID,
		Title:      conv.Title,
		ShareToken: conv.ShareToken,
		IsShared:   conv.IsShared,
		CreatedAt:  conv.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:  conv.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// List 获取用户对话列表
// GET /api/chat/conversations
func (h *ChatHandler) List(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "登录状态已失效，请重新登录后操作")
		return
	}

	convs, err := h.convRepo.GetByUserID(userID.(int64))
	if err != nil {
		response.InternalError(c, "会话列表加载失败，请稍后重试")
		return
	}

	result := make([]ConversationResponse, len(convs))
	for i, conv := range convs {
		result[i] = buildConversationResponse(&conv)
	}

	response.Success(c, result)
}

// Create 创建新对话
// POST /api/chat/conversations
func (h *ChatHandler) Create(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "登录状态已失效，请重新登录后操作")
		return
	}

	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "会话标题不能为空")
		return
	}

	conv := &model.Conversation{
		UserID: userID.(int64),
		Title:  req.Title,
	}

	if err := h.convRepo.Create(conv); err != nil {
		response.InternalError(c, "创建会话失败，请稍后重试")
		return
	}

	response.Success(c, buildConversationResponse(conv))
}

// Get 获取对话详情（含消息）
// GET /api/chat/conversations/:id
func (h *ChatHandler) Get(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "登录状态已失效，请重新登录后操作")
		return
	}

	convIDInt, ok := parseConversationID(c)
	if !ok {
		return
	}

	conv, err := h.convRepo.GetByID(convIDInt)
	if err != nil {
		response.NotFound(c, "会话不存在或已被删除")
		return
	}

	if conv.UserID != userID.(int64) {
		response.Forbidden(c, "你没有权限查看这个会话")
		return
	}

	msgs, err := h.msgRepo.GetByConversationID(convIDInt)
	if err != nil {
		response.InternalError(c, "会话消息加载失败，请稍后重试")
		return
	}

	var steps []model.ConversationRunStep
	if h.runStepRepo != nil {
		steps, _ = h.runStepRepo.GetByConversationID(convIDInt)
	}
	var activeRun *model.ConversationRun
	if h.runRepo != nil {
		activeRun, _ = h.runRepo.GetActiveByConversationID(convIDInt)
	}

	stepResponses := make([]StepResponse, len(steps))
	stepsByRunID := make(map[int64][]StepResponse)
	for i, step := range steps {
		stepResponse := buildStepResponse(step)
		stepResponses[i] = stepResponse
		stepsByRunID[step.RunID] = append(stepsByRunID[step.RunID], stepResponse)
	}

	msgResponses := make([]MessageResponse, len(msgs))
	for i, msg := range msgs {
		msgResponse := MessageResponse{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			RunID:          msg.RunID,
			Role:           msg.Role,
			Content:        msg.Content,
			CreatedAt:      msg.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if msg.RunID != nil {
			msgResponse.ProcessSteps = stepsByRunID[*msg.RunID]
		}
		msgResponses[i] = msgResponse
	}

	var activeRunResponse *ActiveRunResponse
	activeStepResponses := make([]StepResponse, 0)
	if isResumableRun(activeRun, msgs) {
		var heartbeatAt string
		if activeRun.HeartbeatAt != nil {
			heartbeatAt = activeRun.HeartbeatAt.Format("2006-01-02 15:04:05")
		}
		activeRunResponse = &ActiveRunResponse{
			ID:              activeRun.ID,
			Status:          activeRun.Status,
			CurrentStage:    activeRun.CurrentStage,
			PendingQuestion: activeRun.PendingQuestion,
			LastAnswer:      activeRun.LastAnswer,
			HeartbeatAt:     heartbeatAt,
			CanResume:       activeRun.Status == "running" || activeRun.Status == "waiting_user",
		}
		if h.runStepRepo != nil {
			activeSteps, _ := h.runStepRepo.GetByRunID(activeRun.ID)
			activeStepResponses = make([]StepResponse, len(activeSteps))
			for i, step := range activeSteps {
				activeStepResponses[i] = buildStepResponse(step)
			}
		}
	}

	response.Success(c, ConversationDetailResponse{
		Conversation: buildConversationResponse(conv),
		Messages:     msgResponses,
		Steps:        stepResponses,
		ActiveRun:    activeRunResponse,
		ActiveSteps:  activeStepResponses,
	})
}

func isResumableRun(run *model.ConversationRun, messages []model.ChatMessage) bool {
	if run == nil {
		return false
	}
	if run.Status != "running" && run.Status != "waiting_user" {
		return false
	}
	if run.Status != "running" {
		return true
	}
	for _, msg := range messages {
		if msg.Role == "assistant" && !msg.CreatedAt.Before(run.CreatedAt) {
			return false
		}
	}
	return true
}

// Update 更新对话标题
// PATCH /api/chat/conversations/:id
func (h *ChatHandler) Update(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "登录状态已失效，请重新登录后操作")
		return
	}

	convIDInt, ok := parseConversationID(c)
	if !ok {
		return
	}

	var req UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "会话标题不能为空")
		return
	}

	conv, err := h.convRepo.GetByID(convIDInt)
	if err != nil {
		response.NotFound(c, "会话不存在或已被删除")
		return
	}

	if conv.UserID != userID.(int64) {
		response.Forbidden(c, "你没有权限修改这个会话")
		return
	}

	conv.Title = req.Title
	conv.UpdatedAt = time.Now()
	if err := h.convRepo.Update(conv); err != nil {
		response.InternalError(c, "更新会话标题失败，请稍后重试")
		return
	}

	response.Success(c, buildConversationResponse(conv))
}

// Delete 删除对话
// DELETE /api/chat/conversations/:id
func (h *ChatHandler) Delete(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "登录状态已失效，请重新登录后操作")
		return
	}

	convIDInt, ok := parseConversationID(c)
	if !ok {
		return
	}

	conv, err := h.convRepo.GetByID(convIDInt)
	if err != nil {
		response.NotFound(c, "会话不存在或已被删除")
		return
	}

	if conv.UserID != userID.(int64) {
		response.Forbidden(c, "你没有权限删除这个会话")
		return
	}

	if err := h.msgRepo.DeleteByConversationID(convIDInt); err != nil {
		response.InternalError(c, "删除会话消息失败，请稍后重试")
		return
	}

	if h.runStepRepo != nil {
		if err := h.runStepRepo.DeleteByConversationID(convIDInt); err != nil {
			response.InternalError(c, "删除会话运行步骤失败，请稍后重试")
			return
		}
	}

	if h.runRepo != nil {
		if err := h.runRepo.DeleteByConversationID(convIDInt); err != nil {
			response.InternalError(c, "删除会话运行记录失败，请稍后重试")
			return
		}
	}

	if h.memoryRepo != nil {
		if err := h.memoryRepo.DeleteByConversationID(convIDInt); err != nil {
			response.InternalError(c, "删除会话记忆失败，请稍后重试")
			return
		}
	}

	if err := h.convRepo.Delete(convIDInt); err != nil {
		response.InternalError(c, "删除会话失败，请稍后重试")
		return
	}

	response.Success(c, gin.H{"message": "会话删除成功"})
}

// Share 切换对话分享状态
// POST /api/chat/conversations/:id/share
func (h *ChatHandler) Share(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "登录状态已失效，请重新登录后操作")
		return
	}

	convIDInt, ok := parseConversationID(c)
	if !ok {
		return
	}

	conv, err := h.convRepo.GetByID(convIDInt)
	if err != nil {
		response.NotFound(c, "会话不存在或已被删除")
		return
	}

	if conv.UserID != userID.(int64) {
		response.Forbidden(c, "你没有权限修改这个会话的分享状态")
		return
	}

	var req struct {
		Share bool `json:"share"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "分享状态参数不正确：share 必须是布尔值")
		return
	}

	token := conv.ShareToken
	if req.Share && token == "" {
		token = generateShareToken()
	}
	if !req.Share {
		token = ""
	}

	if err := h.convRepo.UpdateShare(convIDInt, req.Share, token); err != nil {
		response.InternalError(c, "更新会话分享状态失败，请稍后重试")
		return
	}

	conv.ShareToken = token
	conv.IsShared = req.Share

	response.Success(c, buildConversationResponse(conv))
}

// GetShared 获取公开分享的会话
// GET /api/shared/conversations/:token
func (h *ChatHandler) GetShared(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		response.InvalidParams(c, "缺少分享令牌，请使用完整的分享链接")
		return
	}

	conv, err := h.convRepo.GetByShareToken(token)
	if err != nil {
		response.NotFound(c, "分享会话不存在、已取消分享或链接无效")
		return
	}

	msgs, err := h.msgRepo.GetByConversationID(conv.ID)
	if err != nil {
		response.InternalError(c, "分享会话消息加载失败，请稍后重试")
		return
	}

	var steps []model.ConversationRunStep
	if h.runStepRepo != nil {
		steps, _ = h.runStepRepo.GetByConversationID(conv.ID)
	}

	stepResponses := make([]StepResponse, len(steps))
	stepsByRunID := make(map[int64][]StepResponse)
	for i, step := range steps {
		stepResponse := buildStepResponse(step)
		stepResponses[i] = stepResponse
		stepsByRunID[step.RunID] = append(stepsByRunID[step.RunID], stepResponse)
	}

	msgResponses := make([]MessageResponse, len(msgs))
	for i, msg := range msgs {
		msgResponse := MessageResponse{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			RunID:          msg.RunID,
			Role:           msg.Role,
			Content:        msg.Content,
			CreatedAt:      msg.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if msg.RunID != nil {
			msgResponse.ProcessSteps = stepsByRunID[*msg.RunID]
		}
		msgResponses[i] = msgResponse
	}

	// 构建分享者信息（精简版）
	sharedBy := gin.H{
		"username":   conv.User.Username,
		"avatar_url": conv.User.AvatarURL,
	}

	response.Success(c, gin.H{
		"conversation": buildConversationResponse(conv),
		"messages":     msgResponses,
		"steps":        stepResponses,
		"shared_by":    sharedBy,
	})
}

// Export 导出对话为 Markdown
// GET /api/chat/conversations/:id/export
func (h *ChatHandler) Export(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "登录状态已失效，请重新登录后操作")
		return
	}

	convIDInt, ok := parseConversationID(c)
	if !ok {
		return
	}

	conv, err := h.convRepo.GetByID(convIDInt)
	if err != nil {
		response.NotFound(c, "会话不存在或已被删除")
		return
	}

	if conv.UserID != userID.(int64) {
		response.Forbidden(c, "你没有权限导出这个会话")
		return
	}

	msgs, err := h.msgRepo.GetByConversationID(convIDInt)
	if err != nil {
		response.InternalError(c, "会话消息加载失败，无法导出")
		return
	}

	var steps []model.ConversationRunStep
	if h.runStepRepo != nil {
		steps, _ = h.runStepRepo.GetByConversationID(convIDInt)
	}

	stepsByRunID := make(map[int64][]StepResponse)
	for _, step := range steps {
		stepResponse := buildStepResponse(step)
		stepsByRunID[step.RunID] = append(stepsByRunID[step.RunID], stepResponse)
	}

	var md strings.Builder
	md.WriteString(fmt.Sprintf("# %s\n\n", conv.Title))
	md.WriteString(fmt.Sprintf("> 导出时间：%s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	siteURL := ""
	if h.cfg != nil {
		siteURL = strings.TrimRight(h.cfg.Site.URL, "/")
	}

	for _, msg := range msgs {
		content := resolveRelativeLinks(msg.Content, siteURL)
		if msg.Role == "user" {
			md.WriteString(fmt.Sprintf("## 👤 用户\n\n%s\n\n", content))
		} else {
			md.WriteString(fmt.Sprintf("## 🤖 AI 助手\n\n%s\n\n", content))
			if msg.RunID != nil {
				if runSteps, ok := stepsByRunID[*msg.RunID]; ok && len(runSteps) > 0 {
					md.WriteString("<details>\n<summary>处理过程</summary>\n\n")
					for _, step := range runSteps {
						statusIcon := "✅"
						if step.Status == "running" {
							statusIcon = "⏳"
						} else if step.Status == "failed" {
							statusIcon = "❌"
						}
						md.WriteString(fmt.Sprintf("- %s **%s** (%s): %s\n", statusIcon, step.AgentName, step.Type, step.Summary))
						if step.Detail != "" {
							md.WriteString(fmt.Sprintf("  > %s\n", step.Detail))
						}
					}
					md.WriteString("\n</details>\n\n")
				}
			}
		}
	}

	filename := fmt.Sprintf("conversation-%d.md", convIDInt)
	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "private, max-age=300")
	c.String(200, md.String())
}

var reArticleLink = regexp.MustCompile(`\]\((/article/[^)]+)\)`)

func resolveRelativeLinks(content string, siteURL string) string {
	if siteURL == "" {
		return content
	}
	return reArticleLink.ReplaceAllString(content, "]("+siteURL+"$1)")
}

func generateShareToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
