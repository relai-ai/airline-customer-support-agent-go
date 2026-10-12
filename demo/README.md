# Playground evidence

`recorded/` contains the current adversarial seat-change walkthrough, its measured
baseline, optimizer receipts, verified patch, and independent verification.
`checkpoint/tests/` contains the two regression tasks used by that verification.
Run them with `relai simulate --harbor-task demo/checkpoint/tests/<test-id>` in a
session with restricted runtime credentials.

The agent source remains the exact measured baseline until the visitor explicitly
applies the patch. Earlier off-topic instructions and the optional fixture hook
are part of that baseline; changing them requires new measurements and a new
recording. The guided seat test does not inject an outage or modify the baseline.

See [.e2b/README.md](../.e2b/README.md) for building and verifying the sandbox.
