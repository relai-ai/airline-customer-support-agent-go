The playground conversation runs the real airline adapter through `try-agent.py`, using temporary RELAI session credentials supplied by the service. No provider key or personal login is needed. CLI preferences are initialized with the supported `relai setup --theme light` command; credentials stay in its child environment.

`optimizer-scope.json` is the unmodified nonsecret output of an interactive `relai optimize` scope selection approved for this prepared sample. It allows prompt/code/tool changes, keeps models fixed, prohibits additional paid tools, and disables cost optimization. Do not hand-edit CLI-owned scope state. Reconfigure it through RELAI's interactive terminal if the prepared sample's scope changes.

IDE startup also writes the issued temporary session settings into the ignored
CLI config and simulator environment with mode 0600. Nested test validation can
therefore run without asking users to configure keys. On local Docker Desktop,
persona trials remap the host alias to the inner Docker host; a bridge bound only
to the private inner Docker gateway preserves local model routing. It publishes
no workstation port. Hosted HTTPS gateways do not start this bridge.

Checks: `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s .e2b/ide -p 'test_*.py'`.
