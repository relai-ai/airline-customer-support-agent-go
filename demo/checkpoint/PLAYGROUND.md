# From a plausible answer to evidence of improvement

This prepared airline agent has already gone through one RELAI improvement cycle.
The recorded target was to decline off-topic requests while continuing to help
with legitimate airline questions, including after an off-topic turn.

## Recorded result — not a live run

The saved evaluation score improved from **0.4444 to 1.0** on that recorded task.
The two saved regression runs completed with their listed evaluator scores at 1.0.
These are individual recorded runs using **gpt-4.1**, not a statistical reliability
estimate, a speed benchmark, or proof of performance on the current model profile.

- [Results and limitations](results.json)
- [Off-topic and recovery conversation](airline-off-topic-refusal.trajectory.json)
- [Seat-change grounding conversation](airline-seat-fee-grounding.trajectory.json)

## Try the same process on a new scenario

1. Choose a challenge in the RELAI assistant panel.
2. Describe what you expect in your own words.
3. Review the proposed test and approve its execution.
4. Compare the baseline with the result after optimization.
5. Inspect the code changes and rerun regressions.

A scenario that already passes is a useful result. The assistant must not invent a
failure or weaken a test to manufacture an improvement. Airline data is synthetic;
new commands, file changes and execution results are real.
