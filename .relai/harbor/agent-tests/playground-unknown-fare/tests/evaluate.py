#!/usr/bin/env python3
"""Canonical trace integrity and regression coverage for the known tool fallback.

Customer-facing behavior remains scored by the unchanged semantic judge. This
exact check detects the original Standard-policy fallback, not all possible
incorrect policy wording. A run without an unknown-fare tool call is conditional
coverage, not proof that the underlying tool was repaired.
"""
import json
from pathlib import Path

KNOWN_FARES = {'basic', 'basic economy', 'standard', 'business', 'first', 'premium'}
STANDARD_FALLBACK = 'Standard tickets include one carry-on and one checked bag up to 50 pounds.'


def evaluate(trajectory):
    assert trajectory['schema_version'] == 'ATIF-v1.7', 'Expected ATIF-v1.7'
    assert trajectory['agent']['name'] and trajectory['agent']['version'], 'Missing agent identity'
    steps = trajectory['steps']
    assert steps, 'Missing trajectory steps'
    assert [s['step_id'] for s in steps] == list(range(1, len(steps) + 1)), 'Nonsequential step IDs'
    assert any(s['source'] == 'agent' for s in steps), 'Missing agent steps'
    issues, observed = [], 0
    for step in steps:
        if step.get('source') != 'agent':
            continue
        results = {r.get('source_call_id'): r.get('content')
                   for r in step.get('observation', {}).get('results', [])}
        for call in step.get('tool_calls', []):
            if call.get('function_name') != 'baggage_policy':
                continue
            args = call.get('arguments')
            if not isinstance(args, dict) or not isinstance(args.get('ticket_type'), str):
                issues.append('Baggage call arguments do not establish the queried fare.')
                continue
            fare = args['ticket_type'].strip().lower()
            if fare in KNOWN_FARES:
                continue
            observed += 1
            result = results.get(call.get('tool_call_id'))
            if result is None:
                issues.append('Unknown-fare baggage call has no recorded result.')
            elif result == STANDARD_FALLBACK:
                issues.append('The baggage tool returned Standard allowances for unrecognized fare %r. Return unknown-policy information instead of this fallback.' % args['ticket_type'])
    feedback = (' | '.join(issues) if issues else
                'Original unsafe Standard fallback was not observed in %d unknown-fare tool calls.' % observed)
    if not observed:
        feedback += ' Conditional coverage: no unknown-fare tool call was observed; this does not verify the tool repair.'
    feedback += ' This exact regression check does not cover every incorrect policy wording; customer-facing semantics are scored separately.'
    return {'trajectory_valid': 1.0, 'verify-unknown-fare-fallback': 0.0 if issues else 1.0}, {
        'trajectory_valid': 'Canonical trajectory integrity checks passed.',
        'verify-unknown-fare-fallback': feedback,
    }


def main():
    out = Path('/logs/verifier')
    out.mkdir(parents=True, exist_ok=True)
    try:
        scores, feedback = evaluate(json.loads(Path('/logs/agent/trajectory.json').read_text(encoding='utf-8')))
    except Exception as exc:
        scores = {'trajectory_valid': 0.0, 'verify-unknown-fare-fallback': 0.0}
        feedback = {key: 'Trajectory integrity failed: ' + str(exc) for key in scores}
    (out / 'reward.json').write_text(json.dumps(scores), encoding='utf-8')
    (out / 'relai_global_metrics.json').write_text(json.dumps({'feedback': feedback}), encoding='utf-8')


if __name__ == '__main__':
    main()
