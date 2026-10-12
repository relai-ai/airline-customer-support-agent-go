const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");
const { execFile } = require("node:child_process");
const exec = require("node:util").promisify(execFile);
const {
  prompts,
  digest,
  loadRecording,
  applyRecording,
  reviewFile,
} = require("./recorded");
async function fixture(t) {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), "relai-recorded-"));
  t.after(() => fs.rm(root, { recursive: true, force: true }));
  await fs.mkdir(path.join(root, "airline"));
  await fs.mkdir(path.join(root, "demo/recorded"), { recursive: true });
  await exec("git", ["init", "-q"], { cwd: root });
  const before = "package airline\n\nfunc seat() bool { return true }\n",
    after = "package airline\n\nfunc seat() bool { return false }\n";
  await fs.writeFile(path.join(root, "airline/tools.go"), before);
  await exec("git", ["add", "airline/tools.go"], { cwd: root });
  await fs.writeFile(path.join(root, "airline/tools.go"), after);
  const { stdout: patch } = await exec(
    "git",
    ["diff", "--", "airline/tools.go"],
    { cwd: root },
  );
  await fs.writeFile(path.join(root, "airline/tools.go"), before);
  await fs.writeFile(path.join(root, "demo/recorded/verified.patch"), patch);
  const manifest = {
    schemaVersion: 1,
    verificationStatus: "verified",
    prompts,
    steps: Array.from({ length: 3 }, () => ({
      text: "Measured result",
      durationSeconds: 60,
    })),
    turns: Array.from({ length: 12 }, () => ({
      user: "Traveler request",
      agent: "Actual response",
    })),
    files: [
      {
        path: "airline/tools.go",
        before: digest(before),
        after: digest(after),
        beforeContent: before,
        afterContent: after,
      },
    ],
    patchSha256: digest(patch),
    diff: patch,
    comparison: { before: "Before", after: "After", regressions: "Passed" },
  };
  await fs.writeFile(
    path.join(root, "demo/recorded/walkthrough.json"),
    JSON.stringify(manifest),
  );
  return { root, manifest, before, after };
}
test("recorded patch changes actual code exactly once", async (t) => {
  const { root, after } = await fixture(t);
  const manifest = await loadRecording(root);
  assert.match(await applyRecording(root, manifest), /Applied/);
  assert.equal(
    await fs.readFile(path.join(root, "airline/tools.go"), "utf8"),
    after,
  );
  assert.match(await applyRecording(root, manifest), /already applied/);
});
test("native review uses measured snapshots even when the visitor edits the working file", async (t) => {
  const { root, manifest, before, after } = await fixture(t);
  await fs.writeFile(path.join(root, "airline/tools.go"), "// visitor draft\n");
  assert.deepEqual(reviewFile(manifest, "airline/tools.go"), { before, after });
  assert.throws(
    () => reviewFile(manifest, ".relai/config.toml"),
    /not part of the recorded change/,
  );
  manifest.files[0].afterContent = "// different proposed change\n";
  assert.throws(() => reviewFile(manifest, "airline/tools.go"), /do not match/);
});
test("post-apply example must match an actual recorded reply, not a tool-call message", async (t) => {
  const { root, manifest } = await fixture(t);
  const trajectory = "demo/recorded/verified-example.json";
  manifest.fixedExample = { trajectory, user: "Request 99Z", agent: "Rejected; no change." };
  await fs.writeFile(path.join(root, trajectory), JSON.stringify({ steps: [
    { source: "user", message: "Request 99Z" },
    { source: "agent", message: "lookup", tool_calls: [{ function_name: "lookup_booking" }] },
    { source: "agent", message: "Rejected; no change." },
  ] }));
  const save = () => fs.writeFile(path.join(root, "demo/recorded/walkthrough.json"), JSON.stringify(manifest));
  await save();
  assert.equal((await loadRecording(root)).fixedExample.agent, "Rejected; no change.");
  manifest.fixedExample.agent = "Invented success";
  await save();
  await assert.rejects(loadRecording(root), /differs from its actual recorded response/);
});
test("inventory reads cannot be displayed as initialization timings", async (t) => {
  const { root, manifest } = await fixture(t);
  manifest.initialization = {
    receipt: "demo/recorded/init-receipt.json",
    durationSeconds: 1,
  };
  await fs.writeFile(
    path.join(root, "demo/recorded/walkthrough.json"),
    JSON.stringify(manifest),
  );
  const receipt = {
    argv: ["relai", "wiki", "list", "risks"],
    exitCode: 0,
    durationSeconds: 1,
  };
  await fs.writeFile(
    path.join(root, manifest.initialization.receipt),
    JSON.stringify(receipt),
  );
  await assert.rejects(() => loadRecording(root), /completed relai init/);
  receipt.argv = ["relai", "init"];
  await fs.writeFile(
    path.join(root, manifest.initialization.receipt),
    JSON.stringify(receipt),
  );
  assert.equal((await loadRecording(root)).initialization.durationSeconds, 1);
});
test("conflicting user edits are preserved", async (t) => {
  const { root, manifest } = await fixture(t);
  const edit = "// user edit\n";
  await fs.writeFile(path.join(root, "airline/tools.go"), edit);
  await assert.rejects(
    () => applyRecording(root, manifest),
    /edits were preserved/,
  );
  assert.equal(
    await fs.readFile(path.join(root, "airline/tools.go"), "utf8"),
    edit,
  );
});
test("pending verification permits preview but prevents applying or releasing the patch", async (t) => {
  const { root, manifest, before } = await fixture(t);
  manifest.verificationStatus = "pending";
  await fs.writeFile(
    path.join(root, "demo/recorded/walkthrough.json"),
    JSON.stringify(manifest),
  );
  assert.equal((await loadRecording(root)).verificationStatus, "pending");
  await assert.rejects(
    () => applyRecording(root, manifest),
    /verification is pending/,
  );
  await assert.rejects(
    () => require("./validate-recording").validate(root),
    /before building the release template/,
  );
  assert.equal(
    await fs.readFile(path.join(root, "airline/tools.go"), "utf8"),
    before,
  );
});
test("modified evidence and short simulations cannot be presented as the recording", async (t) => {
  const { root, manifest } = await fixture(t);
  manifest.turns = manifest.turns.slice(0, 9);
  await fs.writeFile(
    path.join(root, "demo/recorded/walkthrough.json"),
    JSON.stringify(manifest),
  );
  await assert.rejects(() => loadRecording(root), /ten real exchanges/);
  manifest.turns = Array.from({ length: 10 }, () => ({
    user: "request",
    agent: "reply",
  }));
  await fs.writeFile(
    path.join(root, "demo/recorded/walkthrough.json"),
    JSON.stringify(manifest),
  );
  await fs.appendFile(
    path.join(root, "demo/recorded/verified.patch"),
    "tampered",
  );
  await assert.rejects(() => loadRecording(root), /integrity/);
});
test("symlink targets cannot overwrite another workspace", async (t) => {
  const { root, manifest } = await fixture(t);
  await fs.rename(
    path.join(root, "airline/tools.go"),
    path.join(root, "airline/other.go"),
  );
  await fs.symlink("other.go", path.join(root, "airline/tools.go"));
  await assert.rejects(() => applyRecording(root, manifest), /symlinks/);
});
test("verified new Go files apply once and never replace an existing user file", async (t) => {
  const { root, manifest } = await fixture(t);
  const content = "package airline\n\n// Verified new file.\n";
  const patch =
    "diff --git a/airline/validation.go b/airline/validation.go\nnew file mode 100644\nindex 0000000..1234567\n--- /dev/null\n+++ b/airline/validation.go\n@@ -0,0 +1,3 @@\n+package airline\n+\n+// Verified new file.\n";
  const updated = {
    ...manifest,
    files: [
      { path: "airline/validation.go", before: null, after: digest(content) },
    ],
    patchSha256: digest(patch),
    diff: patch,
  };
  await fs.writeFile(path.join(root, "demo/recorded/verified.patch"), patch);
  await fs.writeFile(
    path.join(root, "demo/recorded/walkthrough.json"),
    JSON.stringify(updated),
  );
  const recording = await loadRecording(root);
  assert.match(await applyRecording(root, recording), /Applied/);
  assert.equal(
    await fs.readFile(path.join(root, "airline/validation.go"), "utf8"),
    content,
  );
  assert.match(await applyRecording(root, recording), /already applied/);
  await fs.writeFile(
    path.join(root, "airline/validation.go"),
    "// user-owned file\n",
  );
  await assert.rejects(
    () => applyRecording(root, recording),
    /edits were preserved/,
  );
  assert.equal(
    await fs.readFile(path.join(root, "airline/validation.go"), "utf8"),
    "// user-owned file\n",
  );
});

test("image validation requires the original baseline and rejects an invented displayed diff", async (t) => {
  const { root, manifest } = await fixture(t);
  const { validate } = require("./validate-recording");
  await validate(root);
  manifest.files[0].afterContent = "// invented review snapshot\n";
  await fs.writeFile(
    path.join(root, "demo/recorded/walkthrough.json"),
    JSON.stringify(manifest),
  );
  await assert.rejects(() => validate(root), /snapshots do not match/);
  manifest.files[0].afterContent =
    "package airline\n\nfunc seat() bool { return false }\n";
  await fs.writeFile(
    path.join(root, "demo/recorded/walkthrough.json"),
    JSON.stringify(manifest),
  );
  await fs.writeFile(
    path.join(root, "airline/tools.go"),
    "// newer template\n",
  );
  await assert.rejects(() => validate(root), /measured baseline/);
  manifest.diff = "unrelated displayed changes";
  await fs.writeFile(
    path.join(root, "demo/recorded/walkthrough.json"),
    JSON.stringify(manifest),
  );
  await assert.rejects(() => loadRecording(root), /Displayed changes/);
});
