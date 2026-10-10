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
