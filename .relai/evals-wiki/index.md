# EvalsWiki

Repository requirements and evaluation knowledge for the airline support demo. These pages distill documented and tested contracts; challenge obligations retain their explicit scope.

## Airline assistance

- [Off-topic refusal](relai://wiki/requirements/off-topic-refusal.md)
- [Supported-turn recovery](relai://wiki/requirements/supported-turn-recovery.md)
- [Mixed requests](relai://wiki/requirements/mixed-request-handling.md)
- [Booking verification](relai://wiki/requirements/booking-verification.md)
- [Baggage grounding](relai://wiki/requirements/baggage-policy-grounding.md)
- [Commercial grounding](relai://wiki/requirements/commercial-fact-grounding.md)
- [Concise replies](relai://wiki/requirements/concise-replies.md)
- [Lookup accuracy](relai://wiki/requirements/booking-lookup-accuracy.md)
- [Seat changes](relai://wiki/requirements/requested-seat-change.md)

## Synthetic challenges

- [Unknown fares](relai://wiki/requirements/challenge-unknown-fare.md)
- [Seat layout](relai://wiki/requirements/challenge-seat-layout.md)
- [Booking outage](relai://wiki/requirements/challenge-booking-outage.md)
- [Unknown-fare baseline question](relai://wiki/hypotheses/open-questions/unknown-fare-baseline.md)
- [Invalid-seat baseline question](relai://wiki/hypotheses/open-questions/invalid-seat-baseline.md)

## Demonstration operations

- [Session isolation](relai://wiki/requirements/demo-session-isolation.md)
- [Evidence integrity](relai://wiki/requirements/demo-evidence-integrity.md)

## Runtime and evaluation context

The [Go agent](relai://repo/airline/agent.go) supports OpenAI and Anthropic; OpenAI takes precedence when both keys exist. The [README](relai://repo/README.md) documents provider-specific model overrides and defaults of gpt-4.1 and claude-sonnet-5. These are runtime context, not airline-response requirements. The [launcher](relai://repo/start.sh) invokes a [terminal entrypoint](relai://repo/cmd/airline-support/main.go), saving conversations as JSONL. Named logs are overwritten on rerun; unnamed runs create fresh sessions. [Session code](relai://repo/airline/session.go) validates filenames and supports a log-directory override. [Booking tools](relai://repo/airline/tools.go) use synthetic in-memory data; seat changes mutate that data rather than a real reservation system.

[Agent unit tests](relai://repo/airline/agent_test.go) check instruction content and propagation, not semantic model compliance. The [benchmark](relai://repo/benchmarks/airline_support_benchmark.csv) covers refund grounding, known and unknown booking lookup, and seat changes.

The [saved checkpoint result](relai://repo/demo/checkpoint/results.json) reports gpt-4.1 at revision dceb3dcbcf1936e3ae2c2bf885b7884f90cae9a5, dated 2026-10-10: targeted optimization increased from approximately 0.4444 to 1.0. Separate post-optimization regressions report 1.0 for off-topic refusal, redirection, airline helpfulness, and seat-commercial grounding, with no evaluator errors listed. This is repository-recorded historical evidence, not a new local simulation or a reliability estimate; it records one run per regression. [Checkpoint guidance](relai://repo/demo/README.md) describes portable generated packages and sanitized conversations, not an initialized runtime.

The [challenge catalog](relai://repo/demo/challenges/catalog.json) distinguishes new requirements from a controlled outage. [Fixture selection](relai://repo/airline/challenges.go) leaves unknown-fare and invalid-seat behavior untouched and replaces only booking lookup for booking-outage. A suspicious tool path alone does not demonstrate agent failure.


## Challenge analysis

- [Risk hypotheses](relai://wiki/hypotheses/risks.md)
