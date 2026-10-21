package airline

import (
	"context"
	"fmt"
	"io"
)

// AgentName is the display name of the demo agent.
const AgentName = "SkyServe Airline Support"

// AirlineAgentInstructions is the system prompt for the support agent.
const AirlineAgentInstructions = "You are a concise airline customer support agent for a demo airline. " +
	"Help with booking lookups, baggage policy, seat changes, and flight-change guidance. " +
	"Before changing seats or discussing a specific booking, ask for and verify the confirmation code. " +
	"Do not invent refund amounts, flight availability, fees, or policy exceptions. " +
	"Answer naturally and keep responses brief."

// runner runs one agent response (a full tool-call loop) for a provider,
// streaming assistant text to out and returning the final assistant text.
type runner interface {
	run(ctx context.Context, instructions string, tools []Tool, history []Message, out io.Writer) (string, error)
}

// Agent is a terminal airline support agent bound to a model provider.
type Agent struct {
	Name         string
	Instructions string
	Tools        []Tool
	runner       runner
	trace        *relaiTrace
}

// NewAgent builds the airline support agent using the provider selected from
// the environment (OpenAI preferred, otherwise Anthropic).
func NewAgent() (*Agent, error) {
	selection, err := SelectModel()
	if err != nil {
		return nil, err
	}

	var r runner
	switch selection.Provider {
	case "openai":
		r = newOpenAIRunner(selection.Model)
	case "anthropic":
		r = newAnthropicRunner(selection.Model)
	default:
		return nil, fmt.Errorf("unsupported provider %q", selection.Provider)
	}

	return &Agent{
		Name:         AgentName,
		Instructions: AirlineAgentInstructions,
		Tools:        airlineTools(),
		trace:        newRelaiTrace(),
		runner:       r,
	}, nil
}

// StreamResponse runs a single agent turn over the conversation history,
// streaming assistant text deltas to out and returning the final text.
func (a *Agent) StreamResponse(ctx context.Context, history []Message, out io.Writer) (string, error) {
	if a.trace != nil {
		if len(history) > 0 && history[len(history)-1].Role == "user" { a.trace.step("user", history[len(history)-1].Content, "") }
		ctx = context.WithValue(ctx, traceKey{}, a.trace)
	}
	return a.runner.run(ctx, a.Instructions, a.Tools, history, out)
}
