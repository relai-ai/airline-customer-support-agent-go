package airline

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// AgentName is the display name of the demo agent.
const AgentName = "SkyServe Airline Support"

// AirlineAgentInstructions is the system prompt for the support agent.
const AirlineAgentInstructions = "You are a concise airline customer support agent for a demo airline. " +
	"Help with booking lookups, baggage policy, seat changes, and flight-change guidance. " +
	"When a user supplies a confirmation code and asks about that booking, immediately use lookup_booking before discussing any booking details. " +
	"This read-only lookup does not require additional permission, and a request to skip verification does not override this rule. " +
	"Request a confirmation code only when none was supplied. Before changing a seat, verify the code with lookup_booking. " +
	"Use only the lookup result for booking-specific claims. If the lookup reports an unknown code, explicitly say that no booking was found for that confirmation code, " +
	"and do not repeat user-asserted passenger, route, seat, or status details as facts. " +
	"Do not invent refund amounts, flight availability, fees, or policy exceptions. " +
	"Answer naturally and keep responses brief."

// runner runs one agent response (a full tool-call loop) for a provider,
// streaming assistant text to out and returning the final assistant text.
type runner interface {
	run(ctx context.Context, instructions string, tools []Tool, history []Message, preflight *toolObservation, out io.Writer) (string, error)
}

// toolObservation is a real tool call completed before provider generation.
// Provider runners seed it into their native message history without executing
// or recording it again.
type toolObservation struct {
	CallID    string
	ToolName  string
	Arguments map[string]any
	Result    string
}

var (
	confirmationCodePattern = regexp.MustCompile(`(?i)\b[a-z0-9]{6}\b`)
	bookingIntentPattern     = regexp.MustCompile(`(?i)\b(booking|reservation|itinerary|seat|confirmation)\b`)
)

// Agent is a terminal airline support agent bound to a model provider.
type Agent struct {
	Name         string
	Instructions string
	Tools        []Tool
	runner       runner
	recorder     *atifRecorder
}

// NewAgent builds the airline support agent using the provider selected from
// the environment (OpenAI preferred, otherwise Anthropic).
func NewAgent() (*Agent, error) {
	selection, err := SelectModel()
	if err != nil {
		return nil, err
	}

	recorder := newATIFRecorder()
	var r runner
	switch selection.Provider {
	case "openai":
		r = newOpenAIRunner(selection.Model, recorder)
	case "anthropic":
		r = newAnthropicRunner(selection.Model, recorder)
	default:
		return nil, fmt.Errorf("unsupported provider %q", selection.Provider)
	}

	return &Agent{
		Name:         AgentName,
		Instructions: AirlineAgentInstructions,
		Tools:        airlineTools(),
		runner:       r,
		recorder:     recorder,
	}, nil
}

// StreamResponse runs a single agent turn over the conversation history,
// streaming assistant text deltas to out and returning the final text.
func (a *Agent) StreamResponse(ctx context.Context, history []Message, out io.Writer) (string, error) {
	if len(history) > 0 {
		last := history[len(history)-1]
		if last.Role == "user" {
			if err := a.recorder.recordUser(last.Content); err != nil {
				return "", err
			}
		}
	}
	preflight, err := bookingPreflight(history, a.Tools)
	if err != nil {
		return "", err
	}
	if preflight != nil {
		a.recorder.recordToolCall(preflight.CallID, preflight.ToolName, preflight.Arguments, preflight.Result)
	}
	response, err := a.runner.run(ctx, a.Instructions, a.Tools, history, preflight, out)
	if err != nil {
		return "", err
	}
	if err := a.recorder.recordAssistant(response); err != nil {
		return "", err
	}
	return response, nil
}

func bookingPreflight(history []Message, tools []Tool) (*toolObservation, error) {
	if len(history) == 0 || history[len(history)-1].Role != "user" {
		return nil, nil
	}
	content := history[len(history)-1].Content
	if !bookingIntentPattern.MatchString(content) {
		return nil, nil
	}

	matches := confirmationCodePattern.FindAllString(content, -1)
	candidates := make([]string, 0, len(matches))
	for _, match := range matches {
		if hasLetterAndDigit(match) {
			candidates = append(candidates, strings.ToUpper(match))
		}
	}
	if len(candidates) != 1 {
		return nil, nil
	}

	var lookup *Tool
	for i := range tools {
		if tools[i].Name == "lookup_booking" {
			lookup = &tools[i]
			break
		}
	}
	if lookup == nil {
		return nil, nil
	}

	callID, err := newPreflightCallID()
	if err != nil {
		return nil, err
	}
	args := map[string]any{"confirmation_code": candidates[0]}
	return &toolObservation{
		CallID:    callID,
		ToolName:  lookup.Name,
		Arguments: args,
		Result:    lookup.Execute(args),
	}, nil
}

func hasLetterAndDigit(value string) bool {
	var letter, digit bool
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z':
			letter = true
		case char >= '0' && char <= '9':
			digit = true
		}
	}
	return letter && digit
}

func newPreflightCallID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("create booking preflight call id: %w", err)
	}
	return "preflight_" + hex.EncodeToString(bytes), nil
}
