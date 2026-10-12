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

## Seat-change walkthrough

- [Seat layout](relai://wiki/requirements/challenge-seat-layout.md)

## Demonstration operations

- [Session isolation](relai://wiki/requirements/demo-session-isolation.md)
- [Evidence integrity](relai://wiki/requirements/demo-evidence-integrity.md)

## Runtime and evaluation context

The [Go agent](relai://repo/airline/agent.go) supports OpenAI and Anthropic; OpenAI takes precedence when both keys exist. The [README](relai://repo/README.md) documents provider-specific model overrides and defaults of gpt-4.1 and claude-sonnet-5. These are runtime context, not airline-response requirements. The [launcher](relai://repo/start.sh) invokes a [terminal entrypoint](relai://repo/cmd/airline-support/main.go), saving conversations as JSONL. Named logs are overwritten on rerun; unnamed runs create fresh sessions. [Session code](relai://repo/airline/session.go) validates filenames and supports a log-directory override. [Booking tools](relai://repo/airline/tools.go) use synthetic in-memory data; seat changes mutate that data rather than a real reservation system.

[Agent unit tests](relai://repo/airline/agent_test.go) check instruction content and propagation, not semantic model compliance. The [benchmark](relai://repo/benchmarks/airline_support_benchmark.csv) covers refund grounding, known and unknown booking lookup, and seat changes.



## Risk analysis

- [Risk hypotheses](relai://wiki/hypotheses/risks.md)

The [current recorded walkthrough](relai://repo/demo/recorded/README.md) includes the measured adversarial seat-change baseline, independent verification, and a patch applied only on explicit request. The two [regression packages](relai://repo/demo/checkpoint/tests) remain available for reruns.
