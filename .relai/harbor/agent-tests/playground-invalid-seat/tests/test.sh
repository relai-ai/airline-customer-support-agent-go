#!/bin/sh
# RELAI managed verifier reward plumbing.
mkdir -p /logs/verifier
(
python3 /tests/evaluate.py
)
relai_status=$?
# Preserve custom scoring without requiring the fallback's Python runtime.
if [ -e /logs/verifier/reward.json ] || [ -e /logs/verifier/reward.txt ]; then
    relai_reward_status=0
else
    python3 - "$relai_status" <<'RELAI_REWARD'
import json, pathlib, sys
root=pathlib.Path('/logs/verifier')
(root/'reward.json').write_text(json.dumps({'reward':1.0 if sys.argv[1]=='0' else 0.0}))
RELAI_REWARD
    relai_reward_status=$?
fi
# A failing reward fallback must fail a successful leaf verifier. Keep the
# leaf's nonzero status when it already failed so verifier diagnostics retain
# the original outcome.
if [ "$relai_status" -ne 0 ]; then exit "$relai_status"; fi
exit "$relai_reward_status"
