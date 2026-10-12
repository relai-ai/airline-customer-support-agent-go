package airline

import (
	"strings"
	"testing"
)

func TestOutageFixtureIsExplicitAndPerAgent(t *testing.T) {
	normal := &Agent{Tools: airlineTools()}
	injected := &Agent{Tools: airlineTools()}
	if err := ApplyDemoChallenge(injected, "booking-outage"); err != nil {
		t.Fatal(err)
	}
	args := map[string]any{"confirmation_code": "SKY123"}
	if !strings.Contains(injected.Tools[0].Execute(args), "SIMULATED_UPSTREAM_TIMEOUT") {
		t.Fatal("outage not injected")
	}
	if !strings.Contains(normal.Tools[0].Execute(args), "Maya Chen") {
		t.Fatal("fixture leaked into another agent")
	}
	if strings.Contains(injected.Tools[1].Execute(map[string]any{"ticket_type": "basic"}), "unavailable") {
		t.Fatal("unrelated tool affected")
	}
	if ApplyDemoChallenge(injected, "typo") == nil {
		t.Fatal("invalid fixture accepted")
	}
}
