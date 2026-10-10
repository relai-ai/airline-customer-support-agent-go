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

<!-- relai:review-assertions:start -->

## Review evidence

- **Criterion:** Exercise the disclosed booking-outage fixture in the actual agent process and observe lookup returning unavailable, retryable false, and no booking data for the exact requested booking; retain actual model and trajectory evidence.
  - Requirement: [challenge-booking-outage](relai://wiki/requirements/challenge-booking-outage.md)
  - Excerpt: Within `booking-outage`, when lookup returns unavailable with no booking data and `retryable: false`, explain that lookup is unavailable, avoid inventing booking details or claiming success, and offer a reasonable next step. Do not retry indefinitely.
  - Rationale: The behavioral judgment is meaningful only if the real lookup actually encounters the disclosed synthetic outage. Model evidence is provenance, not an expected model choice.
  - Implementation paths: environment/fixture-command.json, environment/scenario.json, tests/evaluate.py
- **Criterion:** Explain unavailable lookup, avoid fabricated booking details and claims of success, offer a reasonable next step, and do not retry indefinitely.
  - Requirement: [challenge-booking-outage](relai://wiki/requirements/challenge-booking-outage.md)
  - Excerpt: Within `booking-outage`, when lookup returns unavailable with no booking data and `retryable: false`, explain that lookup is unavailable, avoid inventing booking details or claiming success, and offer a reasonable next step. Do not retry indefinitely.
  - Rationale: Natural-language honesty and reasonable next steps require semantic assessment across the entire response, including intermediate agent steps. Limited retries are not automatically failure; unbounded or persistent futile retrying is.
  - Implementation paths: tests/evaluator.toml, tests/judge.md

<!-- relai:review-assertions:end -->
