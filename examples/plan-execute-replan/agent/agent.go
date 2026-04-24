/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/adk/multiagent/plan-execute-replan/tools"
)

func NewPlanner(ctx context.Context) (adk.Agent, error) {

	return planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: model.NewChatModel(),
	})
}

// 预先定义好的提示模板
var executorPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(`You are a diligent and meticulous travel research executor, Follow the given plan and execute your tasks carefully and thoroughly.
Execute each planning step by using available tools.
For weather queries, use get_weather tool.
For flight searches, use search_flights tool.
For hotel searches, use search_hotels tool.
For attraction research, use search_attractions tool.
For user's clarification, use ask_for_clarification tool. In summary, repeat the questions and results to confirm with the user, try to avoid disturbing users'
Provide detailed results for each task.
Cloud Call multiple tools to get the final result.`),
	schema.UserMessage(`## OBJECTIVE
{input}
## Given the following plan:
{plan}
## COMPLETED STEPS & RESULTS
{executed_steps}
## Your task is to execute the first step, which is: 
{step}`))

func formatInput(in []adk.Message) string {
	return in[0].Content
}

func formatExecutedSteps(in []planexecute.ExecutedStep) string {
	var sb strings.Builder
	for idx, m := range in {
		sb.WriteString(fmt.Sprintf("## %d. Step: %v\n  Result: %v\n\n", idx+1, m.Step, m.Result))
	}
	return sb.String()
}

func NewExecutor(ctx context.Context) (adk.Agent, error) {
	// Get travel tools for the executor
	travelTools, err := tools.GetAllTravelTools(ctx)
	if err != nil {
		return nil, err
	}

	return planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: travelTools,
			},
		},

		// 目的：这是 NewExecutor 中最核心、最关键的部分。GenInputFn (Generate Input Function) 的作用是在每次执行任务前，动态生成发送给大语言模型（LLM）的完整指令（即 Prompt）。
		//  `func(...)`：这里定义了一个匿名函数，它会在每次 Executor 需要行动时被调用。

		//  in.UserInput: 用户的原始请求。
		//  in.Plan: Planner 生成的完整计划。
		//  in.ExecutedSteps: 已经执行过的步骤及其结果。
		GenInputFn: func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
			planContent, err_ := in.Plan.MarshalJSON()
			if err_ != nil {
				return nil, err_
			}

			// 这里的“第一步”指的是剩余未执行步骤中的第一步。
			firstStep := in.Plan.FirstStep()

			msgs, err_ := executorPrompt.Format(ctx, map[string]any{
				"input":          formatInput(in.UserInput),
				"plan":           string(planContent),
				"executed_steps": formatExecutedSteps(in.ExecutedSteps),
				"step":           firstStep,
			})
			if err_ != nil {
				return nil, err_
			}

			return msgs, nil
		},
		// 最终，这个 GenInputFn 函数返回一个被完全填充、包含所有最新上下文的、准备发给大语言模型的指令 (msgs)。
	})
}

func NewReplanAgent(ctx context.Context) (adk.Agent, error) {
	return planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: model.NewChatModel(),
	})
}
