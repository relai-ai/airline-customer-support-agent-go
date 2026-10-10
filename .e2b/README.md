# Airline E2B environment

This template prepares the airline source, Go, Node, Python, Codex, RELAI, uv,
Docker and Compose, plus the optimized source and sanitized regression evidence.
Model access, RELAI session seeding and the second demonstrated failure remain
separate. See `demo/README.md` for measured results and limitations.

Authenticate on the host with `npx --yes @e2b/cli@2.21.2 auth login`, then run
`bash .e2b/build.sh`. Record the resulting template/build IDs before promoting it.
The development alias is mutable; production must pin a verified build.

Builds never run model calls or copy host credentials. Only explicitly listed
source paths are uploaded. Each runtime session must receive restricted,
expiring credentials from the playground service. Never snapshot a session after
injecting credentials or visitor data.

Create runtime sandboxes with public traffic disabled and a finite kill timeout.
The dashboard service must authenticate users and proxy any editor access; a raw
E2B URL is not a ready onboarding experience. The service's `probe:sandbox` checks
prerequisites and destroys its disposable sandbox afterward.

Startup risk: Docker starts inside the VM. The readiness command checks its live
daemon rather than accepting a successful image build as proof of readiness.

## Verified development build

On 2026-10-10, template `i1atd35xm9d495f89hp7` / build
`01db49e9-d807-41b9-874b-702976b9758f` passed startup, fresh-instance reset,
checkpoint integrity, and automatic 30-second kill verification in RELAI
`playground-dev`. All three probes were cleaned up. See `verification.json`.
The CLI remote-exec operation extends the sandbox timeout; do not use it during
an expiry-only probe. These results do not validate a browser editor or model
credentials. Build selection is guarded against the personal E2B project.

## Follow-up challenge package (pending new build)

The template definition now includes three labeled challenge scenarios and prepared
RELAI test definitions. Recorded GPT-4.1 baselines are under `demo/challenges`:
wrong fare attribution and invalid seat mutation were observed; the controlled
outage was handled without fabrication. The existing verified build above does
not contain this new package. Rebuild and validate it after backend credential and
requested-model integration are ready. The package does not contain host auth,
`.relai/config.toml`, `.relai/simulator.env`, or internal run directories.

## Browser IDE (requires a new template build)

The image now includes pinned code-server 4.141.0 and the RELAI Playground
extension. The service injects a temporary gateway token outside the repository,
then starts the IDE. Browser HTTP/WebSocket access is authenticated by the
playground service on an isolated per-session origin; code-server itself is
reachable only through that proxy. Do not enable public E2B traffic.

The panel offers three scenarios with their common agent-problem categories,
then one choice between the RELAI playground coding assistant and a manual
terminal. The assistant simulates the workflow used with Codex, Claude Code or
Cursor; it is not an embedded instance of those products. Its editable starting
request authorizes the selected scenario's real test, evaluation, optimization
when needed, and verification in the temporary sample. Commands continue without
repeated approval clicks; Stop revokes continuation and cancels running commands.
Older sessions without workflow consent still require command approval.

Routine commands and output stay in collapsed details. The final comparison
quotes completed command results, with source command receipts and explicit
missing checks. Model interpretation is labeled separately; no score is inferred
from command success and a passing baseline does not force optimization.
The terminal, files, tests, and changes are real. Recorded GPT-4.1 scores are not evidence of current
production-profile performance. The Go agent supports explicit
`AIRLINE_MODEL_PROVIDER=relai` routing through the same restricted gateway.

Build a separate local image for fixture testing:
`docker build --platform linux/amd64 -f .e2b/Dockerfile -t relai-airline-ide:local .`
Run the dashboard repository's `scripts/ide-smoke.ts` with Node/tsx to verify
startup, the extension, WebSockets and access revocation without live AI calls.
The earlier E2B build IDs above do not contain this IDE.
