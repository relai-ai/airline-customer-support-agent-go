package airline

import (
	"strings"
	"testing"
)

// These guard against a model omitting a field or sending null for a tool
// argument: the tools must return a graceful response, never panic.
func TestLookupBookingText(t *testing.T) {
	for _, code := range []string{"", "   "} {
		if got := LookupBookingText(code); !strings.Contains(strings.ToLower(got), "confirmation code") {
			t.Errorf("LookupBookingText(%q) = %q, want a prompt for a code", code, got)
		}
	}
	if got := LookupBookingText("ZZZ999"); !strings.Contains(strings.ToLower(got), "no booking was found") {
		t.Errorf("LookupBookingText(ZZZ999) = %q, want not-found", got)
	}
	got := LookupBookingText(" sky123 ")
	if !strings.Contains(got, "Maya Chen") || !strings.Contains(got, "JFK to SFO") {
		t.Errorf("LookupBookingText(sky123) = %q, want Maya Chen / JFK to SFO", got)
	}
}

func TestBaggagePolicyText(t *testing.T) {
	if got := BaggagePolicyText(""); !strings.Contains(strings.ToLower(got), "standard tickets") {
		t.Errorf("BaggagePolicyText(\"\") = %q, want standard default", got)
	}
	if got := BaggagePolicyText("basic"); !strings.Contains(strings.ToLower(got), "basic economy") {
		t.Errorf("BaggagePolicyText(basic) = %q, want basic economy", got)
	}
	if got := BaggagePolicyText("BUSINESS"); !strings.Contains(strings.ToLower(got), "premium cabins") {
		t.Errorf("BaggagePolicyText(BUSINESS) = %q, want premium cabins", got)
	}
}

func TestChangeSeatText(t *testing.T) {
	if got := ChangeSeatText("", ""); !strings.Contains(strings.ToLower(got), "valid confirmation code") {
		t.Errorf("ChangeSeatText(empty) = %q, want valid-code message", got)
	}
	if got := ChangeSeatText("ZZZ999", "14C"); !strings.Contains(strings.ToLower(got), "valid confirmation code") {
		t.Errorf("ChangeSeatText(ZZZ999) = %q, want valid-code message", got)
	}
	if got := ChangeSeatText("SKY123", ""); !strings.Contains(strings.ToLower(got), "which seat") {
		t.Errorf("ChangeSeatText(no seat) = %q, want which-seat prompt", got)
	}
	if got := ChangeSeatText(" bay777 ", " 20f "); got != "Seat updated to 20F for booking BAY777." {
		t.Errorf("ChangeSeatText update = %q", got)
	}
}

func TestArgTextHandlesMissingAndNonString(t *testing.T) {
	args := map[string]any{"a": "  hi  ", "b": nil, "c": 42}
	if got := argText(args, "a"); got != "hi" {
		t.Errorf("argText a = %q, want hi", got)
	}
	if got := argText(args, "b"); got != "" {
		t.Errorf("argText nil = %q, want empty", got)
	}
	if got := argText(args, "c"); got != "" {
		t.Errorf("argText non-string = %q, want empty", got)
	}
	if got := argText(args, "missing"); got != "" {
		t.Errorf("argText missing = %q, want empty", got)
	}
}
