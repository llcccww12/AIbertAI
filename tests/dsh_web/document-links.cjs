const assert = require('node:assert/strict')
const { chromium } = require(process.env.DSH_PLAYWRIGHT_CORE || '../../AIbertAI/code/app/node_modules/playwright-core')

async function run() {
  const base = process.env.DSH_WEB_BASE || 'http://127.0.0.1:18926'
  const browser = await chromium.launch({ executablePath: process.env.DSH_CHROME, headless: true })
  const context = await browser.newContext({ locale: 'zh-CN', viewport: { width: 1550, height: 920 } })
  const page = await context.newPage()
  page.setDefaultTimeout(12000)
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  try {
    const { sessionId } = await (await context.request.get(base + '/fixture')).json()
    const url = base + '/sessions/' + sessionId
    await page.goto(url)
    const login = page.getByRole('button', { name: '登录', exact: true })
    await login.or(page.getByRole('button', { name: /会话的工具/ })).first().waitFor()
    if (await login.isVisible()) {
      await page.locator('input').first().fill(base)
      await login.click()
      await page.waitForURL(address => address.pathname !== '/login')
      await page.goto(url)
    }
    const markdown = page.locator('[data-document-markdown]')
    const guide = page.getByRole('button', { name: '查看开发指南', exact: true })
    await guide.click()
    await markdown.getByRole('heading', { name: '工作区指南' }).waitFor()
    const colors = selector => page.waitForFunction(selector => {
      const tokens = Array.from(document.querySelectorAll(selector))
      const palette = new Set(tokens.map(el => getComputedStyle(el).color))
      return palette.size >= 2
    }, selector)
    await colors('[data-document-markdown] code .line span')
    await markdown.getByText('跳至验证章节', { exact: true }).click()
    await page.waitForFunction(() => document.activeElement?.textContent === '验证章节')
    await markdown.getByText('另一份文档', { exact: true }).click()
    await markdown.getByRole('heading', { name: '参考资料' }).waitFor()
    await page.waitForFunction(() => document.activeElement?.textContent === '验证章节')
    await page.reload()
    await markdown.getByRole('heading', { name: '参考资料' }).waitFor()
    await page.waitForFunction(() => document.activeElement?.textContent === '验证章节')
    await guide.click()
    const external = 'https://docs.example.invalid/demo#setup'
    await context.route('https://docs.example.invalid/**', route => route.fulfill({ contentType: 'text/html; charset=utf-8', body: '<h1>External document</h1>' }))
    const opened = context.waitForEvent('page')
    await markdown.getByRole('link', { name: '查看网页' }).click()
    const popup = await opened
    await popup.getByRole('heading', { name: 'External document' }).waitFor({ timeout: 10000 }).catch(error => { throw new Error(popup.url() + ': ' + error.message) })
    assert.equal(popup.url(), external)
    await popup.close()
    const viewer = page.locator('[data-document-viewer-menu]')
    await viewer.click()
    await page.getByRole('menuitem', { name: '代码', exact: true }).click()
    await colors('[data-code-preview] code .line span')
    await viewer.click()
    await page.getByRole('menuitem', { name: 'Markdown', exact: true }).click()
    await markdown.getByRole('button', { name: '定位配置' }).click()
    const target = page.locator('[data-code-preview] .line[aria-current="location"]')
    await target.waitFor()
    assert.match(await target.innerText(), /"label": "演示"/)
    await colors('[data-code-preview] code .line span')
    await guide.click()
    await markdown.getByText('打开长文', { exact: true }).click()
    await markdown.getByRole('heading', { name: '长文首页' }).waitFor()
    await markdown.getByRole('button', { name: '分页中的本地链接' }).click()
    await page.locator('[data-textpreview-plain]').getByText('同目录定位成功。').waitFor()
    assert.deepEqual(errors, [])
    console.log('Passed native Markdown token colors, Unicode same-page and cross-document anchors, reload, HTTP link and highlighted JSON line navigation.')
  } finally {
    await context.request.post(base + '/fixture/finish').catch(() => {})
    await browser.close()
  }
}
run().catch(error => { console.error(error); process.exitCode = 1 })
