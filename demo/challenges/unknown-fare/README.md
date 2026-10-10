# An unfamiliar ticket type

Type: **new-requirement**. Status: **not yet verified as an agent failure**.

## Scenario

My ticket says Flex Saver Plus. How many checked bags are included?

## Expected behavior

The synthetic airline recognizes only basic/basic economy, standard, business, first, and premium. For any other fare name, do not assign standard allowances. Explain that the policy is unknown and ask for clarification or offer escalation.

## Fixture

Set `RELAI_DEMO_CHALLENGE=unknown-fare` for the target agent process. All business data is synthetic. This adds an explicit requirement to investigate against the existing optimized checkpoint; no defect is secretly inserted.

## Investigation prompt

Investigate the an unfamiliar ticket type challenge. Read demo/challenges/unknown-fare/README.md and inspect the relevant agent code and tool behavior. Unknown ticket types must not silently inherit the standard baggage allowance. Explain the cause, then propose a repeatable RELAI test for the expected behavior. Show me the proposed test before running it. After approval, evaluate the baseline, optimize if it fails, inspect the real diff, and rerun the target test and existing regressions. If it already passes, report that honestly. Do not change the expected behavior or evaluator merely to obtain a pass.

## Evidence policy

A suspicious code path is not a demonstrated agent failure. Record the exact checkpoint, scenario, model profile, command, trajectory and evaluated outcome before changing the catalog status. A passing baseline is a resilience result, not a failed challenge.
