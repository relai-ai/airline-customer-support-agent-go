package airline

import (
	"fmt"
	"strings"
)

// Booking is a demo airline booking record.
type Booking struct {
	Passenger string
	Route     string
	Departure string
	Seat      string
	Status    string
}

// bookings is the in-memory demo data. change_seat mutates it in place.
var bookings = map[string]*Booking{
	"SKY123": {
		Passenger: "Maya Chen",
		Route:     "JFK to SFO",
		Departure: "2026-07-14 09:30",
		Seat:      "12A",
		Status:    "confirmed",
	},
	"BAY777": {
		Passenger: "Noah Patel",
		Route:     "LAX to SEA",
		Departure: "2026-07-18 16:45",
		Seat:      "18C",
		Status:    "confirmed",
	},
}

// Tool is a provider-neutral function tool the agent can call.
type Tool struct {
	Name        string
	Description string
	// Parameters is a JSON Schema object describing the tool arguments.
	Parameters map[string]any
	// Execute runs the tool with decoded JSON arguments and returns the
	// string result handed back to the model.
	Execute func(args map[string]any) string
}

// argText coerces a decoded JSON argument to a trimmed string.
//
// A model may omit a field or send null for a tool argument, so the decoded
// value may be missing or not a string. Normalizing here keeps the tools from
// panicking and lets them return a graceful response instead.
func argText(args map[string]any, key string) string {
	value, ok := args[key]
	if !ok {
		return ""
	}
	str, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(str)
}

// LookupBookingText returns a summary for a booking confirmation code.
func LookupBookingText(confirmationCode string) string {
	code := strings.ToUpper(strings.TrimSpace(confirmationCode))
	if code == "" {
		return "Please share a confirmation code so I can look up the booking."
	}
	booking, ok := bookings[code]
	if !ok {
		return "No booking was found for that confirmation code."
	}
	return fmt.Sprintf(
		"%s is booked on %s departing %s. Seat %s. Status: %s.",
		booking.Passenger, booking.Route, booking.Departure, booking.Seat, booking.Status,
	)
}

// BaggagePolicyText returns the demo baggage policy for a ticket type.
func BaggagePolicyText(ticketType string) string {
	normalized := strings.ToLower(strings.TrimSpace(ticketType))
	if normalized == "" {
		normalized = "standard"
	}
	switch normalized {
	case "basic", "basic economy":
		return "Basic economy includes one personal item. Carry-on bags and checked bags require a fee."
	case "business", "first", "premium":
		return "Premium cabins include one carry-on and two checked bags up to 70 pounds each."
	default:
		return "Standard tickets include one carry-on and one checked bag up to 50 pounds."
	}
}

// ChangeSeatText changes a seat when the confirmation code is valid.
func ChangeSeatText(confirmationCode, requestedSeat string) string {
	code := strings.ToUpper(strings.TrimSpace(confirmationCode))
	booking, ok := bookings[code]
	if code == "" || !ok {
		return "Seat changes require a valid confirmation code."
	}
	seat := strings.ToUpper(strings.TrimSpace(requestedSeat))
	if seat == "" {
		return "Please tell me which seat you would like."
	}
	booking.Seat = seat
	return fmt.Sprintf("Seat updated to %s for booking %s.", seat, code)
}

// airlineTools returns the tool set exposed to the model.
func airlineTools() []Tool {
	return []Tool{
		{
			Name:        "lookup_booking",
			Description: "Authoritative read-only source for verifying a demo booking by confirmation code. Use it before making booking-specific claims whenever a code is supplied; no additional user permission is required.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"confirmation_code": map[string]any{"type": "string"},
				},
				"required": []string{"confirmation_code"},
			},
			Execute: func(args map[string]any) string {
				return LookupBookingText(argText(args, "confirmation_code"))
			},
		},
		{
			Name:        "baggage_policy",
			Description: "Return the demo airline baggage policy for a ticket type.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ticket_type": map[string]any{"type": "string"},
				},
			},
			Execute: func(args map[string]any) string {
				return BaggagePolicyText(argText(args, "ticket_type"))
			},
		},
		{
			Name:        "change_seat",
			Description: "Change a seat for a demo booking when the confirmation code is valid.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"confirmation_code": map[string]any{"type": "string"},
					"requested_seat":    map[string]any{"type": "string"},
				},
				"required": []string{"confirmation_code", "requested_seat"},
			},
			Execute: func(args map[string]any) string {
				return ChangeSeatText(argText(args, "confirmation_code"), argText(args, "requested_seat"))
			},
		},
	}
}
