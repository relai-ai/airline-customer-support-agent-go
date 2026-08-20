package airline

import (
	"context"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// AgentName is the display name of the demo agent.
const AgentName = "SkyServe Airline Support"

// DefaultCustomerServiceSignoff is the standard closing for customer-facing replies.
const DefaultCustomerServiceSignoff = "Please let me know if you have any questions."

// AirlineAgentInstructions is the system prompt for the support agent.
const AirlineAgentInstructions = "You are a concise airline customer support agent for a demo airline. " +
	"Help with booking lookups, baggage policy, seat changes, and flight-change guidance. " +
	"Before changing seats or discussing a specific booking, ask for and verify the confirmation code. " +
	"Do not invent refund amounts, flight availability, fees, or policy exceptions. " +
	"Answer naturally and keep responses brief. " +
	"End every customer-facing response naturally with \"Please let me know if you have any questions.\""

// runner runs one agent response (a full tool-call loop) for a provider,
// streaming assistant text to out and returning the final assistant text.
type runner interface {
	run(ctx context.Context, instructions string, tools []Tool, history []Message, out io.Writer) (string, error)
}

// RunnerFunc adapts a plain function to the runner interface, letting callers
// inject a deterministic agent turn (e.g. for local simulation) without a
// model provider.
type RunnerFunc func(ctx context.Context, instructions string, tools []Tool, history []Message, out io.Writer) (string, error)

func (f RunnerFunc) run(ctx context.Context, instructions string, tools []Tool, history []Message, out io.Writer) (string, error) {
	return f(ctx, instructions, tools, history, out)
}

// Agent is a terminal airline support agent bound to a model provider.
type Agent struct {
	Name         string
	Instructions string
	Signoff      string
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
		Signoff:      DefaultCustomerServiceSignoff,
		Tools:        airlineTools(),
		runner:       r,
	}, nil
}

// NewAgentWithRunnerFunc builds the airline support agent with run in place
// of a model-provider runner, for deterministic construction without
// provider calls.
func NewAgentWithRunnerFunc(run RunnerFunc) *Agent {
	return &Agent{
		Name:         AgentName,
		Instructions: AirlineAgentInstructions,
		Signoff:      DefaultCustomerServiceSignoff,
		Tools:        airlineTools(),
		runner:       run,
	}
}

// StreamResponse runs a single agent turn over the conversation history,
// streaming assistant text deltas to out and returning the final text.
func (a *Agent) StreamResponse(ctx context.Context, history []Message, out io.Writer) (string, error) {
	response, err := a.runner.run(ctx, a.Instructions, a.Tools, history, out)
	if err != nil || strings.TrimSpace(response) == "" || strings.TrimSpace(a.Signoff) == "" {
		return response, err
	}
	if hasEquivalentClosing(response, a.Signoff) {
		return response, nil
	}

	finisher := "\n\n" + a.Signoff
	if _, err := io.WriteString(out, finisher); err != nil {
		return response, err
	}
	return response + finisher, nil
}

func hasEquivalentClosing(response, closing string) bool {
	responseWords := normalizedWords(response)
	closingWords := normalizedWords(closing)
	if len(closingWords) == 0 || len(responseWords) < len(closingWords) {
		return false
	}

	offset := len(responseWords) - len(closingWords)
	for i, word := range closingWords {
		if responseWords[offset+i] != word {
			return false
		}
	}
	return true
}

func normalizedWords(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}
