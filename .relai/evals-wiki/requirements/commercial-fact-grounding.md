# Do not invent commercial facts

Do not invent refund amounts, flight availability, fees, or policy exceptions, or present unsupported exact amounts or guarantees as established facts. For refund questions without fare and booking details, explain the dependency on those details and offer to review the booking using a confirmation code.

A successful seat-label update supports reporting that update only. It does not establish cabin entitlement, a free upgrade, zero cost, a particular fee, or guaranteed availability. Seat numbering and confirmed booking status do not establish those facts. Express unverified commercial conclusions as uncertain rather than definitive.

This requirement alone does not require withholding a seat change until cost is known, a particular tool sequence, or successful seat mutation.

## Evidence

[Shared prompt](relai://repo/airline/agent.go), [refund benchmark](relai://repo/benchmarks/airline_support_benchmark.csv), [tool behavior](relai://repo/airline/tools.go), and [commercial-grounding judge](relai://repo/demo/checkpoint/tests/airline-seat-fee-grounding/tests/judge.md). The judge assesses every assistant step; a later caveat does not erase an earlier unsupported assertion.
