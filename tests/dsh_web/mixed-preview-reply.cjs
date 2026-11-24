const assert = require('node:assert/strict');
const { chromium } = require(process.env.DSH_PLAYWRIGHT_CORE || '../../AIbertAI/code/app/node_modules/playwright-core');

// TestDSHWebFixture with DSH_WEB_STRUCTURED=1. The existing Markdown preview
// chunk also imports the employee shell. Loading it must reuse the mounted React.
async function run() {
  const base = process.env.DSH_WEB_BASE || 'http://127.0.0.1:18767';
  const browser = await chromium.launch({ executablePath: process.env.DSH_CHROME, headless: true });
  try {
    const page = await browser.newPage({ locale: 'zh-CN' });
    const { sessionId } = await (await page.request.get(base + '/fixture')).json();
    const url = base + '/sessions/' + sessionId;
    const login = page.getByRole('button', { name: '登录', exact: true });
    const role = page.locator('header [data-dsh-preset]').getByRole('button');
    await page.goto(url);
    await login.or(role).first().waitFor();
    if (await login.isVisible()) {
      await page.locator('input').first().fill(base);
      await login.click();
      await page.waitForURL(location => location.pathname !== '/login');
      await page.goto(url);
    }
    for (let visit = 0; visit < 2; visit++) {
      if (visit) await page.reload();
      await role.waitFor();
      await page.locator('.dsh-native-reply').first().waitFor();
      await page.evaluate(() => import('/assets/markstream-L8TZCC5v.js'));
      const instances = await page.evaluate(() => performance.getEntriesByType('resource')
        .filter(resource => new URL(resource.name).pathname.endsWith('/index-zDHqw1Zn.js'))
        .map(resource => resource.name));
      assert.equal(new Set(instances).size, 1, 'one React runtime for previews and messages');
      const summary = page.locator('[data-turn-process]').first();
      if (await summary.getAttribute('aria-expanded') !== 'true') await summary.click();
      await page.locator('[data-dsh-tool=skill] [data-disclosure-row]').first().waitFor();
      assert.equal(await page.locator('[role=alert]:has(pre)').count(), 0);
    }
    console.log('History and its DSH process render after loading the Markdown preview and reopening the employee session.');
  } finally {
    await browser.close();
  }
}
run().catch(error => { console.error(error); process.exitCode = 1; });
