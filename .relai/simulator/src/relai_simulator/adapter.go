package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/relai-ai/airline-customer-support-agent-go/airline"
	relai "github.com/relai-ai/relai-cli/sdk/go"
)

type generatedAdapter struct{}

func BuildAgentAdapter(_ *relai.AdapterRuntime) any {
	return generatedAdapter{}
}

func (generatedAdapter) AdapterCapabilities() []relai.AdapterCapability {
	return []relai.AdapterCapability{relai.AdapterCapabilityRunTurn}
}

func (generatedAdapter) RunTurn(ctx context.Context, input relai.JSONValue) (relai.AgentTurnResult, error) {
	userTurn, ok := input.(string)
	if !ok {
		return relai.AgentTurnResult{}, fmt.Errorf("airline simulator input must be a string, got %T", input)
	}
	if strings.TrimSpace(userTurn) == "" {
		return relai.AgentTurnResult{}, fmt.Errorf("airline simulator input must not be empty")
	}

	if hasLiveProviderCredentials() {
		return runLiveTurn(ctx, userTurn)
	}

	assistant, err := runOfflineValidationTurn(userTurn)
	if err != nil {
		return relai.AgentTurnResult{}, err
	}

	return relai.AgentTurnResult{
		AssistantMessage: assistant,
		Metadata: map[string]any{
			"mode": "offline_validation_fallback",
		},
	}, nil
}

func hasLiveProviderCredentials() bool {
	return strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) != "" || strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")) != ""
}

func runLiveTurn(ctx context.Context, userTurn string) (relai.AgentTurnResult, error) {
	agent, err := airline.NewAgent()
	if err != nil {
		return relai.AgentTurnResult{}, err
	}

	assistant, err := agent.StreamResponse(ctx, []airline.Message{{Role: "user", Content: userTurn}}, io.Discard)
	if err != nil {
		return relai.AgentTurnResult{}, err
	}

	return relai.AgentTurnResult{
		AssistantMessage: assistant,
		Metadata: map[string]any{
			"mode": "live_provider",
		},
	}, nil
}

func runOfflineValidationTurn(userTurn string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(userTurn))

	if strings.Contains(normalized, "baggage") || strings.Contains(normalized, "carry-on") || strings.Contains(normalized, "checked bag") || strings.Contains(normalized, "checked baggage") {
		ticketType := "standard"
		switch {
		case strings.Contains(normalized, "basic"):
			ticketType = "basic"
		case strings.Contains(normalized, "business"), strings.Contains(normalized, "first"), strings.Contains(normalized, "premium"):
			ticketType = "business"
		}
		return airline.BaggagePolicyText(ticketType), nil
	}

	return "", fmt.Errorf("set OPENAI_API_KEY or ANTHROPIC_API_KEY for live turns; without provider credentials this simulator only supports baggage-policy validation prompts")
}
