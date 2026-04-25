package chat

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/internal/model"
	"wenDao/internal/pkg/response"
	"wenDao/internal/repository"
)

// ChatHandler 对话处理器
type ChatHandler struct {
	convRepo    repository.ConversationRepository
	msgRepo     repository.ChatMessageRepository
	runRepo     repository.ConversationRunRepository
	runStepRepo repository.ConversationRunStepRepository
	memoryRepo  repository.ConversationMemoryRepository
}

// NewChatHandler 创建对话处理器
func NewChatHandler(
	convRepo repository.ConversationRepository,
	msgRepo repository.ChatMessageRepository,
	runRepo repository.ConversationRunRepository,
	runStepRepo repository.ConversationRunStepRepository,
	memoryRepo repository.ConversationMemoryRepository,
) *ChatHandler {
	return &ChatHandler{
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
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// MessageResponse 消息响应
type MessageResponse struct {
	ID             int64  `json:"id"`
	ConversationID int64  `json:"conversation_id"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	CreatedAt      string `json:"created_at"`
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

func parseConversationID(c *gin.Context) (int64, bool) {
	convID := c.Param("id")
	var convIDInt int64
	if _, err := fmt.Sscanf(convID, "%d", &convIDInt); err != nil {
		response.InvalidParams(c, "Invalid conversation ID")
		return 0, false
	}
	return convIDInt, true
}

func buildConversationResponse(conv *model.Conversation) ConversationResponse {
	return ConversationResponse{
		ID:        conv.ID,
		UserID:    conv.UserID,
		Title:     conv.Title,
		CreatedAt: conv.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: conv.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// List 获取用户对话列表
// GET /api/chat/conversations
func (h *ChatHandler) List(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Missing user ID")
		return
	}

	convs, err := h.convRepo.GetByUserID(userID.(int64))
	if err != nil {
		response.InternalError(c, "Failed to get conversations")
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
		response.Unauthorized(c, "Missing user ID")
		return
	}

	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "Invalid request: title is required")
		return
	}

	conv := &model.Conversation{
		UserID: userID.(int64),
		Title:  req.Title,
	}

	if err := h.convRepo.Create(conv); err != nil {
		response.InternalError(c, "Failed to create conversation")
		return
	}

	response.Success(c, buildConversationResponse(conv))
}

// Get 获取对话详情（含消息）
// GET /api/chat/conversations/:id
func (h *ChatHandler) Get(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Missing user ID")
		return
	}

	convIDInt, ok := parseConversationID(c)
	if !ok {
		return
	}

	conv, err := h.convRepo.GetByID(convIDInt)
	if err != nil {
		response.NotFound(c, "Conversation not found")
		return
	}

	if conv.UserID != userID.(int64) {
		response.Forbidden(c, "Access denied")
		return
	}

	msgs, err := h.msgRepo.GetByConversationID(convIDInt)
	if err != nil {
		response.InternalError(c, "Failed to get messages")
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

	msgResponses := make([]MessageResponse, len(msgs))
	for i, msg := range msgs {
		msgResponses[i] = MessageResponse{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			Role:           msg.Role,
			Content:        msg.Content,
			CreatedAt:      msg.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	stepResponses := make([]StepResponse, len(steps))
	for i, step := range steps {
		stepResponses[i] = StepResponse{
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

	var activeRunResponse *ActiveRunResponse
	activeStepResponses := make([]StepResponse, 0)
	if activeRun != nil && (activeRun.Status == "running" || activeRun.Status == "waiting_user") {
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
				activeStepResponses[i] = StepResponse{
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

// Update 更新对话标题
// PATCH /api/chat/conversations/:id
func (h *ChatHandler) Update(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Missing user ID")
		return
	}

	convIDInt, ok := parseConversationID(c)
	if !ok {
		return
	}

	var req UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "Invalid request: title is required")
		return
	}

	conv, err := h.convRepo.GetByID(convIDInt)
	if err != nil {
		response.NotFound(c, "Conversation not found")
		return
	}

	if conv.UserID != userID.(int64) {
		response.Forbidden(c, "Access denied")
		return
	}

	conv.Title = req.Title
	conv.UpdatedAt = time.Now()
	if err := h.convRepo.Update(conv); err != nil {
		response.InternalError(c, "Failed to update conversation")
		return
	}

	response.Success(c, buildConversationResponse(conv))
}

// Delete 删除对话
// DELETE /api/chat/conversations/:id
func (h *ChatHandler) Delete(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Missing user ID")
		return
	}

	convIDInt, ok := parseConversationID(c)
	if !ok {
		return
	}

	conv, err := h.convRepo.GetByID(convIDInt)
	if err != nil {
		response.NotFound(c, "Conversation not found")
		return
	}

	if conv.UserID != userID.(int64) {
		response.Forbidden(c, "Access denied")
		return
	}

	if err := h.msgRepo.DeleteByConversationID(convIDInt); err != nil {
		response.InternalError(c, "Failed to delete messages")
		return
	}

	if h.runStepRepo != nil {
		if err := h.runStepRepo.DeleteByConversationID(convIDInt); err != nil {
			response.InternalError(c, "Failed to delete conversation run steps")
			return
		}
	}

	if h.runRepo != nil {
		if err := h.runRepo.DeleteByConversationID(convIDInt); err != nil {
			response.InternalError(c, "Failed to delete conversation runs")
			return
		}
	}

	if h.memoryRepo != nil {
		if err := h.memoryRepo.DeleteByConversationID(convIDInt); err != nil {
			response.InternalError(c, "Failed to delete conversation memories")
			return
		}
	}

	if err := h.convRepo.Delete(convIDInt); err != nil {
		response.InternalError(c, "Failed to delete conversation")
		return
	}

	response.Success(c, gin.H{"message": "Conversation deleted successfully"})
}
