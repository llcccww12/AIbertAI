// node tests/dsh_web/markdown-layout.cjs /path/to/prepared-DSH/resources/app
const assert = require('node:assert/strict')
const fs = require('node:fs/promises')
const path = require('node:path')
const { createRequire } = require('node:module')
const work = path.resolve(__dirname, '../..')
const web = path.join(work, 'AIbertAI/code/web')
const { chromium } = require(path.join(work, 'AIbertAI/code/app/node_modules/playwright-core'))
const sdk = path.resolve(process.argv[2])
const esbuild = createRequire(path.join(sdk, 'packages/boot/hmr/package.json'))('esbuild')

async function run() {
  const runtime = await esbuild.build({ stdin: {
    contents: "import * as react from 'react';import * as jsx from 'react/jsx-runtime';import * as dom from 'react-dom';import {createRoot} from 'react-dom/client';export {react,jsx,dom,createRoot};",
    resolveDir: path.join(sdk, 'packages/client/ui-chat') }, format: 'esm', bundle: true, write: false,
    define: { 'process.env.NODE_ENV': '"production"' } })
  const html = await fs.readFile(path.join(web, 'index.html'), 'utf8')
  const browser = await chromium.launch({ executablePath: process.env.DSH_CHROME, headless: true })
  try {
    const page = await browser.newPage()
    await page.route('http://layout.test/**', async route => {
      const url = new URL(route.request().url())
      if (url.pathname === '/') return route.fulfill({ contentType: 'text/html', body: html
        .replace(/<script\b[^>]*><\/script>/g, '').replace('<div id="root"></div>', '<div id="root" class="chat-message"></div>') })
      if (url.pathname === '/fixture.js') return route.fulfill({ contentType: 'text/javascript', body: runtime.outputFiles[0].text })
      const file = path.join(web, url.pathname)
      if (!file.startsWith(web + path.sep)) return route.abort()
      await route.fulfill({ path: file })
    })
    await page.goto('http://layout.test/')
    await page.evaluate(async () => {
      const runtime = await import('/fixture.js')
      const { bindChatFramework } = await import('/dsh/chat-framework.mjs')
      bindChatFramework(runtime.react, runtime.jsx, runtime.dom)
      const { default: Reply, PresentedFileCard, EmployeeOpenTarget, deliverableZh } = await import('/dsh/native-chat/reply.js')
      const markdown = ['已转成 Markdown 版本 ✅', '📝 [**难忘的一天.md**](./难忘的一天.md)',
        '之前的 Word 版 [难忘的一天.docx](./short.docx) 也还保留在工作区。',
        '- 设计稿\n- [交付文件](./delivery.pdf)\n  1. 内层项', '1. 第一步\n2. 第二步',
        '附件 [2026年联合设计评审报告ABCDEFGHIJKLMNOPQRSTUVWXYZ详细修订记录.docx](./long.docx) 。',
        'See [design review notes](./notes.md) for details.'].join('\n\n')
      const reply = runtime.jsx.jsx(Reply, {
        message: { id:'markdown', parts: [{ type:'text', text:markdown }] }, toolCalls:{}, language:'zh-CN',
        openFile: p => { window.openedPath = p }, renderHostPart: () => null,
      })
      const openTarget = path => runtime.jsx.jsx(EmployeeOpenTarget, {
        scope: { accountId:'account', sessionId:'session' }, path, language:'zh-CN',
        bridge: async (scope, file, action) => {
          if (action === 'inspect') return { application: file === 'without.md' ? null : { id:'default', name:'Reader', icon:null } }
          window.nativeAction = { ...scope, file, action }
        }, reportError: message => { throw Error(message) },
      })
      const card = runtime.jsx.jsx(PresentedFileCard, {
        file: { path: './难忘的一天.md', description: '《难忘的一天》Markdown 版' }, cwd: '/work',
        contextMenuProps: {}, t: (key, params) => (deliverableZh[key] ?? key).replace('{name}', params?.name ?? ''),
        onPreview: () => { window.previewedCard = true }, actions: openTarget('难忘的一天.md'),
      })
      runtime.createRoot(document.querySelector('#root')).render(runtime.jsx.jsxs('div', {
        className: 'dsh-native-theme', children: [reply, card, openTarget('without.md')],
      }))
    })
    const link = page.getByRole('button', { name:'难忘的一天.docx' })
    await link.waitFor()
    async function measure(employee) {
      await page.evaluate(async enable => {
        document.querySelector('link[href*="index-BvV"]').disabled = !enable
        await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))
      }, employee)
      return page.evaluate(() => {
        const root = document.querySelector('#root')
        root.style.width = '430px'
        const md = root.querySelector('[class*=MarkdownText_markdown]')
        const button = [...md.querySelectorAll('button')].find(b => b.textContent === '难忘的一天.md'), icon = button.querySelector('svg')
        const text = button.querySelector('strong')
        const range = new Range(); range.selectNodeContents(text)
        const line = range.getBoundingClientRect(), glyph = icon.getBoundingClientRect()
        const css = element => getComputedStyle(element)
        return { glyph: css(icon).display, aligned: glyph.top < line.bottom && glyph.bottom > line.top,
          ul: css(md.querySelector('ul')).listStyleType, ol: css(md.querySelector('ol')).listStyleType,
          indent: css(md.querySelector('ul')).paddingLeft, paragraph: css(md.querySelector('p')).lineHeight,
          margin: css(md.querySelector('ul')).marginTop }
      }, employee)
    }
    const host = await measure(true), native = await measure(false); await measure(true)
    console.log('DSH', native, 'Employee', host)
    assert.equal(native.aligned, true)
    assert.deepEqual(host, native)
    await link.click()
    assert.match(await page.evaluate(() => window.openedPath), /short\.docx$/)
    await page.evaluate(() => { document.querySelector('#root').style.width = '178px' })
    const long = page.getByRole('button', { name: /2026年联合设计/ })
    assert.ok(await long.evaluate(b => b.offsetHeight > 32 && b.scrollWidth <= b.clientWidth + 1
      && b.getBoundingClientRect().right <= document.querySelector('#root').getBoundingClientRect().right + 4))
    await long.click()
    assert.match(await page.evaluate(() => window.openedPath), /long\.docx$/)
    await page.getByRole('button', {name: 'design review notes'}).click()
    assert.match(await page.evaluate(() => window.openedPath), /notes\.md$/)
    const card = page.locator('[data-presented-file]')
    await card.locator('[data-open-path-open]').click()
    assert.deepEqual(await page.evaluate(() => window.nativeAction),
      {accountId:'account', sessionId:'session', file:'难忘的一天.md', action:'open'})
    assert.equal(await page.evaluate(() => window.previewedCard), undefined)
    await card.locator('[data-open-path-more]').click()
    await page.getByRole('menuitem', { name:'显示文件位置' }).click()
    assert.equal(await page.evaluate(() => window.nativeAction.action), 'reveal')
    await card.locator(':scope > button').click()
    assert.equal(await page.evaluate(() => window.previewedCard), true)
    await page.getByRole('button', { name:'显示文件位置', exact:true }).click()
    assert.equal(await page.evaluate(() => window.nativeAction.file), 'without.md')
    assert.equal(await page.evaluate(() => window.nativeAction.action), 'reveal')
    console.log('Native DSH default app, file location, independent preview, inline links, list markers, paragraph metrics and narrow wrapping verified in the employee styles.')
  } finally { await browser.close() }
}
run().catch(err => { console.error(err); process.exitCode = 1 })
