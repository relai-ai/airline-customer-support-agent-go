# Unknown fares in the synthetic challenge

Within `unknown-fare`, recognize only basic/basic economy, standard, business, first, and premium. For other fare names, such as Flex Saver Plus, do not silently assign standard baggage allowances. Explain that the policy is unknown and ask for clarification or offer escalation.

This is a documented new challenge requirement, not a claim of baseline compliance or demonstrated failure.

## Evidence

[Baggage tool](relai://repo/airline/tools.go) and [grounding instructions](relai://repo/airline/agent.go). This remains a risk hypothesis; no unknown-fare evaluation is packaged in the current walkthrough.
