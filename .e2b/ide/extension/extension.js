const vscode = require('vscode');
const fs = require('node:fs/promises');
const crypto = require('node:crypto');
const path = require('node:path');
function activate(context) {
  let panel, terminal;
  context.subscriptions.push(vscode.window.onDidCloseTerminal(closed => { if (closed === terminal) terminal = undefined; }));
  const open = async () => {
    if (panel) { panel.reveal(); return; }
    panel = vscode.window.createWebviewPanel('relai', 'RELAI · Investigate & improve', vscode.ViewColumn.One, { enableScripts: true, retainContextWhenHidden: true, localResourceRoots: [vscode.Uri.file(context.extensionPath)] });
    const current = panel;
    current.onDidDispose(() => { panel = undefined; });
    const nonce = crypto.randomBytes(24).toString('hex');
    const env = JSON.parse(await fs.readFile('/home/user/.playground/environment.json', 'utf8'));
    const base = env.PLAYGROUND_API_BASE + '/sessions/' + env.PLAYGROUND_SESSION_ID;
    const api = async (suffix, method = 'GET', body) => {
      const response = await fetch(base + suffix, { method, headers: { Authorization: 'Token ' + env.RELAI_TEAM_API_KEY, 'Content-Type': 'application/json' }, redirect: 'error', signal: AbortSignal.timeout(130000), ...(body ? { body: JSON.stringify(body) } : {}) });
      if (!response.ok) throw Error(response.status === 401 ? 'This session has expired. Return to onboarding to reset it.' : 'The request could not finish. Refresh before retrying; no result was fabricated.');
      return response.json();
    };
    current.webview.onDidReceiveMessage(async msg => {
      try {
        // Explicit action allowlist. Webview cannot choose URLs, paths or credentials.
        if (msg.action === 'load') {
          const [catalog, evidence, investigation] = await Promise.all([api('/challenges'), fs.readFile('/home/user/airline/demo/checkpoint/results.json', 'utf8').then(JSON.parse), api('/investigation')]);
          current.webview.postMessage({ type: 'loaded', catalog, evidence, ...investigation });
        } else if (msg.action === 'choose') {
          await api('/investigation', 'POST', { challengeId: msg.id });
          current.webview.postMessage({ type: 'state', ...await api('/investigation') });
        } else if (msg.action === 'send' && typeof msg.prompt === 'string' && msg.prompt.trim()) {
          current.webview.postMessage({ type: 'state', ...await api('/investigation/turn', 'POST', { prompt: msg.prompt, turnId: crypto.randomUUID() }) });
        } else if (msg.action === 'approve' && typeof msg.approved === 'boolean') {
          await api('/investigation/approval', 'POST', { callId: msg.id, approved: msg.approved });
          current.webview.postMessage({ type: 'state', ...await api('/investigation') });
        } else if (msg.action === 'continue') {
          current.webview.postMessage({ type: 'state', ...await api('/investigation/continue', 'POST', {}) });
        } else if (msg.action === 'refresh') {
          current.webview.postMessage({ type: 'state', ...await api('/investigation') });
        } else if (msg.action === 'cancel') {
          await api('/investigation/cancel', 'POST', { jobId: msg.id });
          current.webview.postMessage({ type: 'state', ...await api('/investigation') });
        } else if (msg.action === 'assistant') {
          await vscode.commands.executeCommand('workbench.action.closePanel');
        } else if (msg.action === 'terminal') {
          if (!terminal) terminal = vscode.window.createTerminal({ name: 'RELAI · Terminal', cwd: '/home/user/airline' });
          terminal.show();
        } else if (msg.action === 'evidence') {
          await vscode.commands.executeCommand('markdown.showPreview', vscode.Uri.file('/home/user/airline/demo/checkpoint/PLAYGROUND.md'));
        } else if (msg.action === 'changes') {
          await vscode.commands.executeCommand('workbench.view.scm');
        }
      } catch (error) { current.webview.postMessage({ type: 'error', message: error.message }); }
    });
    current.webview.html = (await fs.readFile(path.join(context.extensionPath, 'panel.html'), 'utf8')).replaceAll('NONCE', nonce);
  };
  context.subscriptions.push(vscode.commands.registerCommand('relai.playground', open));
  void open().catch(() => vscode.window.showErrorMessage('RELAI playground could not open. Check session provisioning.'));
}
module.exports = { activate };
