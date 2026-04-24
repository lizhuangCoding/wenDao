package service

import (
	"context"
	"fmt"
	"strings"

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
