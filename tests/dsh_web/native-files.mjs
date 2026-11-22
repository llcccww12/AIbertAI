// node tests/dsh_web/native-files.mjs /path/to/prepared-DSH/resources/app
import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { resolve } from 'node:path'

const sdk = resolve(process.argv[2])
const deps = createRequire(resolve(sdk, 'package.json'))
const ui = createRequire(resolve(sdk, 'packages/client/ui-chat/package.json'))
const { window } = new (deps('jsdom').JSDOM)('<!doctype html><html><body><main></main></body></html>', { url: 'https://fixture.invalid' })
for (const key of ['document', 'DOMParser', 'HTMLElement', 'HTMLButtonElement', 'HTMLIFrameElement', 'Element', 'Node', 'MutationObserver']) globalThis[key] = window[key]
globalThis.window = window
globalThis.IS_REACT_ACT_ENVIRONMENT = true
const React = ui('react')
const jsx = ui('react/jsx-runtime')
const { bindChatFramework } = await import('../../AIbertAI/code/web/dsh/chat-framework.mjs')
bindChatFramework(React, jsx, ui('react-dom'))
const { default: Files, EmployeeFileTabs, EmployeePreview, EmployeeText } = await import('../../AIbertAI/code/web/dsh/native-files/files.js')
const root = ui('react-dom/client').createRoot(document.querySelector('main'))
const find = selector => document.querySelector(selector)
const file = { path: '设计/结果.txt', fileRef: { device: 'fixture-device', path: '/work/设计/结果.txt' }, origin: { sessionId: 'session-1', cwd: '/work' } }
const directory = { name: '设计', path: '设计', kind: 'directory', children: [
  { path: file.path, name: '结果.txt', kind: 'file', file, children: [], operation: 'modified' },
] }
let selected, paged, refreshes = 0
const labels = key => key
function Fixture() {
  const [expanded, expand] = React.useState(new Set())
  const [query, onQuery] = React.useState('')
  const [fileScope, onScope] = React.useState('all')
  return jsx.jsx(Files, {
    sessionId: 'session-1', cwd: '/work', label: labels, language: 'zh', nodes: [directory], outsideNodes: [],
    directories: { '设计': { data: { nextOffset: 200 } } }, expanded, query, fileScope, git: true,
    selected: file.path, dragProps: node => ({ draggable: node.kind === 'file' }),
    onToggle: path => expand(prev => prev.has(path) ? new Set() : new Set([path])), onSelect: node => { selected = node },
    onQuery, onScope, onLoad: (path, offset) => { paged = [path, offset] }, onRefresh: () => { refreshes++ },
  })
}
const render = element => React.act(() => root.render(element))
const click = selector => React.act(() => { assert.ok(find(selector), selector); find(selector).click() })
try {
  await render(jsx.jsx(Fixture, {}))
  assert.ok(find('[data-files-page]'))
  assert.equal(find('[data-path-label]').getAttribute('title'), '/work')
  await click('[data-files-reload]')
  assert.equal(refreshes, 1)
  await click('[data-files-search-toggle]')
  assert.equal(find('[data-files-search]'), document.activeElement)
  await click('[data-files-entry="directory"] button')
  await click('[data-files-entry="file"] button')
  assert.equal(selected.file, file, 'keep the original host file reference and session identity')
  assert.equal(find('[data-files-entry="file"] button').draggable, true)
  assert.ok(find('[data-operation="modified"]'))
  await click('.files-tree-more')
  assert.deepEqual(paged, ['设计', 200])
  await click('[data-files-entry="directory"] button')
  assert.equal(find('[data-files-entry="file"]'), null)
  await click('[data-files-scope]')
  const options = [...document.querySelectorAll('[role=menuitem]')]
  assert.deepEqual(options.map(option => option.textContent.trim()), ['workspace.fileTree.all', 'workspace.fileTree.session', 'workspace.fileTree.git'])
  await React.act(() => options[1].click())
  assert.ok(find('[role="tree"][aria-label="workspace.fileTree.session"]'))
  await render(jsx.jsx(EmployeePreview, {
    path: file.path, tools: jsx.jsx('button', { children: '下载' }),
    children: jsx.jsx(EmployeeText, { content: '甲\n乙\n丙', position: { line: 2 } }),
  }))
  assert.equal(find('[data-textpreview-path]').title, file.path)
  assert.equal(find('[data-textpreview-target]').textContent.trim(), '乙')
  assert.ok(find('[data-textpreview-plain]'))
  let active, closed, browsing = false
  const items = ['简报.docx', '插图.png'].map(title => ({ id: title, title, path: '项目/' + title }))
  await render(jsx.jsx(EmployeeFileTabs, {
    items, active: items[0].id, onSelect: id => { active = id }, onClose: id => { closed = id },
    onBrowse: () => { browsing = true }, directoryLabel: '工作区文件', language: 'zh',
  }))
  assert.equal(document.querySelectorAll('[data-dockkit-tab]').length, 2)
  assert.ok(find('[data-dockkit-tab-title] svg'), 'native title includes a file-type icon')
  assert.equal(find('[aria-selected=true] [data-dockkit-tab-title]').textContent, '简报.docx')
  await click('[data-dockkit-tab="插图.png"]')
  assert.equal(active, '插图.png')
  await click('[data-dockkit-tab-close="简报.docx"]')
  assert.equal(closed, '简报.docx')
  await click('[data-dockkit-add-tab]')
  assert.equal(browsing, true)
  console.log('Verified native DockKit file tabs, directory controls, pagination, scope, scoped opens and line preview.')
} finally {
  await React.act(() => root.unmount())
  window.close()
}
