// Shared by the webview and its regression tests; no runtime or credentials.
const workflowSteps = {
  prepare: { label: '1. Prepare the test', action: 'Prepare test', guidance: 'Review or edit this request. The assistant will create the test and show its assertions before evaluation.', prompt: 'Can you use RELAI to check this behavior and fix it if it fails?' },
  evaluate: { label: '2. Evaluate the agent', action: 'Run evaluation', guidance: 'Measure the starting agent. The assistant will show its actual response, tool trace, score, and missed assertions.', prompt: 'Run the RELAI test and show me how the agent actually behaves.' },
  optimize: { label: '3. Improve the agent', action: 'Run optimization', guidance: 'Use the measured failure to improve the agent. Review the optimizer’s changes and report before verification.', prompt: 'Go ahead and fix it.' },
  verify: { label: '4. Verify and check regressions', action: 'Run verification', guidance: 'Check the optimizer’s resulting branch and existing tests. If the baseline passed, verify the unchanged agent.', prompt: 'Check that the fix works and hasn’t broken anything else. If verified, apply it here so I can try it.' },
  compare: { label: '5. Compare the results', action: 'Compare results', guidance: 'Review what changed, whether it worked, and which existing behaviors still pass.', prompt: 'Show me what changed and how the agent behaves now.' },
};
function startingRequest(id) {
  return ({
    'invalid-seat': 'The agent may let me book seat 99Z, which doesn’t exist. Can you use RELAI to check and fix this?',
    'unknown-fare': 'Can you use RELAI to check whether the agent guesses baggage allowances for unfamiliar fares, and fix it if it does?',
    'booking-outage': 'Can you use RELAI to check how the agent handles a failed booking lookup, and fix any problems?'
  })[id] || workflowSteps.prepare.prompt;
}
const stepValue = {
  prepare: 'Create an evaluation from your expected behavior, prepare its learning environment, and simulate the agent to measure what happens.',
  evaluate: 'Run the evaluation in the learning environment and inspect the actual response, tool calls, and outcomes.',
  optimize: 'RELAI uses evaluation feedback to test candidate improvements. Review the accepted changes and their measured results.',
  verify: 'Rerun the same evaluation and existing regressions before applying the verified change.',
  compare: 'Compare actual responses, evaluation results, and code changes, then try the improved agent yourself.',
  complete: 'Inspect the comparison or send the original request to the updated airline agent.'
};
function workflowView(i) {
  const step = i.nextStep || i.step || 'prepare';
  return step === 'complete' ? { step, label: 'Results ready', action: 'Send message', guidance: 'Review the comparison below or ask a follow-up question.', prompt: '' } : { step, ...workflowSteps[step] };
}
function shouldPrefill(draft, previousPreset) { return !draft.trim() || draft === previousPreset; }
if (typeof module !== 'undefined') module.exports = { workflowSteps, workflowView, shouldPrefill, startingRequest, stepValue };
