# Verify booking confirmation codes

Before changing a seat or discussing a specific booking, obtain and verify a confirmation code. A valid code already supplied in the request is sufficient; do not demand repetition or unnecessary extra information. An invalid code must not authorize a seat change.

Generic baggage-policy questions are exempt; see [baggage grounding](relai://wiki/requirements/baggage-policy-grounding.md).

## Evidence

[Shared prompt](relai://repo/airline/agent.go), [benchmark](relai://repo/benchmarks/airline_support_benchmark.csv), and [tool tests](relai://repo/airline/tools_test.go). The benchmark explicitly accepts SKY123 as verified for its seat-change scenario; no further identity-authentication process is established.
