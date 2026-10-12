# Recorded developer walkthrough

`demo/recorded/walkthrough.json` contains three precomputed steps (risk review,
fresh adversarial test creation and baseline simulation, optimization and
independent verification) followed by an explicit application request. It must
come from completed real RELAI commands, never the browser integration fixtures.

Keep the sample's Go source at the measured original checkpoint. Store only the
verified optimizer patch in `demo/recorded/verified.patch`. The manifest records
SHA-256 values for every before/after Go file; `before: null` represents a new
file. New files may be created only if the user has not already created them.
Recorded file paths must be regular files inside the workspace, not symlinks.

Required manifest fields: `schemaVersion: 1`, four canonical `prompts`, three
`steps` with text, measured `durationSeconds` and a `durationLabel` naming
what was timed (inventory read, baseline simulation, optimization plus verification), at least ten real completed
`turns` containing user and final agent text, `comparison` with before/after and
regression results, `diff` equal to the stored patch, `files` and `patchSha256`.
Include public evidence paths and commands in the step metadata. Preserve full
redacted transcripts and evaluator feedback alongside the readable conversation.
Disclose incomplete checks; do not infer improvement from a zero command exit.

Record timings from actual command receipts. Label whether a time measures full
creation, certification, simulation or optimization. Reading an existing risk
report is not a newly measured risk-discovery run. Model generation can take
several minutes; replay has no artificial running indicator or delay.

Run `node .e2b/ide/extension/validate-recording.js` before building the template.
The Dockerfile also runs it after copying the extension. Missing or inconsistent
recordings fail the image build. Applying a recording checks the entire original
file set before applying and confirms the resulting hashes. Preserve user edits
on mismatch. Subsequent freeform prompts use the real workspace and scoped model
gateway, not replay data. No provider keys or private CLI state belong in this
bundle. The comparison establishes the recorded test results only.

Generation checklist:

1. Create the adaptive test through RELAI in the original source workspace.
   Keep its generated public EvalsWiki page and registry record with the task
   package; copying only the task prevents later RELAI repair or updates.
2. Review evaluator criteria before optimization. Criteria must accept correct
   tool-side rejection, preserve valid controls and booking state, and allow
   agent-code changes without changing the model or expected behavior.
3. Simulate the original agent and retain its actual trajectory, scores and
   command receipt. A failed command is not a scored agent failure.
4. Optimize using the same frozen test, export the accepted source patch and
   reports before ending the temporary workspace, then independently rerun the
   target and regressions on the accepted branch.
5. Package only redacted public evidence, complete the hash validation, and
   build/smoke-test the image. Never publish fixture transcripts as real runs.

Native diff review uses `beforeContent` and `afterContent` on each file,
validated against the measured source hashes. Virtual documents are read-only;
review never applies the patch or overwrites visitor edits.

Optional initialization timing must have a saved successful `relai init`
command receipt. The loader checks its command, exit status and duration.
Without that receipt, show no initialization duration. Inventory-read time
is never initialization time.
