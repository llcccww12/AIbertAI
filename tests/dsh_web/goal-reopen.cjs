const assert = require('node:assert/strict');
const { setTimeout: delay } = require('node:timers/promises');
const { chromium } = require(process.env.DSH_PLAYWRIGHT_CORE || '../../AIbertAI/code/app/node_modules/playwright-core');

// Start TestDSHWebFixture with DSH_WEB_SERVE=1, DSH_WEB_GOAL=1 and DSH_WEB_GOAL_HOLD=1.
async function run() {
  const base = process.env.DSH_WEB_BASE || 'http://127.0.0.1:18767';
  const browser = await chromium.launch({ executablePath: process.env.DSH_CHROME, headless: true });
  const context = await browser.newContext({ locale: 'zh-CN' });
  try {
    const read = async () => (await context.request.get(base + '/fixture')).json();
    async function waitForStatus(label, match) {
      const deadline = Date.now() + 75_000;
      let state;
      do {
        state = await read();
        if (match(state)) return state;
        await delay(150);
      } while (Date.now() < deadline);
      throw Error(label + ': ' + JSON.stringify(state));
    }
    const first = await context.newPage();
    const initial = await read();
    const url = base + '/sessions/' + initial.sessionId;
    await first.goto(url);
    const login = first.getByRole('button', { name: '登录', exact: true });
    const more = first.getByRole('button', { name: '更多操作', exact: true });
    await login.or(more).first().waitFor();
    if (await login.isVisible()) {
      await first.locator('input').first().fill(base);
      await login.click();
      await first.waitForURL(address => address.pathname !== '/login');
      await first.goto(url);
    }
    if (process.env.DSH_WEB_TEST_CHAT_STOP === '1') {
      const composer = first.locator('.conversation-composer');
      const input = composer.locator('textarea');
      const primary = composer.locator('.composer-send');
      const action = value => composer.locator('.composer-send[data-action="' + value + '"]');
      await input.fill('验收停止');
      await action('send').click();
      await action('stop').waitFor();
      assert.equal(await primary.count(), 1);
      await input.fill('追加说明');
      await action('send').waitFor();
      assert.equal(await action('send').isEnabled(), true);
      await input.fill('');
      const verifyReceipt = process.env.DSH_WEB_CANCEL_RECOVERY === '1';
      if (verifyReceipt) await first.getByText('已开始处理。', { exact: false }).first().waitFor();
      await action('stop').click();
      await action('send').waitFor();
      assert.equal(await primary.isDisabled(), true);
      if (verifyReceipt) {
        const interrupted = first.getByText('已停止', { exact: true }).first();
        await interrupted.waitFor();
        await first.reload();
        await interrupted.waitFor();
        await input.fill('在原会话继续');
        await action('send').click();
        await first.getByText('模拟任务已完成。', { exact: false }).first().waitFor();
        const finished = await read();
        assert.equal(finished.sessionId, initial.sessionId);
        assert.equal(finished.modelCalls, 2);
      }
      console.log('The same primary button sent, offered queuing a draft, and stopped the running reply');
      return;
    }
    await more.click();
    await first.getByText('目标', { exact: true }).click();
    const objective = '检查工作区文件并汇报结果';
    await first.locator('textarea').fill(objective);
    await first.getByRole('button', { name: '设为目标' }).click();
    const held = await waitForStatus('second invocation did not reach gate',
      state => state.goalHeld === true && state.viewers > 0);
    assert.equal(held.modelCalls, 1);
    assert.equal(held.sessionId, initial.sessionId);
    assert.match(held.dshSessionId, /^[0-9a-f-]{36}$/);
    const liveClock = first.locator('[data-chat-running]');
    const initialClock = await liveClock.getByText(/深度求索中，用时/).innerText();
    await first.waitForFunction(previous => {
      const clock = document.querySelector('[data-chat-running]');
      return clock && !clock.innerText.includes(previous);
    }, initialClock, { timeout: 6000 });
    const primary = first.locator('.conversation-composer .composer-send');
    await first.locator('.composer-send[data-action="stop"]').waitFor();
    assert.equal(await primary.count(), 1);
    assert.equal(await primary.getAttribute('aria-label'), '暂停目标');
    assert.equal(await first.getByRole('button', { name: '暂停目标', exact: true }).count(), 1);
    if (process.env.DSH_WEB_TEST_PAUSE === '1') {
      await primary.click();
      await waitForStatus('pause action did not reach the goal', state => state.goalStatus === 'paused');
      console.log('Paused the active goal through the primary composer button');
      return;
    }
    await first.close();
    await waitForStatus('viewer still connected', state => state.viewers === 0);
    assert.equal(context.pages().length, 0);
    // The same Turn stays held while the observer leaves and comes back.
    await delay(2200);
    const observing = await context.newPage();
    await observing.goto(url);
    const restoredClock = await observing.locator('[data-chat-running]').getByText(/深度求索中，用时/).innerText();
    const seconds = label => {
      const time = /用时\s*(?:(\d+)小时)?(?:(\d+)分)?(\d+)秒/.exec(label);
      assert(time, label);
      return Number(time[1] || 0) * 3600 + Number(time[2] || 0) * 60 + Number(time[3]);
    };
    assert(seconds(restoredClock) >= seconds(initialClock) + 2, restoredClock);
    await observing.close();
    await waitForStatus('returning viewer still connected', state => state.viewers === 0);
    assert.equal((await context.request.post(base + '/goal/release')).status(), 204);
    const done = await waitForStatus('goal did not finish while window was closed',
      state => state.goalStatus === 'completed' && state.modelCalls >= 5 && state.turnStatus !== 'running');
    assert.equal(done.viewers, 0);
    assert.equal(done.sessionId, held.sessionId);
    assert.equal(done.dshSessionId, held.dshSessionId);
    const reopened = await context.newPage();
    await reopened.goto(url);
    await reopened.getByText('模拟目标已完成。', { exact: false }).first().waitFor();
    assert.equal(new URL(reopened.url()).pathname, '/sessions/' + held.sessionId);
    assert.equal(await reopened.locator('[data-chat-running]').count(), 0);
    const reply = reopened.locator('.dsh-native-reply');
    await reply.first().waitFor();
    const summaries = reply.locator('[data-turn-process]');
    for (const summary of await summaries.all()) {
      if (await summary.isEnabled() && await summary.getAttribute('aria-expanded') !== 'true') await summary.click();
    }
    await reply.locator('[data-dsh-tool="read"]').first().waitFor();
    assert.equal(await reopened.locator('.conversation-composer .composer-send').getAttribute('data-action'), 'send');
    await reopened.getByRole('button', { name: '更多操作', exact: true }).click();
    const toolsMenu = reopened.getByRole('menuitem', { name: /DSH 工具调用记录/ });
    await toolsMenu.click();
    await reopened.getByText('read', { exact: true }).last().waitFor();
    await reopened.keyboard.press('Escape');
    await reopened.keyboard.press('Escape');
    const finalState = await read();
    assert.equal(finalState.dshSessionId, held.dshSessionId);
    const { used, window: size, breakdown } = finalState.context;
    assert(Number.isSafeInteger(used) && used >= 0 && size === 100000);
    assert(breakdown && breakdown.messageTokens > 0 && breakdown.toolsTokens > 0);
    await reopened.getByRole('button', { name: '上下文已用 ' + Math.min(100, Math.round(100 * used / size)) + '%' }).click();
    const panel = reopened.locator('.dsh-meter-panel');
    assert.deepEqual(await panel.locator('dt').allTextContents(), ['系统提示词', '工具定义', '对话消息']);
    const { formatContextTokens } = await import('../../AIbertAI/code/web/dsh/context-meter.mjs');
    assert((await panel.innerText()).includes('~' + formatContextTokens(used) + ' / ' + formatContextTokens(size)));
    assert.deepEqual(await panel.locator('dd').allTextContents(),
      [breakdown.systemTokens, breakdown.toolsTokens, breakdown.messageTokens].map(n => '~' + formatContextTokens(n)));
    if (process.env.DSH_GOAL_SCREENSHOT) await reopened.screenshot({ path: process.env.DSH_GOAL_SCREENSHOT, animations: 'disabled' });
    console.log(JSON.stringify({ session: held.sessionId, sdkSession: done.dshSessionId,
      callsAtClose: held.modelCalls, callsBeforeReopen: done.modelCalls, disconnectedStatus: done.goalStatus,
      restoredResponse: true, used, window: size, breakdown }));
  } finally {
    await context.request.post(base + '/fixture/finish').catch(() => {});
    await browser.close();
  }
}
run().catch(error => { console.error(error); process.exitCode = 1; });
