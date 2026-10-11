#!/usr/bin/env python3
"""Start the IDE only after the service provisions a session-scoped environment."""
import json, os, pathlib, subprocess, time, urllib.request
root = pathlib.Path('/home/user/.playground')
env = json.loads((root / 'environment.json').read_text())
from session_config import provision_cli
env = provision_cli(pathlib.Path('/home/user/airline'), env)
from local_gateway import ensure_bridge
ensure_bridge(env['RELAI_API_BASE'])
# Supported CLI setup runs with the session credential in the child environment,
# never in arguments or a provider-key prompt. Scope is prepared in the template.
setup = subprocess.run(['relai', 'setup', '--theme', 'light'],
                       env={**os.environ, **env}, stdin=subprocess.DEVNULL,
                       capture_output=True, timeout=30)
if setup.returncode:
    raise SystemExit('Session CLI setup failed')
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
