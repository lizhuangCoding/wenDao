package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
)

// AIHandler AI Handler
type AIHandler struct {
	aiService service.AIService
}

// NewAIHandler 创建 AI Handler 实例
func NewAIHandler(aiService service.AIService) *AIHandler {
	return &AIHandler{aiService: aiService}
}

// ChatRequest AI 对话请求
type ChatRequest struct {
	Message        string `json:"message" binding:"required"`
	ArticleID      *int64 `json:"article_id"`
	ConversationID *int64 `json:"conversation_id"`
}

// ChatResponse AI 对话响应
type ChatResponse struct {
	Message string   `json:"message"`
	Sources []string `json:"sources,omitempty"`
}

type chatStreamEvent struct {
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

// Chat 处理 AI 对话请求
func (h *AIHandler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "消息内容不能为空")
		return
	}

	answer, err := h.aiService.Chat(req.Message, req.ConversationID, getCurrentUserID(c))
	if err != nil {
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

// GenerateSummary 生成文章摘要
func (h *AIHandler) GenerateSummary(c *gin.Context) {
	var req SummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "内容不能为空")
		return
	}

	summary, err := h.aiService.GenerateSummary(req.Content)
	if err != nil {
		response.InternalError(c, "生成摘要失败，请稍后再试")
		return
	}

	response.Success(c, SummaryResponse{Summary: summary})
}

// ChatStream 处理 AI 流式对话请求
func (h *AIHandler) ChatStream(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "消息内容不能为空")
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

	for eventCh != nil || errCh != nil {
		select {
		case <-c.Request.Context().Done():
			if errors.Is(c.Request.Context().Err(), context.Canceled) || errors.Is(c.Request.Context().Err(), context.DeadlineExceeded) {
				return
			}
			return
		case event, ok := <-eventCh:
			if !ok {
				eventCh = nil
				continue
			}
			switch event.Type {
			case service.StreamEventStage:
				// 阶段事件只更新 UI 顶部状态，不写入最终回答正文。
				if err := writeSSEvent(c, "stage", chatStreamEvent{Stage: event.Stage, Label: event.Label}); err != nil {
					return
				}
			case service.StreamEventQuestion:
				// Planner 认为需要补充条件时，前端会把这条问题显示成 assistant 消息。
				if err := writeSSEvent(c, "question", chatStreamEvent{Stage: event.Stage, Message: event.Message, RequiresUserInput: true}); err != nil {
					return
				}
			case service.StreamEventChunk:
				// chunk 是当前累计答案快照，前端用它覆盖 assistant 占位消息。
				if err := writeSSEvent(c, "chunk", chatStreamEvent{Message: event.Message, Sources: event.Sources}); err != nil {
					return
				}
			case service.StreamEventStep:
				// step 是可展开的多 Agent 过程日志，通常对应一次 Agent 切换或工具调用结果。
				if err := writeSSEvent(c, "step", chatStreamEvent{
					StepID:    event.StepID,
					AgentName: event.AgentName,
					Status:    event.Status,
					Summary:   event.Summary,
					Detail:    event.Detail,
				}); err != nil {
					return
				}
			case service.StreamEventDone:
				if err := writeSSEvent(c, "done", chatStreamEvent{Stage: event.Stage}); err != nil {
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
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			_ = writeSSEvent(c, "error", chatStreamEvent{Error: err.Error()})
			return
		}
	}

	c.Status(http.StatusOK)
}
