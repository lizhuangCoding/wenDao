package chatcore

import (
	"context"
	"fmt"
)

const defaultThinkTankPluginName = "thinktank"

type PluginManifest struct {
	Name         string
	Version      string
	DisplayName  string
	Description  string
	Capabilities []string
}

type AgentRunInput struct {
	Question       string
	ConversationID *int64
	UserID         *int64
}

type AgentResumeInput struct {
	ConversationID int64
	RunID          int64
	UserID         *int64
}

type AgentPlugin interface {
	Manifest() PluginManifest
	Run(ctx context.Context, input AgentRunInput) (*ThinkTankChatResponse, error)
	RunStream(ctx context.Context, input AgentRunInput) (<-chan StreamEvent, <-chan error)
	ResumeStream(ctx context.Context, input AgentResumeInput) (<-chan StreamEvent, <-chan error)
}

type thinkTankPlugin struct {
	service ThinkTankService
}

func NewThinkTankPlugin(service ThinkTankService) AgentPlugin {
	return &thinkTankPlugin{service: service}
}

func (p *thinkTankPlugin) Manifest() PluginManifest {
	return PluginManifest{
		Name:        defaultThinkTankPluginName,
		Version:     "1.0.0",
		DisplayName: "ThinkTank",
		Description: "Default multi-agent research and answer workflow.",
		Capabilities: []string{
			"chat",
			"stream",
			"resume",
			"multi_agent",
		},
	}
}

func (p *thinkTankPlugin) Run(ctx context.Context, input AgentRunInput) (*ThinkTankChatResponse, error) {
	if p == nil || p.service == nil {
		return nil, fmt.Errorf("thinktank plugin service is unavailable")
	}
	return p.service.Chat(ctx, input.Question, input.ConversationID, input.UserID)
}

func (p *thinkTankPlugin) RunStream(ctx context.Context, input AgentRunInput) (<-chan StreamEvent, <-chan error) {
	if p == nil || p.service == nil {
		return closedStreamError(fmt.Errorf("thinktank plugin service is unavailable"))
	}
	return p.service.ChatStream(ctx, input.Question, input.ConversationID, input.UserID)
}

func (p *thinkTankPlugin) ResumeStream(ctx context.Context, input AgentResumeInput) (<-chan StreamEvent, <-chan error) {
	if p == nil || p.service == nil {
		return closedStreamError(fmt.Errorf("thinktank plugin service is unavailable"))
	}
	return p.service.ResumeChatStream(ctx, input.ConversationID, input.RunID, input.UserID)
}

func closedStreamError(err error) (<-chan StreamEvent, <-chan error) {
	eventCh := make(chan StreamEvent)
	errCh := make(chan error, 1)
	errCh <- err
	close(eventCh)
	close(errCh)
	return eventCh, errCh
}
