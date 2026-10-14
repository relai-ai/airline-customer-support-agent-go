The playground conversation runs the real airline adapter through `try-agent.py`, using temporary RELAI session credentials supplied by the service. No provider key or personal login is needed. CLI preferences are initialized with the supported `relai setup --theme light` command; credentials stay in its child environment.

`optimizer-scope.json` is the unmodified nonsecret output of an interactive `relai optimize` scope selection approved for this prepared sample. It allows prompt/code/tool changes, keeps models fixed, prohibits additional paid tools, and disables cost optimization. Do not hand-edit CLI-owned scope state. Reconfigure it through RELAI's interactive terminal if the prepared sample's scope changes.

IDE startup also writes the issued temporary session settings into the ignored
CLI config and simulator environment with mode 0600. Nested test validation can
therefore run without asking users to configure keys. On local Docker Desktop,
persona trials remap the host alias to the inner Docker host; a bridge bound only
to the private inner Docker gateway preserves local model routing. It publishes
no workstation port. Hosted HTTPS gateways do not start this bridge.

Checks: `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s .e2b/ide -p 'test_*.py'`.


## Cloudflare custom events

The extension sends `extension_activated`, `relai_panel_ready`, fixed-name action requests and failures to the authenticated `/sessions/:id/events` API using its existing workspace credential. The Cloudflare Worker writes validated receipts to Analytics Engine. `relai_panel_ready` is acknowledged by the panel after initial data rendering and two animation-frame callbacks, not inferred from socket connection. Durations are measured from this extension activation; they do not include the preceding code-server connection delay. Action events record intent, not completion.

No prompt, terminal text, source, arbitrary fields or credential is included in the event body. Reporting never blocks normal UI/commands, times out after two seconds, has no retries and caps eight in-flight events. The existing Dockerfile copies this module with the extension directory; rebuild the E2B template for new sandboxes. Existing sandboxes are not updated. Requires the companion dashboard API/Cloudflare Analytics Engine binding before events are stored.

Run `node --test .e2b/ide/extension/telemetry.test.js .e2b/ide/extension/workflow.test.js .e2b/ide/extension/recorded.test.js`. These checks use local fixtures only, without live model calls or telemetry delivery.
