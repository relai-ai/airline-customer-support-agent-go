// Shared by the webview and its regression tests; no runtime or credentials.
const workflowSteps = {
  prepare: { label: '1. Prepare the test', action: 'Prepare test', guidance: 'Review or edit this request. The assistant will create the test and show its assertions before evaluation.', prompt: 'Use RELAI to create a new repeatable test for this scenario. Show me what it checks before running it.' },
  evaluate: { label: '2. Evaluate the agent', action: 'Run evaluation', guidance: 'Measure the starting agent. The assistant will show its actual response, tool trace, score, and missed assertions.', prompt: 'Run the RELAI test and show me how the agent actually behaves.' },
  optimize: { label: '3. Improve the agent', action: 'Run optimization', guidance: 'Use the measured failure to improve the agent. Review the optimizer’s changes and report before verification.', prompt: 'Use RELAI to improve this failure. Keep the test criteria unchanged and show me the changes.' },
  verify: { label: '4. Verify and check regressions', action: 'Run verification', guidance: 'Check the optimizer’s resulting branch and existing tests. If the baseline passed, verify the unchanged agent.', prompt: 'Verify the RELAI improvement and check the existing regressions. If it works, apply the verified changes to this demo workspace so I can try the agent again.' },
  compare: { label: '5. Compare the results', action: 'Compare results', guidance: 'Review what changed, whether it worked, and which existing behaviors still pass.', prompt: 'Compare the actual before and after results and show me what RELAI changed.' },
};
function workflowView(i) {
  const step = i.nextStep || i.step || 'prepare';
  return step === 'complete' ? { step, label: 'Results ready', action: 'Send message', guidance: 'Review the comparison below or ask a follow-up question.', prompt: '' } : { step, ...workflowSteps[step] };
}
function shouldPrefill(draft, previousPreset) { return !draft.trim() || draft === previousPreset; }
if (typeof module !== 'undefined') module.exports = { workflowSteps, workflowView, shouldPrefill };
