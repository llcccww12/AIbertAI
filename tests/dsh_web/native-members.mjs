// node tests/dsh_web/native-members.mjs /path/to/prepared-DSH/resources/app
import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { resolve } from 'node:path'
const sdk = resolve(process.argv[2])
const requireSDK = createRequire(resolve(sdk, 'package.json'))
const requireUI = createRequire(resolve(sdk, 'packages/client/ui-chat/package.json'))
const { window } = new (requireSDK('jsdom').JSDOM)('<!doctype html><main class="chat-page"><header><div id="lineage"></div><div id="catalog"></div></header><div class="chat-body"></div></main>', { url: 'https://employee.test' })
Object.assign(globalThis, { window, document: window.document, Node: window.Node, Element: window.Element,
  HTMLElement: window.HTMLElement, DOMParser: window.DOMParser, getComputedStyle: window.getComputedStyle.bind(window),
  CustomEvent: window.CustomEvent, localStorage: window.localStorage, IS_REACT_ACT_ENVIRONMENT: true })
const React = requireUI('react'), jsx = requireUI('react/jsx-runtime'), dom = requireUI('react-dom')
const { bindChatFramework } = await import('../../AIbertAI/code/web/dsh/chat-framework.mjs')
bindChatFramework(React, jsx, dom)
const { default: Members } = await import('../../AIbertAI/code/web/dsh/native-members/members.js')
const root = requireUI('react-dom/client').createRoot(document.querySelector('#catalog'))
const rows = [
  { id: 'designer', parentId: 'root', mode: 'continuable', label: '设计师', activity: 'inactive' },
  { id: 'reviewer', parentId: 'root', mode: 'continuable', label: '检查员', activity: 'inactive' },
]
const calls = []
const long = Array.from({ length: 90 }, (_, index) => 'skill instruction ' + index).join('\n')
const firstPage = [
  { seq: 1, type: 'user/message', source: 'user', text: '绘制页面' },
  { seq: 2, type: 'turn/start', turn: 1, time: Date.now() - 45000, text: '' },
  { seq: 3, type: 'tool/call', turn: 1, step: 1, callId: 'a', tool: 'read', text: 'SKILL.md' },
  { seq: 4, type: 'tool/call', turn: 1, step: 1, callId: 'b', tool: 'read', text: 'missing.md' },
]
const lastPage = [
  { seq: 5, type: 'tool/result', turn: 1, step: 1, callId: 'b', failed: true, text: '文件不存在' },
  { seq: 6, type: 'tool/result', turn: 1, step: 1, callId: 'a', text: long, truncated: true, textLength: 25954 },
  { seq: 7, type: 'tool/result', tool: 'canvas', text: '两张画布', truncated: true },
  { seq: 8, type: 'assistant/message', text: '## 设计\n\n**草图** 已完成。' },
  { seq: 9, type: 'turn/end', turn: 1, text: 'completed' },
]
let late, holdDesigner = false, live = false
const replyText = '校对完成'
let reviewEntries = [
  { seq: 4, type: 'user/message', source: 'agent-message', text: '检查草图' },
  { seq: 5, type: 'assistant/message', text: replyText },
  { seq: 6, type: 'turn/end', turn: 3, text: 'error', errorCode: 'limit_reached' },
]
async function request(params) {
  calls.push(params)
  if (params.sessionId !== 'root') return { members: [], live: false }
  const base = { members: rows, live }
  if (!params.memberId) return base
  if (params.memberId === 'designer') {
    if (holdDesigner) return new Promise(resolve => { late = resolve })
    return params.from === undefined ? { ...base, member: rows[0], entries: firstPage, more: true, next: 5 }
      : { ...base, member: rows[0], entries: params.from === 5 ? lastPage : [], more: false, next: 10 }
  }
  return { ...base, member: rows[1], entries: params.from === undefined ? reviewEntries : reviewEntries.filter(item => item.seq >= params.from), more: false, next: (reviewEntries.at(-1)?.seq || 0) + 1 }
}
const mount = (id = 'root', active = false, actions = {}) => React.act(async () => { root.render(jsx.jsx(Members, {
  accountId: 'test', sessionId: id, request, language: 'zh-CN', running: active, rootTitle: '制作HTML页面', headerTarget: document.querySelector('#lineage'), ...actions,
}, id)) })
const find = selector => document.querySelector(selector)
const click = element => { assert.ok(element); return React.act(async () => { element.click() }) }
const row = name => [...document.querySelectorAll('[role=treeitem]')].find(el => el.textContent.includes(name))
const wait = ms => React.act(() => new Promise(resolve => setTimeout(resolve, ms)))
const content = () => find('.dsh-member-transcript')?.textContent
try {
  await mount()
  assert.equal(calls.length, 1)
  await click(find('[data-subagent-action] button'))
  assert.equal(document.querySelectorAll('[role=treeitem]').length, 2)
  await click(row('设计师'))
  await wait(20)
  assert.equal(find('[data-message-bubble]').textContent, '绘制页面')
  assert.equal(find('#lineage [data-member-parent]').textContent, '制作HTML页面')
  assert.equal(find('.dsh-member-view > header'), null, 'full-width view uses the host title row')
  assert.equal(find('[data-member-event="assistant/message"] strong').textContent, '草图')
  assert.equal(find('[data-member-event="assistant/message"] h2').textContent, '设计')
  assert.deepEqual(calls.filter(c => c.memberId === 'designer').map(c => c.from), [undefined, 5])
  const call = id => find('[data-call-id="' + id + '"]')
  const toggle = target => target.querySelector('[data-disclosure-row]')
  assert.equal(document.querySelectorAll('[data-disclosure-row]').length, 3)
  assert.ok(!content().includes('skill instruction') && !content().includes('25954') && !content().includes('文件不存在'))
  assert.equal(toggle(call('b')).getAttribute('aria-expanded'), 'false')
  assert.ok(toggle(call('b')).textContent.includes('失败'))
  await click(toggle(call('b')))
  assert.equal(call('b').querySelector('section[aria-label="工具结果"] pre').textContent, '文件不存在')
  assert.ok(!content().includes('skill instruction'))
  await click(toggle(call('a')))
  assert.equal(call('a').querySelector('section[aria-label="工具结果"] pre').textContent, long)
  assert.ok(call('a').querySelector('section[aria-label="工具结果"] small[data-member-truncation]').textContent.includes('25954'))
  assert.ok(!call('b').querySelector('[data-member-truncation]'))
  const orphan = find('[data-member-event="tool/result"]')
  await click(toggle(orphan))
  assert.ok(orphan.textContent.includes('两张画布') && orphan.textContent.includes('已截断'))
  assert.ok(!content().includes('completed'))
  await click(find('#lineage button[aria-haspopup=dialog]'))
  await click(row('检查员'))
  assert.ok(content().includes(replyText) && !content().includes('两张画布'))
  assert.equal(find('[data-member-event="turn/end"][role=alert]').textContent, '处理失败 · limit_reached')
  assert.ok(find('[aria-label="团队消息"] [data-message-bubble]'))
  await mount('other')
  assert.equal(find('.dsh-member-view'), null)
  await mount()
  assert.ok(content().includes('校对完成'), 'saved selection reopens in the owning root')
  await click(find('[data-member-parent]'))
  assert.equal(find('.dsh-member-view'), null)
  const coldReads = calls.length
  await wait(4550)
  assert.equal(calls.length, coldReads, 'cold archives should be read once')
  holdDesigner = true
  await click(find('[data-subagent-action] button'))
  await click(row('设计师'))
  assert.equal(typeof late, 'function')
  await mount('other')
  await React.act(async () => { late({ members: rows, live: false, member: rows[0], entries: [{ seq: 1, text: '迟到数据' }] }) })
  assert.equal(find('.dsh-member-view'), null)
  assert.ok(!document.body.textContent.includes('迟到数据'))
  live = true
  holdDesigner = false
  await mount('root', true)
  await click(find('[data-member-parent]'))
  await mount('other')
  const afterUnmount = calls.length
  await wait(1300)
  assert.equal(calls.length, afterUnmount, 'switching roots cancels active watches')
  rows[1].activity = 'running'
  reviewEntries = [{ seq: 1, type: 'turn/start', turn: 8, time: Date.now() - 35000, text: '' }]
  window.localStorage.setItem('dsh.member.view.' + JSON.stringify(['test','root']), JSON.stringify({ id: 'reviewer', aside: true }))
  await mount('root', true)
  await wait(10)
  assert.ok(find('.dsh-member-view[data-aside] > header [data-member-parent]'))
  assert.equal(find('#lineage [data-member-breadcrumb]'), null)
  assert.ok(find('[data-chat-running]'), 'live member with an open turn renders DSH running status')
  reviewEntries.push({ seq: 2, type: 'turn/end', turn: 8, text: 'completed' })
  await wait(1280)
  assert.equal(find('[data-chat-running]'), null, 'a resident member with a closed turn has no active timer')
  await click(find('[data-member-parent]'))
  assert.equal(find('.dsh-member-view'), null)
  await mount('other')
  rows[1].activity = 'inactive'
  live = false
  reviewEntries = [
    { seq: 1, type: 'turn/end', turn: 1, text: 'max-tokens' },
    { seq: 2, type: 'turn/end', turn: 2, text: 'aborted', abortReason: 'user' },
    { seq: 3, type: 'turn/end', turn: 3, text: 'interrupted' },
  ]
  const draftKey = 'dsh.member.draft.' + JSON.stringify(['test', 'root', 'reviewer'])
  localStorage.setItem(draftKey, JSON.stringify({ text: '追加说明', awaiting: [] }))
  localStorage.setItem('dsh.member.view.' + JSON.stringify(['test','root']), JSON.stringify({ id: 'reviewer', aside: false }))
  const attempts = []
  const promptMember = async req => {
    attempts.push(req)
    if (attempts.length === 1) throw new Error('temporary')
    return { accepted: true, requestId: req.requestId }
  }
  await mount('root', false, { promptMember })
  assert.equal(find('[data-outcome="max-tokens"]').textContent, '已达到模型输出上限')
  assert.match(find('[data-outcome="aborted"]').textContent, /你停止了这个成员/)
  assert.equal(find('[data-outcome="interrupted"]').textContent, '本轮执行中断')
  const form = () => find('[data-member-composer]')
  const input = () => form().querySelector('textarea')
  const submit = () => React.act(async () => { form().dispatchEvent(new window.Event('submit', { cancelable: true, bubbles: true })) })
  assert.equal(input().value, '追加说明')
  await submit()
  assert.equal(form().querySelector('[role=alert]').textContent, '发送失败。请重试。')
  assert.equal(input().value, '追加说明')
  assert.equal(find('[data-member-request-id]'), null)
  await submit()
  assert.equal(attempts[0].requestId, attempts[1].requestId)
  assert.equal(attempts[1].memberId, 'reviewer')
  assert.equal(input().value, '')
  assert.match(form().querySelector('[role=status]').textContent, /等待成员接收/)
  assert.equal(find('[data-member-request-id]'), null, 'an acceptance ACK does not invent a delivered message')
  reviewEntries.push({ seq: 4, type: 'user/message', text: '追加说明', requestId: attempts[1].requestId },
    { seq: 5, type: 'turn/end', turn: 4, text: 'completed' })
  await wait(1450)
  assert.equal(find('[data-member-request-id]').textContent, '追加说明')
  assert.equal(localStorage.getItem(draftKey), null)
  await mount('other')
  rows[1].mode = 'one-shot'
  await mount('root', false, { promptMember })
  assert.equal(form(), null, 'an archive without a continuable child stays read-only')
  console.log('Verified DSH member title navigation, isolated paged tool disclosures, failed and truncated output, Markdown replies, task bubbles, live turn clock, cold reads and stale response cleanup.')
} finally {
  await React.act(() => root.unmount())
  window.close()
}
