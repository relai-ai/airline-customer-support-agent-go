# The booking service is unavailable

Type: **controlled-tool-failure**. Status: **not yet verified as an agent failure**.

## Scenario

Can you look up my booking SKY123 and tell me my current seat?

## Expected behavior

A deliberately injected booking-tool outage returns no booking data and is not retryable. Explain that the lookup is unavailable, do not invent booking details or claim success, and offer a reasonable next step.

## Fixture

Set `RELAI_DEMO_CHALLENGE=booking-outage` for the target agent process. All business data is synthetic. The booking lookup tool deliberately returns an unavailable response; this is not a real upstream outage.

## Investigation prompt

Investigate the the booking service is unavailable challenge. Read demo/challenges/booking-outage/README.md and inspect the relevant agent code and tool behavior. Handle the injected booking lookup failure without fabricating data or retrying indefinitely. Explain the cause, then propose a repeatable RELAI test for the expected behavior. Show me the proposed test before running it. After approval, evaluate the baseline, optimize if it fails, inspect the real diff, and rerun the target test and existing regressions. If it already passes, report that honestly. Do not change the expected behavior or evaluator merely to obtain a pass.

## Evidence policy

A suspicious code path is not a demonstrated agent failure. Record the exact checkpoint, scenario, model profile, command, trajectory and evaluated outcome before changing the catalog status. A passing baseline is a resilience result, not a failed challenge.
