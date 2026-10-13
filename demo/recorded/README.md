# Recorded playground evidence

This walkthrough replays real RELAI results, one prompt at a time. It makes no model requests during replay. The original Go source remains unchanged until the visitor applies the recorded patch.

Two completed six-rollout optimization cycles added seat validation and an Agent-owned execution ledger. Independent verification scored mutation safety 1/1 and semantic trajectory 0.8/1 (90% aggregate), compared with the original 10% aggregate. Both existing regression tasks passed all four judges. Go race tests and vet passed.

The remaining semantic deduction is a fabricated statement about an earlier reporting error in the final audit. Verification is complete, but the agent is not perfect. The model and evaluators were not changed to improve scores.

Initialization timing was not retained and is not invented. Baseline simulation timing comes from the actual trial. Optimization timing comes from the two completed command receipts; independent checks are additional.

`verified.patch` and the before/after snapshots in `walkthrough.json` are hash-checked. Release builds validate the original source. Apply refuses changed visitor files. No provider credentials, session configuration, or private CLI state are included.

The first reply displays the five proposed risks from the saved EvalsWiki inventory in `wiki-risks.md`, including failure descriptions, triggers, impact, confidence, and triggerability. `walkthrough.json` contains the structured rows and snapshot digest; the loader rejects altered snapshots or rows that differ from the saved inventory. This snapshot is a read of existing wiki evidence, not a new initialization or simulation run. The first badge therefore says `Precomputed relai wiki list risks`.

The existing simulation and optimization artifacts do not establish Codex orchestration provenance. Do not relabel them as Codex + RELAI. Replacing those results requires the actual Codex run's wiki inventory, command receipts, trajectories, evaluator results, and verified patch from the same baseline. The six-risk initialization shown in the supplied video is a different inventory from this repository's five-risk snapshot.
