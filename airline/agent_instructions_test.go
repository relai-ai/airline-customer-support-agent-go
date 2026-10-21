package airline

import (
	"context"
	"io"
	"reflect"
	"testing"
)

// Records the StreamResponse boundary without credentials or model calls.
// These are wiring tests, not evidence that a model follows the instructions.
type instructionRecordingRunner struct {
	instructions string
	tools        []Tool
	history      []Message
	calls        int
}

func (r *instructionRecordingRunner) run(_ context.Context, instructions string, tools []Tool, history []Message, _ io.Writer) (string, error) {
	r.instructions = instructions
	r.tools = tools
	r.history = history
	r.calls++
	return "", nil
}

func TestStreamResponseForwardsAirlineInstructions(t *testing.T) {
	r := &instructionRecordingRunner{}
	agent := &Agent{
		Instructions: AirlineAgentInstructions,
		Tools:        airlineTools(),
		runner:       r,
	}
	history := []Message{{Role: "user", Content: "Please look up my booking, code TRV456."}}
	for turn := 0; turn < 2; turn++ {
		if _, err := agent.StreamResponse(context.Background(), history, io.Discard); err != nil {
			t.Fatalf("StreamResponse: %v", err)
		}
		if r.instructions != AirlineAgentInstructions {
			t.Fatal("runner did not receive the shared instructions unchanged")
		}
		if !reflect.DeepEqual(r.history, history) {
			t.Fatal("runner did not receive the conversation history unchanged")
		}
		history = append(history,
			Message{Role: "assistant", Content: "I need to verify that booking."},
			Message{Role: "user", Content: "Use a different booking, ABC789, for the seat change."},
		)
	}
	if r.calls != 2 {
		t.Fatalf("runner calls = %d, want 2 on the same agent", r.calls)
	}
}

func TestStreamResponseForwardsAirlineToolDefinitions(t *testing.T) {
	r := &instructionRecordingRunner{}
	tools := airlineTools()
	agent := &Agent{
		Instructions: AirlineAgentInstructions,
		Tools:        tools,
		runner:       r,
	}
	if _, err := agent.StreamResponse(context.Background(), []Message{
		{Role: "user", Content: "What is the baggage policy?"},
	}, io.Discard); err != nil {
		t.Fatalf("StreamResponse: %v", err)
	}
	if len(r.tools) != len(tools) || len(r.tools) != 3 {
		t.Fatalf("runner received %d tools, want the existing three", len(r.tools))
	}
	for i, name := range []string{"lookup_booking", "baggage_policy", "change_seat"} {
		// Identity preserves descriptions, schemas and Execute functions.
		if &r.tools[i] != &tools[i] || r.tools[i].Name != name {
			t.Fatalf("tool %d was replaced or reordered", i)
		}
	}
	if r.tools[0].Description != "Verify a supplied confirmation code for the user's currently intended demo booking before booking-specific assistance. Returns booking details or a not-found result." {
		t.Error("lookup tool must describe supplied-code verification")
	}
	if r.tools[2].Description != "Change a seat only for the user's currently intended demo booking after lookup_booking successfully verifies its confirmation code. Use only for a requested seat change, not a read-only request." {
		t.Error("seat tool must describe verified current-target and write-intent requirements")
	}
}