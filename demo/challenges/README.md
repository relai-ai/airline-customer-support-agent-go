# Follow-up challenge package

Three choices continue from the existing optimized airline checkpoint. All business
data is synthetic. Unknown-fare and seat-layout scenarios add explicit requirements;
they do not secretly alter the agent to force a failure. Booking outage deliberately
replaces one tool in one agent instance with a nonretryable synthetic failure.

`baselines.json` records source hashes, actual models and recorded observations.
The captured GPT-4.1 runs show wrong fare attribution and an invalid seat mutation.
The outage run handled the unavailable lookup without fabricating a seat or retrying.
It is a resilience scenario, not a demonstrated failure. These observations do not
constitute completed semantic grades or validation on GPT-6.1 Sol / Fast / medium.
The catalog therefore remains unverified for release.

Prepared RELAI tests, adapters and requirements are in `.relai/harbor` and
`.relai/evals-wiki`. Inspect each test README for scenario, evaluator and criteria.
Use `relai simulate --help` before evaluating a chosen test; the three IDs are
`playground-unknown-fare`, `playground-invalid-seat`, and `playground-booking-outage`.
Use the normal RELAI optimize workflow only after reviewing the failing baseline.
Keep the expected behavior unchanged and rerun existing checkpoint regressions.

The package contains no credentials or local RELAI configuration. The service must
provision restricted backend resources and configure model routing before visitors
can execute the complete workflow. Building this template alone does not do that.
The previous verified E2B build predates this package; a new build and live checks
are required before release. Do not overwrite recorded evidence with live results.
