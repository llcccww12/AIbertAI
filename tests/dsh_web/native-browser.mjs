// node tests/dsh_web/native-browser.mjs /path/to/prepared-DSH/resources/app
import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { resolve } from 'node:path'

const sdk = resolve(process.argv[2])
const requireSdk = createRequire(resolve(sdk, 'package.json'))
const requireUI = createRequire(resolve(sdk, 'packages/client/ui-chat/package.json'))
const { window } = new (requireSdk('jsdom').JSDOM)('<main></main>', { url: 'https://employee.test' })
Object.assign(globalThis, { window, document: window.document, HTMLElement: window.HTMLElement,
  Node: window.Node, Element: window.Element, IS_REACT_ACT_ENVIRONMENT: true })
const react = requireUI('react')
const jsx = requireUI('react/jsx-runtime')
const dom = requireUI('react-dom')
const { bindChatFramework } = await import('../../AIbertAI/code/web/dsh/chat-framework.mjs')
bindChatFramework(react, jsx, dom)
const { default: Panel } = await import('../../AIbertAI/code/web/dsh/native-browser/panel.js')
const root = requireUI('react-dom/client').createRoot(document.querySelector('main'))
const calls = []
const addressRef = { current: null }
let toolbar
const onToolbar = value => { toolbar = value }
const button = label => document.querySelector('button[aria-label="' + label + '"]')
const render = async (tabId, state) => react.act(() => root.render(jsx.jsx(Panel, {
  tabId, language: 'zh-CN', state, addressRef, onToolbar,
  onNavigate: url => calls.push(url), onAction: action => calls.push(action),
  children: jsx.jsx('div', { 'data-host-viewport': true }),
}, tabId)))
async function submit(value) {
  const input = addressRef.current
  await react.act(() => {
    Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set.call(input, value)
    input.dispatchEvent(new window.Event('input', { bubbles: true }))
  })
  await react.act(() => toolbar.dispatchEvent(new window.Event('submit', { bubbles: true, cancelable: true })))
}
try {
  await render('main', { url: 'about:blank' })
  assert.equal(document.activeElement?.tagName, 'BODY')
  assert.ok(toolbar.contains(addressRef.current))
  assert.ok(button('后退').disabled && button('前进').disabled && button('刷新').disabled)
  assert.ok(document.querySelector('[data-host-viewport]').parentElement.matches('[aria-label="浏览器"]'))
  for (const input of ['file:///tmp/test.html', 'https://user:password@example.test', 'https://employee.test/other']) {
    await submit(input)
    assert.ok(document.querySelector('[role=alert]').textContent)
    assert.equal(calls.length, 0)
  }
  await submit('example.test/one')
  assert.deepEqual(calls, ['https://example.test/one'])
  await render('main', { url: 'https://example.test/two', canGoBack: true, canGoForward: true })
  assert.equal(addressRef.current.value, 'https://example.test/two')
  for (const label of ['后退', '前进', '刷新']) await react.act(() => button(label).click())
  assert.deepEqual(calls.slice(1), ['back', 'forward', 'reload'])
  assert.equal(button('在系统浏览器中打开').disabled, false)
  await render('another', { url: 'about:blank' })
  assert.equal(addressRef.current.value, '')
  assert.equal(button('后退').disabled, true)
  await render('main', { url: 'https://example.test/two', canGoBack: true, error: 'load error' })
  assert.equal(addressRef.current.value, 'https://example.test/two')
  assert.match(document.querySelector('[role=status]').textContent, /加载失败/)
  assert.equal(button('关闭沙箱限制'), null)
  console.log('Verified native DSH browser, validated addresses, history commands, viewport, tab switch, restore and load failure.')
} finally {
  await react.act(() => root.unmount())
  window.close()
}
