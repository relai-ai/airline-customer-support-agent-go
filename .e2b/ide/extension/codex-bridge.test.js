const test = require("node:test");
const assert = require("node:assert/strict");
const { CodexBridge } = require("./codex-bridge");
function fixture() {
  const updates = [];
  const bridge = new CodexBridge({}, "/tmp", (state) => updates.push(state));
  bridge.restore = async () => {};
  bridge.save = async () => {};
  return { bridge, updates };
}
test("submission publishes the prompt and pending state before runtime startup; Stop cancels startup", async () => {
  const { bridge, updates } = fixture();
  let finish;
  bridge.start = () =>
    new Promise((resolve) => {
      finish = resolve;
    });
  bridge.rpc = async () =>
    assert.fail("Cancelled startup must not start a turn");
  const sending = bridge.send("Inspect this repository");
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(updates[0].pendingTurn, true);
  assert.equal(updates[0].messages[0].content, "Inspect this repository");
  await bridge.stop();
  finish();
  await sending;
  assert.equal(bridge.snapshot().pendingTurn, false);
});
test("Stop during turn acknowledgement interrupts once the turn is known", async () => {
  const { bridge } = fixture();
  bridge.start = async () => {};
  const calls = [];
  let acknowledge;
  bridge.rpc = (method, params) => {
    calls.push({ method, params });
    return method === "turn/start"
      ? new Promise((resolve) => {
          acknowledge = resolve;
        })
      : Promise.resolve({});
  };
  const sending = bridge.send("Inspect this repository");
  await new Promise((resolve) => setImmediate(resolve));
  await bridge.stop();
  acknowledge({ turn: { id: "turn-1" } });
  await sending;
  assert.equal(calls[1].method, "turn/interrupt");
  assert.equal(calls[1].params.turnId, "turn-1");
  bridge.receive({
    method: "turn/completed",
    params: { turn: { id: "turn-1", status: "interrupted" } },
  });
  assert.equal(bridge.snapshot().pendingTurn, false);
});

test("approval decline and file-change receipts are real app-server events", async () => {
  const { bridge } = fixture();
  const replies = [];
  bridge.child = { stdin: { write: (line) => replies.push(JSON.parse(line)) } };
  bridge.receive({
    id: 7,
    method: "item/commandExecution/requestApproval",
    params: { command: "go test ./..." },
  });
  await bridge.approve("codex-approval-7", false);
  assert.deepEqual(replies[0], { id: 7, result: { decision: "decline" } });
  assert.equal(bridge.snapshot().messages.at(-1).result.status, "denied");
  await assert.rejects(
    bridge.approve("codex-approval-7", true),
    /no longer pending/,
  );
  bridge.receive({
    method: "item/completed",
    params: {
      item: { type: "fileChange", changes: [{ path: "airline/tools.go" }] },
    },
  });
  assert.deepEqual(bridge.snapshot().changedFiles, ["airline/tools.go"]);
});
