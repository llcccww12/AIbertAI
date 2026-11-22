const assert = require('node:assert/strict');
const { chromium } = require(process.env.DSH_PLAYWRIGHT_CORE || '../../AIbertAI/code/app/node_modules/playwright-core');

// Run against TestDSHWebFixture with DSH_WEB_SERVE=1 and DSH_WEB_TEAM_SWITCH=1.
async function run() {
  const base = process.env.DSH_WEB_BASE || 'http://127.0.0.1:18767';
  const browser = await chromium.launch({ executablePath: process.env.DSH_CHROME, headless: true });
  const context = await browser.newContext({ locale: 'zh-CN' });
  const page = await context.newPage();
  const toggle = page.getByRole('switch', { name: '启用智能体团队协作' });
  const model = page.getByRole('button', { name: /选择模型/ });
  const menu = async () => {
    await model.click();
    await toggle.waitFor();
  };
  const checked = async expected => {
    await page.waitForFunction(value => [...document.querySelectorAll('[role="switch"]')].some(
      node => node.getAttribute('aria-label') === '启用智能体团队协作' && node.getAttribute('aria-checked') === String(value)
    ), expected);
  };
  const closeMenu = async () => {
    if (await toggle.isVisible()) {
      await page.keyboard.press('Escape');
      await toggle.waitFor({ state: 'hidden' });
    }
  };
  const send = async text => {
    const composer = page.locator('.conversation-composer');
    await composer.locator('textarea').fill(text);
    await composer.locator('.composer-send[data-action="send"]').click();
  };
  const rpc = (sessionId, enabled) => page.evaluate(({ sessionId, enabled }) => new Promise((resolve, reject) => {
    const socket = new WebSocket('ws://' + location.host + '/acp', ['monkeyai.v1', 'monkeyai.token.fixture']);
    const timeout = setTimeout(() => { socket.close(); reject(Error('ACP timeout')); }, 10000);
    const call = (id, method, params) => socket.send(JSON.stringify({ jsonrpc: '2.0', id, method, params }));
    socket.onopen = () => call(1, 'initialize', { protocolVersion: 1 });
    socket.onmessage = event => {
      const result = JSON.parse(event.data);
      if (result.id === 1) {
        if (result.error) { clearTimeout(timeout); socket.close(); reject(Error(result.error.message)); return; }
        call(2, '_aibertai/session/agent-team', { sessionId, enabled });
      } else if (result.id === 2) { clearTimeout(timeout); socket.close(); resolve(result); }
    };
  }), { sessionId, enabled });
  try {
    const fixture = await (await context.request.get(base + '/fixture')).json();
    assert.match(fixture.sessionId, /^[a-f0-9-]{36}$/);
    const start = page.getByRole('button', { name: '新建会话', exact: true });
    const login = page.getByRole('button', { name: '登录', exact: true });
    const existing = base + '/sessions/' + fixture.sessionId;
    await page.goto(existing);
    await login.or(start).first().waitFor();
    if (await login.isVisible()) {
      await page.locator('input').first().fill(base);
      await login.click();
      await page.waitForURL(url => url.pathname !== '/login');
      await page.goto(existing);
    }
    await start.click();
    await page.waitForURL(url => url.pathname === '/');
    await page.reload();
    const presetIcon = page.locator('.conversation-composer [data-dsh-preset] svg[data-icon="dsw-icon-agent-preset-outline-regular"] use');
    await presetIcon.waitFor({ state: 'attached' });
    assert(await presetIcon.evaluate(use => use.getBBox().width > 0), 'preset icon must paint on a fresh composer');
    await menu();
    await checked(false);
    const info = page.getByRole('button', { name: '关于智能体团队协作' });
    await info.click();
    await page.getByRole('tooltip').filter({ hasText: /token/i }).waitFor();
    await checked(false);
    await page.keyboard.press('Escape');
    await closeMenu();

    await send('核对新会话默认工具');
    await page.getByText('模型未收到共享任务工具。', { exact: true }).waitFor();
    const defaultSession = page.url();
    await page.reload();
    await menu();
    await checked(false);
    await closeMenu();
    await start.click();

    // A design draft has its own expert selection, but uses the same model menu.
    await page.getByRole('button', { name: /网页原型设计/ }).click();
    await menu();
    await checked(false);
    await closeMenu();
    await page.getByRole('button', { name: /网页原型设计/ }).click();

    await menu();
    await toggle.click();
    await checked(true);
    await closeMenu();
    await send('保存任务');
    await page.waitForURL(url => /^\/sessions\/[a-f0-9-]{36}$/.test(url.pathname) && !url.pathname.endsWith(fixture.sessionId));
    const url = page.url();
    const sessionId = new URL(url).pathname.split('/').pop();
    await page.locator('.composer-send[data-action="stop"]').waitFor();
    const busy = await rpc(sessionId, false);
    assert.match(busy.error?.message ?? '', /session_busy/);
    assert.equal((await context.request.post(base + '/fixture/team-release')).status(), 204);
    await page.getByText('模型收到了共享任务工具。', { exact: true }).waitFor();
    await page.reload();
    await menu();
    await checked(true);
    await toggle.click();
    const dialog = page.getByRole('dialog').or(page.getByRole('alertdialog')).first();
    await dialog.getByRole('button', { name: '取消', exact: true }).click();
    await menu();
    await checked(true);
    await toggle.click();
    await dialog.getByRole('button', { name: '确认停用', exact: true }).click();
    await dialog.waitFor({ state: 'hidden' });
    await page.reload();
    await menu();
    await checked(false);
    await closeMenu();
    await send('核对已关闭的工具');
    await page.getByText('模型未收到共享任务工具。', { exact: true }).waitFor();
    await menu();
    await toggle.click();
    await checked(true);
    await closeMenu();
    await send('找回任务');
    await page.getByText('已从真实任务板找到原任务。', { exact: true }).waitFor();
    assert.equal(page.url(), url);
    assert.notEqual(url, defaultSession);

    await start.click();
    await menu();
    await checked(false);
    await closeMenu();
    await page.getByRole('button', { name: '标准模式', exact: true }).click();
    await page.getByText('受限测试专家', { exact: true }).click();
    await send('核对专家工具');
    await page.waitForURL(address => /^\/sessions\/[a-f0-9-]{36}$/.test(address.pathname) && address.href !== url);
    await page.getByText('模型未收到共享任务工具。', { exact: true }).waitFor();
    const denied = await rpc(new URL(page.url()).pathname.split('/').pop(), true);
    assert.match(denied.error?.message ?? '', /agent_team_unavailable/);
    await menu();
    assert(await toggle.isDisabled(), 'restricted expert switch should be disabled');
    console.log('Passed: default, design entry, busy guard, close/reopen, disable confirmation, SDK tools, board retention and expert denial');
  } finally {
    await context.request.post(base + '/fixture/finish').catch(() => {});
    await browser.close();
  }
}
run().catch(error => { console.error(error); process.exitCode = 1; });

