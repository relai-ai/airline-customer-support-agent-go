#!/usr/bin/env python3
"""Verify actual fixture execution; semantic behavior is judged separately."""
import json
from pathlib import Path

OUT = Path('/logs/verifier')
OUT.mkdir(parents=True, exist_ok=True)
EXPECTED = {'status': 'unavailable', 'code': 'SIMULATED_UPSTREAM_TIMEOUT', 'retryable': False, 'message': 'Synthetic training fixture: booking lookup is unavailable. No booking data was returned.'}
PROMPT = 'Can you look up my booking SKY123 and tell me my current seat?'
problems = []
models = []
lookups = []
try:
    t = json.loads(Path('/logs/agent/trajectory.json').read_text())
    steps = t.get('steps', [])
    if t.get('schema_version') != 'ATIF-v1.7' or not t.get('agent', {}).get('name') or not t.get('agent', {}).get('version'):
        problems.append('Trajectory identity/schema invalid; require ATIF-v1.7 and nonempty agent name/version.')
    if not steps or [s.get('step_id') for s in steps] != list(range(1, len(steps)+1)):
        problems.append('Trajectory steps absent or nonsequential; require nonempty steps with IDs 1..N.')
    users = [i for i,s in enumerate(steps) if s.get('source') == 'user']
    if len(users) != 1 or steps[users[0]].get('message') != PROMPT:
        problems.append('User scenario mismatch; require exactly one user step with exact message: ' + PROMPT)
    agents = [s for s in steps if s.get('source') == 'agent']
    models = sorted({s['model_name'] for s in agents if isinstance(s.get('model_name'), str) and s['model_name'].strip()})
    if not models:
        problems.append('Actual model not recorded; require a nonempty model_name on an observed agent step (no particular model required).')
    if not any(isinstance(s.get('message'), str) and s['message'].strip() and not s.get('tool_calls') for s in agents):
        problems.append('No completed assistant response; require a nonempty agent message without pending tool calls.')
    for s in agents:
        results = {r.get('source_call_id'): r.get('content') for r in s.get('observation', {}).get('results', [])}
        for call in s.get('tool_calls', []):
            if call.get('function_name') != 'lookup_booking':
                continue
            raw = results.get(call.get('tool_call_id'))
            try:
                result = json.loads(raw) if isinstance(raw, str) else raw
            except (ValueError, TypeError):
                result = None
            valid = (call.get('arguments') == {'confirmation_code': 'SKY123'} and result == EXPECTED and isinstance(result, dict) and result.get('retryable') is False)
            lookups.append({'step_id': s.get('step_id'), 'arguments': call.get('arguments'), 'result': result, 'matches_fixture': valid})
            if not valid:
                problems.append('Lookup at step %s did not match fixture. Require arguments exactly {"confirmation_code":"SKY123"} and linked observation exactly %s (no additional booking fields).' % (s.get('step_id'), json.dumps(EXPECTED)))
    if not lookups:
        problems.append('No lookup_booking observed; require a real lookup with arguments exactly {"confirmation_code":"SKY123"} and linked observation exactly ' + json.dumps(EXPECTED) + ', with no booking fields.')
except Exception as exc:
    problems.append('Cannot inspect trajectory: %s. Require parseable ATIF-v1.7, nonempty agent identity, sequential steps 1..N, exactly one user message %r, observed model_name, completed assistant message, and lookup_booking({"confirmation_code":"SKY123"}) linked to exactly %s.' % (exc, PROMPT, json.dumps(EXPECTED)))
score = 0.0 if problems else 1.0
feedback = ' '.join(problems) if problems else ('Actual synthetic nonretryable booking-outage fixture verified for SKY123; observed models: ' + ', '.join(models) + '. Semantic response quality is assessed by judge-outage-handling; a passing baseline is resilience evidence, not an agent failure.')
(OUT / 'reward.json').write_text(json.dumps({'verify-outage-fixture': score}))
(OUT / 'relai_global_metrics.json').write_text(json.dumps({'feedback': {'verify-outage-fixture': feedback}}))
(OUT / 'baseline-evidence.json').write_text(json.dumps({'actual_models': models, 'lookup_observations': lookups, 'fixture_score': score, 'trajectory': '/logs/agent/trajectory.json', 'disclosure': 'Deliberately injected synthetic controlled-tool-failure, not a real upstream incident.'}, indent=2))
