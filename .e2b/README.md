# Airline E2B environment

This template prepares the airline source, Go, Node, Python, Codex, RELAI, uv,
Docker and Compose. It is an environment checkpoint, not the optimized onboarding
sample: model access, RELAI session seeding and the second failure are separate.

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
