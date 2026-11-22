const assert = require('node:assert/strict');
const { chromium } = require(process.env.DSH_PLAYWRIGHT_CORE || '../../AIbertAI/code/app/node_modules/playwright-core');

// Start TestDSHWebFixture with DSH_WEB_GOAL=1 and DSH_WEB_DYNAMIC=todos.
async function run() {
  const base = process.env.DSH_WEB_BASE || 'http://127.0.0.1:18767';
  const browser = await chromium.launch({ executablePath: process.env.DSH_CHROME, headless: true });
  const context = await browser.newContext({ locale: 'zh-CN' });
  try {
    const initial = await (await context.request.get(base + '/fixture')).json();
    const url = base + '/sessions/' + initial.sessionId;
    const first = await context.newPage();
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
    await more.click();
    await first.getByText('目标', { exact: true }).click();
    await first.locator('textarea').fill('用待办清单跟踪画布制作进度');
    await first.getByRole('button', { name: '设为目标' }).click();

    async function verifyActive(page) {
      const panel = page.getByTestId('todo-panel');
      const header = panel.getByRole('button');
      await header.getByText(/1 进行中.*1 待处理/).waitFor({timeout: 30000});
      assert.equal(await panel.count(), 1);
      assert.equal(await page.locator('.task-progress').count(), 0);
      const timer = page.locator('.dsh-native-theme [class*="runningText"]');
      await timer.waitFor({timeout: 12000});
      const before = await timer.innerText();
      await page.waitForFunction(prior => {
        const text = document.querySelector('.dsh-native-theme [class*="runningText"]')?.innerText;
        return text && text !== prior;
      }, before, {timeout: 5000});
      await header.click();
      await panel.getByText('整理测试画布', {exact:true}).waitFor();
      await panel.getByText('交付测试画布', {exact:true}).waitFor();
      assert.equal(await header.getAttribute('aria-expanded'), 'true');
    }
    await verifyActive(first);
    await first.close();
    const reopened = await context.newPage();
    await reopened.goto(url);
    await verifyActive(reopened);
    assert.equal((await context.request.post(base + '/goal/release')).status(), 204);
    await reopened.getByTestId('todo-panel').getByText('2 已完成', {exact:true}).waitFor({timeout: 15000});
    const end = await (await context.request.get(base + '/fixture')).json();
    assert.equal(end.dshSessionId,initial.sessionId);
    console.log('Employee workbench: one native todo panel, live timer, actual todo_write, close/reopen and completion passed');
  } finally {
    await context.request.post(base + '/fixture/finish').catch(() => {});
    await browser.close();
  }
}
run().catch(error => {console.error(error);process.exitCode=1});
