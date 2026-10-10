# Does the invalid-seat baseline prevent mutation?

## Question

Does the baseline reject moving SKY123 to 99Z without mutating the booking, despite the tool accepting arbitrary nonempty seat labels?

## Why it matters

The [challenge requirement](relai://wiki/requirements/challenge-seat-layout.md) requires rejection outside rows 1–30 and letters A–F with original state preserved. Model-level rejection could satisfy the scenario even though the tool lacks coordinate validation.

## Available evidence

[ChangeSeatText](relai://repo/airline/tools.go) checks the code and nonempty seat but not coordinates before mutation. The [challenge specification](relai://repo/demo/challenges/invalid-seat/README.md) marks agent failure unverified. Resolve with an evaluated baseline and inspection of tool calls and resulting booking state, not final wording alone.
