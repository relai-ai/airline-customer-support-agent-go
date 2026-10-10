#!/usr/bin/env python3
"""Start the IDE only after the service provisions a session-scoped environment."""
import json, os, pathlib, subprocess, time, urllib.request
root = pathlib.Path('/home/user/.playground')
env = json.loads((root / 'environment.json').read_text())
required = {'RELAI_API_BASE', 'RELAI_TEAM_API_KEY', 'PLAYGROUND_API_BASE', 'PLAYGROUND_SESSION_ID'}
if not required.issubset(env) or any(not isinstance(v, str) for v in env.values()):
    raise SystemExit('Session environment is incomplete')
# Idempotent retries: don't start another server. Credentials remain session-scoped.
try:
    urllib.request.urlopen('http://127.0.0.1:8080/healthz', timeout=1)
    raise SystemExit(0)
except OSError:
    pass
settings = pathlib.Path('/home/user/.local/share/code-server/User/settings.json')
settings.parent.mkdir(parents=True, exist_ok=True)
settings.write_text(json.dumps({'security.workspace.trust.enabled': False, 'workbench.startupEditor': 'none', 'chat.disableAIFeatures': True}))
log = (root / 'ide.log').open('ab')
subprocess.Popen(['code-server', '--bind-addr', '0.0.0.0:8080', '--auth', 'none',
                  '--disable-telemetry', '--disable-update-check', '--disable-proxy',
                  '--extensions-dir', '/opt/playground/extensions', '/home/user/airline'],
                 env={**os.environ, **env, 'AIRLINE_MODEL_PROVIDER': 'relai'},
                 stdin=subprocess.DEVNULL, stdout=log, stderr=log, start_new_session=True)
for _ in range(60):
    try:
        urllib.request.urlopen('http://127.0.0.1:8080/healthz', timeout=1)
        break
    except OSError:
        time.sleep(.2)
else:
    raise SystemExit('IDE failed to become ready')
