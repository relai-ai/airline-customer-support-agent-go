"""Provision only temporary playground credentials for CLI and nested trials."""
import json
import os
from pathlib import Path


def provision_cli(workspace: Path, session: dict) -> dict:
    required = {'RELAI_API_BASE', 'RELAI_TEAM_API_KEY', 'PLAYGROUND_API_BASE', 'PLAYGROUND_SESSION_ID'}
    if not required.issubset(session) or any(not isinstance(v, str) or '\n' in v or '\r' in v for v in session.values()):
        raise ValueError('Session environment is incomplete')
    env = {**session, 'AIRLINE_MODEL_PROVIDER': 'relai',
           'RELAI_API_URL': session['RELAI_API_BASE'], 'RELAI_API_KEY': session['RELAI_TEAM_API_KEY']}
    root = workspace / '.relai'
    if root.is_symlink():
        raise ValueError('Session configuration cannot use symlinks')
    root.mkdir(parents=True, exist_ok=True)
    config = root / 'config.toml'
    simulator = root / 'simulator.env'
    for file in (config, simulator):
        # Refuse links before writing private session material.
        if file.is_symlink():
            raise ValueError('Session configuration cannot use symlinks')
    def private_write(file, content):
        fd = os.open(file, os.O_WRONLY | os.O_CREAT | os.O_TRUNC | os.O_NOFOLLOW, 0o600)
        with os.fdopen(fd, 'w') as output:
            os.fchmod(output.fileno(), 0o600)
            output.write(content)
    private_write(config, '[api]\nurl = ' + json.dumps(env['RELAI_API_BASE']) + '\nkey = ' + json.dumps(env['RELAI_TEAM_API_KEY']) + '\n')
    names = ['AIRLINE_MODEL_PROVIDER', 'RELAI_API_BASE', 'RELAI_API_KEY',
             'RELAI_API_URL', 'RELAI_SESSION_MODEL', 'RELAI_TEAM_API_KEY']
    private_write(simulator, ''.join(name + '=' + env[name] + '\n' for name in names if name in env))
    return env
