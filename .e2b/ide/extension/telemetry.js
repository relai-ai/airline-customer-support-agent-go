const crypto = require("node:crypto");
const actions = new Set([
  "recorded",
  "folder",
  "source",
  "mode",
  "choose",
  "try-agent",
  "send",
  "approve",
  "continue",
  "refresh",
  "stop",
  "cancel",
  "assistant",
  "terminal",
  "evidence",
  "review-change",
  "changes",
]);
/** Best-effort structured events through the existing session gateway. No webview networking or secrets. */
function createTelemetry(
  readEnvironment,
  request = fetch,
  now = () => performance.now(),
) {
  const start = now(),
    launchId = crypto.randomUUID();
  let environment,
    pending = 0;
  const inflight = new Set();
  function report(event, detail = {}) {
    if (
      pending >= 8 ||
      ![
        "extension_activated",
        "relai_panel_ready",
        "playground_action",
        "playground_failed",
      ].includes(event)
    )
      return;
    if (event === "playground_action" && !actions.has(detail.action)) return;
    const payload = {
      event_id: crypto.randomUUID(),
      launch_id: launchId,
      event,
      duration_ms: Math.min(7200000, Math.max(0, now() - start)),
      ...(event === "playground_action" ? { action: detail.action } : {}),
      ...(event === "playground_failed"
        ? {
            stage: ["action", "panel", "extension"].includes(detail.stage)
              ? detail.stage
              : "action",
          }
        : {}),
    };
    pending++;
    const task = (async () => {
      environment ??= Promise.resolve().then(readEnvironment);
      const env = await environment;
      const base = new URL(env.PLAYGROUND_API_BASE);
      if (
        base.protocol !== "https:" ||
        base.username ||
        base.password ||
        base.search ||
        base.hash ||
        !/^[a-f0-9-]{36}$/.test(env.PLAYGROUND_SESSION_ID) ||
        !/^pg_[A-Za-z0-9_-]+$/.test(env.RELAI_TEAM_API_KEY)
      )
        return;
      const url =
        base.href.replace(/\/$/, "") +
        "/sessions/" +
        env.PLAYGROUND_SESSION_ID +
        "/events";
      const response = await request(url, {
        method: "POST",
        headers: {
          Authorization: "Token " + env.RELAI_TEAM_API_KEY,
          "Content-Type": "application/json",
        },
        redirect: "error",
        signal: AbortSignal.timeout(2000),
        body: JSON.stringify({
          ...payload,
          session_id: env.PLAYGROUND_SESSION_ID,
        }),
      });
      // Do not retain response bodies or delay IDE work for acknowledgement.
      await response.body?.cancel();
    })()
      .catch(() => {})
      .finally(() => {
        pending--;
        inflight.delete(task);
      });
    inflight.add(task);
  }
  return { report, whenIdle: () => Promise.all([...inflight]) };
}
module.exports = { createTelemetry, actions };
