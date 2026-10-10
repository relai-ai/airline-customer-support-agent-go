# Agent test: `playground-invalid-seat`

## Description

Measure the current SkyServe checkpoint on the exact synthetic invalid-seat request without source changes or injected defects. This is new-requirement baseline evidence, not production-profile verification.

## Scenario

One fixed user turn: “My confirmation code is SKY123. Please move me to seat 99Z.” Synthetic layout: rows 1–30, letters A–F; original SKY123 seat 12A. Covers rejection and preserved booking with conditional verification-before-valid-change checks; does not claim coverage of every boundary seat or a separate valid-change interaction.

## Evaluator mechanism

Shared Harbor project runtime and original NewAgent/StreamResponse event boundary; no mocks. A code evaluator inspects every tool call and result in the full canonical ATIF turn using existing tool semantics. An LLM judge assesses the response meaning. The verifier records actual trajectory model_name values in model-evidence.json and feedback. The invalid-seat selector is a no-op, equivalent to default; no task override changes model or tools.

## Evaluation criteria

Full credit requires rejecting 99Z as invalid without a success claim, no invalid-coordinate mutation calls, SKY123 remaining at 12A throughout, and successful prior confirmation lookup before any valid change. Conditional verification may pass vacuously when no change is attempted. Missing or malformed trace evidence cannot establish full credit. Record actual configured model from observed agent steps, checkpoint and baseline outcome; never infer a demonstrated failure from source alone.

### `verify-seat-mutations` (code)

Validate coordinates on every change_seat call; infer SKY123 remains at 12A from complete mutation trace; require prior successful matching lookup before valid changes. Emit all unmet conditions and observed model names.

### `judge-invalid-seat` (llm_judge)

Assess the complete turn for a clear rejection of 99Z as outside the synthetic layout, no misleading claim of successful mutation, and consistency with the observed tools.

## References

- [challenge-seat-layout](relai://wiki/requirements/challenge-seat-layout.md): Authoritative synthetic layout and exact-instance rejection, unchanged-booking and verification-before-change requirements.

<!-- relai:review-assertions:start -->

## Review evidence

- **Criterion:** Reject the requested 99Z as outside the synthetic layout without claiming a completed move or presenting it as valid.
  - Requirement: [challenge-seat-layout](relai://wiki/requirements/challenge-seat-layout.md)
  - Excerpt: Within `invalid-seat`, valid seats have rows 1–30 and letters A–F. Reject seats outside this layout, including 99Z, and leave the original booking unchanged.
  - Rationale: Rejection is an open-ended natural-language behavior judged semantically, not with keywords.
  - Implementation paths: tests/evaluator.toml, tests/judge.md
- **Criterion:** No invalid coordinate is sent to change_seat, and no executed change moves SKY123 away from its original 12A even if later restored.
  - Requirement: [challenge-seat-layout](relai://wiki/requirements/challenge-seat-layout.md)
  - Excerpt: Reject seats outside this layout, including 99Z, and leave the original booking unchanged. Verify the confirmation code before any valid change and validate coordinates before mutation.
  - Rationale: The original change_seat tool mutates any known booking with a nonempty requested seat. Its complete observed call stream can determine whether invalid coordinates were passed and whether SKY123 was changed. No new guard or mocked state is introduced.
  - Implementation paths: tests/evaluate.py
- **Criterion:** Any valid change_seat call must follow a successful lookup_booking for the same confirmation code; this is conditional and does not require lookup merely to reject 99Z.
  - Requirement: [challenge-seat-layout](relai://wiki/requirements/challenge-seat-layout.md)
  - Excerpt: Verify the confirmation code before any valid change and validate coordinates before mutation.
  - Rationale: Call order, confirmation-code equality and recorded lookup output are exact facts. The exact instance does not independently exercise a requested valid change.
  - Implementation paths: tests/evaluate.py

<!-- relai:review-assertions:end -->
