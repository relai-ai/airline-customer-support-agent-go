# Recorded playground evidence

This walkthrough replays real RELAI results, one prompt at a time. It makes no model requests during replay. The original Go source remains unchanged until the visitor applies the recorded patch.

Two completed six-rollout optimization cycles added seat validation and an Agent-owned execution ledger. Independent verification scored mutation safety 1/1 and semantic trajectory 0.8/1 (90% aggregate), compared with the original 10% aggregate. Both existing regression tasks passed all four judges. Go race tests and vet passed.

The remaining semantic deduction is a fabricated statement about an earlier reporting error in the final audit. Verification is complete, but the agent is not perfect. The model and evaluators were not changed to improve scores.

Initialization timing was not retained and is not invented. Baseline simulation timing comes from the actual trial. Optimization timing comes from the two completed command receipts; independent checks are additional.

`verified.patch` and the before/after snapshots in `walkthrough.json` are hash-checked. Release builds validate the original source. Apply refuses changed visitor files. No provider credentials, session configuration, or private CLI state are included.
