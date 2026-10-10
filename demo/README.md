# Optimized airline checkpoint

The included source contains the accepted off-topic handling improvement from
optimizer commit `dceb3dcbcf1936e3ae2c2bf885b7884f90cae9a5`.
`checkpoint/results.json` records the targeted before/after score and a separate
post-optimization regression run. Both regression scenarios passed all their
evaluators. This is limited measured evidence, not a claim of universal reliability.

The checkpoint includes exact generated test packages and sanitized conversations.
`checkpoint/sha256.json` records their hashes. It excludes personal RELAI config,
OAuth sessions, environment files, model keys, and internal runtime caches.

Test packages are portable inputs, not a configured RELAI installation. A new
session still needs its own initialized runtime and restricted credentials before
running them via `relai simulate --harbor-task demo/checkpoint/tests/<test-id>`.
Do not reuse the preparer's authentication or load production keys into this image.

The next demonstrated failure is pending discovery. Until one has a real failing
trajectory and valid evaluator result, this image is an optimized development
sample, not the complete onboarding exercise.

Reset means destroying the visitor sandbox and creating a new one from the same
immutable E2B build. Never snapshot a visitor's mutated or authenticated session.
