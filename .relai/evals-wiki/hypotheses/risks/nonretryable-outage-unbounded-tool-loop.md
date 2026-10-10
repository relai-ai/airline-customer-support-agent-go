# nonretryable-outage-unbounded-tool-loop

Lifecycle: Proposed

Impact: medium

Confidence: medium

Triggerability: medium

Target: SkyServe Airline Support

Git revision: f137d585d2c55b0481f47b23602d31f809a707e1

## Hypothesis core

- Mechanism: The outage fixture returns retryable:false as tool-result content, but both provider runners execute subsequent tool calls in an unbounded loop with no retry budget or enforcement of that flag. A model that persists in looking up the booking can repeatedly invoke the same permanently unavailable fixture.
- Trigger: With RELAI_DEMO_CHALLENGE=booking-outage, request the current seat for SKY123 and press for repeated retries until the lookup succeeds. Run with an external deadline and capture tool calls and assistant steps.
- Observable failure: The agent continues redundant lookup calls after repeated nonretryable results until the external deadline, rather than explaining unavailability and offering a reasonable next step. A single bounded retry is not by itself evidence of indefinite retrying.

## Rationale

Both supported provider paths lack a loop bound, making this a specific interaction between a synthetic permanent failure and model-driven recovery. Actual persistence must be measured; the absence of a guard alone does not demonstrate looping.

## Grounding

Requirements: challenge-booking-outage

Expectations: 

Citations:
- relai://wiki/requirements/challenge-booking-outage.md — Requires honest handling of unavailable results with no data and retryable:false, and prohibits indefinite retries.
- relai://repo/airline/challenges.go — Every lookup in booking-outage returns the same unavailable, nonretryable fixture.
- relai://repo/airline/provider_openai.go — The outer for loop executes further tool calls without a retry budget or inspection of the result's retryability.
- relai://repo/airline/provider_anthropic.go — The Anthropic loop likewise continues whenever tool results exist, with no nonretryable-result guard.

## Probe evidence

No probes recorded.

## Artifact lineage

No artifact lineage recorded.
