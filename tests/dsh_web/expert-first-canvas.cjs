const assert = require('node:assert/strict');
const { chromium } = require(process.env.DSH_PLAYWRIGHT_CORE || '../../AIbertAI/code/app/node_modules/playwright-core');

// TestDSHWebFixture with DSH_WEB_TEAM_SWITCH=1 provides a selectable custom expert.
// The browser adapter supplies only the two Electron canvas APIs used by the chat.
async function run() {
  const base = process.env.DSH_WEB_BASE || 'http://127.0.0.1:18767';
  const browser = await chromium.launch({ executablePath: process.env.DSH_CHROME });
  const context = await browser.newContext({ locale: 'zh-CN', viewport: { width: 1280, height: 840 } });
  const page = await context.newPage();
  await context.addInitScript(() => {
    const listeners = new Set();
    window.fixtureBoard = {
      ready: false, revision: 1, calls: [], reads: [], listeners,
      emit: id => listeners.forEach(listener => listener(id)),
      onPrivateCanvasChanged(listener) {
        listeners.add(listener);
        return () => listeners.delete(listener);
      },
      async privateDesignWorkspace(tool, { sessionId }) {
        const board = window.fixtureBoard;
        const canvasId = 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee';
        board.calls.push(tool);
        if (tool === 'design_state') {
          board.reads.push(sessionId);
          if (!board.ready) throw Error('not_a_design_session');
          return { designSessionId: sessionId, canvasId, revision: 1, localDesignPath: '.aibertai/design/' + canvasId,
            phase: 'BRIEFING', snapshot: { localVersion: 1, remoteVersion: 0, status: 'local' }, provider: { status: 'unavailable' } };
        }
        if (tool === 'canvas_get') return {
          document: { id: canvasId, schemaVersion: 1, title: 'fixture', createdAt: '2026-01-01T00:00:00Z',
            updatedAt: '2026-01-01T00:00:0' + board.revision + 'Z', revision: board.revision,
            nodes: [{ id: 'expert-first-layer', type: 'text', title: '画布输出', content: '画布第' + board.revision + '版',
              contentFormat: 'markdown', textRole: 'note', x: 0, y: 0, width: 200, height: 90, zIndex: 0,
              fontFamily: 'Arial', fontSize: 16, fontWeight: 400, color: '#222222', textAlign: 'left', lineHeight: 1.5 }] },
          generationTasks: [],
        };
        if (tool === 'canvas_select') return { selection: { nodeIds: [] } };
        if (tool === 'canvas_selected_sources') return [];
        throw Error('Unexpected canvas request ' + tool);
      },
    };
  });
  await page.route('**/assets/index-zDHqw1Zn.js*', async route => {
    const response = await route.fetch();
    const body = await response.text();
    const original = 'return window.aibertai ?? QKe();';
    assert(body.includes(original));
    await route.fulfill({ response, body: body.replace(original,
      'return window.aibertai = { ...QKe(), onPrivateCanvasChanged: window.fixtureBoard.onPrivateCanvasChanged, privateDesignWorkspace: window.fixtureBoard.privateDesignWorkspace };') });
  });
  try {
    const { sessionId } = await (await context.request.get(base + '/fixture')).json();
    const existing = base + '/sessions/' + sessionId;
    const currentRole = page.locator('header [data-dsh-preset]').getByRole('button');
    const login = page.getByRole('button', { name: '登录', exact: true });
    await page.goto(existing);
    await login.or(currentRole).first().waitFor();
    if (await login.isVisible()) {
      await page.locator('input').first().fill(base);
      await login.click();
      await page.waitForURL(url => url.pathname !== '/login');
      await page.goto(existing);
    }
    await currentRole.waitFor();
    await page.waitForFunction(() => window.fixtureBoard.listeners.size > 0);
    const layer = page.locator('[data-canvas-node="expert-first-layer"]');

    await page.evaluate(() => window.fixtureBoard.emit('ffffffff-ffff-4fff-8fff-ffffffffffff'));
    assert.deepEqual(await page.evaluate(() => window.fixtureBoard.reads), []);
    await page.evaluate(id => window.fixtureBoard.emit(id), sessionId);
    await page.waitForFunction(() => window.fixtureBoard.reads.length === 1);
    assert.equal(await page.locator('.design-surface-pane').count(), 0);
    assert.equal(await page.evaluate(() => window.fixtureBoard.calls.includes('design_session_open')), false);

    assert.equal(await currentRole.innerText(), '标准模式');
    await currentRole.click();
    const standard = page.getByRole('menuitem', { name: /标准模式.*处理代码、文件和资料/ });
    await standard.waitFor();
    await page.getByText('受限测试专家', { exact: true }).click();
    await page.getByRole('button', { name: '受限测试专家', exact: true }).waitFor();
    await page.evaluate(id => { window.fixtureBoard.ready = true; window.fixtureBoard.emit(id); }, sessionId);
    await layer.waitFor();
    assert.match(await layer.innerText(), /画布第1版/);
    await page.evaluate(id => { window.fixtureBoard.revision = 2; window.fixtureBoard.emit(id); }, sessionId);
    await page.waitForFunction(() => document.querySelector('[data-canvas-node="expert-first-layer"]')?.textContent.includes('画布第2版'));
    await page.getByRole('button', { name: '受限测试专家', exact: true }).waitFor();
    assert.equal(await page.evaluate(() => window.fixtureBoard.calls.includes('design_session_open')), false);
    await currentRole.click();
    await standard.click();
    await page.getByRole('button', { name: '标准模式', exact: true }).waitFor();
    await page.reload();
    await page.locator('header').getByRole('button', { name: '标准模式', exact: true }).waitFor();
    console.log('Passed: foreign session ignored, file-only event does not open canvas, first saved canvas appears, later updates render and selected expert is preserved.');
  } finally {
    await context.request.post(base + '/fixture/finish').catch(() => {});
    await browser.close();
  }
}
run().catch(error => { console.error(error); process.exitCode = 1; });

