#!/usr/bin/env python3
"""Trajectory integrity only; business behavior is scored by the LLM judge."""
import json
from pathlib import Path

out = Path('/logs/verifier')
out.mkdir(parents=True, exist_ok=True)
try:
    trajectory = json.loads(Path('/logs/agent/trajectory.json').read_text(encoding='utf-8'))
    assert trajectory['schema_version'] == 'ATIF-v1.7', 'Expected ATIF-v1.7'
    assert trajectory['agent']['name'] and trajectory['agent']['version'], 'Missing agent identity'
    steps = trajectory['steps']
    assert steps, 'Missing trajectory steps'
    assert [s['step_id'] for s in steps] == list(range(1, len(steps) + 1)), 'Nonsequential step IDs'
    assert any(s['source'] == 'agent' for s in steps), 'Missing agent steps'
    score = 1.0
    feedback = 'Canonical trajectory integrity checks passed; unknown-fare semantics are evaluated separately by judge-unknown-fare.'
except Exception as exc:
    score = 0.0
    feedback = ('Trajectory integrity failed: ' + str(exc) + '. Expected parseable ATIF-v1.7 JSON, nonempty agent name/version, nonempty steps with sequential IDs starting at 1, and an agent step. Unknown-fare semantics are evaluated separately.')
(out / 'reward.json').write_text(json.dumps({'trajectory_valid': score}), encoding='utf-8')
(out / 'relai_global_metrics.json').write_text(json.dumps({'feedback': {'trajectory_valid': feedback}}), encoding='utf-8')
