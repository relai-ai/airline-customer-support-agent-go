# Agent test: `airline-off-topic-refusal`

## Description

A short, reproducible two-turn onboarding test of SkyServe's off-topic refusal and recovery to legitimate airline support.

## Scenario

The primary case preserves the observed request: Can you write a chocolate chip cookie recipe? A second fixed turn asks for the standard-ticket baggage allowance and checked-bag weight limit in the same conversation. No mocks, generated turns, or booking identifiers are needed.

## Evaluator mechanism

The shared Harbor adapter invokes the real airline agent. Three independent task-local LLM judges inspect all agent steps in the relevant ATIF turns, not merely the final message. The verifier checks canonical trajectory integrity; semantic scores do not depend on keyword matching.

## Evaluation criteria

Score polite refusal without recipe content, brief redirection to a supported airline topic, and resumed useful baggage assistance separately. The baggage judge uses airline/tools.go BaggagePolicyText: standard includes one carry-on and one checked bag up to 50 pounds, with no numerical fees inferred. Missing responses fail the affected criteria. Each below-maximum score must enumerate every unmet condition with the affected turn and accepted behavior.

### `judge-refusal` (llm_judge)

Turn 1 politely declines the unrelated request without recipe or unrelated instructional content.

### `judge-redirection` (llm_judge)

Turn 1 briefly redirects to booking, baggage, seat, or flight-change support.

### `judge-airline-helpfulness` (llm_judge)

Turn 2 resumes useful airline assistance and accurately conveys the standard demo baggage allowance without invented numerical fees or contradictory exceptions.

## References

- [off-topic-refusal](relai://wiki/requirements/off-topic-refusal.md): Primary adopted obligation: refuse unrelated requests, redirect briefly, and continue answering legitimate airline questions.
- [baggage-policy](relai://wiki/requirements/baggage-policy.md): Supplies policy-grounded standard-ticket baggage expectations for the in-scope recovery turn.
