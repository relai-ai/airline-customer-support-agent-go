#!/usr/bin/env python3
"""Task-owned integration checks using only exported, canonical ATIF evidence."""
import json
from pathlib import Path


def evaluate():
    trajectory = json.loads(Path('/logs/agent/trajectory.json').read_text())
    assert trajectory['schema_version'] == 'ATIF-v1.7'
    steps = trajectory['steps']
    assert [s['step_id'] for s in steps] == list(range(1, len(steps)+1))
    assert len([s for s in steps if s['source'] == 'user']) >= 2
    assert all(s.get('message') for s in steps if s['source'] == 'agent')
    # Each observed user turn must have a nonempty final assistant response,
    # not just a tool-call message, before the next user arrives.
    turns = []
    current = None
    for step in steps:
        if step['source'] == 'user':
            current = []
            turns.append(current)
        elif step['source'] == 'agent':
            if current is None:
                raise ValueError('Agent step without preceding user')
            current.append(step)
    assert all(t and not t[-1].get('tool_calls') and str(t[-1].get('message', '')).strip() for t in turns)
    found = set()
    for step in steps:
        results = step.get('observation', {}).get('results', [])
        for call in step.get('tool_calls', []):
            matched = [r for r in results if r.get('source_call_id') == call['tool_call_id']]
            assert matched, 'Observed tool call lacks its same-step result'
            args = call.get('arguments', {})
            output = ' '.join(str(r.get('content', '')) for r in matched)
            if call['function_name'] == 'lookup_booking' and args.get('confirmation_code') == 'SKY123' and 'Maya Chen' in output:
                found.add('lookup_booking')
            if call['function_name'] == 'change_seat' and args.get('confirmation_code') == 'SKY123' and args.get('requested_seat') == '14C' and 'Seat updated to 14C' in output:
                found.add('change_seat')
    assert found == {'lookup_booking', 'change_seat'}, 'Expected booking lookup and seat change evidence'


if __name__ == '__main__':
    directory = Path('/logs/verifier')
    directory.mkdir(parents=True, exist_ok=True)
    try:
        evaluate()
    except Exception:
        (directory / 'reward.txt').write_text('0\n')
        print('Smoke failed: missing or invalid booking/seat-change trajectory evidence.')
        raise SystemExit(1)
    (directory / 'reward.txt').write_text('1\n')
    print('Smoke passed: booking lookup and contextual seat change observed.')
