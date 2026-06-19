package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/config"
	"wenDao/internal/pkg/aiobserve"
	"wenDao/internal/pkg/response"
	"wenDao/internal/repository"
	"wenDao/internal/service"
)

// AIHandler AI Handler
type AIHandler struct {
	aiService service.AIService
	cfg       *config.Config
	runRepo   repository.ConversationRunRepository
}

// NewAIHandler 创建 AI Handler 实例
func NewAIHandler(aiService service.AIService, cfg *config.Config, runRepos ...repository.ConversationRunRepository) *AIHandler {
	var runRepo repository.ConversationRunRepository
	if len(runRepos) > 0 {
		runRepo = runRepos[0]
	}
	return &AIHandler{aiService: aiService, cfg: cfg, runRepo: runRepo}
}

// ChatRequest AI 对话请求
type ChatRequest struct {
	Message        string `json:"message" binding:"required"`
	ArticleID      *int64 `json:"article_id"`
	ConversationID *int64 `json:"conversation_id"`
	ModelProvider  string `json:"model_provider"`
	ModelName      string `json:"model_name"`
}

// ChatResponse AI 对话响应
type ChatResponse struct {
	Message string   `json:"message"`
	Sources []string `json:"sources,omitempty"`
}

type chatStreamEvent struct {
	RunID             int64    `json:"run_id,omitempty"`
	Stage             string   `json:"stage,omitempty"`
	Label             string   `json:"label,omitempty"`
	Message           string   `json:"message,omitempty"`
	Error             string   `json:"error,omitempty"`
	Sources           []string `json:"sources,omitempty"`
	RequiresUserInput bool     `json:"requires_user_input,omitempty"`

	// For step updates
	StepID    int64  `json:"step_id,omitempty"`
	AgentName string `json:"agent_name,omitempty"`
	Status    string `json:"status,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type ResumeChatStreamRequest struct {
	ConversationID int64 `json:"conversation_id" binding:"required"`
	RunID          int64 `json:"run_id" binding:"required"`
}

func getCurrentUserID(c *gin.Context) *int64 {
	if uid, exists := c.Get("user_id"); exists {
		if v, ok := uid.(int64); ok {
			return &v
		}
	}
	return nil
}

func writeSSEvent(c *gin.Context, event string, payload interface{}) error {
	writer := c.Writer
	writer.WriteHeaderNow()

	// 前端按 event 名分发到 stage/question/step/chunk/done 等处理器。
	if _, err := fmt.Fprintf(writer, "event: %s\n", event); err != nil {
		return err
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "data: %s\n\n", data); err != nil {
		return err
	}

	c.Writer.Flush()
	return nil
}

// ModelInfo 模型信息
type ModelInfo struct {
	Provider    string `json:"provider"`
	ModelName   string `json:"model_name"`
	DisplayName string `json:"display_name"`
}

// GetModels 获取可用的 AI 模型列表
func (h *AIHandler) GetModels(c *gin.Context) {
	models := []ModelInfo{
		{
			Provider:    h.cfg.AI.Provider,
			ModelName:   h.cfg.AI.LLMModel,
			DisplayName: fmt.Sprintf("%s (%s)", h.cfg.AI.Provider, h.cfg.AI.LLMModel),
		},
	}

	for _, m := range h.cfg.AI.Models {
		displayName := m.DisplayName
		if displayName == "" {
			displayName = fmt.Sprintf("%s/%s", m.Provider, m.ModelName)
		}
		models = append(models, ModelInfo{
			Provider:    m.Provider,
			ModelName:   m.ModelName,
			DisplayName: displayName,
		})
	}

	response.Success(c, gin.H{"models": models})
}

// Chat 处理 AI 对话请求
func (h *AIHandler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "消息内容不能为空")
		return
	}
	if !h.allowAIUsage(c, req.Message) {
		return
	}

	answer, err := h.aiService.Chat(c.Request.Context(), req.Message, req.ConversationID, getCurrentUserID(c))
	if err != nil {
		if errors.Is(err, service.ErrAIDisabled) {
			response.ServiceUnavailable(c, "AI 服务暂时不可用，请稍后再试")
			return
		}
		response.InternalError(c, "生成回答失败，请稍后再试")
		return
	}

	response.Success(c, ChatResponse{Message: answer})
}

// SummaryRequest 摘要生成请求
type SummaryRequest struct {
	Content string `json:"content" binding:"required"`
}

// SummaryResponse 摘要生成响应
type SummaryResponse struct {
	Summary string `json:"summary"`
}

// WritingRequest AI 写作辅助请求
type WritingRequest struct {
	Action  service.WritingAction `json:"action" binding:"required"`
	Content string                `json:"content" binding:"required"`
	Title   string                `json:"title"`
	Summary string                `json:"summary"`
}

// WritingResponse AI 写作辅助响应
type WritingResponse struct {
	Result      string   `json:"result"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// GenerateSummary 生成文章摘要
func (h *AIHandler) GenerateSummary(c *gin.Context) {
	var req SummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "内容不能为空")
		return
	}

	summary, err := h.aiService.GenerateSummary(c.Request.Context(), req.Content)
	if err != nil {
		if errors.Is(err, service.ErrAIDisabled) {
			response.ServiceUnavailable(c, "AI 服务暂时不可用，请稍后再试")
			return
		}
		response.InternalError(c, "生成摘要失败，请稍后再试")
		return
	}

	response.Success(c, SummaryResponse{Summary: summary})
}

// GenerateWriting 生成 Markdown 写作辅助内容
func (h *AIHandler) GenerateWriting(c *gin.Context) {
	var req WritingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "写作内容不能为空")
		return
	}

	result, err := h.aiService.GenerateWriting(c.Request.Context(), service.WritingRequest{
		Action:  req.Action,
		Content: req.Content,
		Title:   req.Title,
		Summary: req.Summary,
	})
	if err != nil {
		if errors.Is(err, service.ErrAIDisabled) {
			response.ServiceUnavailable(c, "AI 服务暂时不可用，请稍后再试")
			return
		}
		if errors.Is(err, service.ErrUnsupportedWritingAction) || errors.Is(err, service.ErrWritingContentEmpty) {
			response.InvalidParams(c, "写作类型或内容无效")
			return
		}
		response.InternalError(c, "生成写作建议失败，请稍后再试")
		return
	}

	response.Success(c, WritingResponse{Result: result.Result, Suggestions: result.Suggestions})
}

// ChatStream 处理 AI 流式对话请求
func (h *AIHandler) ChatStream(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "消息内容不能为空")
		return
	}
	if !h.allowAIUsage(c, req.Message) {
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// AIService 返回两个 channel：一个承载正常流程事件，一个承载执行错误。
	eventCh, errCh := h.aiService.ChatStream(c.Request.Context(), req.Message, req.ConversationID, getCurrentUserID(c))
	if err := writeSSEvent(c, "start", chatStreamEvent{}); err != nil {
		return
	}

	h.streamEvents(c, eventCh, errCh)
	c.Status(http.StatusOK)
}

func (h *AIHandler) ResumeChatStream(c *gin.Context) {
	var req ResumeChatStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "会话或运行参数不能为空")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	eventCh, errCh := h.aiService.ResumeChatStream(c.Request.Context(), req.ConversationID, req.RunID, getCurrentUserID(c))
	h.streamEvents(c, eventCh, errCh)
	c.Status(http.StatusOK)
}

func (h *AIHandler) allowAIUsage(c *gin.Context, message string) bool {
	if h == nil || h.cfg == nil || h.runRepo == nil {
		return true
	}
	userID := getCurrentUserID(c)
	if userID == nil || *userID <= 0 {
		return true
	}
	usage, err := h.runRepo.GetDailyUsageByUser(*userID, time.Now())
	if err != nil {
		response.InternalErrorWithErr(c, "AI usage quota check failed", err)
		return false
	}
	if h.cfg.AI.DailyRunLimit > 0 && usage.RunCount >= int64(h.cfg.AI.DailyRunLimit) {
		response.TooManyRequests(c, "今日 AI 对话次数已达上限，请明天再试")
		return false
	}
	requestTokens := aiobserve.EstimateTokens(message, "").PromptTokens
	usedTokens := usage.PromptTokens + usage.CompletionTokens
	if h.cfg.AI.DailyTokenLimit > 0 && usedTokens+requestTokens > h.cfg.AI.DailyTokenLimit {
		response.TooManyRequests(c, "今日 AI token 额度已达上限，请明天再试")
		return false
	}
	return true
}

func (h *AIHandler) streamEvents(c *gin.Context, eventCh <-chan service.StreamEvent, errCh <-chan error) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	lastRunID := int64(0)
	lastStage := ""
	lastStatus := ""

	for eventCh != nil || errCh != nil {
		select {
		case <-c.Request.Context().Done():
			if errors.Is(c.Request.Context().Err(), context.Canceled) || errors.Is(c.Request.Context().Err(), context.DeadlineExceeded) {
				return
			}
			return
		case <-ticker.C:
			if lastRunID > 0 {
				if err := writeSSEvent(c, "heartbeat", chatStreamEvent{RunID: lastRunID, Stage: lastStage, Status: lastStatus}); err != nil {
					return
				}
			}
		case event, ok := <-eventCh:
			if !ok {
				eventCh = nil
				continue
			}
			if event.RunID > 0 {
				lastRunID = event.RunID
			}
			if event.Stage != "" {
				lastStage = event.Stage
			}
			if event.Status != "" {
				lastStatus = event.Status
			}
			switch event.Type {
			case service.StreamEventStage:
				// 阶段事件只更新 UI 顶部状态，不写入最终回答正文。
				if err := writeSSEvent(c, "stage", chatStreamEvent{RunID: event.RunID, Stage: event.Stage, Label: event.Label}); err != nil {
					return
				}
			case service.StreamEventQuestion:
				// Planner 认为需要补充条件时，前端会把这条问题显示成 assistant 消息。
				if err := writeSSEvent(c, "question", chatStreamEvent{RunID: event.RunID, Stage: event.Stage, Status: event.Status, Message: event.Message, RequiresUserInput: true}); err != nil {
					return
				}
			case service.StreamEventChunk:
				// chunk 是当前累计答案快照，前端用它覆盖 assistant 占位消息。
				if err := writeSSEvent(c, "chunk", chatStreamEvent{RunID: event.RunID, Stage: event.Stage, Status: event.Status, Message: event.Message, Sources: event.Sources}); err != nil {
					return
				}
			case service.StreamEventStep:
				// step 是可展开的多 Agent 过程日志，通常对应一次 Agent 切换或工具调用结果。
				if err := writeSSEvent(c, "step", chatStreamEvent{
					RunID:     event.RunID,
					StepID:    event.StepID,
					AgentName: event.AgentName,
					Status:    event.Status,
					Summary:   event.Summary,
					Detail:    event.Detail,
				}); err != nil {
					return
				}
			case service.StreamEventResume:
				if err := writeSSEvent(c, "resume", chatStreamEvent{RunID: event.RunID, Stage: event.Stage, Status: event.Status}); err != nil {
					return
				}
			case service.StreamEventSnapshot:
				if err := writeSSEvent(c, "snapshot", chatStreamEvent{RunID: event.RunID, Stage: event.Stage, Status: event.Status, Message: event.Message}); err != nil {
					return
				}
			case service.StreamEventHeartbeat:
				if err := writeSSEvent(c, "heartbeat", chatStreamEvent{RunID: event.RunID, Stage: event.Stage, Status: event.Status}); err != nil {
					return
				}
			case service.StreamEventDone:
				if err := writeSSEvent(c, "done", chatStreamEvent{RunID: event.RunID, Stage: event.Stage, Status: event.Status}); err != nil {
					return
				}
			}
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			if err == nil {
				continue
			}
			if errors.Is(err, service.ErrAIDisabled) {
				_ = writeSSEvent(c, "error", chatStreamEvent{Error: "AI 服务暂时不可用，请稍后再试"})
				return
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			_ = writeSSEvent(c, "error", chatStreamEvent{Error: err.Error()})
			return
		}
	}
}
