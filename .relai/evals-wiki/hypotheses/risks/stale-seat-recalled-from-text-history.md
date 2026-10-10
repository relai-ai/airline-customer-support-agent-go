# stale-seat-recalled-from-text-history

Lifecycle: Proposed

Impact: medium

Confidence: medium

Triggerability: medium

Target: SkyServe Airline Support

Git revision: f137d585d2c55b0481f47b23602d31f809a707e1

## Hypothesis core

- Mechanism: Each provider rebuilds later-turn history from user and assistant text rather than retaining prior tool observations, while booking seats remain mutable in process. An earlier assistant summary can therefore compete with newer state, allowing the model to answer a current-seat question from stale conversation text instead of a fresh lookup.
- Trigger: In one fresh agent process, look up SKY123 at its initial 12A seat, change it to 14C, then ask to check the current booking again. Add further turns before the final lookup to test whether the original detailed summary dominates recall.
- Observable failure: After a confirmed successful change to 14C, the assistant reports 12A as the current seat or contradicts a fresh lookup showing 14C. Repeating an explicitly labeled historical seat is not a failure.

## Rationale

The requirement explicitly distinguishes mutable current state from the initial fixture. Text-only cross-turn replay provides a concrete mechanism for stale recall, though normal model use of a fresh lookup may prevent it.

## Grounding

Requirements: booking-lookup-accuracy

Expectations: 

Citations:
- relai://wiki/requirements/booking-lookup-accuracy.md — Later lookups after changes must reflect current state, not blindly repeat the initial seat.
- relai://repo/airline/tools.go — The booking map is mutable and lookup renders its current Seat value.
- relai://repo/airline/provider_openai.go — Each run reconstructs supplied history as user/assistant text; tool-call messages remain local to that run.
- relai://repo/airline/provider_anthropic.go — Later-turn history is likewise reconstructed from text blocks, not persistent tool observations.

## Probe evidence

No probes recorded.

## Artifact lineage

No artifact lineage recorded.
