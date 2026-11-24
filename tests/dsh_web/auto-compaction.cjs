const assert = require('node:assert/strict');
const { chromium } = require(process.env.DSH_PLAYWRIGHT_CORE || '../../AIbertAI/code/app/node_modules/playwright-core');

// TestDSHWebFixture with DSH_WEB_NATIVE_COMPACT=auto holds the real summarizer briefly.
async function run() {
  const base = process.env.DSH_WEB_BASE || 'http://127.0.0.1:18767';
  const browser = await chromium.launch({ executablePath: process.env.DSH_CHROME, headless: true });
  const context = await browser.newContext({locale: 'zh-CN'});
  try {
    const {sessionId} = await (await context.request.get(base + '/fixture')).json();
    const url = base + '/sessions/' + sessionId;
    let page = await context.newPage();
    await page.goto(url);
    const login = page.getByRole('button', {name: '登录', exact: true});
    const composer = page.locator('.conversation-composer');
    await login.or(composer).first().waitFor();
    if (await login.isVisible()) {
      await page.locator('input').first().fill(base);
      await login.click();
      await page.waitForURL(address => address.pathname !== '/login');
      await page.goto(url);
    }
    const marker = '.message-compaction[data-kind="auto"]';
    const pending = page.locator(marker + '[data-status="in_progress"]');
    for (let i = 0; i < 10; i++) {
      await composer.locator('textarea').fill('资料 ' + i + ' ' + 'historical material '.repeat(500));
      await composer.locator('.composer-send[data-action="send"]').click();
      await composer.locator('.composer-send[data-action="stop"]').or(pending).first().waitFor();
      await composer.locator('.composer-send[data-action="send"]').or(pending).first().waitFor();
      if (await pending.isVisible()) break;
    }
    await pending.waitFor({timeout: 4000});
    assert.equal(await pending.count(), 1);
    await page.close();
    page = await context.newPage();
    await page.goto(url);
    const finished = page.getByText('上下文已压缩（自动）', {exact: true});
    await finished.waitFor();
    await page.reload();
    await finished.waitFor();
    assert.equal(await page.locator(marker).count(), 1);
    assert.equal(await page.getByText(/模拟摘要：已核对隔离会话资料/).count(), 0);
    console.log('Native automatic compaction was visible live, finished after close, and appeared once on reopen.');
  } finally {
    await context.request.post(base + '/fixture/finish').catch(() => {});
    await browser.close();
  }
}
run().catch(err => {console.error(err); process.exitCode = 1});
