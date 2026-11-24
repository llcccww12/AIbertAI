// Isolated adapter for exercising the Go desktop bridge with the real worker.
const { app, utilityProcess } = require('electron');
const readline = require('readline');
const fs = require('fs'); const path = require('path'); let dropId = null;
app.whenReady().then(() => {
  const child = utilityProcess.fork(process.argv[2], [], { stdio: 'pipe' });
  child.stderr.on('data', data => process.stderr.write(data));
  child.on('message', message => {
    if (message.type === 'private-catalog-result') {
      if (message.id === dropId) { child.kill(); app.exit(0); return; }
      const result = { type: 'design_tool_result', id: message.id };
      if (message.error) result.error = 'design_' + message.error.replace(/^design_/, '');
      else result.result = message.data;
      process.stdout.write(JSON.stringify(result) + '\n');
    }
  });
  const sharp = require(require.resolve('sharp', { paths: [path.dirname(process.argv[2])] }));
  sharp({ create: { width: 1, height: 1, channels: 3, background: 'white' } }).png().toBuffer().then(
    () => child.postMessage({ type: 'image-preflight', result: { state: 'passed' } }),
    () => child.postMessage({ type: 'image-preflight', result: { state: 'failed', reason: 'probe_failed' } }),
  );
  child.postMessage({ type: 'init-private-catalog', origin: process.env.DSH_TEST_CANVAS_CATALOG_ORIGIN || 'http://127.0.0.1:9' });
  readline.createInterface({ input: process.stdin }).on('line', line => {
    const event = JSON.parse(line);
    if (event.name === 'canvas_create_text' && process.env.DSH_TEST_CANVAS_SCENARIO === 'drop') dropId = event.id;
    if (event.name === 'canvas_create_text' && process.env.DSH_TEST_CANVAS_SCENARIO === 'conflict') { const file=path.join(event.cwd,event.design.localDesignPath,'canvas.json'); const canvas=JSON.parse(fs.readFileSync(file)); canvas.revision=(canvas.revision||0)+1; fs.writeFileSync(file,JSON.stringify(canvas)); }
    const open = event.name === 'design_session_open';
    const style = event.name === 'design_test_style';
    const direct = open || style || event.name === 'design_tool_catalog' || event.name === 'design_guard_capabilities';
    const args = open ? { cwd: event.cwd, sessionId: event.sessionId, selection: event.arguments } : style ? { cwd: event.cwd, payload: event.design, params: { sessionId: event.sessionId, decision: 'skipped' } } : {
      cwd: event.cwd, sessionId: event.sessionId, design: event.design, name: event.name, args: event.arguments,
      ...(event.expectedRevision === undefined ? {} : { expectedRevision: event.expectedRevision, expectedCanvasDigest: event.expectedCanvasDigest, expectedArtifactsDigest: event.expectedArtifactsDigest, expectedContextDigest: event.expectedContextDigest, expectedSourceDigest: event.expectedSourceDigest })
    };
    child.postMessage({ type: 'private-catalog', id: event.id, tool: style ? 'design_style_decision' : direct ? event.name : 'design_model_tool', args });
  }).on('close', () => { child.kill(); app.quit(); });
});
