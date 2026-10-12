// Recorded runs are evidence, never live command execution.
const fs = require("node:fs/promises");
const path = require("node:path");
const crypto = require("node:crypto");
const { execFile } = require("node:child_process");
const { promisify } = require("node:util");
const exec = promisify(execFile);
const prompts = [
  "What are the risks identified for this agent by RELAI?",
  "Use RELAI to create a new agent test for the unvalidated seat-change risk with at least 10 turns and a sophisticated adversarial user. Show me the readable results of the agent simulation against this test.",
  "Use RELAI to optimize the agent.",
  "These changes look great. Apply them.",
];
const digest = (data) => crypto.createHash("sha256").update(data).digest("hex");
async function regular(root, name) {
  if (
    !/^(airline\/[a-zA-Z0-9_-]+\.go|demo\/recorded\/[a-zA-Z0-9_.-]+)$/.test(
      name,
    )
  )
    throw Error("Invalid recorded artifact path");
  const file = path.join(await fs.realpath(root), name);
  if ((await fs.realpath(file)) !== file)
    throw Error("Recorded artifacts cannot use symlinks");
  if (!(await fs.stat(file)).isFile())
    throw Error("Recorded artifacts must be regular files");
  return fs.readFile(file);
}
async function loadRecording(root) {
  const manifest = JSON.parse(
    await regular(root, "demo/recorded/walkthrough.json"),
  );
  if (
    manifest.schemaVersion !== 1 ||
    !["pending", "verified"].includes(manifest.verificationStatus) ||
    manifest.steps?.length !== 3 ||
    !Array.isArray(manifest.files) ||
    !manifest.files.length
  )
    throw Error("Recorded walkthrough is incomplete");
  for (const step of manifest.steps)
    if (
      !Number.isFinite(step.durationSeconds) ||
      step.durationSeconds < 0 ||
      typeof step.text !== "string"
    )
      throw Error("Recorded run metadata unavailable");
  if (manifest.initialization) {
    const init = manifest.initialization;
    if (!/^demo\/recorded\/[a-zA-Z0-9_.-]+\.json$/.test(init.receipt ?? ""))
      throw Error("Initialization timing requires a saved command receipt");
    const receipt = JSON.parse(await regular(root, init.receipt));
    if (
      receipt.argv?.[0] !== "relai" ||
      receipt.argv?.[1] !== "init" ||
      receipt.exitCode !== 0 ||
      !Number.isFinite(receipt.durationSeconds) ||
      receipt.durationSeconds <= 0 ||
      init.durationSeconds !== receipt.durationSeconds
    )
      throw Error(
        "Initialization timing must come from a completed relai init command",
      );
  }
  if (
    typeof manifest.diff !== "string" ||
    !["before", "after", "regressions"].every(
      (key) => typeof manifest.comparison?.[key] === "string",
    )
  )
    throw Error("Recorded comparison is incomplete");
  for (const file of manifest.files)
    if (
      !/^airline\/[a-zA-Z0-9_-]+\.go$/.test(file.path) ||
      (file.before !== null && !/^[a-f0-9]{64}$/.test(file.before)) ||
      !/^[a-f0-9]{64}$/.test(file.after)
    )
      throw Error("Recorded source hashes are invalid");
  if (
    new Set(manifest.files.map((file) => file.path)).size !==
    manifest.files.length
  )
    throw Error("Recorded files must be unique");
  if (
    !Array.isArray(manifest.turns) ||
    manifest.turns.length < 10 ||
    !manifest.turns.every(
      (t) => typeof t.user === "string" && typeof t.agent === "string",
    )
  )
    throw Error("Recording requires ten real exchanges");
  if (
    !Array.isArray(manifest.prompts) ||
    manifest.prompts.length !== 4 ||
    !manifest.prompts.every((p, i) => p === prompts[i])
  )
    throw Error("Recorded prompts do not match the walkthrough");
  if (manifest.fixedExample) {
    const example = manifest.fixedExample;
    if (!/^demo\/recorded\/[a-zA-Z0-9_.-]+\.json$/.test(example.trajectory ?? ""))
      throw Error("Verification example requires a recorded trajectory");
    const trajectory = JSON.parse(await regular(root, example.trajectory));
    const user = trajectory.steps?.find((step) => step.source === "user");
    const agent = trajectory.steps?.find(
      (step) => step.source === "agent" && !step.tool_calls?.length && step.message,
    );
    if (example.user !== user?.message || example.agent !== agent?.message)
      throw Error("Verification example differs from its actual recorded response");
  }
  const patch = await regular(root, "demo/recorded/verified.patch");
  if (digest(patch) !== manifest.patchSha256)
    throw Error("Recorded patch integrity check failed");
  if (manifest.diff !== patch.toString())
    throw Error("Displayed changes do not match the recorded patch");
  return manifest;
}
async function applyRecording(root, manifest) {
  if (manifest.verificationStatus !== "verified")
    throw Error(
      "Independent verification is pending. The recorded patch cannot be applied yet.",
    );
  const patch = await regular(root, "demo/recorded/verified.patch");
  if (digest(patch) !== manifest.patchSha256)
    throw Error("Recorded patch integrity check failed");
  let before = true,
    after = true;
  for (const file of manifest.files) {
    if (!/^airline\/[a-zA-Z0-9_-]+\.go$/.test(file.path))
      throw Error("Only recorded agent code can be applied");
    let hash;
    try {
      hash = digest(await regular(root, file.path));
    } catch (error) {
      if (error.code !== "ENOENT") throw error;
      const parent = path.join(await fs.realpath(root), "airline");
      if ((await fs.realpath(parent)) !== parent)
        throw Error("Recorded artifacts cannot use symlinks");
      hash = null;
    }
    before &&= hash === file.before;
    after &&= hash === file.after;
  }
  if (after)
    return "The verified changes are already applied to this workspace.";
  if (!before)
    throw Error(
      "These files have changed since the recorded baseline. Your edits were preserved; review the diff before applying.",
    );
  const patchPaths = [
    ...patch.toString().matchAll(/^diff --git a\/(.+) b\/(.+)$/gm),
  ];
  if (
    !patchPaths.length ||
    patchPaths.some(
      (m) => m[1] !== m[2] || !manifest.files.some((f) => f.path === m[1]),
    )
  )
    throw Error("Recorded patch contains an unexpected file");
  const patchFile = path.join(root, "demo/recorded/verified.patch");
  await exec("git", ["apply", "--check", "--", patchFile], { cwd: root });
  await exec("git", ["apply", "--", patchFile], { cwd: root });
  for (const file of manifest.files)
    if (digest(await regular(root, file.path)) !== file.after)
      throw Error("Applied file does not match verified evidence");
  return "Applied the recorded, verified improvement to this workspace. New requests now use these files.";
}
function reviewFile(manifest, name) {
  const file = manifest.files.find((item) => item.path === name);
  if (!file || !/^airline\/[a-zA-Z0-9_-]+\.go$/.test(name))
    throw Error("This file is not part of the recorded change");
  if (
    typeof file.beforeContent !== "string" ||
    typeof file.afterContent !== "string"
  )
    throw Error("Recorded diff snapshots are unavailable");
  if (
    (file.before === null
      ? file.beforeContent !== ""
      : digest(file.beforeContent) !== file.before) ||
    digest(file.afterContent) !== file.after
  )
    throw Error("Recorded diff snapshots do not match the measured source");
  return { before: file.beforeContent, after: file.afterContent };
}
module.exports = { prompts, loadRecording, applyRecording, reviewFile, digest };
