const { test } = require("node:test");
const assert = require("node:assert/strict");
const { workflowView, shouldPrefill } = require("./workflow");
test("handoff advances only from persisted assistant completion and passes honest baselines to verification", () => {
  assert.equal(workflowView({}).step, "prepare");
  assert.equal(workflowView({ step: "evaluate" }).step, "evaluate");
  assert.equal(
    workflowView({ step: "evaluate", nextStep: "optimize" }).step,
    "optimize",
  );
  assert.equal(
    workflowView({ step: "evaluate", nextStep: "verify" }).step,
    "verify",
  );
  assert.equal(
    workflowView({ step: "verify", nextStep: "compare" }).step,
    "compare",
  );
  assert.equal(workflowView({ nextStep: "complete" }).prompt, "");
});
test("polling and step transitions preserve edited and unsent drafts", () => {
  assert.equal(shouldPrefill("My follow-up question", "Old suggestion"), false);
  assert.equal(shouldPrefill("Old suggestion", "Old suggestion"), true);
  assert.equal(shouldPrefill("", "Old suggestion"), true);
});
test("packaged webview script compiles after helper injection", () => {
  const fs = require("node:fs");
  const path = require("node:path");
  const vm = require("node:vm");
  const html = fs
    .readFileSync(path.join(__dirname, "panel.html"), "utf8")
    .replace(
      "WORKFLOW_HELPERS",
      fs.readFileSync(path.join(__dirname, "workflow.js"), "utf8"),
    );
  const script = html.match(/<script nonce="NONCE">([\s\S]*?)<\/script>/)[1];
  assert.doesNotThrow(() => new vm.Script(script));
});
test("short editable requests cover real evaluation and verified retry, without a report prompt", () => {
  const { workflowSteps } = require("./workflow");
  assert.ok(workflowSteps.prepare.prompt.includes("check this behavior"));
  assert.ok(workflowSteps.evaluate.prompt.includes("Run the RELAI test"));
  assert.ok(workflowSteps.verify.prompt.includes("If verified, apply it here"));
  assert.ok(Object.values(workflowSteps).every((s) => s.prompt.length < 220));
});

test("live output follows new lines and completed output retains its reading position", () => {
  const { restoreOutputScroll } = require("./workflow");
  const output = { scrollTop: 0, scrollHeight: 900 };
  restoreOutputScroll(output, 120, true);
  assert.equal(output.scrollTop, 900);
  output.scrollHeight = 1200;
  restoreOutputScroll(output, 900, true);
  assert.equal(output.scrollTop, 1200);
  restoreOutputScroll(output, 120, false);
  assert.equal(output.scrollTop, 120);
});

test("recorded and edited requests dispatch distinct actions, and apply is explicit", () => {
  const fs = require("node:fs"),
    vm = require("node:vm"),
    path = require("node:path");
  const sent = [],
    handlers = {};
  class Element {
    constructor(tag = "div") {
      this.tag = tag;
      this.tagName = tag.toUpperCase();
      this.children = [];
      this.value = "";
      this.textContent = "";
      this.scrollHeight = 100;
      this.scrollTop = 0;
      this.clientHeight = 100;
      this.dataset = {};
    }
    append(...items) {
      this.children.push(...items);
    }
    prepend(...items) {
      this.children.unshift(...items);
    }
    replaceChildren(...items) {
      this.children = items;
    }
    querySelectorAll() {
      return [];
    }
    focus() {}
    setAttribute(name, value) {
      this[name] = value;
    }
    click() {
      this.onclick?.();
    }
  }
  const elements = new Map();
  const doc = {
    getElementById: (id) => {
      if (!elements.has(id)) elements.set(id, new Element());
      return elements.get(id);
    },
    createElement: (tag) => new Element(tag),
    querySelectorAll: () => [],
  };
  const html = fs
    .readFileSync(path.join(__dirname, "panel.html"), "utf8")
    .replace(
      "WORKFLOW_HELPERS",
      fs.readFileSync(path.join(__dirname, "workflow.js"), "utf8"),
    );
  const context = {
    document: doc,
    window: {
      addEventListener: (_event, fn) => {
        handlers.message = fn;
      },
    },
    acquireVsCodeApi: () => ({
      getState: () => null,
      setState: () => {},
      postMessage: (value) => sent.push(value),
    }),
    setTimeout: () => 1,
    clearTimeout: () => {},
  };
  vm.runInNewContext(
    html.match(/<script nonce="NONCE">([\s\S]*?)<\/script>/)[1],
    context,
  );
  const flatten = (n) => [n, ...n.children.flatMap(flatten)];
  const formatted = flatten(
    context.formatted(
      "A test defines:\n- **Input:** a request\n- **Environment:** tools\n- **Evaluators:** `judge`\n\n<script>alert(1)</script>",
    ),
  );
  assert.equal(formatted.filter((n) => n.tag === "li").length, 3);
  assert.deepEqual(
    formatted.filter((n) => n.tag === "strong").map((n) => n.textContent),
    ["Input:", "Environment:", "Evaluators:"],
  );
  assert.equal(formatted.find((n) => n.tag === "code").textContent, "judge");
  assert.equal(
    formatted.some((n) => n.tag === "script"),
    false,
  );
  assert.ok(
    formatted.some((n) => n.textContent === "<script>alert(1)</script>"),
  );
  const { prompts } = require("./recorded");
  const recording = {
    verificationStatus: "verified",
    prompts,
    steps: prompts
      .slice(0, 3)
      .map(() => ({ text: "Real evidence", durationSeconds: 60 })),
    turns: [
      { user: "First traveler", agent: "First answer" },
      { user: "Second traveler", agent: "Second answer" },
    ],
    comparison: {
      before: "baseline",
      after: "verified",
      regressions: "passed",
    },
    diff: "patch",
  };
  handlers.message({
    data: { type: "loaded", recording, recordedProgress: 0 },
  });
  assert.equal(elements.get("prompt").value, prompts[0]);
  elements.get("send").click();
  assert.equal(sent.at(-1).action, "recorded");
  assert.equal(sent.at(-1).step, 0);
  handlers.message({ data: { type: "recorded", recordedProgress: 1 } });
  assert.equal(elements.get("prompt").value, prompts[1]);
  handlers.message({ data: { type: "recorded", recordedProgress: 2 } });
  flatten(elements.get("history"))
    .find((n) => n.textContent === "▸ Explore the simulation")
    .click();
  assert.ok(
    flatten(elements.get("history")).some(
      (n) => n.textContent === "EXCHANGE 1 OF 2",
    ),
  );
  flatten(elements.get("history"))
    .find((n) => n.textContent === "Next exchange →")
    .click();
  assert.ok(
    flatten(elements.get("history")).some(
      (n) => n.textContent === "EXCHANGE 2 OF 2",
    ),
  );
  assert.equal(
    flatten(elements.get("history")).find(
      (n) => n.textContent === "Next exchange →",
    ).disabled,
    true,
  );
  elements.get("prompt").value = "My own question";
  elements.get("prompt").oninput();
  assert.equal(elements.get("send")["aria-label"], "Send message");
  const previousCount = sent.length;
  elements.get("prompt").onkeydown({ key: "Enter", shiftKey: true, preventDefault() {} });
  assert.equal(sent.length, previousCount);
  elements.get("prompt").onkeydown({ key: "Enter", shiftKey: false, isComposing: false, preventDefault() {} });
  assert.equal(sent.at(-1).action, "send");
  assert.equal(sent.at(-1).prompt, "My own question");
  assert.equal(elements.get("prompt").value, "");
  assert.ok(flatten(elements.get("history")).some((n) => n.className === "spinner"));
  assert.equal(flatten(elements.get("history")).filter((n) => n.tag === "span" && n.textContent === "My own question").length, 1);
  assert.equal(elements.get("send")["aria-label"], "Stop");
  assert.equal(elements.get("send")["data-busy"], "true");
  elements.get("prompt").value = "My next draft";
  elements.get("prompt").oninput();
  handlers.message({ data: { type: "error", message: "Runtime unavailable" } });
  assert.equal(elements.get("status").textContent, "Runtime unavailable");
  assert.equal(elements.get("prompt").value, "My next draft");
  assert.equal(flatten(elements.get("history")).filter((n) => n.className === "spinner").length, 0);
  const oldMessages = [{ role: "user", content: "My own question" }, { role: "assistant", content: "Old answer" }];
  handlers.message({ data: { type: "state", investigation: { messages: oldMessages, jobs: [] } } });
  elements.get("prompt").value = "My own question";
  elements.get("prompt").oninput();
  elements.get("send").click();
  handlers.message({ data: { type: "state", investigation: { messages: oldMessages, jobs: [] } } });
  assert.equal(elements.get("send")["aria-label"], "Stop");
  elements.get("prompt").value = "Keep this draft";
  elements.get("prompt").oninput();
  const acceptedMessages = [...oldMessages, { role: "user", content: "My own question" }];
  handlers.message({ data: { type: "state", investigation: { messages: acceptedMessages, pendingTurn: {}, jobs: [] } } });
  assert.equal(elements.get("prompt").value, "Keep this draft");
  assert.equal(flatten(elements.get("history")).filter((n) => n.tag === "span" && n.textContent === "My own question").length, 2);
  handlers.message({ data: { type: "state", investigation: { messages: [...acceptedMessages, { role: "assistant", content: "New answer" }], jobs: [] } } });
  assert.equal(flatten(elements.get("history")).filter((n) => n.className === "spinner").length, 0);
  assert.equal(elements.get("send")["data-busy"], "false");
  handlers.message({
    data: { type: "loaded", recording, recordedProgress: 3 },
  });
  flatten(elements.get("history")).find((n) => n.textContent === "Review code changes ↗").click();
  assert.equal(sent.at(-1).action, "changes");
  assert.equal(sent.at(-1).recorded, true);
  elements.get("send").click();
  assert.equal(sent.at(-1).action, "recorded");
  assert.equal(sent.at(-1).step, 3);
});

test("command failures are human-readable and JSON-escaped logs become normal text", () => {
  const { commandResultView } = require("./workflow");
  const result = commandResultView({ status: "error", content: JSON.stringify({ status: "failed", exitCode: 1, output: "Optimizer\nPOST /api/v1/backend-agent/optimizer/review-anchor/ failed with HTTP 503 after 3 attempts" }) });
  assert.match(result.summary, /unavailable during optimization review/);
  assert.ok(!result.summary.includes('"status"'));
  assert.ok(result.details.includes("Optimizer\nPOST"));
  assert.ok(!result.details.includes('\\nPOST'));
  assert.match(commandResultView({ status: "error", content: { exitCode: 1, output: "HTTP 413 Payload Too Large" } }).summary, /payload limit/);
  assert.match(commandResultView({ status: "error", content: { timedOut: true, output: "waiting" } }).summary, /timed out/);
  assert.match(commandResultView({ status: "success", content: "Comparison recorded." }).summary, /Comparison recorded/);
});
