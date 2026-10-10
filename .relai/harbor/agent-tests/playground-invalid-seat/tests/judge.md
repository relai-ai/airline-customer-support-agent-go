# Exact invalid-seat challenge: semantic rejection

Source: requirement challenge-seat-layout. The exact user request is: My confirmation code is SKY123. Please move me to seat 99Z.
The synthetic layout is rows 1–30, letters A–F. Original SKY123 seat is 12A. This is a new requirement, not an injected defect. Evaluate the actual baseline; never assume failure from source code.

Read the entire canonical ATIF turn: every agent message and tool call/result after the user step and before the next user, including agent steps before the first user as belonging to the first turn. Do not score only the final message. Tool-call JSON is not itself a user-facing rejection.

Score 1.0 only when the assistant clearly declines the requested move because 99Z is not a valid seat / is outside the layout, does not present 99Z as valid, and does not misleadingly claim a completed move or unchanged booking contrary to observed tool evidence. Equivalent wording is accepted; no literal phrase is required. The assistant need not recite the full layout or original seat, offer alternatives, or look up a booking just to reject an impossible seat. If it describes the layout, that description must be consistent with rows 1–30 and letters A–F. An offer to help select a valid seat is acceptable but not required.

Score 0.5 if it declines or defers the move without claiming success but fails to recognize that 99Z is outside the layout (for example, merely asks for booking verification). Score 0.0 if it accepts 99Z as valid, claims the invalid move succeeded, gives contradictory or misleading success/unchanged-state claims, or supplies no meaningful response. A later rejection does not erase an earlier misleading success claim. Exact mutation safety is independently scored by verify-seat-mutations; do not infer a refusal merely from absence of calls.

Return a finite score in [0,1] and specific actionable feedback for every score. For any score below 1, enumerate ALL independent unmet criteria with the affected step/response, observed deficiency and what would satisfy it, explicitly explaining that equivalent rejection wording is accepted. Do not claim that this one instance proves all coordinate boundaries, valid-change verification, or production-profile correctness. The configured target model is whatever the trajectory's model_name records, not the judge model.
