# Airline playground runtime

The E2B image packages the airline agent, Go, Node, Python, Codex, RELAI,
Docker, code-server, and the playground extension. `.e2b/setup.sh` installs
pinned tools without authentication or model calls. The copy allowlist excludes
host credentials, OAuth state, logs, and local RELAI configuration.

## Build and verify

Build locally with:

```sh
docker build -f .e2b/Dockerfile -t relai-airline-ide:local .
```

For E2B, authenticate on the host with
`npx --yes @e2b/cli@2.21.2 auth login`, then run `bash .e2b/build.sh`.
Record the template/build IDs and pin a verified build for release.

Run the offline checks:

```sh
go test -race ./...
go vet ./...
node --test .e2b/ide/extension/*.test.js
node .e2b/ide/extension/validate-recording.js .
python3 -m unittest discover -s .e2b/ide -p 'test_*.py'
```

Image validation checks the recorded patch and exact baseline source hashes.
A successful image build does not prove runtime readiness: start the real
container and verify its Docker daemon, code-server, and extension. The companion
playground service's `ide-smoke.ts` checks browser access, WebSockets, and access
revocation using fixture responses. Hosted startup and live optimization must be
verified separately before promoting a rebuilt template.

## Session integration

Create private sandboxes with a finite kill timeout. The companion service must
authenticate visitors and proxy HTTP/WebSocket access on isolated session origins.
Do not expose raw code-server or E2B URLs publicly.

The service injects restricted, expiring gateway credentials outside the repository.
`session_config.py` configures the CLI and nested simulations; `provider_relai.go`
routes agent calls through that gateway. Provider keys stay on the backend.
`local_gateway.py` bridges Docker-only local trials; hosted HTTPS routing is direct.
The service and backend must both support the configured session lifetime and
live conversation continuation. Reset by destroying the sandbox and creating a
fresh one; never snapshot credentials or visitor data.

## Guided example and evidence

The extension introduces the repository and initialized RELAI context, then guides
one adversarial seat-change example: risk analysis, adaptive evaluation,
optimization, and explicit application. Recorded prompts replay actual measured
results without model calls or source changes. Edited prompts use live conversation.
Code review opens native read-only diffs; application checks source hashes and
`git apply --check` to preserve visitor edits.

`demo/recorded/` contains the measured baseline, optimizer receipts, verified patch,
and independent verification. The aggregate improved from 10% to 90% in that
trial; the semantic judge retained one deduction. The two regression packages in
`demo/checkpoint/tests/` were used by that verification. These are individual
measurements, not universal reliability or evidence for a different baseline.

The source retains the earlier off-topic instructions and optional fixture hook
because they are part of the measured baseline. The guided seat-change example
injects no outage. Replacing that baseline requires a new recording and independent
verification. Optimizer scope comes from the CLI's interactive selection; tests
and evaluators are not optimizer edit targets.

`staging-build-verification.json` is the historical receipt for the October 12
build from `452ebd9`; it does not verify later source or packaging changes.

`demo/checkpoint/results.json` is a historical receipt retained for the companion
service readiness check. `demo/challenges/catalog.json` exposes only the seat-change
scenario for the service catalog endpoint. Neither is a new measurement.
