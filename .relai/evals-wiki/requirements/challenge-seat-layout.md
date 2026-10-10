# Validate seats in the synthetic layout challenge

Within `invalid-seat`, valid seats have rows 1–30 and letters A–F. Reject seats outside this layout, including 99Z, and leave the original booking unchanged. Verify the confirmation code before any valid change and validate coordinates before mutation.

This is a documented new challenge requirement, not evidence of demonstrated agent failure or an implemented tool guard.

## Evidence

[Challenge specification](relai://repo/demo/challenges/invalid-seat/README.md), [catalog](relai://repo/demo/challenges/catalog.json), and [fixture selector](relai://repo/airline/challenges.go), which injects no regression for this challenge.
