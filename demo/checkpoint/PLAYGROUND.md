# Seat-change playground

The walkthrough starts with the measured airline source, investigates seat-change
risks, replays an adaptive adversarial test, and shows the independently verified
optimization patch. Review the diff and explicitly apply it before trying the
improved agent. Editable requests can also use the live session gateway.

- [Recording and provenance](../recorded/README.md)
- [Measured verification and limitations](../recorded/verification.json)
- [Workspace and runtime setup](../../.e2b/README.md)

The independent aggregate improved from 10% to 90% in the recorded trial. The
semantic judge retained one deduction; this is evidence of one measured trial,
not universal reliability. The regression tasks in `tests/` cover off-topic
handling and commercial grounding and remain available for reruns.

Destroy the sandbox and create a fresh one to reset it. Never snapshot injected
credentials or visitor data.
