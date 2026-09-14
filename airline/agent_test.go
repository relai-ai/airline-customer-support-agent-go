package airline

import (
	"strings"
	"testing"
)

func TestNewAgentUsesStandardClosingPolicy(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("ANTHROPIC_API_KEY", "")

	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}

	if agent.Instructions != AirlineAgentInstructions {
		t.Fatal("NewAgent() did not use AirlineAgentInstructions")
	}
	const closingPolicy = `Every final customer-facing answer must end with exactly "Please let me know if you have any questions."`
	if !strings.Contains(agent.Instructions, closingPolicy) {
		t.Fatalf("NewAgent() instructions do not contain the standard closing policy: %q", agent.Instructions)
	}
}