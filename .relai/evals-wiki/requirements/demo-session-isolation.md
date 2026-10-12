# Isolate demo sessions and credentials

Images, prebuilds, and committed checkpoints must exclude personal authentication, provider keys, private traces, and local runtime configuration. Runtime sessions must use their own restricted credentials; E2B credentials must expire. Never snapshot a session after injecting credentials or visitor data.

Fresh visitors must receive new disposable workspaces from the rehearsed revision or same immutable E2B build, not a previous visitor's modified or authenticated workspace. Restore corresponding backend state separately when needed; Git reset alone is insufficient. Concurrent demos must use independent workspaces and validated backend identities.

E2B sandboxes must disable public traffic and have finite kill timeouts. The dashboard service must authenticate users and proxy editor access. Staff must clean up disposable resources and revoke temporary credentials; stopping a Codespace is not deletion or revocation.

## Evidence

[Playground runtime](relai://repo/.e2b/README.md).
