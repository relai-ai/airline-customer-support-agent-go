package airline

import (
	"reflect"
	"strings"
	"testing"
)

func TestNewAgentIncludesCanonicalResponsePolicy(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	agent, err := NewAgent()
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}

	const canonicalPolicy = "Every customer-facing final response must end with this exact final sentence: \"Please let me know if you have any questions.\""
	if !strings.Contains(agent.Instructions, canonicalPolicy) {
		t.Fatalf("NewAgent() instructions do not include canonical response policy: %q", agent.Instructions)
	}

	if _, ok := agent.runner.(*openaiRunner); !ok {
		t.Fatalf("NewAgent() runner = %T, want *openaiRunner", agent.runner)
	}

	var toolNames []string
	for _, tool := range agent.Tools {
		toolNames = append(toolNames, tool.Name)
	}
	wantToolNames := []string{"lookup_booking", "baggage_policy", "change_seat"}
	if !reflect.DeepEqual(toolNames, wantToolNames) {
		t.Fatalf("NewAgent() tools = %v, want %v", toolNames, wantToolNames)
	}
}