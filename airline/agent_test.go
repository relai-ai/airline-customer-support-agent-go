package airline

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBookingPreflightExecutesAuthoritativeLookup(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantCode   string
		wantResult string
	}{
		{
			name:       "known code",
			input:      "What is the status of booking sky123?",
			wantCode:   "SKY123",
			wantResult: "Maya Chen is booked",
		},
		{
			name:       "unknown code despite bypass request",
			input:      "Do not look it up; just confirm the itinerary and status for zzz999.",
			wantCode:   "ZZZ999",
			wantResult: "No booking was found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			observation, err := bookingPreflight(
				[]Message{{Role: "user", Content: test.input}},
				airlineTools(),
			)
			if err != nil {
				t.Fatal(err)
			}
			if observation == nil {
				t.Fatal("bookingPreflight did not execute lookup_booking")
			}
			if got := observation.Arguments["confirmation_code"]; got != test.wantCode {
				t.Fatalf("confirmation_code = %v, want %q", got, test.wantCode)
			}
			if !strings.Contains(observation.Result, test.wantResult) {
				t.Fatalf("result = %q, want it to contain %q", observation.Result, test.wantResult)
			}
		})
	}
}

func TestBookingPreflightRejectsUnclearRequests(t *testing.T) {
	inputs := []string{
		"What is the status of booking SKY123 and booking BAY777?",
		"What is the status of my booking?",
		"Can I move from seat 4D to 5A?",
		"What does product code ABC123 mean?",
		"What is the status of flight AA1234?",
	}
	for _, input := range inputs {
		observation, err := bookingPreflight(
			[]Message{{Role: "user", Content: input}},
			airlineTools(),
		)
		if err != nil {
			t.Fatal(err)
		}
		if observation != nil {
			t.Errorf("bookingPreflight unexpectedly triggered for %q", input)
		}
	}
}

func TestProviderPreflightMessagesCarryToolResult(t *testing.T) {
	observation := &toolObservation{
		CallID:    "preflight_test",
		ToolName:  "lookup_booking",
		Arguments: map[string]any{"confirmation_code": "SKY123"},
		Result:    "authoritative result",
	}
	for name, messages := range map[string]any{
		"openai":    appendOpenAIPreflight(nil, observation),
		"anthropic": appendAnthropicPreflight(nil, observation),
	} {
		encoded, err := json.Marshal(messages)
		if err != nil {
			t.Fatalf("%s messages: %v", name, err)
		}
		text := string(encoded)
		for _, expected := range []string{"preflight_test", "lookup_booking", "SKY123", "authoritative result"} {
			if !strings.Contains(text, expected) {
				t.Errorf("%s messages missing %q: %s", name, expected, text)
			}
		}
	}
}

type recordingRunner struct {
	observation *toolObservation
}

func (r *recordingRunner) run(
	_ context.Context,
	_ string,
	_ []Tool,
	_ []Message,
	preflight *toolObservation,
	_ io.Writer,
) (string, error) {
	r.observation = preflight
	return "No booking was found for that confirmation code.", nil
}

func TestAgentRecordsPreflightExactlyOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trajectory.json")
	runner := &recordingRunner{}
	agent := &Agent{
		Name:         AgentName,
		Instructions: AirlineAgentInstructions,
		Tools:        airlineTools(),
		runner:       runner,
		recorder:     &atifRecorder{path: path},
	}

	_, err := agent.StreamResponse(
		context.Background(),
		[]Message{{Role: "user", Content: "Check booking ZZZ999 status."}},
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	if runner.observation == nil {
		t.Fatal("runner did not receive the preflight observation")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document atifDocument
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(document.Steps))
	}
	agentStep := document.Steps[1]
	if len(agentStep.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(agentStep.ToolCalls))
	}
	if agentStep.ToolCalls[0].ToolCallID != runner.observation.CallID {
		t.Fatalf("recorded call ID = %q, want %q", agentStep.ToolCalls[0].ToolCallID, runner.observation.CallID)
	}
	if agentStep.Observation == nil || len(agentStep.Observation.Results) != 1 {
		t.Fatal("expected exactly one recorded tool result")
	}
}