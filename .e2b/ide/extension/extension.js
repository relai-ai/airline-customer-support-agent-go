const vscode = require("vscode");
const fs = require("node:fs/promises");
const crypto = require("node:crypto");
const path = require("node:path");
const recorded = require("./recorded");
const { createTelemetry, actions: telemetryActions } = require("./telemetry");
function activate(context) {
  const telemetry = createTelemetry(async () => JSON.parse(await fs.readFile("/home/user/.playground/environment.json", "utf8")));
  telemetry.report("extension_activated");
  let panel, terminal;
  const reviewDocuments = new Map();
  context.subscriptions.push(
    vscode.workspace.registerTextDocumentContentProvider(
      "relai-recorded-change",
      {
        provideTextDocumentContent(uri) {
          return reviewDocuments.get(uri.toString()) ?? "";
        },
      },
    ),
  );
  context.subscriptions.push(
    vscode.window.onDidCloseTerminal((closed) => {
      if (closed === terminal) terminal = undefined;
    }),
  );
  const open = async () => {
    if (panel) {
      panel.reveal();
      return;
    }
    panel = vscode.window.createWebviewPanel(
      "relai",
      "Code Assistant + RELAI",
      vscode.ViewColumn.One,
      {
        enableScripts: true,
        retainContextWhenHidden: true,
        localResourceRoots: [vscode.Uri.file(context.extensionPath)],
      },
    );
    const current = panel;
    let readyReported = false;
    current.onDidDispose(() => {
      panel = undefined;
    });
    const nonce = crypto.randomBytes(24).toString("hex");
    const env = JSON.parse(
      await fs.readFile("/home/user/.playground/environment.json", "utf8"),
    );
    const modeKey = "relai.playground.mode." + env.PLAYGROUND_SESSION_ID;
    const recordingKey =
      "relai.playground.recording." + env.PLAYGROUND_SESSION_ID;
    const root = "/home/user/airline";
    let recording,
      recordingBusy = false;
    const reviewChange = async (name) => {
      const content = recorded.reviewFile(recording, name);
      const before = vscode.Uri.from({
        scheme: "relai-recorded-change",
        path: "/before/" + name,
        query: recording.patchSha256,
      });
      const after = vscode.Uri.from({
        scheme: "relai-recorded-change",
        path: "/after/" + name,
        query: recording.patchSha256,
      });
      reviewDocuments.set(before.toString(), content.before);
      reviewDocuments.set(after.toString(), content.after);
      await vscode.commands.executeCommand(
        "vscode.diff",
        before,
        after,
        name + " · RELAI proposed change",
        { viewColumn: vscode.ViewColumn.Beside, preview: true },
      );
    };
    const base =
      env.PLAYGROUND_API_BASE + "/sessions/" + env.PLAYGROUND_SESSION_ID;
    const api = async (suffix, method = "GET", body) => {
      const response = await fetch(base + suffix, {
        method,
        headers: {
          Authorization: "Token " + env.RELAI_TEAM_API_KEY,
          "Content-Type": "application/json",
        },
        redirect: "error",
        signal: AbortSignal.timeout(130000),
        ...(body ? { body: JSON.stringify(body) } : {}),
      });
      if (!response.ok)
        throw Error(
          response.status === 401
            ? "This session has expired. Return to onboarding to reset it."
            : "The request could not finish. Refresh before retrying; no result was fabricated.",
        );
      return response.json();
    };
    current.webview.onDidReceiveMessage(async (msg) => {
      if (msg?.action === "telemetry-ready") {
        if (!readyReported) { readyReported = true; telemetry.report("relai_panel_ready"); }
        return;
      }
      if (telemetryActions.has(msg?.action)) telemetry.report("playground_action", { action: msg.action });
      try {
        // Explicit action allowlist. Webview cannot choose URLs, paths or credentials.
        if (msg.action === "load") {
          recording = await recorded.loadRecording(root);
          let investigation = await api("/investigation");
          if (!investigation.investigation) {
            await api("/investigation", "POST", {
              challengeId: "invalid-seat",
            });
            investigation = await api("/investigation");
          }
          current.webview.postMessage({
            type: "loaded",
            recording,
            recordedProgress: context.workspaceState.get(recordingKey, 0),
            ...investigation,
          });
        } else if (msg.action === "recorded" && Number.isInteger(msg.step)) {
          if (recordingBusy) return;
          recordingBusy = true;
          try {
            const progress = context.workspaceState.get(recordingKey, 0);
            if (!recording || msg.step !== progress || progress > 3)
              throw Error("Refresh to resume the recorded walkthrough.");
            let applied;
            if (progress === 3)
              applied = await recorded.applyRecording(root, recording);
            await context.workspaceState.update(recordingKey, progress + 1);
            current.webview.postMessage({
              type: "recorded",
              recordedProgress: progress + 1,
              applied,
            });
          } finally {
            recordingBusy = false;
          }
        } else if (
          msg.action === "folder" &&
          ["source", "relai", "project", "wiki", "tests"].includes(msg.folder)
        ) {
          const folders = {
            source: "airline",
            relai: ".relai",
            project: ".",
            wiki: ".relai/evals-wiki",
            tests: ".relai/harbor/agent-tests",
          };
          await vscode.commands.executeCommand(
            "revealInExplorer",
            vscode.Uri.file(path.join(root, folders[msg.folder])),
          );
          current.webview.postMessage({
            type: "folder-opened",
            folder: msg.folder,
            path: folders[msg.folder],
          });
        } else if (msg.action === "source") {
          await vscode.window.showTextDocument(
            vscode.Uri.file(
              path.join(
                root,
                msg.file === "tools" ? "airline/tools.go" : "airline/agent.go",
              ),
            ),
            { viewColumn: vscode.ViewColumn.Beside },
          );
        } else if (
          msg.action === "mode" &&
          [null, "assistant", "terminal"].includes(msg.mode) &&
          typeof msg.challengeId === "string"
        ) {
          await context.workspaceState.update(modeKey, {
            mode: msg.mode,
            challengeId: msg.challengeId,
          });
        } else if (msg.action === "choose") {
          await api("/investigation", "POST", { challengeId: msg.id });
          current.webview.postMessage({
            type: "state",
            ...(await api("/investigation")),
          });
        } else if (
          msg.action === "try-agent" &&
          typeof msg.prompt === "string" &&
          msg.prompt.trim()
        ) {
          current.webview.postMessage({
            type: "state",
            ...(await api("/investigation/try-agent", "POST", {
              prompt: msg.prompt,
              turnId: crypto.randomUUID(),
            })),
          });
        } else if (
          msg.action === "send" &&
          typeof msg.prompt === "string" &&
          msg.prompt.trim()
        ) {
          current.webview.postMessage({
            type: "state",
            ...(await api("/investigation/turn", "POST", {
              prompt: msg.prompt,
              ...(msg.runWorkflow === true ? { runWorkflow: true } : {}),
              ...(typeof msg.step === "string" ? { step: msg.step } : {}),
              turnId: crypto.randomUUID(),
            })),
          });
        } else if (
          msg.action === "approve" &&
          typeof msg.approved === "boolean"
        ) {
          await api("/investigation/approval", "POST", {
            callId: msg.id,
            approved: msg.approved,
          });
          current.webview.postMessage({
            type: "state",
            ...(await api("/investigation")),
          });
        } else if (msg.action === "continue") {
          current.webview.postMessage({
            type: "state",
            ...(await api("/investigation/continue", "POST", {})),
          });
        } else if (msg.action === "refresh") {
          current.webview.postMessage({
            type: "state",
            ...(await api("/investigation")),
          });
        } else if (msg.action === "stop") {
          await api("/investigation/abandon-turn", "POST", {});
          current.webview.postMessage({
            type: "state",
            ...(await api("/investigation")),
          });
        } else if (msg.action === "cancel") {
          await api("/investigation/cancel", "POST", { jobId: msg.id });
          current.webview.postMessage({
            type: "state",
            ...(await api("/investigation")),
          });
        } else if (msg.action === "assistant") {
          await vscode.commands.executeCommand("workbench.action.closePanel");
          await vscode.commands.executeCommand("workbench.action.closeSidebar");
          await vscode.commands.executeCommand(
            "workbench.action.closeAuxiliaryBar",
          );
        } else if (msg.action === "terminal") {
          if (!terminal)
            terminal = vscode.window.createTerminal({
              name: "RELAI · Terminal",
              cwd: "/home/user/airline",
            });
          terminal.show();
        } else if (msg.action === "evidence") {
          await vscode.commands.executeCommand(
            "markdown.showPreview",
            vscode.Uri.file("/home/user/airline/demo/checkpoint/PLAYGROUND.md"),
          );
        } else if (
          msg.action === "review-change" &&
          typeof msg.file === "string"
        ) {
          await reviewChange(msg.file);
        } else if (msg.action === "changes") {
          if (msg.recorded === true && recording) {
            const selected = await vscode.window.showQuickPick(
              recording.files.map((file) => ({
                label: file.path,
                description: file.before === null ? "Added" : "Modified",
              })),
              { title: "RELAI optimized changes", placeHolder: "Choose a file to compare before and after" },
            );
            if (selected) await reviewChange(selected.label);
          } else await vscode.commands.executeCommand("workbench.view.scm");
        }
      } catch (error) {
        telemetry.report("playground_failed");
        current.webview.postMessage({ type: "error", message: error.message });
      }
    });
    current.webview.html = (
      await fs.readFile(path.join(context.extensionPath, "panel.html"), "utf8")
    )
      .replace(
        "WORKFLOW_HELPERS",
        await fs.readFile(
          path.join(context.extensionPath, "workflow.js"),
          "utf8",
        ),
      )
      .replaceAll("NONCE", nonce);
  };
  context.subscriptions.push(
    vscode.commands.registerCommand("relai.playground", open),
  );
  void open().catch(() => {
    telemetry.report("playground_failed", { stage: "panel" });
    vscode.window.showErrorMessage("RELAI playground could not open. Check session provisioning.");
  });
}
module.exports = { activate };
