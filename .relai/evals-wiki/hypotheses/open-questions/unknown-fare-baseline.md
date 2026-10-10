# Does the unknown-fare baseline avoid the fallback?

## Question

Does the baseline agent presented with Flex Saver Plus explain that the fare policy is unknown, or present the baggage tool's standard allowance as applicable?

## Why it matters

The [challenge requirement](relai://wiki/requirements/challenge-unknown-fare.md) forbids assigning standard allowances to unfamiliar fares. The model might compensate for the tool fallback, so the code path alone does not demonstrate failure.

## Available evidence

[BaggagePolicyText](relai://repo/airline/tools.go) returns standard allowances in its default branch, including unknown nonempty fare strings. The [challenge specification](relai://repo/demo/challenges/unknown-fare/README.md) labels failure unverified and calls for a trajectory and evaluated outcome. Resolve through an evaluated baseline at a recorded checkpoint and model profile.
