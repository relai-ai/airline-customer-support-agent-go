# A seat outside the aircraft layout

Type: **new-requirement**. Status: **not yet verified as an agent failure**.

## Scenario

My confirmation code is SKY123. Please move me to seat 99Z.

## Expected behavior

The synthetic aircraft has rows 1–30 and seats A–F. Reject seats outside that layout and leave the original booking unchanged. Verify the confirmation code before any valid change.

## Fixture

Set `RELAI_DEMO_CHALLENGE=invalid-seat` for the target agent process. All business data is synthetic. This adds an explicit requirement to investigate against the existing optimized checkpoint; no defect is secretly inserted.

## Investigation prompt

Investigate the a seat outside the aircraft layout challenge. Read demo/challenges/invalid-seat/README.md and inspect the relevant agent code and tool behavior. Validate seat coordinates before mutating booking state. Explain the cause, then propose a repeatable RELAI test for the expected behavior. Show me the proposed test before running it. After approval, evaluate the baseline, optimize if it fails, inspect the real diff, and rerun the target test and existing regressions. If it already passes, report that honestly. Do not change the expected behavior or evaluator merely to obtain a pass.

## Evidence policy

A suspicious code path is not a demonstrated agent failure. Record the exact checkpoint, scenario, model profile, command, trajectory and evaluated outcome before changing the catalog status. A passing baseline is a resilience result, not a failed challenge.
