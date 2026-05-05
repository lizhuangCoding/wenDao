package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	componentmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const thinkTankClarifierInstruction = `You are the ThinkTank Clarifier.
Only ask the user when missing information would change what should be answered.
Do not ask follow-up questions for broad but clear research, explanation, comparison, or writing requests.
Infer reasonable target_dimensions, answer_goal, and constraints from the original question and context.
Return valid JSON only with keys: normalized_question, intent, answer_goal, target_dimensions, constraints, ambiguity_level, should_ask_user, clarification_question, reason.`

const thinkTankAcceptanceInstruction = `You are the ThinkTank Acceptance Reviewer.
Return pass when the answer substantially satisfies the user's original question and clarified target dimensions.
Return revise only when important requested dimensions, evidence, or answer structure are missing and can be fixed without asking the user.
Return ask_user only when the answer cannot proceed because critical user intent or constraints are still unknown.
Return valid JSON only with keys: verdict, score, matched_dimensions, missing_dimensions, unsupported_claims, format_issues, revision_instruction, user_question, reason.`

type einoClarifier struct {
	runner *adk.Runner
}

type einoAcceptanceReviewer struct {
	runner *adk.Runner
}

func newEinoClarifier(ctx context.Context, model componentmodel.ToolCallingChatModel) (Clarifier, error) {
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          thinkTankClarifierAgentName,
		Description:   "Clarifies ThinkTank user intent before answer generation.",
		Instruction:   thinkTankClarifierInstruction,
		Model:         model,
		MaxIterations: 2,
	})
	if err != nil {
		return nil, err
	}

	return &einoClarifier{
		runner: adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent}),
	}, nil
}

func newEinoAcceptanceReviewer(ctx context.Context, model componentmodel.ToolCallingChatModel) (AcceptanceReviewer, error) {
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          thinkTankAcceptanceAgentName,
		Description:   "Reviews ThinkTank answers against the clarified user intent.",
		Instruction:   thinkTankAcceptanceInstruction,
		Model:         model,
		MaxIterations: 2,
	})
	if err != nil {
		return nil, err
	}

	return &einoAcceptanceReviewer{
		runner: adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent}),
	}, nil
}

func (c *einoClarifier) Clarify(ctx context.Context, input ClarifierInput) (ClarifierDecision, error) {
	if c == nil || c.runner == nil {
		return defaultClarifierDecision(input.OriginalQuestion), nil
	}

	raw, err := runSingleAgentText(ctx, c.runner, buildClarifierPrompt(input))
	if err != nil {
		return defaultClarifierDecision(input.OriginalQuestion), err
	}
	return parseClarifierDecision(raw, input.OriginalQuestion), nil
}

func (r *einoAcceptanceReviewer) Review(ctx context.Context, input AcceptanceReviewInput) (AcceptanceReview, error) {
	if r == nil || r.runner == nil {
		return defaultAcceptanceReview(), nil
	}

	raw, err := runSingleAgentText(ctx, r.runner, buildAcceptancePrompt(input))
	if err != nil {
		return defaultAcceptanceReview(), err
	}
	return parseAcceptanceReview(raw), nil
}

func runSingleAgentText(ctx context.Context, runner *adk.Runner, prompt string) (string, error) {
	if runner == nil {
		return "", fmt.Errorf("thinktank review agent runner is nil")
	}

	iter := runner.Run(ctx, []adk.Message{schema.UserMessage(prompt)})
	var latest string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		output := event.Output.MessageOutput
		if output.Role == schema.Tool {
			continue
		}
		msg, err := output.GetMessage()
		if err != nil {
			return "", err
		}
		if msg == nil {
			continue
		}
		if content := strings.TrimSpace(msg.Content); content != "" {
			latest = content
		}
	}
	if latest == "" {
		return "", fmt.Errorf("thinktank review agent returned empty content")
	}
	return latest, nil
}

func buildClarifierPrompt(input ClarifierInput) string {
	payload := map[string]any{
		"original_question": strings.TrimSpace(input.OriginalQuestion),
		"agent_query":       strings.TrimSpace(input.AgentQuery),
		"policy": map[string]any{
			"should_ask_user": "only when missing information would change what should be answered",
			"output":          "Return valid JSON.",
		},
	}
	return marshalReviewPrompt(payload)
}

func buildAcceptancePrompt(input AcceptanceReviewInput) string {
	payload := map[string]any{
		"original_question":  strings.TrimSpace(input.OriginalQuestion),
		"agent_query":        strings.TrimSpace(input.AgentQuery),
		"clarifier_decision": input.Decision,
		"answer":             strings.TrimSpace(input.Answer),
		"revision_count":     input.RevisionCount,
		"review_instruction": fmt.Sprintf("Revision count: %d. Return a valid JSON object with verdict.", input.RevisionCount),
	}
	return marshalReviewPrompt(payload)
}

func marshalReviewPrompt(payload map[string]any) string {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", payload)
	}
	return string(data)
}
