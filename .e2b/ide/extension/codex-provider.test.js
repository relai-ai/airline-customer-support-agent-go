const test = require("node:test");
const assert = require("node:assert/strict");
const { startProvider } = require("./codex-provider");
test("native reasoning and custom tool state survives scoped gateway and SSE without translation", async () => {
  let calls = 0;
  const input = [
    { type: "reasoning", encrypted_content: "opaque" },
    {
      type: "custom_tool_call",
      call_id: "patch",
      name: "apply_patch",
      input: "real patch",
    },
    { type: "custom_tool_call_output", call_id: "patch", output: "Done" },
  ];
  const output = [
    {
      id: "r",
      type: "reasoning",
      encrypted_content: "opaque_next",
      summary: [],
    },
    {
      id: "m",
      type: "message",
      role: "assistant",
      content: [
        { type: "output_text", text: "Actual result", annotations: [] },
        { type: "output_text", text: "Second real part", annotations: [] },
      ],
    },
  ];
  const env = {
    RELAI_SESSION_MODEL: "openai/gpt-6.1-sol",
    RELAI_API_BASE: "https://gateway.example/runtime/session/",
    RELAI_TEAM_API_KEY: "dummy-scope-token",
  };
  const provider = await startProvider(env, async (url, options) => {
    calls++;
    assert.equal(
      String(url),
      "https://gateway.example/runtime/session/api/v1/model-proxy/codex/responses/",
    );
    assert.equal(options.headers.Authorization, "Token dummy-scope-token");
    assert.deepEqual(JSON.parse(options.body).input, input);
    assert.equal(JSON.parse(options.body).stream, false);
    assert.equal(options.redirect, "error");
    return Response.json({
      id: "resp",
      status: "completed",
      output,
      usage: { input_tokens: 10, output_tokens: 5, total_tokens: 15 },
    });
  });
  try {
    assert.equal(
      (await fetch(provider.url + "/responses", { method: "POST", body: "{}" }))
        .status,
      401,
    );
    const result = await fetch(provider.url + "/responses", {
      method: "POST",
      headers: { Authorization: "Bearer " + provider.token },
      body: JSON.stringify({ input, tools: [], stream: true }),
    });
    const events = (await result.text())
      .split("\n")
      .filter((l) => l.startsWith("data: "))
      .map((l) => JSON.parse(l.slice(6)));
    assert.deepEqual(events.at(-1).response.output, output);
    assert.equal(
      "input" in
        events.find(
          (e) =>
            e.type === "response.output_item.added" &&
            e.item.type === "reasoning",
        ).item,
      false,
    );
    assert.equal(
      events.filter((e) => e.type === "response.output_text.delta").length,
      2,
    );
    assert.equal(events.at(-1).response.usage.total_tokens, 15);
    assert.equal(
      (
        await fetch(provider.url + "/responses/compact", {
          method: "POST",
          headers: { Authorization: "Bearer " + provider.token },
          body: "{}",
        })
      ).status,
      404,
    );
    assert.equal(calls, 1);
  } finally {
    provider.close();
  }
});
test("backend denial is a readable error and never a completed result", async () => {
  const provider = await startProvider(
    {
      RELAI_SESSION_MODEL: "openai/model",
      RELAI_API_BASE: "https://example.test/",
      RELAI_TEAM_API_KEY: "scope",
    },
    async () =>
      Response.json(
        { error: { message: "Session budget exhausted." } },
        { status: 429 },
      ),
  );
  try {
    const result = await fetch(provider.url + "/responses", {
      method: "POST",
      headers: { Authorization: "Bearer " + provider.token },
      body: JSON.stringify({ input: "hello", stream: true }),
    });
    assert.equal(result.status, 502);
    assert.equal(
      (await result.json()).error.message,
      "Session budget exhausted.",
    );
  } finally {
    provider.close();
  }
});
