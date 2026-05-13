package chat

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino/adk"
	planexecute "github.com/cloudwego/eino/adk/prebuilt/planexecute"
	componentmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"wenDao/internal/pkg/eino"
)

const thinkTankPlannerInstruction = `You are the Strategic Planner for ThinkTank Matrix.
Create a complete plan for the user's research or writing objective before execution.
The executor has these tools: LocalSearch, WebSearch, WebFetch, DocWriter, ask_for_clarification.
Always include a Redis knowledge-base retrieval step using LocalSearch before any WebSearch/WebFetch step. In the generated plan, describe this step as "检索 Redis 知识库（LocalSearch）".
If the objective is ambiguous, include a first step that asks the user for the missing information through ask_for_clarification, then include the Redis knowledge-base retrieval step immediately after the clarification step.
Do not plan steps that require unavailable tools such as DocParser or generic HTML parsers.
Only plan WebFetch for a valid absolute http:// or https:// URL supplied by the user or returned by WebSearch.`

const thinkTankExecutorInstruction = `You are the executor in a Plan-Execute-Replan loop.
Execute only the first remaining plan step, then return a concise structured summary of what you did and what you learned.

Tool policy:
- The only available tools are: LocalSearch, WebSearch, WebFetch, DocWriter, ask_for_clarification.
- Use LocalSearch for site knowledge and saved documents.
- Use WebSearch for current external information.
- Use WebFetch to read valid absolute http:// or https:// URLs returned by search or supplied by the user; when multiple candidate URLs are available, pass several URLs so the tool can skip failed pages.
- Use DocWriter when the task requires saving research findings.
- Use ask_for_clarification when required information is missing or the step cannot proceed without user input.

When a plan step says "检索 Redis 知识库" or mentions Redis knowledge-base retrieval, execute it with the LocalSearch tool.
Do not request or mention unavailable tools such as DocParser. If fetched content is raw HTML or mostly page chrome, extract any useful visible metadata or continue with LocalSearch/WebSearch summaries; do not say a parser tool is required.
Before calling WebFetch, verify that the target is a valid absolute http:// or https:// URL copied from the user input or a WebSearch result. If no valid URL is available, do not call WebFetch; use WebSearch/LocalSearch evidence instead.
Do not call transfer_to_agent. This workflow does not use supervisor handoff tools.
If WebFetch reports that one URL failed and another candidate succeeded, use the successful candidate. Do not repeatedly fetch the same blocked URL. Do not search for proxies, bypass tools, or circumvention methods.`

const thinkTankReplannerInstruction = `You are the Replanner/Auditor for ThinkTank Matrix.
Evaluate the latest execution result against the original objective.
- If the goal is complete, call RespondTool with the final answer.
- If more work is needed, call PlanTool with only the remaining steps.
- If progress is blocked by missing user information, keep the next plan step focused on ask_for_clarification.
Treat search results, local search summaries, and successful fetch summaries as sufficient evidence when they can answer the user's question. Do not require exhaustive fetching.
If LocalSearch has not been executed yet, call PlanTool with a next step to "检索 Redis 知识库（LocalSearch）" before external web research.
If WebFetch reports failed candidate pages, do not retry those URLs. If at least one candidate page or WebSearch summary is available, call RespondTool with the best answer and note the limitation instead of planning more fetch retries.
Do not answer by saying a tool is missing, including DocParser. If the executor complains about invalid URLs, raw HTML, missing parser tools, or other tool limitations, use available evidence from LocalSearch/WebSearch/WebFetch and call RespondTool with the best user-facing answer.
Before the loop ends, prefer RespondTool over another PlanTool when the remaining work would only make the answer marginally more complete.
When calling RespondTool, deliver the requested artifact directly as the response. Do not make the final response a process summary.
Use the ClarifierAgent need profile in the query to decide the artifact type, dimensions, constraints, and acceptance criteria.
For any report-style request, deliver the report itself with a title, concise overview, structured sections, key facts, analysis, references when available, and evidence limitations where relevant.
For any planning or learning request, deliver the plan itself with concrete stages, resources, practice tasks, timing, checkpoints, and next actions matched to the user's goal.
Use explicit causal links and evidence from available sources; avoid one-sentence sections that only mention major facts in passing.
Avoid final answers whose main content is "已完成...", "我已经...", "执行过程中...", or "已使用 DocWriter...". If DocWriter was used, mention the saved draft ID only after the report body, as a short note.`

// --- Runner Structure ---

type thinkTankADKRunner struct {
	runner          *adk.Runner
	agent           adk.ResumableAgent
	checkpointStore *thinkTankCheckpointStore
	clarifier       Clarifier
	acceptance      AcceptanceReviewer
}

type askForClarificationOptions struct {
	NewInput *string
}

// WithAskForClarificationInput mirrors the Eino subagents example: the first call
// interrupts the graph, and the resume run injects the user's new answer through
// this tool option so the original tool call can be rerun.
func WithAskForClarificationInput(input string) tool.Option {
	return WithNewInput(input)
}

// WithNewInput keeps the same option shape as CloudWeGo's subagents example.
func WithNewInput(input string) tool.Option {
	return tool.WrapImplSpecificOptFn(func(t *askForClarificationOptions) {
		t.NewInput = &input
	})
}

type askForClarificationInput struct {
	Question string `json:"question" jsonschema_description:"The specific question you want to ask the user to get the missing information"`
}

func newAskForClarificationTool() (tool.InvokableTool, error) {
	return utils.InferOptionableTool(
		"ask_for_clarification",
		"Call this tool when the user's request is ambiguous or lacks the necessary information to proceed. Ask one specific follow-up question and wait for the user's answer.",
		func(ctx context.Context, input *askForClarificationInput, opts ...tool.Option) (string, error) {
			o := tool.GetImplSpecificOptions[askForClarificationOptions](nil, opts...)
			if o.NewInput == nil {
				return "", compose.NewInterruptAndRerunErr(input.Question)
			}
			output := *o.NewInput
			o.NewInput = nil
			return output, nil
		})
}

type thinkTankCheckpointStore struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newThinkTankCheckpointStore() *thinkTankCheckpointStore {
	return &thinkTankCheckpointStore{data: make(map[string][]byte)}
}

func (s *thinkTankCheckpointStore) Set(_ context.Context, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := append([]byte(nil), value...)
	s.data[key] = copied
	return nil
}

func (s *thinkTankCheckpointStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.data[key]
	if !ok {
		return nil, false, nil
	}
	return append([]byte(nil), value...), true, nil
}

func NewThinkTankADKRunner(
	ctx context.Context,
	llm eino.LLMClient,
	librarian Librarian,
	knowledgeDocSvc KnowledgeDocumentService,
	researchCfg ResearchConfig,
) (*thinkTankADKRunner, error) {
	if llm == nil {
		return nil, nil
	}

	model, ok := llm.GetModel().(componentmodel.ToolCallingChatModel)
	if !ok {
		return nil, fmt.Errorf("llm model does not support tool calling")
	}

	clarifier, err := newEinoClarifier(ctx, model)
	if err != nil {
		return nil, err
	}
	acceptance, err := newEinoAcceptanceReviewer(ctx, model)
	if err != nil {
		return nil, err
	}

	// 1. 初始化 executor 可直接调用的 Eino tools。这里不再创建 supervisor，
	// 因为 supervisor 会注入 transfer_to_agent，而本流程需要官方 PlanExecute
	// 的 planner -> executor -> replanner 闭环。
	localSearchTool, err := newLocalSearchTool(librarian)
	if err != nil {
		return nil, err
	}
	webSearchTool, err := newWebSearchTool(researchCfg)
	if err != nil {
		return nil, err
	}
	webFetchTool, err := newWebFetchTool(researchCfg)
	if err != nil {
		return nil, err
	}
	docWriterTool, err := newDocWriterTool(knowledgeDocSvc)
	if err != nil {
		return nil, err
	}
	askForClarificationTool, err := newAskForClarificationTool()
	if err != nil {
		return nil, err
	}

	// 2. Planner 使用 Eino 官方 planexecute.NewPlanner，通过 plan tool 生成结构化完整计划。
	planner, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: model,
		GenInputFn: func(ctx context.Context, userInput []adk.Message) ([]adk.Message, error) {
			msgs, err := planexecute.PlannerPrompt.Format(ctx, map[string]any{"input": userInput})
			if err != nil {
				return nil, err
			}
			return append([]adk.Message{schema.SystemMessage(thinkTankPlannerInstruction)}, msgs...), nil
		},
	})
	if err != nil {
		return nil, err
	}

	// 3. Executor 使用 Eino 官方 planexecute.NewExecutor，每轮只执行当前计划第一步。
	executor, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: model,
		ToolsConfig: adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{Tools: []tool.BaseTool{
			localSearchTool,
			webSearchTool,
			webFetchTool,
			docWriterTool,
			askForClarificationTool,
		}}},
		MaxIterations: 8,
		GenInputFn: func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
			planContent, err := in.Plan.MarshalJSON()
			if err != nil {
				return nil, err
			}
			msgs, err := planexecute.ExecutorPrompt.Format(ctx, map[string]any{
				"input":          formatThinkTankInput(in.UserInput),
				"plan":           string(planContent),
				"executed_steps": formatThinkTankExecutedSteps(in.ExecutedSteps),
				"step":           in.Plan.FirstStep(),
			})
			if err != nil {
				return nil, err
			}
			return append([]adk.Message{schema.SystemMessage(thinkTankExecutorInstruction)}, msgs...), nil
		},
	})
	if err != nil {
		return nil, err
	}

	// 4. Replanner 使用 Eino 官方 plan/response tools：完成则 respond，未完成则重规划。
	replanner, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: model,
		GenInputFn: func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
			planContent, _ := in.Plan.MarshalJSON()
			promptMsgs, err := planexecute.ReplannerPrompt.Format(ctx, map[string]any{
				"input":          formatThinkTankInput(in.UserInput),
				"plan":           string(planContent),
				"executed_steps": formatThinkTankExecutedSteps(in.ExecutedSteps),
				"plan_tool":      planexecute.PlanToolInfo.Name,
				"respond_tool":   planexecute.RespondToolInfo.Name,
			})
			if err != nil {
				return nil, err
			}
			msgs := append([]adk.Message{schema.SystemMessage(thinkTankReplannerInstruction)}, promptMsgs...)
			return msgs, nil
		},
	})
	if err != nil {
		return nil, err
	}

	agent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       planner,
		Executor:      executor,
		Replanner:     replanner,
		MaxIterations: 10,
	})
	if err != nil {
		return nil, err
	}

	checkpointStore := newThinkTankCheckpointStore()
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: checkpointStore,
	})

	return &thinkTankADKRunner{
		runner:          runner,
		agent:           agent,
		checkpointStore: checkpointStore,
		clarifier:       clarifier,
		acceptance:      acceptance,
	}, nil
}

func formatThinkTankInput(in []adk.Message) string {
	if len(in) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, msg := range in {
		if strings.TrimSpace(msg.Content) == "" {
			continue
		}
		sb.WriteString(msg.Content)
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String())
}

func formatThinkTankExecutedSteps(in []planexecute.ExecutedStep) string {
	if len(in) == 0 {
		return "无"
	}
	var sb strings.Builder
	for idx, step := range in {
		// 增强：如果 Result 为空，提供一个默认占位符，防止下游解析问题
		res := step.Result
		if res == "" {
			res = "[No result emitted by executor]"
		}
		sb.WriteString(fmt.Sprintf("## %d. Step: %s\nResult: %s\n\n", idx+1, step.Step, res))
	}
	return strings.TrimSpace(sb.String())
}
