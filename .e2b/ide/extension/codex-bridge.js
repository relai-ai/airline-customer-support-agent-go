const { spawn } = require("node:child_process");
const fs = require("node:fs/promises");
const readline = require("node:readline");
const { startProvider } = require("./codex-provider");
class CodexBridge {
  constructor(env, root, changed) {
    this.env = env;
    this.root = root;
    this.changed = changed;
    this.state = {
      messages: [],
      jobs: [],
      pendingTurn: false,
      runWorkflow: false,
      engine: "codex",
    };
    this.requests = new Map();
    this.approvals = new Map();
    this.sequence = 0;
    this.file = "/home/user/.playground/codex-thread.json";
  }
  async restore() {
    if (this.restored) return;
    try {
      const saved = JSON.parse(await fs.readFile(this.file, "utf8"));
      this.thread = saved.thread;
      this.state.messages = saved.messages || [];
      this.state.changedFiles = saved.changedFiles || [];
    } catch (e) {
      if (e.code !== "ENOENT") throw e;
    }
    this.restored = true;
  }
  snapshot() {
    return structuredClone(this.state);
  }
  publish() {
    this.changed(this.snapshot());
  }
  rpc(method, params) {
    const id = ++this.sequence;
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.requests.delete(id);
        reject(Error("Codex did not acknowledge " + method + "."));
      }, 30000);
      this.requests.set(id, { resolve, reject, timer });
      this.child.stdin.write(JSON.stringify({ id, method, params }) + "\n");
    });
  }
  async start() {
    if (this.ready) return this.ready;
    this.ready = (async () => {
      await this.restore();
      this.provider = await startProvider(this.env);
      const config = {
        model_provider: "relai_playground",
        model: this.env.RELAI_SESSION_MODEL.split("/").at(-1),
        "model_providers.relai_playground.name": "RELAI scoped session gateway",
        "model_providers.relai_playground.base_url": this.provider.url,
        "model_providers.relai_playground.env_key": "RELAI_CODEX_TOKEN",
        "model_providers.relai_playground.wire_api": "responses",
        "model_providers.relai_playground.requires_openai_auth": false,
        "model_providers.relai_playground.supports_websockets": false,
        model_reasoning_effort:
          this.env.RELAI_SESSION_REASONING_EFFORT || "medium",
        approval_policy: "on-request",
        sandbox_mode: "workspace-write",
        web_search: "disabled",
      };
      const args = ["app-server", "--listen", "stdio://"];
      for (const [k, v] of Object.entries(config))
        args.push("-c", k + "=" + JSON.stringify(v));
      // Never inherit host/provider keys. The guest has only scoped playground access.
      const childEnv = {
        PATH: process.env.PATH,
        HOME: "/home/user",
        USER: "user",
        LANG: "C.UTF-8",
        RELAI_CODEX_TOKEN: this.provider.token,
        ...this.env,
      };
      this.child = spawn("codex", args, {
        cwd: this.root,
        env: childEnv,
        stdio: ["pipe", "pipe", "pipe"],
      });
      this.child.stderr.on("data", () => {});
      this.child.on("error", (error) => this.fail(error.message));
      this.child.on("exit", () => {
        this.ready = null;
        this.provider?.close();
        if (this.state.pendingTurn)
          this.fail("Codex stopped before completing this request.");
        for (const pending of this.requests.values()) {
          clearTimeout(pending.timer);
          pending.reject(Error("Codex process exited."));
        }
        this.requests.clear();
      });
      readline
        .createInterface({ input: this.child.stdout })
        .on("line", (line) => {
          try {
            this.receive(JSON.parse(line));
          } catch {
            this.fail("Codex returned an unreadable event.");
          }
        });
      await this.rpc("initialize", {
        clientInfo: {
          name: "relai_playground",
          title: "Code Assistant + RELAI",
          version: "0.1.0",
        },
      });
      this.child.stdin.write(JSON.stringify({ method: "initialized" }) + "\n");
      const response = await this.rpc(
        this.thread ? "thread/resume" : "thread/start",
        { ...(this.thread ? { threadId: this.thread } : {}), cwd: this.root },
      );
      this.thread = response.thread.id;
      await this.save();
    })().catch((error) => {
      this.close();
      this.ready = null;
      throw error;
    });
    return this.ready;
  }
  async save() {
    await fs.writeFile(
      this.file,
      JSON.stringify({
        thread: this.thread,
        messages: this.state.messages,
        changedFiles: this.state.changedFiles,
      }),
      { mode: 0o600 },
    );
  }
  async send(prompt) {
    await this.restore();
    if (this.state.pendingTurn)
      throw Error(
        "Stop the current Codex turn before sending another request.",
      );
    this.state.messages.push({ role: "user", content: prompt });
    this.cancelRequested = false;
    this.state.pendingTurn = true;
    this.publish();
    const handoff =
      this.state.messages.filter((m) => m.role === "user").length === 1
        ? "You are the real Codex coding assistant in this airline repository. RELAI is initialized and its Codex plugin is installed. Use its CLI/skills when requested; inspect actual files and receipts. The UI walkthrough was prerecorded, and its patch may have been explicitly applied. Read demo/recorded/README.md for evidence and limitations; do not claim you ran recorded tests. Do not commit, push, deploy, or expose credentials. Present concise real findings and reviewable changes; commands need not be repeated in your prose.\n\n"
        : "";
    try {
      await this.start();
      if (this.cancelRequested) {
        this.fail("Codex stopped before starting this request.");
        await this.save();
        return;
      }
      await this.save();
      const r = await this.rpc("turn/start", {
        threadId: this.thread,
        input: [{ type: "text", text: handoff + prompt }],
      });
      if (this.state.pendingTurn) this.turn = r.turn.id;
      if (this.cancelRequested && this.turn) await this.stop();
    } catch (e) {
      this.fail(e.message);
      throw e;
    }
  }
  receive(event) {
    if (event.id !== undefined && !event.method) {
      const p = this.requests.get(event.id);
      if (p) {
        clearTimeout(p.timer);
        this.requests.delete(event.id);
        event.error
          ? p.reject(Error(event.error.message))
          : p.resolve(event.result);
      }
      return;
    }
    const p = event.params || {};
    if (event.id !== undefined && event.method?.includes("requestApproval")) {
      const id = "codex-approval-" + event.id;
      this.approvals.set(id, event.id);
      this.state.messages.push({
        role: "assistant",
        tool_calls: [
          {
            id,
            name: "run_command",
            arguments: {
              command: Array.isArray(p.command)
                ? p.command.join(" ")
                : typeof p.command === "string"
                  ? p.command
                  : p.reason || "Review proposed workspace changes.",
            },
          },
        ],
      });
      this.publish();
      return;
    }
    if (event.id !== undefined) {
      this.child.stdin.write(
        JSON.stringify({
          id: event.id,
          error: {
            code: -32601,
            message: "This playground does not support this interaction yet.",
          },
        }) + "\n",
      );
      return;
    }
    if (event.method === "item/agentMessage/delta") {
      let message = this.state.messages.find((m) => m.codexId === p.itemId);
      if (!message) {
        message = { role: "assistant", content: "", codexId: p.itemId };
        this.state.messages.push(message);
      }
      message.content += p.delta;
      this.publish();
    }
    if (event.method === "item/completed" && p.item?.type === "agentMessage") {
      let message = this.state.messages.find((m) => m.codexId === p.item.id);
      if (!message) {
        message = { role: "assistant", codexId: p.item.id };
        this.state.messages.push(message);
      }
      message.content = p.item.text;
      this.publish();
    }
    if (event.method === "turn/started") this.turn = p.turn.id;
    if (event.method === "item/completed" && p.item?.type === "fileChange") {
      this.state.changedFiles = [
        ...new Set([
          ...(this.state.changedFiles || []),
          ...p.item.changes.map((c) => c.path),
        ]),
      ];
      this.publish();
    }
    if (event.method === "turn/completed") {
      this.state.pendingTurn = false;
      this.turn = null;
      if (p.turn.status === "failed")
        this.state.messages.push({
          role: "assistant",
          content:
            "Codex could not complete this request: " +
            (p.turn.error?.message || "runtime error"),
        });
      if (p.turn.status === "interrupted")
        this.state.messages.push({
          role: "assistant",
          content:
            "Codex stopped. Completed workspace changes remain available for review.",
        });
      for (const id of this.approvals.keys())
        this.state.messages.push({
          role: "tool",
          result: {
            call_id: id,
            status: "denied",
            content: "The Codex turn ended.",
          },
        });
      this.approvals.clear();
      this.publish();
      void this.save().catch(() =>
        this.fail(
          "Codex history could not be saved. Workspace files remain available.",
        ),
      );
    }
  }
  async approve(id, approved) {
    const rpcId = this.approvals.get(id);
    if (rpcId === undefined) throw Error("This approval is no longer pending.");
    this.approvals.delete(id);
    this.child.stdin.write(
      JSON.stringify({
        id: rpcId,
        result: { decision: approved ? "accept" : "decline" },
      }) + "\n",
    );
    this.state.messages.push({
      role: "tool",
      result: {
        call_id: id,
        status: approved ? "success" : "denied",
        content: approved ? "Approved." : "Declined.",
      },
    });
    this.publish();
  }
  async stop() {
    this.cancelRequested = true;
    if (this.turn)
      await this.rpc("turn/interrupt", {
        threadId: this.thread,
        turnId: this.turn,
      });
  }
  fail(message) {
    this.state.pendingTurn = false;
    this.state.messages.push({ role: "assistant", content: message });
    this.publish();
  }
  close() {
    this.child?.kill("SIGTERM");
    this.provider?.close();
  }
}
module.exports = { CodexBridge };
