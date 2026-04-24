package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

// StreamEventType SSE 流事件类型
type StreamEventType string

const (
	StreamEventStage    StreamEventType = "stage"
	StreamEventQuestion StreamEventType = "question"
	StreamEventChunk    StreamEventType = "chunk"
	StreamEventStep     StreamEventType = "step"
	StreamEventDone     StreamEventType = "done"

	maxStepDetailRunes = 6000

	ConversationMemoryScopeSummary     = "conversation_summary"
	ConversationMemoryScopePreference  = "user_preference"
	ConversationMemoryScopeProjectFact = "project_fact"
	ConversationMemoryScopeDecision    = "decision"
	ConversationMemoryScopeOpenThread  = "open_thread"
	recentMemoryMessageCount           = 6
	maxMemorySnippetRunes              = 80
)

// StreamEvent ThinkTank 流事件
type StreamEvent struct {
	Type    StreamEventType
	Stage   string
	Label   string
	Message string
	Sources []string

	StepID    int64
	AgentName string
	Status    string
	Summary   string
	Detail    string
}

// ThinkTankChatResponse ThinkTank 对话响应
type ThinkTankChatResponse struct {
	Message           string
	Sources           []string
	Stage             string
	RequiresUserInput bool
	Steps             []model.ConversationRunStep
}

// PlannerDecision 只保留为执行状态摘要，不再代表旧的自写 planner。
// 真正的规划、执行、重规划由 Eino planexecute 负责。
type PlannerDecision struct {
	PlanSummary           string
	ExecutionStrategy     string
	RequiresClarification bool
	ClarificationQuestion string
}

// ThinkTankService ThinkTank 编排服务接口
type ThinkTankService interface {
	Chat(ctx context.Context, question string, conversationID *int64, userID *int64) (*ThinkTankChatResponse, error)
	ChatStream(ctx context.Context, question string, conversationID *int64, userID *int64) (<-chan StreamEvent, <-chan error)
}

type thinkTankService struct {
	librarian   Librarian
	journalist  Journalist
	synthesizer ThinkTankSynthesizer

	logger AILogger

	conversations *thinkTankConversationManager
	memories      *thinkTankMemoryManager
	runs          *thinkTankRunRecorder
	streams       *thinkTankStreamEmitter
	researchDraft *thinkTankResearchDraftSink
	orchestrator  *thinkTankOrchestrator

	adkRunner        *thinkTankADKRunner
	adkAnswerFetcher func(ctx context.Context, question string) (string, error)
}

// NewThinkTankService 创建 ThinkTank 服务
func NewThinkTankService(
	librarian Librarian,
	journalist Journalist,
	synthesizer ThinkTankSynthesizer,
	runRepo repository.ConversationRunRepository,
	runStepRepo repository.ConversationRunStepRepository,
	memoryRepo repository.ConversationMemoryRepository,
	convRepo repository.ConversationRepository,
	msgRepo repository.ChatMessageRepository,
	knowledgeSvc KnowledgeDocumentService,
	logger AILogger,
	options ...any,
) ThinkTankService {
	var runner *thinkTankADKRunner
	var memorySummarizer ConversationMemorySummarizer
	for _, option := range options {
		switch v := option.(type) {
		case *thinkTankADKRunner:
			runner = v
		case ConversationMemorySummarizer:
			memorySummarizer = v
		}
	}

	svc := &thinkTankService{
		librarian:     librarian,
		journalist:    journalist,
		synthesizer:   synthesizer,
		logger:        logger,
		conversations: newThinkTankConversationManager(convRepo, msgRepo, logger),
		memories:      newThinkTankMemoryManager(memoryRepo, memorySummarizer, logger),
		runs:          newThinkTankRunRecorder(runRepo, runStepRepo, logger),
		streams:       newThinkTankStreamEmitter(),
		researchDraft: newThinkTankResearchDraftSink(knowledgeSvc),
		adkRunner:     runner,
	}
	svc.orchestrator = newThinkTankOrchestrator(svc)
	svc.adkAnswerFetcher = svc.collectADKRunnerAnswer
	return svc
}

func (s *thinkTankService) Chat(ctx context.Context, question string, conversationID *int64, userID *int64) (*ThinkTankChatResponse, error) {
	return s.orchestrator.chat(ctx, question, conversationID, userID)
}

func (s *thinkTankService) ChatStream(ctx context.Context, question string, conversationID *int64, userID *int64) (<-chan StreamEvent, <-chan error) {
	return s.orchestrator.chatStream(ctx, question, conversationID, userID)
}

func (s *thinkTankService) collectADKRunnerAnswer(ctx context.Context, question string) (string, error) {
	if s.adkRunner == nil || s.adkRunner.runner == nil {
		return "", nil
	}
	iter := s.adkRunner.runner.Run(ctx, []adk.Message{schema.UserMessage(question)})
	finalAnswer := ""
	localNotes := make([]string, 0)
	webNotes := make([]string, 0)
	articleSources := make([]SourceRef, 0)
	webSources := make([]SourceRef, 0)
	for {
		e, ok := iter.Next()
		if !ok {
			break
		}
		if e == nil {
			continue
		}
		if e.Err != nil {
			return "", e.Err
		}
		msg, _, err := adk.GetMessage(e)
		if err == nil && msg != nil && strings.TrimSpace(msg.Content) != "" {
			if strings.TrimSpace(msg.ToolName) == "LocalSearch" {
				articleSources = mergeSourceRefs(articleSources, extractLocalSearchArticleSources(msg.Content))
				localNotes = appendNonEmptyNote(localNotes, extractLocalSearchSummary(msg.Content))
			}
			if strings.TrimSpace(msg.ToolName) == "WebSearch" {
				webSources = mergeSourceRefs(webSources, extractWebSearchSources(msg.Content))
				webNotes = appendNonEmptyNote(webNotes, summarizeWebSearchResult(msg.Content))
			}
			if e.AgentName == "executor" && strings.TrimSpace(msg.ToolName) == "" && len(msg.ToolCalls) == 0 {
				webNotes = appendNonEmptyNote(webNotes, msg.Content)
			}
			if response, ok := extractPlanExecuteFinalResponse(msg.Content); ok {
				if !isNonFinalToolLimitationAnswer(response) {
					finalAnswer = response
				} else {
					webNotes = appendNonEmptyNote(webNotes, "replanner returned a tool limitation instead of a user-facing answer: "+response)
				}
			}
		}
	}
	if strings.TrimSpace(finalAnswer) == "" {
		return s.composeADKFallbackAnswer(ctx, question, localNotes, webNotes, articleSources, webSources)
	}
	return finalAnswer, nil
}

func (s *thinkTankService) composeADKFallbackAnswer(
	ctx context.Context,
	question string,
	localNotes []string,
	webNotes []string,
	articleSources []SourceRef,
	webSources []SourceRef,
) (string, error) {
	localSummary := compactNotes(localNotes)
	webSummary := compactNotes(webNotes)
	if localSummary == "" && webSummary == "" && len(articleSources) == 0 && len(webSources) == 0 {
		return "", fmt.Errorf("no ADK evidence available for fallback answer")
	}

	local := LibrarianResult{
		CoverageStatus: "partial",
		Summary:        localSummary,
		Sources:        articleSources,
	}
	web := &JournalistResult{
		Summary: webSummary,
		Sources: webSources,
	}
	if s.synthesizer != nil {
		answer, _, err := s.synthesizer.Compose(ctx, question, local, web)
		if err == nil && strings.TrimSpace(answer) != "" {
			return answer, nil
		}
	}

	var builder strings.Builder
	builder.WriteString("基于当前已完成的检索结果，先给出可用回答：")
	if localSummary != "" {
		builder.WriteString("\n\n站内知识：\n")
		builder.WriteString(localSummary)
	}
	if webSummary != "" {
		builder.WriteString("\n\n外部资料：\n")
		builder.WriteString(webSummary)
	}
	return appendGroupedReferences(builder.String(), articleSources, webSources), nil
}
