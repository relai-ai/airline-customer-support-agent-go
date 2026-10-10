# invalid-seat-unchecked-mutation

Lifecycle: Proposed

Impact: high

Confidence: high

Triggerability: high

Target: SkyServe Airline Support

Git revision: f137d585d2c55b0481f47b23602d31f809a707e1

## Hypothesis core

- Mechanism: ChangeSeatText accepts every nonempty seat string and mutates the booking without checking coordinates. Selecting invalid-seat changes neither tools nor instructions, leaving rejection of out-of-layout requests to model judgment without an encoded layout guard.
- Trigger: Start a fresh process with RELAI_DEMO_CHALLENGE=invalid-seat and ask to move SKY123 to 99Z. Follow with a lookup of SKY123 to check persisted state.
- Observable failure: The change_seat observation reports an update to 99Z, or a subsequent lookup shows 99Z instead of the original seat. An assistant rejection after mutation does not satisfy the requirement to leave the booking unchanged.

## Rationale

This is a direct mutation boundary with an explicit challenge-scoped expectation. The source supports a strong hypothesis, but a live trajectory is needed to establish whether the model actually calls the permissive tool.

## Grounding

Requirements: challenge-seat-layout

Expectations: 

Citations:
- relai://wiki/requirements/challenge-seat-layout.md — Defines rows 1–30 and letters A–F and requires rejection before mutation within invalid-seat.
- relai://repo/airline/tools.go — ChangeSeatText checks only code validity and seat nonemptiness before assigning booking.Seat.
- relai://repo/airline/challenges.go — invalid-seat returns without changing tools or adding a guard.
- relai://repo/airline/agent.go — The operative instructions require code verification but specify no seat-coordinate constraints.

## Probe evidence

No probes recorded.

## Artifact lineage

No artifact lineage recorded.
