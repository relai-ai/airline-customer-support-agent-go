// Image build check only: no credentials, models, writes or patch application.
const fs = require("node:fs/promises");
const path = require("node:path");
const { loadRecording, reviewFile, digest } = require("./recorded");

async function validate(root) {
  const recording = await loadRecording(root);
  if (recording.verificationStatus !== "verified")
    throw Error(
      "Independent verification must finish before building the release template",
    );
  for (const file of recording.files) {
    reviewFile(recording, file.path);
    const target = path.join(root, file.path);
    if (file.before === null) {
      try {
        await fs.lstat(target);
      } catch (error) {
        if (error.code === "ENOENT") continue;
        throw error;
      }
      throw Error("New recorded file already exists in the original template");
    }
    if (digest(await fs.readFile(target)) !== file.before)
      throw Error("Template source differs from the measured baseline");
  }
  console.log(
    "Recorded walkthrough and original source validated; no model calls.",
  );
}

if (require.main === module) {
  validate(process.argv[2] || process.cwd()).catch((error) => {
    console.error(error.message);
    process.exitCode = 1;
  });
}
module.exports = { validate };
