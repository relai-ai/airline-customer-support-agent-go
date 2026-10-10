#!/usr/bin/env python3
"""Trajectory integrity only; task behavior is scored by independent LLM judges."""
import json
from pathlib import Path

out = Path('/logs/verifier')
out.mkdir(parents=True, exist_ok=True)
try:
    trajectory = json.loads(Path('/logs/agent/trajectory.json').read_text(encoding='utf-8'))
    assert trajectory['schema_version'] == 'ATIF-v1.7', 'Expected ATIF-v1.7.'
    assert trajectory['agent']['name'] and trajectory['agent']['version'], 'Missing agent identity.'
    steps = trajectory['steps']
    assert steps, 'Expected nonempty steps.'
    assert [s['step_id'] for s in steps] == list(range(1, len(steps) + 1)), 'Expected sequential step IDs starting at 1.'
    assert any(s['source'] == 'agent' for s in steps), 'Expected at least one agent step.'
    score = 1.0
    feedback = 'Canonical trajectory integrity checks passed. Behavioral scores are supplied by the three independent LLM judges.'
except Exception as exc:
    score = 0.0
    feedback = ('Trajectory integrity failed: ' + str(exc) + ' Required: readable ATIF-v1.7 JSON, nonempty agent name and version, nonempty steps with sequential step IDs starting at 1, and at least one agent step. Semantic behavior is scored separately.')
(out / 'reward.json').write_text(json.dumps({'reward': score}), encoding='utf-8')
(out / 'relai_global_metrics.json').write_text(json.dumps({'feedback': {'trajectory-integrity': feedback}}), encoding='utf-8')
