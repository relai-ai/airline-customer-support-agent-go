# Airline E2B environment

This template prepares the original airline source, Go, Node, Python, Codex,
RELAI, uv, Docker and Compose. The developer walkthrough stores measured
optimization evidence and its patch separately; only an explicit application
request changes the visitor's agent files. Model access uses the scoped session
gateway, with provider keys kept on the backend.

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

The panel introduces the SkyServe agent and source, EvalsWiki and evaluation folders,
then opens **Code assistant + RELAI**. Its first three editable requests replay a
clearly labeled precomputed run: risk analysis, a genuinely adaptive ten-or-more-turn
seat-change evaluation and simulation, and optimization with independent verification.
Results lead with measured outcomes; commands, transcripts and evidence remain under one collapsed details control. Initialization duration is shown only with a completed `relai init` receipt. Folder cards explicitly open the real repository folders in the Explorer. Proposed changes open in read-only native VS Code diffs before application.
No recorded step starts a live command or displays a fake running indicator.

The fourth request explicitly applies the verified code patch to this workspace.
Source hashes and `git apply --check` protect user edits; conflicts are preserved.
After applying, freeform requests run live through the
session-scoped gateway. Editing a recorded request changes its send action to **Send message**;
it never attaches a recorded answer to a different user prompt. Users may also skip
the replay and use the live conversation or the real terminal directly.

The original source remains unchanged in the template; recorded optimizer changes
are stored separately until application. A recorded result proves only its measured
trial, not universal reliability. The guided example focuses on unvalidated seat
changes. No provider API-key configuration is needed
in the playground. New workspace lifetimes default to 120 minutes when the companion
service and backend lifetime support have been deployed together. Lifetime is
separate from the backend request budget; replay consumes no model requests.
The recording process may use separate normally capped authoring and optimization
sessions. Preserve that provenance and measure independent verification before
publishing an accepted patch.

Build the local image from the verified recording bundle:
`docker build -f .e2b/Dockerfile -t relai-airline-ide:local .`
The build validates the original source hashes and recording integrity. Browser
smoke tests use a fixture model response; they do not establish optimization quality.
Run the dashboard repository's `apps/playground-api/scripts/ide-smoke.ts` with Node/tsx to verify
startup, the extension, WebSockets and access revocation without live AI calls.
The earlier E2B build IDs above do not contain this IDE.


The sample baggage tool currently falls back to standard allowances for unknown
fares. This is existing implementation behavior, not an injected defect; the
prepared unknown-fare test evaluates whether the full agent turn handles it
correctly. A model may still pass, so only live evaluation establishes a failure.
Optimizer setup must use the CLI's interactive scope selection, never a hand-written
scope artifact. Tests and evaluators are not optimizer edit targets.

Extension checks: `node --test .e2b/ide/extension/*.test.js`.
The companion service must support live conversation continuation before promoting
this template. The backend must accept two-hour scoped grants.
