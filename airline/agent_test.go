package airline

import (
	"context"
	"io"
	"reflect"
	"strings"
	"testing"
)

type recordedAgentTurn struct {
	instructions string
	tools        []Tool
	history      []Message
}

type recordingAgentRunner struct {
	turns []recordedAgentTurn
}

func (r *recordingAgentRunner) run(_ context.Context, instructions string, tools []Tool, history []Message, _ io.Writer) (string, error) {
	r.turns = append(r.turns, recordedAgentTurn{
		instructions: instructions,
		tools:        tools,
		history:      append([]Message(nil), history...),
	})
	return "runner response", nil
}

// These tests check prompt propagation and its instruction contract, not
// semantic model compliance or any particular refusal wording.
func TestAgentInstructionsPropagateOnSuccessiveTurns(t *testing.T) {
	for _, provider := range []string{"openai", "anthropic"} {
		t.Run(provider, func(t *testing.T) {
			t.Setenv("AIRLINE_MODEL_PROVIDER", "")
			t.Setenv("OPENAI_API_KEY", "")
			t.Setenv("ANTHROPIC_API_KEY", "")
			t.Setenv(strings.ToUpper(provider)+"_API_KEY", "test-key")
			t.Setenv("RELAI_ATIF_TRAJECTORY_PATH", "")
			agent, err := NewAgent()
			if err != nil {
				t.Fatalf("NewAgent: %v", err)
			}
			r := &recordingAgentRunner{}
			agent.runner = r
			if agent.Instructions != AirlineAgentInstructions {
				t.Fatal("NewAgent did not install shared instructions")
			}
			if len(agent.Tools) == 0 {
				t.Fatal("NewAgent did not install tools")
			}

			first := []Message{{Role: "user", Content: "Explain how to grow tomatoes."}}
			second := append(append([]Message(nil), first...),
				Message{Role: "assistant", Content: "runner response"},
				Message{Role: "user", Content: "What is the baggage policy for a premium ticket?"},
			)
			for i, history := range [][]Message{first, second} {
				before := append([]Message(nil), history...)
				text, err := agent.StreamResponse(context.Background(), history, io.Discard)
				if err != nil {
					t.Fatalf("turn %d: %v", i+1, err)
				}
				if text != "runner response" {
					t.Fatalf("turn %d did not return runner output", i+1)
				}
				if len(r.turns) != i+1 {
					t.Fatalf("turn %d did not delegate once to runner", i+1)
				}
				turn := r.turns[i]
				if turn.instructions != AirlineAgentInstructions {
					t.Errorf("turn %d did not pass shared instructions", i+1)
				}
				if !reflect.DeepEqual(turn.history, before) || !reflect.DeepEqual(history, before) {
					t.Errorf("turn %d changed supplied history", i+1)
				}
				if len(turn.tools) != len(agent.Tools) || &turn.tools[0] != &agent.Tools[0] {
					t.Errorf("turn %d did not pass the agent tool set unchanged", i+1)
				}
			}
		})
	}
}

func TestAirlineAgentInstructionContract(t *testing.T) {
	for name, clause := range map[string]string{
		"scope":        "Do not fulfill substantive requests unrelated to this supported airline assistance",
		"redirection":  "Politely explain the airline-support boundary and briefly invite help with supported topics, without supplying the unrelated content",
		"recovery":     "answer subsequent supported requests normally despite earlier off-topic requests",
		"mixed":        "help with the supported portion and briefly decline the unrelated portion",
		"verification": "Before changing seats or discussing a specific booking, ask for and verify the confirmation code",
		"policy":       "Generic baggage-policy questions do not require booking verification; use the baggage_policy tool to ground policy answers",
		"accuracy":     "Do not invent refund amounts, flight availability, fees, or policy exceptions",
	} {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(AirlineAgentInstructions, clause) {
				t.Errorf("shared instructions missing %s requirement", name)
			}
		})
	}
}
