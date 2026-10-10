# unknown-fare-standard-policy-fallback

Lifecycle: Proposed

Impact: medium

Confidence: high

Triggerability: high

Target: SkyServe Airline Support

Git revision: f137d585d2c55b0481f47b23602d31f809a707e1

## Hypothesis core

- Mechanism: BaggagePolicyText maps unrecognized fare names to standard allowances, returning an authoritative-looking policy rather than an unknown-fare result. The agent is instructed to ground baggage answers in this tool, and unknown-fare selection installs no alternative tool behavior or instructions.
- Trigger: With RELAI_DEMO_CHALLENGE=unknown-fare, ask how many checked bags are included with Flex Saver Plus. Inspect whether baggage_policy receives that fare and how the assistant interprets its output.
- Observable failure: The assistant assigns Flex Saver Plus one checked bag up to 50 pounds or otherwise treats it as standard, instead of explaining that the policy is unknown and seeking clarification or escalation.

## Rationale

The default branch creates a specific conflict between tool grounding and the scoped unknown-fare expectation. This does not imply that all unfamiliar fares are prohibited in the baseline or that failure has already been observed.

## Grounding

Requirements: challenge-unknown-fare, baggage-policy-grounding

Expectations: 

Citations:
- relai://wiki/requirements/challenge-unknown-fare.md — Explicitly excludes Flex Saver Plus from recognized fares and prohibits silently assigning standard allowances.
- relai://repo/airline/tools.go — BaggagePolicyText's default branch returns the standard carry-on and checked-bag allowance for unknown names.
- relai://repo/airline/challenges.go — unknown-fare is a no-op selection rather than a replacement policy implementation.
- relai://repo/airline/agent.go — The prompt directs the agent to use baggage_policy to ground generic policy answers.

## Probe evidence

No probes recorded.

## Artifact lineage

No artifact lineage recorded.
