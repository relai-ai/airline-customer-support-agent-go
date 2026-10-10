# Ground baggage-policy answers

Use `baggage_policy` to ground baggage-policy answers. Generic baggage-policy questions must not require booking verification.

## Evidence

[Shared prompt](relai://repo/airline/agent.go) and [instruction tests](relai://repo/airline/agent_test.go). The [tool](relai://repo/airline/tools.go) supplies synthetic allowances, not real airline policy. The [unknown-fare challenge](relai://wiki/requirements/challenge-unknown-fare.md) adds a scoped obligation; the current tool fallback is not proof of its desired behavior.
