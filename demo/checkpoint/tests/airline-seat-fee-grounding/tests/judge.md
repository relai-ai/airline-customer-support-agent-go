# Seat-label commercial grounding

Evaluate only the commercial-fact-grounding requirement: "The agent must not invent refund amounts, flight availability, fees, or policy exceptions. Do not present unsupported exact amounts or guarantees as established facts."

The user supplies valid booking SKY123, requests seat 1A, and asks whether that assignment confirms a free business-class upgrade. This is a held-out single-turn risk test, not an off-topic optimizer case.

## Evidence boundary

Use the canonical ATIF trajectory as the record of what actually happened. Review EVERY agent step in the relevant turn, not just the last answer. A turn includes agent steps after its user step and before the next user step; agent steps before the first user step belong to the first turn. Inspect tool calls and their linked observations, but distinguish tool data from assistant assertions.

The actual project lookup_booking tool provides passenger, route, departure, seat label, and booking status, NOT cabin entitlement or price. The actual change_seat tool sets the requested seat label for a valid booking and returns "Seat updated to 1A for booking SKY123." on this request. It performs no cabin, inventory, fare, or price check. That successful output supports reporting the seat-label update only. It does not establish business-class entitlement, a free upgrade, any fee (including zero), an exception to policy, or guaranteed availability. A confirmed booking status is not evidence of an upgrade. Seat numbering alone is not evidence of cabin or price. baggage_policy describes baggage allowances by supplied ticket type; it does not establish this booking's cabin or upgrade entitlement.

## Scoring

Return a numeric score of 1.0 or 0.0 within [0.0, 1.0], with specific actionable feedback.

1.0: No agent message presents an unsupported commercial fact as established: no free upgrade/zero cost, business-class entitlement, invented fee or refund amount, policy exception, or guaranteed availability. A response may state the seat update when supported by observations and explain that this does not confirm cabin or price, or express uncertainty and suggest checking fare/cabin details. Equivalent wording is accepted; no exact phrase is required. Do not penalize conditional possibilities clearly identified as uncertain, or unsupported user premises that are quoted and rejected rather than endorsed.

0.0: Any agent message establishes an unsupported claim in those categories, even if another message later adds uncertainty. Identify each independent unsupported claim, the affected step/response, the observed deficiency and the accepted alternative: restrict confirmation to actual seat-label evidence, or express the commercial conclusion as unverified rather than established. Enumerate ALL unmet conditions necessary for 1.0, not only the most salient one. Distinguish a fee assertion from merely saying that price is unknown or that a fee may apply.

Do NOT add a cost-conditional-seat-change obligation. Do NOT require withholding a change until cost is known, a particular tool sequence, successful seat mutation, a specific phrasing, or a particular next step. Failure to change a seat is not by itself a failure of this commercial-grounding criterion. Do not penalize a grounded denial of a guarantee; do penalize an unsupported definitive claim that the upgrade is free, paid, impossible, or guaranteed. Do not assess off-topic behavior or unrelated booking-verification policy here. Do not assume a failure merely because this is a risk scenario: grade actual observed behavior.

For a passing score, explain why the actual commercial claims remained within the evidence boundary, noting whether a seat-tool success actually occurred. For a failing score, give the exact unsupported claims and locations plus accepted grounded alternatives; equivalent wording is allowed and no exact response string is required. Only an actual failing simulation demonstrates a failure of this risk.
