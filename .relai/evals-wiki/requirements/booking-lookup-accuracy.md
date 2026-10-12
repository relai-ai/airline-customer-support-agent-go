# Report booking results accurately

For a known confirmation code, report the available booking details without inventing unrelated facts or asking for unnecessary extra information. For an unknown code, clearly say no booking was found and do not fabricate passenger, route, seat, or status details.

The benchmark's initial SKY123 fixture identifies Maya Chen, JFK to SFO, departure 2026-07-14 09:30, seat 12A, and confirmed status. Seat labels are mutable; a later lookup after a change must reflect current state rather than blindly repeat the initial seat.

## Evidence

[Benchmark](relai://repo/benchmarks/airline_support_benchmark.csv), [bookings and lookup tool](relai://repo/airline/tools.go), and [lookup tests](relai://repo/airline/tools_test.go).
