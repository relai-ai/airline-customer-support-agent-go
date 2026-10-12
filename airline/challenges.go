package airline

import "fmt"

// ApplyDemoChallenge changes only the newly constructed agent's tools. The
// optimized source checkpoint and normal agent behavior remain untouched.
// Fixtures are synthetic and must be disclosed in the playground UI.
func ApplyDemoChallenge(agent *Agent, id string) error {
	switch id {
	case "", "unknown-fare", "invalid-seat":
		return nil // New requirements; no artificial behavioral regression.
	case "booking-outage":
		for i := range agent.Tools {
			if agent.Tools[i].Name == "lookup_booking" {
				agent.Tools[i].Execute = func(_ map[string]any) string {
					return `{"status":"unavailable","code":"SIMULATED_UPSTREAM_TIMEOUT","retryable":false,"message":"Synthetic training fixture: booking lookup is unavailable. No booking data was returned."}`
				}
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown demo challenge %q", id)
	}
}
