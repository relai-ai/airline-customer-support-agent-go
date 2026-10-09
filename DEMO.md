# Staff-guided airline playground

## Readiness

This configuration prepares a browser editor, terminal, Go, Docker, uv, Codex
CLI, and the RELAI plugin. It does **not** contain a verified initialized or
optimized RELAI workspace. Do not present installation success as demo readiness.
The existing README includes historical learning-env commands; use the installed
RELAI plugin and current command help for the demonstration.

## Launch

1. In GitHub, select the branch containing this configuration, then **Code →
   Codespaces → Create codespace**. Check the billing owner. Use a staff-owned
   codespace on the demo laptop; do not give visitors the staff GitHub login.
2. Wait for the setup hook. Open a new terminal and run `relai setup --check`.
   Run `bash .devcontainer/verify.sh` to check installed tools, Docker readiness,
   and the Go build, vet, and tests. This does not make model calls.
3. Run `relai setup --agent codex` and complete browser authorization with the
   intended demo RELAI account/workspace. Never copy personal OAuth files into
   the repository or prebuild.
4. Authenticate Codex using its supported login flow, or configure a limited
   demo provider credential. Configure the sample agent's provider credential
   separately. Use Codespaces secrets; never put values in this document or Git.
   Terminal users can read environment variables, so these are not hidden keys.
5. Open `codex` in the terminal and verify the RELAI plugin is available.

## Prepare before visitors arrive

- Use the RELAI plugin to initialize this sample and prepare the first validated
  optimization. Inspect the generated files and verify the associated backend
  agent/tests/results belong to the demo identity.
- Choose a second, reproducible failure and save a sanitized trace. A candidate
  is changing a booking only if the change is free, but this is not a verified
  failure in this checkout. Confirm it before using it in the demo.
- Save a known-good checkpoint and record the backend state it requires. Do not
  commit credentials, local configuration, private traces or generated caches.
- Rehearse the full next loop and record its actual duration. Keep an explicitly
  labeled recorded result as a fallback if the live service is unavailable.

Suggested visitor prompt, after a real failure is selected:

> Investigate the supplied failure trace. Use RELAI to create a regression test,
> improve the agent, and verify the fix while preserving the existing tests.
> Explain the evidence and show the changed files.

The visitor starts after setup and the first optimization, with the file explorer
and coding-agent terminal visible together. Staff explains which artifacts are
preloaded and which work is happening live.

## Reusing this setup for other agents

This repository owns the airline environment only. A shared playground service
can live in the dashboard monorepo and select this repository as one template.
Other agents should provide their own workspace configuration and demo fixtures.
Keep session-management services and cloud deployment infrastructure out of this
agent repository.

Record the full commit SHA used for each rehearsed demo (`git rev-parse HEAD`).
Use that same revision for subsequent sessions; a moving branch is not a stable
demo checkpoint. The base image and devcontainer features still receive updates,
so retain the rehearsed workspace and check recorded tool versions after rebuilding.

## Between sessions

For a fresh visitor session, create a **new** Codespace from the rehearsed
revision. Do not share staff account credentials or reuse a previous visitor's
modified files. GitHub account access and billing still apply; the workspace URL
is not an anonymous visitor link.

Preserve useful feedback and sanitized results. Restore the prepared checkpoint
only in the disposable demo workspace, and restore or recreate corresponding
backend state as needed. A Git reset alone does not reset backend records.
Use independent workspaces and validated backend identities for concurrent demos.
Stop unused codespaces and revoke temporary credentials after the event.
Delete disposable Codespaces when finished to remove their files and storage;
stopping alone preserves them. GitHub idle timeout is not an absolute session
expiration, and deleting a Codespace does not delete RELAI backend records or
revoke provider credentials. Those steps remain staff responsibilities until the
shared playground service implements them.

## Prebuilds and versions

After one successful cold launch, enable a prebuild for the selected branch under
repository **Settings → Codespaces → Prebuilds**. Open and verify the codespace
before the event. Prebuilds do not complete interactive login or RELAI preparation.

Codex and uv are version-pinned. The official RELAI installer is checksum-pinned
but downloads the current production CLI; its binary is not version-pinned.
Installed versions are recorded in `~/.local/share/relai-demo/tool-versions.txt`.
Retain the rehearsed workspace for the event and rehearse again after any rebuild
or CLI update. If the installer changes, review it before updating its checksum.

## Verification

Run `go build ./...`, `go vet ./...`, `go test ./...`, and `bash -n start.sh`.
Check `.devcontainer/setup.sh` and `.devcontainer/verify.sh` with `bash -n` as well.
Then verify a real Codespaces
cold launch, Docker readiness, login, plugin availability, and the complete live
loop. Credentials and model usage are required for the live loop; installation
and unit checks do not prove it.

Startup risk: the devcontainer lifecycle hook and Docker feature affect environment
startup. Verify them in Codespaces; local Go tests alone are insufficient. This
repository has no Django/ASGI service, so the backend `/health/` smoke path does
not apply to this configuration change.
