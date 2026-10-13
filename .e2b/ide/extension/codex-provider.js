// Session-scoped native Responses adapter. No provider keys enter the sandbox.
const http = require("node:http");
const crypto = require("node:crypto");
function emitResponse(res, response) {
  let sequence_number = 0;
  const event = (type, data) =>
    res.write(
      "event: " +
        type +
        "\ndata: " +
        JSON.stringify({ type, sequence_number: sequence_number++, ...data }) +
        "\n\n",
    );
  res.writeHead(200, {
    "Content-Type": "text/event-stream",
    "Cache-Control": "no-store",
  });
  event("response.created", {
    response: { ...response, status: "in_progress", output: [] },
  });
  for (const [output_index, item] of response.output.entries()) {
    event("response.output_item.added", {
      output_index,
      item: {
        ...item,
        ...(item.type === "message"
          ? { content: [] }
          : item.type === "function_call"
            ? { arguments: "" }
            : { input: "" }),
      },
    });
    if (item.type === "message") {
      const part = item.content[0];
      event("response.content_part.added", {
        item_id: item.id,
        output_index,
        content_index: 0,
        part: { ...part, text: "" },
      });
      // The current backend returns a complete response. Emit its real text once,
      // not a fabricated token-by-token animation.
      event("response.output_text.delta", {
        item_id: item.id,
        output_index,
        content_index: 0,
        delta: part.text,
      });
      event("response.output_text.done", {
        item_id: item.id,
        output_index,
        content_index: 0,
        text: part.text,
      });
      event("response.content_part.done", {
        item_id: item.id,
        output_index,
        content_index: 0,
        part,
      });
    } else if (["function_call", "custom_tool_call"].includes(item.type)) {
      const kind =
        item.type === "function_call"
          ? "function_call_arguments"
          : "custom_tool_call_input";
      const value = item.type === "function_call" ? item.arguments : item.input;
      event("response." + kind + ".delta", {
        item_id: item.id,
        output_index,
        delta: value,
      });
      event("response." + kind + ".done", {
        item_id: item.id,
        output_index,
        ...(item.type === "function_call"
          ? { arguments: value }
          : { input: value }),
      });
    }
    event("response.output_item.done", { output_index, item });
  }
  event(
    response.status === "completed"
      ? "response.completed"
      : response.status === "incomplete"
        ? "response.incomplete"
        : "response.failed",
    { response },
  );
  res.end();
}
async function startProvider(env, request = fetch) {
  const token = crypto.randomBytes(32).toString("hex");
  const model = env.RELAI_SESSION_MODEL?.split("/").at(-1);
  if (!model || !env.RELAI_API_BASE || !env.RELAI_TEAM_API_KEY)
    throw Error("Codex session model configuration is missing.");
  const server = http.createServer(async (req, res) => {
    if (req.headers.authorization !== "Bearer " + token) {
      res.writeHead(401);
      res.end();
      return;
    }
    if (req.method !== "POST" || req.url !== "/v1/responses") {
      res.writeHead(404);
      res.end(
        JSON.stringify({
          error: {
            message:
              "This session supports stateless Responses turns; remote compaction is unavailable.",
          },
        }),
      );
      return;
    }
    const abort = new AbortController();
    res.on("close", () => {
      if (!res.writableEnded) abort.abort();
    });
    try {
      let raw = "";
      for await (const chunk of req) {
        raw += chunk;
        if (Buffer.byteLength(raw) > 1024 * 1024) {
          res.writeHead(413);
          res.end();
          return;
        }
      }
      const payload = JSON.parse(raw);
      if (payload.previous_response_id)
        throw Error(
          "Server-stored response chaining is not supported by this session adapter.",
        );
      const upstream = await request(
        new URL("api/v1/model-proxy/codex/responses/", env.RELAI_API_BASE),
        {
          method: "POST",
          headers: {
            Authorization: "Token " + env.RELAI_TEAM_API_KEY,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ ...payload, stream: false, store: false }),
          signal: AbortSignal.any([abort.signal, AbortSignal.timeout(120000)]),
          redirect: "error",
        },
      );
      const response = await upstream.json();
      if (!upstream.ok)
        throw Error(
          response.error?.message ||
            "The scoped RELAI model gateway returned HTTP " +
              upstream.status +
              ".",
        );
      if (!Array.isArray(response.output))
        throw Error(
          "The scoped model gateway returned an invalid Responses result.",
        );
      if (payload.stream) emitResponse(res, response);
      else {
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify(response));
      }
    } catch (error) {
      if (!res.headersSent) {
        res.writeHead(502, { "Content-Type": "application/json" });
        res.end(
          JSON.stringify({
            error: {
              message: error.message,
              type: "codex_session_gateway_error",
            },
          }),
        );
      } else res.end();
    }
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  return {
    url: "http://127.0.0.1:" + server.address().port + "/v1",
    token,
    close: () => server.close(),
  };
}
module.exports = { startProvider, emitResponse };
