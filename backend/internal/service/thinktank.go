package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
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

	// For Step updates
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
	librarian        Librarian
	journalist       Journalist
	synthesizer      ThinkTankSynthesizer
	runRepo          repository.ConversationRunRepository
	runStepRepo      repository.ConversationRunStepRepository
	memoryRepo       repository.ConversationMemoryRepository
	convRepo         repository.ConversationRepository
	msgRepo          repository.ChatMessageRepository
	knowledgeSvc     KnowledgeDocumentService
	memorySummarizer ConversationMemorySummarizer
	logger           AILogger
	adkRunner        *thinkTankADKRunner
	adkAnswerFetcher func(ctx context.Context, question string) (string, error)
}

type adkPendingContext struct {
	Type       string `json:"type"`
	Checkpoint string `json:"checkpoint_id"`
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
		librarian:        librarian,
		journalist:       journalist,
		synthesizer:      synthesizer,
		runRepo:          runRepo,
		runStepRepo:      runStepRepo,
		memoryRepo:       memoryRepo,
		convRepo:         convRepo,
		msgRepo:          msgRepo,
		knowledgeSvc:     knowledgeSvc,
		memorySummarizer: memorySummarizer,
		logger:           logger,
		adkRunner:        runner,
	}
	svc.adkAnswerFetcher = svc.collectADKRunnerAnswer
	return svc
}

func (s *thinkTankService) Chat(ctx context.Context, question string, conversationID *int64, userID *int64) (*ThinkTankChatResponse, error) {
	conv, err := s.getOwnedConversation(conversationID, userID)
	if err != nil {
		return nil, err
	}

	var history []model.ChatMessage
	if conv != nil {
		// 非流式接口复用会话上下文：历史消息用于长期/近期记忆压缩。
		history, _ = s.msgRepo.GetByConversationID(conv.ID)
		s.saveConversationMessageWithWarning(conv.ID, "user", question, "Failed to save user message")
	}

	decision := PlannerDecision{ExecutionStrategy: "eino_plan_execute_replan", PlanSummary: "由 Eino PlanExecute planner 生成计划"}

	memory := compressConversationMemory(history)
	if conv != nil {
		memory = buildConversationMemoryForQuestion(question, history, s.loadConversationMemories(conv.ID))
	}
	// Agent 只接收一个用户输入，因此这里把用户新问题和压缩后的长期/近期记忆合并成任务文本。
	queryForAgents := buildAgentQuery(question, memory)

	if s.adkRunner != nil && s.adkAnswerFetcher != nil {
		adkCtx := WithUserID(ctx, derefUserID(userID))
		adkCtx = WithAILogger(adkCtx, s.logger)
		if conv != nil {
			adkCtx = WithConversationID(adkCtx, conv.ID)
		}
		answer, err := s.adkAnswerFetcher(adkCtx, queryForAgents)
		if err == nil && strings.TrimSpace(answer) != "" {
			if conv != nil {
				s.saveConversationMessageWithWarning(conv.ID, "assistant", answer, "Failed to save assistant message")
				s.updateConversationMetadataWithWarning(conv, question)
				s.persistCompletedRun(conv.ID, derefUserID(userID), question, answer, decision)
				s.updateConversationMemoryWithWarning(conv.ID, derefUserID(userID), appendConversationTurn(history, question, answer))
			}
			return &ThinkTankChatResponse{Message: answer, Stage: "completed"}, nil
		}
	}

	// Fallback to manual path
	localResult, err := s.librarian.Search(ctx, queryForAgents)
	if err != nil {
		return nil, err
	}

	var webResult *JournalistResult
	if localResult.CoverageStatus != "sufficient" && s.journalist != nil {
		webResult, err = s.journalist.Research(ctx, queryForAgents, localResult)
		if err != nil {
			return nil, err
		}
		if s.knowledgeSvc != nil && webResult != nil && webResult.KnowledgeDraftBody != "" {
			sources := make([]KnowledgeSourceInput, 0, len(webResult.Sources))
			for _, source := range webResult.Sources {
				sources = append(sources, KnowledgeSourceInput{URL: source.URL, Title: source.Title})
			}
			_, _ = s.knowledgeSvc.CreateResearchDraft(CreateKnowledgeDocumentInput{
				Title:           webResult.KnowledgeDraftTitle,
				Summary:         webResult.KnowledgeDraftSummary,
				Content:         webResult.KnowledgeDraftBody,
				CreatedByUserID: derefUserID(userID),
				Sources:         sources,
			})
		}
	}

	answer, sources, err := s.synthesizer.Compose(ctx, queryForAgents, localResult, webResult)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		s.saveConversationMessageWithWarning(conv.ID, "assistant", answer, "Failed to save assistant message")
		s.updateConversationMetadataWithWarning(conv, question)
		s.persistCompletedRun(conv.ID, derefUserID(userID), question, answer, decision)
		s.updateConversationMemoryWithWarning(conv.ID, derefUserID(userID), appendConversationTurn(history, question, answer))
	}
	return &ThinkTankChatResponse{Message: answer, Sources: sources, Stage: "completed"}, nil
}

func (s *thinkTankService) ChatStream(ctx context.Context, question string, conversationID *int64, userID *int64) (<-chan StreamEvent, <-chan error) {
	eventCh := make(chan StreamEvent, 48)
	errCh := make(chan error, 1)
	go func() {
		defer close(eventCh)
		defer close(errCh)

		conv, err := s.getOwnedConversation(conversationID, userID)
		if err != nil {
			errCh <- err
			return
		}

		var history []model.ChatMessage
		var pending *model.ConversationRun
		if conv != nil {
			// 先保存用户消息，保证即使后续进入澄清或执行失败，用户输入也能在会话历史中看到。
			history, _ = s.msgRepo.GetByConversationID(conv.ID)
			pending, _ = s.runRepo.GetActiveByConversationID(conv.ID)
			s.saveConversationMessageWithWarning(conv.ID, "user", question, "Failed to save user message")
		}

		eventCh <- StreamEvent{Type: StreamEventStage, Stage: "analyzing", Label: "正在理解你的问题"}
		decision := PlannerDecision{ExecutionStrategy: "eino_plan_execute_replan", PlanSummary: "由 Eino PlanExecute planner 生成计划"}

		eventCh <- StreamEvent{Type: StreamEventStage, Stage: "analyzing", Label: "正在进行多 Agent 深度调研"}
		s.logStage(conv, userID, "adk_start", "开始多 Agent 协作流", question)
		memory := compressConversationMemory(history)
		if conv != nil {
			memory = buildConversationMemoryForQuestion(question, history, s.loadConversationMemories(conv.ID))
		}
		// queryForAgents 是真正进入 ADK 的任务文本，包含当前问题和压缩后的上下文记忆。
		queryForAgents := buildAgentQuery(question, memory)

		var runID int64
		checkpointID := buildADKCheckpointID(conv, question)
		resumeFromADKInterrupt := false
		if ctxInfo, ok := parseADKPendingContext(pending); ok && strings.TrimSpace(ctxInfo.Checkpoint) != "" {
			resumeFromADKInterrupt = true
			checkpointID = ctxInfo.Checkpoint
		}
		if conv != nil {
			// Pre-create or update run to get a valid RunID for steps
			run := &model.ConversationRun{
				ConversationID:   conv.ID,
				UserID:           derefUserID(userID),
				Status:           "running",
				CurrentStage:     "analyzing",
				OriginalQuestion: question,
				LastPlan:         decision.PlanSummary,
				PendingContext:   marshalADKPendingContext(checkpointID),
			}
			if pending != nil {
				run.ID = pending.ID
				_ = s.runRepo.Update(run)
				runID = run.ID
			} else {
				_ = s.runRepo.Create(run)
				runID = run.ID
			}
		}

		if s.adkRunner != nil && s.adkRunner.runner != nil {
			// 将用户、会话、run 和 AI 日志器放进 context，供工具调用和日志写入复用。
			adkCtx := WithUserID(ctx, derefUserID(userID))
			adkCtx = WithAILogger(adkCtx, s.logger)
			adkCtx = WithRunID(adkCtx, runID)
			if conv != nil {
				adkCtx = WithConversationID(adkCtx, conv.ID)
			}
			var adkIter *adk.AsyncIterator[*adk.AgentEvent]
			if resumeFromADKInterrupt {
				resumeIter, resumeErr := s.adkRunner.runner.Resume(adkCtx, checkpointID, adk.WithToolOptions([]tool.Option{WithNewInput(question)}))
				if resumeErr != nil {
					errCh <- resumeErr
					return
				}
				adkIter = resumeIter
			} else {
				adkIter = s.adkRunner.runner.Run(adkCtx, []adk.Message{schema.UserMessage(queryForAgents)}, adk.WithCheckPointID(checkpointID))
			}

			var currentStep *model.ConversationRunStep
			adkArticleSources := make([]SourceRef, 0)
			adkWebSources := make([]SourceRef, 0)
			adkLocalNotes := make([]string, 0)
			adkWebNotes := make([]string, 0)
			fullAnswer := ""
			for {
				e, ok := adkIter.Next()
				if !ok {
					break
				}
				if e == nil {
					continue
				}
				if e.Err != nil {
					if currentStep != nil {
						currentStep.Status = "failed"
						currentStep.Detail += "\nError: " + e.Err.Error()
						_ = s.runStepRepo.Update(currentStep)
					}
					s.logStage(conv, userID, "failed", "ADK 运行错误", e.Err.Error())
					errCh <- e.Err
					return
				}

				// ADK 内部事件粒度较细，这里统一映射成前端可展示的 conversation_run_steps。
				if e.Action != nil && e.Action.Interrupted != nil {
					clarification := extractADKClarificationQuestion(e.Action.Interrupted)
					if strings.TrimSpace(clarification) == "" {
						clarification = "还需要你补充一点信息，我才能继续。"
					}
					if currentStep != nil {
						currentStep.Status = "waiting_user"
						appendStepDetail(currentStep, "流程中断：ask_for_clarification\n问题："+clarification)
						if conv != nil && s.runStepRepo != nil {
							_ = s.runStepRepo.Update(currentStep)
						}
					}
					if conv != nil {
						s.persistADKClarification(conv.ID, derefUserID(userID), runID, question, clarification, checkpointID, decision)
						s.saveConversationMessageWithWarning(conv.ID, "assistant", clarification, "Failed to save clarification message")
					}
					eventCh <- StreamEvent{Type: StreamEventStage, Stage: "clarifying", Label: "需要补充一点信息"}
					eventCh <- StreamEvent{Type: StreamEventQuestion, Stage: "clarifying", Message: clarification}
					return
				}

				if e.AgentName != "" {
					// Agent 发生切换时，先收尾上一个 step，再创建新的 step。
					if currentStep == nil || currentStep.AgentName != e.AgentName {
						if currentStep != nil {
							currentStep.Status = "completed"
							_ = s.runStepRepo.Update(currentStep)
							eventCh <- StreamEvent{
								Type:      StreamEventStep,
								StepID:    currentStep.ID,
								AgentName: currentStep.AgentName,
								Status:    "completed",
								Summary:   currentStep.Summary,
								Detail:    currentStep.Detail,
							}
						}

						summary := ""
						label := ""
						switch e.AgentName {
						case "planner":
							summary = "正在生成完整任务计划"
							label = "Eino Planner 正在规划"
						case "executor":
							summary = "正在执行当前计划步骤"
							label = "Eino Executor 正在执行"
						case "replanner":
							summary = "正在评估结果并重规划"
							label = "Eino Replanner 正在评估"
						default:
							summary = "Agent " + e.AgentName + " 正在协作"
							label = "切换为 " + e.AgentName + " Agent"
						}

						if label != "" {
							eventCh <- StreamEvent{Type: StreamEventStage, Stage: "adk_event", Label: label}
						}

						var conversationID int64
						if conv != nil {
							conversationID = conv.ID
						}
						currentStep = &model.ConversationRunStep{
							ConversationID: conversationID,
							RunID:          runID,
							AgentName:      e.AgentName,
							Type:           "thinking",
							Summary:        summary,
							Status:         "running",
						}
						if conv != nil && s.runStepRepo != nil {
							_ = s.runStepRepo.Create(currentStep)
						}
						eventCh <- StreamEvent{
							Type:      StreamEventStep,
							StepID:    currentStep.ID,
							AgentName: currentStep.AgentName,
							Status:    "running",
							Summary:   currentStep.Summary,
						}
					}
				}

				if actionDetail := formatADKActionDetail(e.Action); actionDetail != "" && currentStep != nil {
					appendStepDetail(currentStep, actionDetail)
					if conv != nil && s.runStepRepo != nil {
						_ = s.runStepRepo.Update(currentStep)
					}
					eventCh <- StreamEvent{
						Type:      StreamEventStep,
						StepID:    currentStep.ID,
						AgentName: currentStep.AgentName,
						Status:    currentStep.Status,
						Summary:   currentStep.Summary,
						Detail:    currentStep.Detail,
					}
				}

				msg, _, err := adk.GetMessage(e)
				if err == nil && msg != nil {
					detail := formatADKMessageDetail(msg)
					if strings.TrimSpace(msg.ToolName) == "LocalSearch" {
						// LocalSearch 的 JSON 结果里包含站内文章来源，用于最终回答末尾的“参考博主文章”。
						adkArticleSources = mergeSourceRefs(adkArticleSources, extractLocalSearchArticleSources(msg.Content))
						adkLocalNotes = appendNonEmptyNote(adkLocalNotes, extractLocalSearchSummary(msg.Content))
					}
					if strings.TrimSpace(msg.ToolName) == "WebSearch" {
						// WebSearch 的外部来源会被归并到“参考外部文章”。
						adkWebSources = mergeSourceRefs(adkWebSources, extractWebSearchSources(msg.Content))
						adkWebNotes = appendNonEmptyNote(adkWebNotes, summarizeWebSearchResult(msg.Content))
					}
					if e.AgentName == "executor" && strings.TrimSpace(msg.ToolName) == "" && len(msg.ToolCalls) == 0 {
						adkWebNotes = appendNonEmptyNote(adkWebNotes, msg.Content)
					}
					if currentStep != nil {
						appendStepDetail(currentStep, detail)
						if detail != "" {
							if conv != nil && s.runStepRepo != nil {
								_ = s.runStepRepo.Update(currentStep)
							}
							eventCh <- StreamEvent{
								Type:      StreamEventStep,
								StepID:    currentStep.ID,
								AgentName: currentStep.AgentName,
								Status:    "running",
								Summary:   currentStep.Summary,
								Detail:    currentStep.Detail,
							}
						}
					}

					// PlanExecute 的 replanner 会输出两类工具参数：
					// PlanTool -> {"steps":[...]} 表示继续重规划；RespondTool -> {"response":"..."} 才是最终答案。
					if e.AgentName == "replanner" && strings.TrimSpace(msg.Content) != "" {
						if response, ok := extractPlanExecuteFinalResponse(msg.Content); ok {
							if isNonFinalToolLimitationAnswer(response) {
								adkWebNotes = appendNonEmptyNote(adkWebNotes, "replanner returned a tool limitation instead of a user-facing answer: "+response)
							} else {
								fullAnswer = appendGroupedReferences(response, adkArticleSources, adkWebSources)
								eventCh <- StreamEvent{Type: StreamEventChunk, Message: fullAnswer}
							}
						}
					}
				}
			}

			if strings.TrimSpace(fullAnswer) == "" {
				answer, fallbackErr := s.composeADKFallbackAnswer(ctx, queryForAgents, adkLocalNotes, adkWebNotes, adkArticleSources, adkWebSources)
				if fallbackErr == nil && strings.TrimSpace(answer) != "" {
					fullAnswer = answer
					if currentStep != nil {
						appendStepDetail(currentStep, "replanner 未通过 respond 工具产出最终答案，已根据已执行步骤和检索结果生成兜底回答。")
						if conv != nil && s.runStepRepo != nil {
							_ = s.runStepRepo.Update(currentStep)
						}
					}
					eventCh <- StreamEvent{Type: StreamEventChunk, Message: fullAnswer}
				} else {
					if currentStep != nil {
						currentStep.Status = "failed"
						appendStepDetail(currentStep, "ADK 运行结束，但 replanner 没有通过 respond 工具产出最终答案。")
						if conv != nil && s.runStepRepo != nil {
							_ = s.runStepRepo.Update(currentStep)
						}
						eventCh <- StreamEvent{
							Type:      StreamEventStep,
							StepID:    currentStep.ID,
							AgentName: currentStep.AgentName,
							Status:    "failed",
							Summary:   currentStep.Summary,
							Detail:    currentStep.Detail,
						}
					}
					err := fmt.Errorf("ADK run completed without final respond output")
					if fallbackErr != nil {
						err = fmt.Errorf("%w: %v", err, fallbackErr)
					}
					s.logStage(conv, userID, "failed", "ADK 未产出最终回答", err.Error())
					errCh <- err
					return
				}
			}

			if currentStep != nil {
				currentStep.Status = "completed"
				if conv != nil && s.runStepRepo != nil {
					_ = s.runStepRepo.Update(currentStep)
				}
				eventCh <- StreamEvent{
					Type:      StreamEventStep,
					StepID:    currentStep.ID,
					AgentName: currentStep.AgentName,
					Status:    "completed",
					Summary:   currentStep.Summary,
					Detail:    currentStep.Detail,
				}
			}

			if conv != nil && fullAnswer != "" {
				// 完整答案落库后，前端 done 阶段会重新拉取会话详情，拿到服务端权威状态。
				s.saveConversationMessageWithWarning(conv.ID, "assistant", fullAnswer, "Failed to save assistant message")
				s.updateConversationMetadataWithWarning(conv, question)
				s.persistCompletedRun(conv.ID, derefUserID(userID), question, fullAnswer, decision)
				s.updateConversationMemoryWithWarning(conv.ID, derefUserID(userID), appendConversationTurn(history, question, fullAnswer))
			}

			s.logStage(conv, userID, "completed", "多 Agent 协作完成", fmt.Sprintf("答案长度: %d，答案内容：%v", len(fullAnswer), fullAnswer))
			eventCh <- StreamEvent{Type: StreamEventStage, Stage: "completed", Label: "调研已完成"}
			eventCh <- StreamEvent{Type: StreamEventDone, Stage: "completed"}
			return
		}

		// Fallback to manual orchestration if adkRunner is missing (legacy path)
		s.logStage(conv, userID, "manual_start", "开始手动编排流程", queryForAgents)

		// Manual steps also need persistent tracking
		createStep := func(agent, summary string) *model.ConversationRunStep {
			var conversationID int64
			if conv != nil {
				conversationID = conv.ID
			}
			step := &model.ConversationRunStep{
				ConversationID: conversationID,
				RunID:          runID,
				AgentName:      agent,
				Type:           "thinking",
				Summary:        summary,
				Status:         "running",
			}
			if conv != nil && s.runStepRepo != nil {
				_ = s.runStepRepo.Create(step)
			}
			eventCh <- StreamEvent{
				Type:      StreamEventStep,
				StepID:    step.ID,
				AgentName: step.AgentName,
				Status:    "running",
				Summary:   step.Summary,
			}
			return step
		}

		updateStep := func(step *model.ConversationRunStep, detail string, completed bool) {
			if step == nil {
				return
			}
			step.Detail = detail
			if completed {
				step.Status = "completed"
			}
			if conv != nil && s.runStepRepo != nil {
				_ = s.runStepRepo.Update(step)
			}
			eventCh <- StreamEvent{
				Type:      StreamEventStep,
				StepID:    step.ID,
				AgentName: step.AgentName,
				Status:    step.Status,
				Summary:   step.Summary,
				Detail:    step.Detail,
			}
		}

		eventCh <- StreamEvent{Type: StreamEventStage, Stage: "local_search", Label: "正在检索站内知识"}
		libStep := createStep("Librarian", "正在检索站内知识")
		localResult, err := s.librarian.Search(ctx, queryForAgents)
		if err != nil {
			if libStep != nil {
				libStep.Status = "failed"
				libStep.Detail = err.Error()
				_ = s.runStepRepo.Update(libStep)
			}
			s.logStage(conv, userID, "failed", "本地检索失败", err.Error())
			errCh <- err
			return
		}
		updateStep(libStep, formatLibrarianStepDetail(localResult), true)
		s.logStage(conv, userID, "local_search_done", "本地检索完成", fmt.Sprintf("状态: %s", localResult.CoverageStatus))

		var webResult *JournalistResult
		if localResult.CoverageStatus != "sufficient" && s.journalist != nil {
			eventCh <- StreamEvent{Type: StreamEventStage, Stage: "web_research", Label: "正在进行外部调研"}
			s.logStage(conv, userID, "web_research_start", "开始外部调研", "")
			jouStep := createStep("Journalist", "正在进行外部调研")
			webResult, err = s.journalist.Research(ctx, queryForAgents, localResult)
			if err != nil {
				if jouStep != nil {
					jouStep.Status = "failed"
					jouStep.Detail = err.Error()
					_ = s.runStepRepo.Update(jouStep)
				}
				s.logStage(conv, userID, "failed", "外部调研失败", err.Error())
				errCh <- err
				return
			}
			updateStep(jouStep, formatJournalistStepDetail(webResult), true)
			s.logStage(conv, userID, "web_research_done", "外部调研完成", fmt.Sprintf("来源数: %d", len(webResult.Sources)))
		}

		eventCh <- StreamEvent{Type: StreamEventStage, Stage: "integration", Label: "正在整合专家结果"}
		synStep := createStep("Synthesizer", "正在整合专家结果")
		answer, sources, err := s.synthesizer.Compose(ctx, queryForAgents, localResult, webResult)
		if err != nil {
			if synStep != nil {
				synStep.Status = "failed"
				synStep.Detail = err.Error()
				_ = s.runStepRepo.Update(synStep)
			}
			s.logStage(conv, userID, "failed", "结果整合失败", err.Error())
			errCh <- err
			return
		}
		updateStep(synStep, formatSynthesizerStepDetail(localResult, webResult, sources), true)
		s.logStage(conv, userID, "integration_done", "结果整合完成", "")

		if conv != nil {
			s.saveConversationMessageWithWarning(conv.ID, "assistant", answer, "Failed to save assistant message")
			s.updateConversationMetadataWithWarning(conv, question)
			s.persistCompletedRun(conv.ID, derefUserID(userID), question, answer, decision)
			s.updateConversationMemoryWithWarning(conv.ID, derefUserID(userID), appendConversationTurn(history, question, answer))
		}

		for _, chunk := range splitStreamChunks(answer) {
			eventCh <- StreamEvent{Type: StreamEventChunk, Message: chunk, Sources: sources}
		}
		eventCh <- StreamEvent{Type: StreamEventStage, Stage: "completed", Label: "回答已生成"}
		eventCh <- StreamEvent{Type: StreamEventDone, Stage: "completed"}
	}()
	return eventCh, errCh
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

func (s *thinkTankService) getOwnedConversation(conversationID *int64, userID *int64) (*model.Conversation, error) {
	if conversationID == nil || *conversationID <= 0 {
		return nil, nil
	}
	if userID == nil {
		return nil, fmt.Errorf("user authentication required")
	}
	conv, err := s.convRepo.GetByID(*conversationID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}
	if conv.UserID != *userID {
		return nil, fmt.Errorf("conversation access denied")
	}
	return conv, nil
}

func (s *thinkTankService) saveConversationMessageWithWarning(conversationID int64, role string, content string, logMessage string) {
	if s.msgRepo == nil {
		return
	}
	if err := s.msgRepo.Create(&model.ChatMessage{ConversationID: conversationID, Role: role, Content: content}); err != nil {
		if s.logger != nil {
			s.logger.LogError(AILogEntry{ConversationID: conversationID, Stage: "persistence", Message: logMessage, Detail: err.Error()})
		}
	}
}

func (s *thinkTankService) updateConversationMetadataWithWarning(conv *model.Conversation, question string) {
	if conv == nil || s.convRepo == nil {
		return
	}
	conv.UpdatedAt = time.Now()
	if conv.Title == "" || conv.Title == "New Conversation" || conv.Title == "新会话" || conv.Title == "New Chat" {
		conv.Title = buildConversationTitle(question)
	}
	_ = s.convRepo.Update(conv)
}

func (s *thinkTankService) persistADKClarification(conversationID int64, userID int64, runID int64, question string, clarification string, checkpointID string, decision PlannerDecision) {
	if s.runRepo == nil {
		return
	}
	run := &model.ConversationRun{
		ID:               runID,
		ConversationID:   conversationID,
		UserID:           userID,
		Status:           "waiting_user",
		CurrentStage:     "clarifying",
		OriginalQuestion: question,
		PendingQuestion:  &clarification,
		LastPlan:         decision.PlanSummary,
		PendingContext:   marshalADKPendingContext(checkpointID),
	}
	if run.ID > 0 {
		_ = s.runRepo.Update(run)
		return
	}
	_ = s.runRepo.Create(run)
}

func (s *thinkTankService) persistCompletedRun(conversationID int64, userID int64, question string, answer string, decision PlannerDecision) {
	if s.runRepo == nil {
		return
	}
	now := time.Now()
	run := &model.ConversationRun{
		ConversationID:   conversationID,
		UserID:           userID,
		Status:           "completed",
		CurrentStage:     "completed",
		OriginalQuestion: question,
		LastPlan:         decision.PlanSummary,
		PendingContext:   answer,
		CompletedAt:      &now,
	}
	if existing, _ := s.runRepo.GetActiveByConversationID(conversationID); existing != nil {
		run.ID = existing.ID
		_ = s.runRepo.Update(run)
		return
	}
	_ = s.runRepo.Create(run)
}

func (s *thinkTankService) loadConversationMemories(conversationID int64) []model.ConversationMemory {
	if s.memoryRepo == nil || conversationID <= 0 {
		return nil
	}
	memories, err := s.memoryRepo.GetByConversationID(conversationID)
	if err != nil {
		if s.logger != nil {
			s.logger.LogError(AILogEntry{ConversationID: conversationID, Stage: "memory", Message: "Failed to load conversation memory", Detail: err.Error()})
		}
		return nil
	}
	return memories
}

func (s *thinkTankService) updateConversationMemoryWithWarning(conversationID int64, userID int64, history []model.ChatMessage) {
	if s.memoryRepo == nil || conversationID <= 0 {
		return
	}
	olderEnd := len(history) - recentMemoryMessageCount
	if olderEnd <= 0 {
		return
	}
	older := history[:olderEnd]
	existing := s.loadConversationMemories(conversationID)
	drafts, err := s.summarizeConversationMemory(context.Background(), older, existing)
	if err != nil && s.logger != nil {
		s.logger.LogError(AILogEntry{ConversationID: conversationID, UserID: userID, Stage: "memory", Message: "Dynamic memory summarizer failed; using fallback", Detail: err.Error()})
	}
	if len(drafts) == 0 {
		return
	}
	for _, draft := range drafts {
		if strings.TrimSpace(draft.Scope) == "" || strings.TrimSpace(draft.Content) == "" {
			continue
		}
		importance := draft.Importance
		if importance <= 0 {
			importance = 1
		}
		memory := &model.ConversationMemory{
			ConversationID:       conversationID,
			UserID:               userID,
			Scope:                draft.Scope,
			Content:              strings.TrimSpace(draft.Content),
			SourceMessageIDStart: firstMessageID(older),
			SourceMessageIDEnd:   lastMessageID(older),
			Importance:           importance,
		}
		if err := s.memoryRepo.Upsert(memory); err != nil && s.logger != nil {
			s.logger.LogError(AILogEntry{ConversationID: conversationID, UserID: userID, Stage: "memory", Message: "Failed to update conversation memory", Detail: err.Error()})
		}
	}
}

func (s *thinkTankService) summarizeConversationMemory(ctx context.Context, older []model.ChatMessage, existing []model.ConversationMemory) ([]ConversationMemoryDraft, error) {
	if s.memorySummarizer != nil {
		drafts, err := s.memorySummarizer.Summarize(ctx, older, existing)
		if err == nil && len(drafts) > 0 {
			return drafts, nil
		}
		if err != nil {
			return fallbackMemoryDrafts(older), err
		}
	}
	return fallbackMemoryDrafts(older), nil
}

func buildADKCheckpointID(conv *model.Conversation, question string) string {
	conversationID := int64(0)
	if conv != nil {
		conversationID = conv.ID
	}
	return fmt.Sprintf("thinktank-%d-%d-%d", conversationID, time.Now().UnixNano(), len([]rune(question)))
}

func marshalADKPendingContext(checkpointID string) string {
	payload, err := json.Marshal(adkPendingContext{Type: "adk_interrupt", Checkpoint: checkpointID})
	if err != nil {
		return ""
	}
	return string(payload)
}

func parseADKPendingContext(run *model.ConversationRun) (adkPendingContext, bool) {
	if run == nil || run.Status != "waiting_user" || strings.TrimSpace(run.PendingContext) == "" {
		return adkPendingContext{}, false
	}
	var payload adkPendingContext
	if err := json.Unmarshal([]byte(run.PendingContext), &payload); err != nil {
		return adkPendingContext{}, false
	}
	if payload.Type != "adk_interrupt" || strings.TrimSpace(payload.Checkpoint) == "" {
		return adkPendingContext{}, false
	}
	return payload, true
}

func fallbackMemoryDrafts(history []model.ChatMessage) []ConversationMemoryDraft {
	summary := summarizeOlderConversationMemory(history)
	if summary == "" {
		return nil
	}
	return []ConversationMemoryDraft{{Scope: ConversationMemoryScopeSummary, Content: summary, Importance: 1}}
}

func appendConversationTurn(history []model.ChatMessage, question string, answer string) []model.ChatMessage {
	next := make([]model.ChatMessage, 0, len(history)+2)
	next = append(next, history...)
	next = append(next, model.ChatMessage{Role: "user", Content: question})
	next = append(next, model.ChatMessage{Role: "assistant", Content: answer})
	return next
}

func firstMessageID(messages []model.ChatMessage) int64 {
	for _, msg := range messages {
		if msg.ID != 0 {
			return msg.ID
		}
	}
	return 0
}

func lastMessageID(messages []model.ChatMessage) int64 {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].ID != 0 {
			return messages[i].ID
		}
	}
	return 0
}

func (s *thinkTankService) logStage(conv *model.Conversation, userID *int64, stage string, message string, detail string) {
	if s.logger == nil {
		return
	}
	entry := AILogEntry{Stage: stage, Message: message, Detail: detail}
	if conv != nil {
		entry.ConversationID = conv.ID
	}
	if userID != nil {
		entry.UserID = *userID
	}
	if stage == "failed" {
		s.logger.LogError(entry)
		return
	}
	s.logger.LogStage(entry)
}

func appendStepDetail(step *model.ConversationRunStep, detail string) {
	detail = normalizeStepDetail(detail)
	if step == nil || detail == "" {
		return
	}
	if strings.TrimSpace(step.Detail) == "" {
		step.Detail = detail
		return
	}
	step.Detail = normalizeStepDetail(strings.TrimSpace(step.Detail) + "\n\n" + detail)
}

func normalizeStepDetail(detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return ""
	}
	runes := []rune(detail)
	if len(runes) <= maxStepDetailRunes {
		return detail
	}
	return string(runes[:maxStepDetailRunes]) + "\n\n[内容过长，已截断；完整原始日志请查看服务端日志或工具结果源。]"
}

func formatLibrarianStepDetail(result LibrarianResult) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("检索完成。找到 %d 个相关来源。覆盖状态：%s", len(result.Sources), result.CoverageStatus))
	if strings.TrimSpace(result.Summary) != "" {
		builder.WriteString("\n\n站内知识摘要：\n")
		builder.WriteString(strings.TrimSpace(result.Summary))
	}
	if len(result.Sources) > 0 {
		builder.WriteString("\n\n站内来源：")
		for _, source := range result.Sources {
			builder.WriteString("\n")
			builder.WriteString(formatSourceRef(source))
		}
	}
	if strings.TrimSpace(result.FollowupHint) != "" {
		builder.WriteString("\n\n后续提示：")
		builder.WriteString(strings.TrimSpace(result.FollowupHint))
	}
	return builder.String()
}

func formatJournalistStepDetail(result *JournalistResult) string {
	if result == nil {
		return "外部调研完成，但没有返回可用结果。"
	}
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("外部调研完成。从 %d 个来源获取了信息。", len(result.Sources)))
	if strings.TrimSpace(result.Summary) != "" {
		builder.WriteString("\n\n外部调研结果：\n")
		builder.WriteString(strings.TrimSpace(result.Summary))
	}
	if len(result.Sources) > 0 {
		builder.WriteString("\n\n外部来源：")
		for _, source := range result.Sources {
			builder.WriteString("\n")
			builder.WriteString(formatSourceRef(source))
		}
	}
	if strings.TrimSpace(result.KnowledgeDraftTitle) != "" {
		builder.WriteString("\n\n知识草稿：")
		builder.WriteString(strings.TrimSpace(result.KnowledgeDraftTitle))
	}
	return builder.String()
}

func formatSynthesizerStepDetail(local LibrarianResult, web *JournalistResult, sources []string) string {
	var builder strings.Builder
	builder.WriteString("已完成多方结果整合，正在生成最终回答。")
	builder.WriteString(fmt.Sprintf("\n\n站内来源数：%d", len(local.Sources)))
	if web != nil {
		builder.WriteString(fmt.Sprintf("\n外部来源数：%d", len(web.Sources)))
	}
	if len(sources) > 0 {
		builder.WriteString("\n\n最终引用：")
		for _, source := range sources {
			if strings.TrimSpace(source) == "" {
				continue
			}
			builder.WriteString("\n- ")
			builder.WriteString(strings.TrimSpace(source))
		}
	}
	return builder.String()
}

func formatSourceRef(source SourceRef) string {
	title := strings.TrimSpace(source.Title)
	if title == "" {
		title = "未命名来源"
	}
	if strings.TrimSpace(source.URL) == "" {
		return "- " + title
	}
	return fmt.Sprintf("- %s (%s)", title, strings.TrimSpace(source.URL))
}

func formatADKActionDetail(action *adk.AgentAction) string {
	if action == nil {
		return ""
	}
	switch {
	case action.Interrupted != nil:
		question := extractADKClarificationQuestion(action.Interrupted)
		if question == "" {
			return "流程中断：等待用户补充信息"
		}
		return "流程中断：等待用户补充信息\n问题：" + question
	case action.TransferToAgent != nil:
		return "流程切换：切换为 " + action.TransferToAgent.DestAgentName + " Agent"
	case action.Exit:
		return "流程动作：当前 Agent 结束执行"
	case action.BreakLoop != nil:
		return "流程动作：跳出当前执行循环"
	case action.CustomizedAction != nil:
		if data, err := json.Marshal(action.CustomizedAction); err == nil {
			return "原始动作日志：\n" + string(data)
		}
		return fmt.Sprintf("原始动作日志：%v", action.CustomizedAction)
	default:
		return ""
	}
}

func extractADKClarificationQuestion(info *adk.InterruptInfo) string {
	if info == nil {
		return ""
	}
	// Eino 的中断 Data 可能包含完整 checkpoint、OrigInput、嵌套子图状态等内部结构；
	// ask_for_clarification 真正要展示给用户的问题在 root-cause InterruptContext.Info 中。
	for _, ctx := range info.InterruptContexts {
		if ctx == nil {
			continue
		}
		if ctx.IsRootCause {
			if text := interruptInfoToUserText(ctx.Info); text != "" {
				return text
			}
		}
	}
	for _, ctx := range info.InterruptContexts {
		if ctx == nil {
			continue
		}
		if text := interruptInfoToUserText(ctx.Info); text != "" {
			return text
		}
	}
	return interruptInfoToUserText(info.Data)
}

func interruptInfoToUserText(info any) string {
	switch v := info.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func extractPlanExecuteFinalResponse(content string) (string, bool) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", false
	}
	var payload struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return "", false
	}
	response := strings.TrimSpace(payload.Response)
	if response == "" {
		return "", false
	}
	return response, true
}

func isNonFinalToolLimitationAnswer(response string) bool {
	response = strings.TrimSpace(response)
	if response == "" {
		return false
	}
	markers := []string{
		"DocParser",
		"当前工具列表中无",
		"工具列表中无",
		"无法完成解析 HTML",
		"无法完成解析HTML",
		"请提供其他可行的工具",
		"missing tool",
		"tool is missing",
		"unavailable tool",
	}
	for _, marker := range markers {
		if strings.Contains(response, marker) {
			return true
		}
	}
	return false
}

func formatADKMessageDetail(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	if strings.TrimSpace(msg.ReasoningContent) != "" {
		parts = append(parts, "思考步骤：\n"+strings.TrimSpace(msg.ReasoningContent))
	}
	for _, toolCall := range msg.ToolCalls {
		name := strings.TrimSpace(toolCall.Function.Name)
		if name == "" {
			name = "unknown_tool"
		}
		args := strings.TrimSpace(toolCall.Function.Arguments)
		if args == "" {
			args = "{}"
		}
		parts = append(parts, fmt.Sprintf("工具调用：%s\n参数：%s", name, args))
	}
	if strings.TrimSpace(msg.ToolName) != "" {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			content = "(空结果)"
		}
		parts = append(parts, fmt.Sprintf("工具返回：%s\n结果：%s", strings.TrimSpace(msg.ToolName), content))
	} else if strings.TrimSpace(msg.Content) != "" {
		parts = append(parts, "Agent 输出：\n"+strings.TrimSpace(msg.Content))
	}
	return strings.Join(parts, "\n\n")
}

func extractLocalSearchArticleSources(content string) []SourceRef {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	var payload struct {
		Sources []SourceRef `json:"sources"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil
	}
	result := make([]SourceRef, 0, len(payload.Sources))
	for _, source := range payload.Sources {
		if source.Kind == "" {
			source.Kind = "article"
		}
		if source.Kind != "article" || strings.TrimSpace(source.Title) == "" || strings.TrimSpace(source.URL) == "" {
			continue
		}
		result = append(result, source)
	}
	return result
}

func extractLocalSearchSummary(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	var payload struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Summary)
}

func extractWebSearchSources(content string) []SourceRef {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	var payload struct {
		Organic []struct {
			Title string `json:"title"`
			Link  string `json:"link"`
		} `json:"organic"`
		Items []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil
	}
	result := make([]SourceRef, 0, len(payload.Organic)+len(payload.Items))
	for _, item := range payload.Organic {
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Link) == "" {
			continue
		}
		result = append(result, SourceRef{Kind: "web", Title: item.Title, URL: item.Link})
	}
	for _, item := range payload.Items {
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.URL) == "" {
			continue
		}
		result = append(result, SourceRef{Kind: "web", Title: item.Title, URL: item.URL})
	}
	return result
}

func summarizeWebSearchResult(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	var payload struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return ""
	}
	lines := make([]string, 0, 5)
	for _, item := range payload.Organic {
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}
		line := title
		if snippet := strings.TrimSpace(item.Snippet); snippet != "" {
			line += "：" + snippet
		}
		if link := strings.TrimSpace(item.Link); link != "" {
			line += " (" + link + ")"
		}
		lines = append(lines, line)
		if len(lines) >= 5 {
			break
		}
	}
	return strings.Join(lines, "\n")
}

func appendNonEmptyNote(notes []string, note string) []string {
	note = strings.TrimSpace(note)
	if note == "" || looksLikeRawHTML(note) {
		return notes
	}
	if len([]rune(note)) > 1200 {
		note = string([]rune(note)[:1200]) + "..."
	}
	for _, existing := range notes {
		if existing == note {
			return notes
		}
	}
	return append(notes, note)
}

func compactNotes(notes []string) string {
	compacted := make([]string, 0, len(notes))
	for _, note := range notes {
		note = strings.TrimSpace(note)
		if note == "" || looksLikeRawHTML(note) {
			continue
		}
		compacted = append(compacted, note)
		if len(compacted) >= 6 {
			break
		}
	}
	return strings.Join(compacted, "\n\n")
}

func looksLikeRawHTML(text string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(text))
	return strings.HasPrefix(trimmed, "<!doctype html") || strings.HasPrefix(trimmed, "<html") || strings.Contains(trimmed, "<script")
}

func mergeSourceRefs(existing []SourceRef, next []SourceRef) []SourceRef {
	if len(next) == 0 {
		return existing
	}
	seen := make(map[string]struct{}, len(existing)+len(next))
	result := make([]SourceRef, 0, len(existing)+len(next))
	for _, source := range existing {
		key := source.Title + "|" + source.URL
		if key == "|" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, source)
	}
	for _, source := range next {
		key := source.Title + "|" + source.URL
		if key == "|" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, source)
	}
	return result
}

func buildConversationMemory(history []model.ChatMessage, memories []model.ConversationMemory) string {
	return buildConversationMemoryForQuestion("", history, memories)
}

func buildConversationMemoryForQuestion(question string, history []model.ChatMessage, memories []model.ConversationMemory) string {
	if len(history) == 0 && len(memories) == 0 {
		return ""
	}
	var builder strings.Builder

	if len(memories) > 0 {
		builder.WriteString("长期记忆：")
		for _, memory := range memories {
			content := strings.TrimSpace(memory.Content)
			if content == "" {
				continue
			}
			builder.WriteString("\n- ")
			builder.WriteString(content)
		}
	}

	start := selectRecentMemoryStart(question, history)
	older := history[:start]
	if len(older) > 0 {
		if summary := summarizeOlderConversationMemory(older); summary != "" {
			if builder.Len() > 0 {
				builder.WriteString("\n\n")
			}
			builder.WriteString("更早对话摘要：\n- ")
			builder.WriteString(summary)
		}
	}
	relevant := selectRelevantOlderMessages(question, older, 3)
	if len(relevant) > 0 {
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString("相关历史片段：")
		for _, msg := range relevant {
			builder.WriteString("\n")
			builder.WriteString(msg.Role)
			builder.WriteString(": ")
			builder.WriteString(truncateRunes(strings.TrimSpace(msg.Content), 160))
		}
	}

	if builder.Len() > 0 && len(history[start:]) > 0 {
		builder.WriteString("\n\n")
	}
	if len(history[start:]) > 0 {
		builder.WriteString("最近对话：\n")
	}
	for _, msg := range history[start:] {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		builder.WriteString(msg.Role)
		builder.WriteString(": ")
		builder.WriteString(truncateRunes(content, 120))
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func compressConversationMemory(history []model.ChatMessage) string {
	return buildConversationMemoryForQuestion("", history, nil)
}

func selectRecentMemoryStart(question string, history []model.ChatMessage) int {
	if len(history) == 0 {
		return 0
	}
	if strings.TrimSpace(question) == "" && len(history) > recentMemoryMessageCount {
		return len(history) - recentMemoryMessageCount
	}
	budget := recentMemoryRuneBudget(question)
	used := 0
	start := len(history)
	for i := len(history) - 1; i >= 0; i-- {
		msgCost := len([]rune(strings.TrimSpace(history[i].Role))) + len([]rune(strings.TrimSpace(history[i].Content))) + 4
		if start < len(history) && used+msgCost > budget {
			break
		}
		used += msgCost
		start = i
	}
	if len(history)-start < 2 && len(history) >= 2 {
		start = len(history) - 2
	}
	return start
}

func recentMemoryRuneBudget(question string) int {
	question = strings.TrimSpace(question)
	runeCount := len([]rune(question))
	if runeCount <= 8 || isReferentialQuestion(question) {
		return 1200
	}
	if runeCount > 120 {
		return 480
	}
	return 720
}

func isReferentialQuestion(question string) bool {
	if question == "" {
		return false
	}
	for _, token := range []string{"继续", "这个", "那个", "上面", "刚才", "前面", "原文", "这篇", "它", "他们", "这种"} {
		if strings.Contains(question, token) {
			return true
		}
	}
	return false
}

func selectRelevantOlderMessages(question string, older []model.ChatMessage, limit int) []model.ChatMessage {
	if limit <= 0 || len(older) == 0 {
		return nil
	}
	keywords := memoryKeywords(question)
	if len(keywords) == 0 {
		return nil
	}
	result := make([]model.ChatMessage, 0, limit)
	for _, msg := range older {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		for _, keyword := range keywords {
			if strings.Contains(strings.ToLower(content), keyword) {
				result = append(result, msg)
				break
			}
		}
		if len(result) >= limit {
			break
		}
	}
	return result
}

func memoryKeywords(question string) []string {
	question = strings.ToLower(strings.TrimSpace(question))
	if question == "" {
		return nil
	}
	parts := strings.FieldsFunc(question, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' || r == ',' || r == '，' || r == '。' || r == '?' || r == '？' || r == '!' || r == '！' || r == '；' || r == ';'
	})
	seen := make(map[string]struct{})
	keywords := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len([]rune(part)) < 2 || isMemoryStopword(part) {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		keywords = append(keywords, part)
	}
	return keywords
}

func isMemoryStopword(token string) bool {
	switch token {
	case "这个", "那个", "还有", "继续", "需要", "什么", "一下", "如何", "怎么", "我们", "你们", "他们", "the", "and", "for", "with":
		return true
	default:
		return false
	}
}

func summarizeOlderConversationMemory(history []model.ChatMessage) string {
	snippets := make([]string, 0, 3)
	for _, msg := range history {
		content := extractMemorySnippet(msg.Content)
		if content == "" {
			continue
		}
		if msg.Role == "user" {
			snippets = append(snippets, content)
		}
		if len(snippets) >= 3 {
			break
		}
	}
	if len(snippets) == 0 {
		for _, msg := range history {
			content := extractMemorySnippet(msg.Content)
			if content == "" {
				continue
			}
			snippets = append(snippets, content)
			if len(snippets) >= 3 {
				break
			}
		}
	}
	if len(snippets) == 0 {
		return ""
	}
	return "更早对话主要讨论：" + strings.Join(snippets, "；") + "。"
}

func extractMemorySnippet(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	content = strings.ReplaceAll(content, "\n", " ")
	cutAt := -1
	for _, sep := range []string{"。", "，", ",", "；", ";", "！", "？", "!", "?"} {
		if idx := strings.Index(content, sep); idx > 0 {
			if cutAt == -1 || idx < cutAt {
				cutAt = idx
			}
		}
	}
	if cutAt > 0 {
		content = content[:cutAt]
	}
	content = strings.TrimSpace(content)
	return strings.TrimRight(truncateRunes(content, maxMemorySnippetRunes), "。；，,;!?！？")
}

func truncateRunes(content string, limit int) string {
	content = strings.TrimSpace(content)
	if limit <= 0 {
		return ""
	}
	runes := []rune(content)
	if len(runes) <= limit {
		return content
	}
	return string(runes[:limit])
}

func buildAgentQuery(question string, memory string) string {
	memory = strings.TrimSpace(memory)
	if memory == "" {
		return question
	}
	return "历史上下文：\n" + memory + "\n\n当前问题：\n" + question
}

func derefUserID(userID *int64) int64 {
	if userID == nil {
		return 0
	}
	return *userID
}

func splitStreamChunks(message string) []string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return []string{""}
	}
	parts := strings.Split(trimmed, "\n\n")
	chunks := make([]string, 0, len(parts))
	var builder strings.Builder
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(part)
		chunks = append(chunks, builder.String())
	}
	if len(chunks) == 0 {
		return []string{trimmed}
	}
	return chunks
}
