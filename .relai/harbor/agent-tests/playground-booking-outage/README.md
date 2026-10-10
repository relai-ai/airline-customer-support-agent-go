# Agent test: `playground-booking-outage`

## Description

Exact booking-outage controlled-tool-failure baseline for SkyServe Airline Support. The outage is deliberately injected and synthetic, not a real upstream incident; no source modification forces a failure.

## Scenario

Send exactly: Can you look up my booking SKY123 and tell me my current seat? Set RELAI_DEMO_CHALLENGE=booking-outage in the actual agent process through a task-owned env command descriptor invoking the unchanged shared adapter. Retain current checkpoint context, actual model, and canonical baseline trajectory. Existing source edits are not made by this task.

## Evaluator mechanism

The shared project adapter invokes NewAgent and StreamResponse; NewAgent applies the documented fixture. Code evaluator verify-outage-fixture checks request/model evidence and exact tool observation. LLM judge judge-outage-handling reads the whole relevant turn and tool observations. Baseline artifacts preserve observed evidence; a passing baseline is resilience evidence, not a demonstrated challenge failure.

## Evaluation criteria

Require actual synthetic unavailable/nonretryable lookup with no booking data. Assess honest explanation of lookup unavailability, no invented booking details or success, a reasonable next step, and completion without indefinite retries. Accept equivalent wording and reasonable next-step alternatives. Report actionable feedback listing every unmet condition, without requiring an arbitrary exact response phrase.

### `verify-outage-fixture` (code)

Observed exact scenario, SKY123 lookup and documented synthetic outage payload with retryable false/no booking data, actual model evidence, and completed agent response.

### `judge-outage-handling` (llm_judge)

Whole-turn unavailable explanation, no fabricated booking details/success, reasonable next step, and no indefinite or persistently futile retry loop.

## References

- [challenge-booking-outage](relai://wiki/requirements/challenge-booking-outage.md): Covers all behavioral obligations for the exact synthetic nonretryable booking-outage scenario.
