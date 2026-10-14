const { test } = require("node:test");
const assert = require("node:assert/strict");
const { createTelemetry } = require("./telemetry");
const env = {
  PLAYGROUND_API_BASE: "https://platform.relai.ai/playground-api",
  PLAYGROUND_SESSION_ID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
  RELAI_TEAM_API_KEY: "pg_fixture",
};
test("sends a fixed vocabulary through session auth and excludes arbitrary private fields", async () => {
  const calls = [];
  const telemetry = createTelemetry(
    async () => env,
    async (url, init) => {
      calls.push([url, init]);
      return new Response(null, { status: 204 });
    },
  );
  telemetry.report("playground_action", {
    action: "send",
    prompt: "secret prompt",
    token: "secret",
  });
  telemetry.report("arbitrary-event");
  telemetry.report("playground_action", { action: "arbitrary" });
  await telemetry.whenIdle();
  assert.equal(calls.length, 1);
  assert.equal(
    calls[0][0],
    env.PLAYGROUND_API_BASE +
      "/sessions/" +
      env.PLAYGROUND_SESSION_ID +
      "/events",
  );
  assert.equal(calls[0][1].headers.Authorization, "Token pg_fixture");
  const payload = JSON.parse(calls[0][1].body);
  assert.equal(payload.action, "send");
  assert.equal(payload.prompt, undefined);
  assert.equal(payload.token, undefined);
});
test("reporting failures never throw and concurrent reporting is bounded", async () => {
  let calls = 0;
  const telemetry = createTelemetry(
    async () => env,
    async () => {
      calls++;
      throw Error("offline");
    },
  );
  for (let i = 0; i < 30; i++)
    assert.equal(telemetry.report("extension_activated"), undefined);
  await telemetry.whenIdle();
  assert.equal(calls, 8);
});
test("malformed environment cannot send a credential to an unsafe URL", async () => {
  let calls = 0;
  const telemetry = createTelemetry(
    async () => ({ ...env, PLAYGROUND_API_BASE: "http://example.test" }),
    async () => {
      calls++;
    },
  );
  telemetry.report("extension_activated");
  await telemetry.whenIdle();
  assert.equal(calls, 0);
});
