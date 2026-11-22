const assert = require('node:assert/strict')
const { chromium } = require(process.env.DSH_PLAYWRIGHT_CORE || '../../AIbertAI/code/app/node_modules/playwright-core')

async function run() {
  const base = process.env.DSH_WEB_BASE || 'http://127.0.0.1:18767'
  const browser = await chromium.launch({ executablePath: process.env.DSH_CHROME, headless: true })
  const context = await browser.newContext({ locale: 'zh-CN', viewport: { width: 1600, height: 900 } })
  const page = await context.newPage()
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  try {
    const { sessionId } = await (await context.request.get(base + '/fixture')).json()
    const url = base + '/sessions/' + sessionId
    await page.goto(url)
    const login = page.getByRole('button', { name: '登录', exact: true })
    const guide = page.getByRole('button', { name: '查看开发指南', exact: true })
    await login.or(guide).first().waitFor()
    if (await login.isVisible()) {
      await page.locator('input').first().fill(base)
      await login.click()
      await page.waitForURL(u => u.pathname !== '/login')
      await page.goto(url)
    }
    await guide.click()
    await page.getByRole('button', { name: '打开长文' }).click()
    const markdown = page.locator('[data-document-markdown]')
    const heading = markdown.getByRole('heading', { name: '第二页目标' })
    await markdown.getByRole('heading', { name: '长文首页' }).waitFor()
    assert.equal(await heading.count(), 0)
    await page.getByRole('button', { name: '加载更多' }).click()
    await heading.waitFor()
    await page.locator('[role=tab]').filter({ hasText:'README.md' }).click()
    await page.getByRole('button', { name: '定位大文件行' }).click()
    const target = page.locator('[data-code-preview] .line[aria-current="location"]')
    await target.waitFor()
    assert.match(await target.innerText(), /target750/)
    assert.ok(await page.getByRole('button', { name: '加载更多' }).isVisible())
    assert.deepEqual(errors, [])
    console.log('Passed paged 1+ MiB Markdown, later heading and source line 750 with additional pages available.')
  } finally {
    await context.request.post(base + '/fixture/finish').catch(() => {})
    await browser.close()
  }
}
run().catch(error => { console.error(error); process.exitCode = 1 })
