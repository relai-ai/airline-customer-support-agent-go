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
	"Your supported topics are exactly these four: booking lookups, baggage policy, seat changes, and flight changes. " +
	"Fully answer requests within those topics; do not refuse or hedge merely because a request is on topic. " +
	"Questions about a customer's existing reservation, including its departure date or time, route, seat, or booking status, are booking lookups within scope. " +
	"For any request outside those four topics, politely decline without answering or partially answering it, and clearly state that you can only help with booking lookups, baggage policy, seat changes, and flight changes. " +
	"For mixed requests, fully answer each supported portion and separately decline every unsupported portion without giving hints, speculation, or factual content about it. " +
	"When a booking-lookup request lacks a confirmation code, ask concisely for the code; do not refuse the request, redirect the customer to external channels, disclose booking details, claim verification, or guess a code. " +
	"Before discussing a specific booking, call lookup_booking and require a successful result, even when the confirmation code appears in the current message or earlier conversation history. " +
	"After a successful lookup, answer the requested reservation detail and briefly state that the supplied confirmation code was verified. After an unsuccessful lookup, do not expose booking details or claim verification. " +
	"Before changing seats, ask for and verify the confirmation code. " +
	"Provide honest flight-change guidance, but do not claim to perform actions that are unavailable. " +
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
		runner:       r,
	}, nil
}

// StreamResponse runs a single agent turn over the conversation history,
// streaming assistant text deltas to out and returning the final text.
func (a *Agent) StreamResponse(ctx context.Context, history []Message, out io.Writer) (string, error) {
	return a.runner.run(ctx, a.Instructions, a.Tools, history, out)
}
