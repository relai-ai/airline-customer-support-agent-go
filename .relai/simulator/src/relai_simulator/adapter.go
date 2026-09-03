package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/relai-ai/airline-customer-support-agent-go/airline"
	relai "github.com/relai-ai/relai-cli/sdk/go"
)

type airlineAdapter struct {
	agent   *airline.Agent
	history []airline.Message
	initErr error
}

// BuildAgentAdapter constructs the application's exported airline agent. The
// existing application does not expose construction-time tool injection, so
// this adapter supports only direct agent turns.
func BuildAgentAdapter(_ *relai.AdapterRuntime) any {
	agent, err := airline.NewAgent()
	return &airlineAdapter{agent: agent, initErr: err}
}

func (a *airlineAdapter) AdapterCapabilities() []relai.AdapterCapability {
	return []relai.AdapterCapability{relai.AdapterCapabilityRunTurn}
}

func (a *airlineAdapter) RunTurn(ctx context.Context, input relai.JSONValue) (relai.AgentTurnResult, error) {
	if a.initErr != nil {
		return relai.AgentTurnResult{}, fmt.Errorf("construct airline agent: %w", a.initErr)
	}

	text, ok := input.(string)
	if !ok {
		return relai.AgentTurnResult{}, fmt.Errorf("airline agent input must be a string")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return relai.AgentTurnResult{}, fmt.Errorf("airline agent input must not be empty")
	}

	a.history = append(a.history, airline.Message{Role: "user", Content: text})
	response, err := a.agent.StreamResponse(ctx, a.history, &strings.Builder{})
	if err != nil {
		return relai.AgentTurnResult{}, err
	}
	a.history = append(a.history, airline.Message{Role: "assistant", Content: response})
	return relai.AgentTurnResult{AssistantMessage: response}, nil
}
