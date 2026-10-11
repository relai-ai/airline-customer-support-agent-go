const { test } = require('node:test');
const assert = require('node:assert/strict');
const { workflowView, shouldPrefill } = require('./workflow');
test('handoff advances only from persisted assistant completion and passes honest baselines to verification', () => {
  assert.equal(workflowView({}).step, 'prepare');
  assert.equal(workflowView({step:'evaluate'}).step, 'evaluate');
  assert.equal(workflowView({step:'evaluate',nextStep:'optimize'}).step,'optimize');
  assert.equal(workflowView({step:'evaluate',nextStep:'verify'}).step,'verify');
  assert.equal(workflowView({step:'verify',nextStep:'compare'}).step,'compare');
  assert.equal(workflowView({nextStep:'complete'}).prompt,'');
});
test('polling and step transitions preserve edited and unsent drafts', () => {
  assert.equal(shouldPrefill('My follow-up question','Old suggestion'),false);
  assert.equal(shouldPrefill('Old suggestion','Old suggestion'),true);
  assert.equal(shouldPrefill('','Old suggestion'),true);
});
test('packaged webview script compiles after helper injection', () => {
  const fs = require('node:fs');
  const path = require('node:path');
  const vm = require('node:vm');
  const html = fs.readFileSync(path.join(__dirname,'panel.html'),'utf8').replace('WORKFLOW_HELPERS',fs.readFileSync(path.join(__dirname,'workflow.js'),'utf8'));
  const script = html.match(/<script nonce="NONCE">([\s\S]*?)<\/script>/)[1];
  assert.doesNotThrow(()=>new vm.Script(script));
});
test('short editable requests cover real evaluation and verified retry, without a report prompt', () => {
  const {workflowSteps}=require('./workflow');
  assert.ok(workflowSteps.evaluate.prompt.includes('Run the RELAI test'));
  assert.ok(workflowSteps.verify.prompt.includes('apply the verified changes'));
  assert.ok(Object.values(workflowSteps).every(s=>s.prompt.length<220));
});
