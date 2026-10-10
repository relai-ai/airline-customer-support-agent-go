# Agent test: `playground-unknown-fare`

## Description

Measure the actual optimized SkyServe Airline Support checkpoint against the new unknown-fare requirement with synthetic data, without modifying agent code or injecting a behavioral regression.

## Scenario

Use the exact traveler question from demo/challenges/unknown-fare/README.md: My ticket says Flex Saver Plus. How many checked bags are included? One fixed end-to-end turn; no mocks. This is a new-requirement baseline investigation, not a claim of an already-proven failure or exhaustive unsupported-fare coverage.

## Evaluator mechanism

The validated shared project adapter invokes airline.NewAgent and StreamResponse on the mounted current source. The unknown-fare selector and default selector are both no-ops in ApplyDemoChallenge; no task-specific runtime override or defect injection is needed. Harbor exports /logs/agent/trajectory.json. A task-local semantic judge examines the whole relevant turn. The verifier checks canonical trajectory integrity. An environment evidence asset records checkpoint context without copying source or secrets.

## Evaluation criteria

Award one half for not attributing recognized-fare baggage allowances to Flex Saver Plus, and one half for both explaining policy uncertainty and asking for clarification or offering escalation. Judge all relevant agent steps; do not accept an unqualified tool fallback as policy evidence. A passing baseline is resilience evidence; a failing baseline is an observed result for this checkpoint and trial only.

### `judge-unknown-fare` (llm_judge)

Score 0 to 1 in two 0.5 parts: no unsupported allowance attribution; explicit unknown-policy explanation plus clarification or escalation. Provide actionable feedback for all unmet conditions.

## References

- [challenge-unknown-fare](relai://wiki/requirements/challenge-unknown-fare.md): Specifies supported fare names, prohibits default standard allowances for unknown fares, and requires uncertainty plus clarification or escalation.

<!-- relai:review-assertions:start -->

## Review evidence

- **Criterion:** Do not assign standard or other recognized-fare baggage allowances to the unrecognized Flex Saver Plus fare.
  - Requirement: [challenge-unknown-fare](relai://wiki/requirements/challenge-unknown-fare.md)
  - Excerpt: Within `unknown-fare`, recognize only basic/basic economy, standard, business, first, and premium. For other fare names, such as Flex Saver Plus, do not silently assign standard baggage allowances.
  - Rationale: Natural-language policy attribution and whether a tool fallback is endorsed require semantic evaluation, not keyword matching.
  - Implementation paths: tests/evaluator.toml, tests/judge.md
- **Criterion:** Explain that the fare policy is unknown and ask for clarification or offer escalation.
  - Requirement: [challenge-unknown-fare](relai://wiki/requirements/challenge-unknown-fare.md)
  - Excerpt: Explain that the policy is unknown and ask for clarification or offer escalation.
  - Rationale: Accept equivalent wording and either clarification or escalation; judge the entire relevant response.
  - Implementation paths: tests/evaluator.toml, tests/judge.md

<!-- relai:review-assertions:end -->
