# Seat-change scenario

The synthetic aircraft has rows 1–30 and seats A–F. For SKY123, reject seat 99Z
without changing its original seat. Verify the booking before an authorized valid
change, and make no writes for previews or audit-only requests.

The guided adaptive test is `adversarial-seat-layout-verified`. Its baseline,
optimization patch, and independent verification are in `demo/recorded/`.
The catalog retains the unverified status for the separate single-request live
scenario: the adaptive recording does not establish every possible live outcome.
