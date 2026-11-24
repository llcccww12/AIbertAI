const assert = require('node:assert/strict')
const { chromium } = require(process.env.DSH_PLAYWRIGHT_CORE || '../../AIbertAI/code/app/node_modules/playwright-core')

// Run TestDSHWebFixture with DSH_WEB_SERVE=1 and DSH_WEB_MEMBERS=chat, chat-stop, chat-cold-stop or chat-limit.
async function run() {
  const base = process.env.DSH_WEB_BASE || 'http://127.0.0.1:18767'
  const browser = await chromium.launch({ executablePath: process.env.DSH_CHROME, headless: true })
  const context = await browser.newContext({ locale: 'zh-CN', viewport: { width: 1440, height: 860 } })
  const errors = []
  context.on('page', page => page.on('pageerror', error => errors.push(error.message)))
  let page = await context.newPage()
  const rpc = (method, params) => page.evaluate(({ method, params }) => new Promise((resolve, reject) => {
    const socket = new WebSocket('ws://' + location.host + '/acp', ['monkeyai.v1', 'monkeyai.token.fixture'])
    const timer = setTimeout(() => { socket.close(); reject(Error('member RPC timeout')) }, 30000)
    const call = (id, method, args) => socket.send(JSON.stringify({ jsonrpc: '2.0', id, method, params: args }))
    socket.onopen = () => call(1, 'initialize', { protocolVersion: 1 })
    socket.onmessage = event => {
      const frame = JSON.parse(event.data)
      if (frame.id === 1 && !frame.error) call(2, method, params)
      else if (frame.id === 2 || frame.id === 1 && frame.error) { clearTimeout(timer); socket.close(); resolve(frame) }
    }
  }), { method, params })
  const state = async () => (await context.request.get(base + '/fixture')).json()
  const waitState = async (predicate, timeout = 45000) => {
    const deadline = Date.now() + timeout
    while (Date.now() < deadline) {
      if (predicate(await state())) return
      await new Promise(resolve => setTimeout(resolve, 120))
    }
    assert.fail('Fixture condition timed out: ' + JSON.stringify(await state()))
  }
  const view = () => page.locator('.dsh-member-view')
  const input = () => view().locator('[data-member-composer] textarea')
  const archiveMessage = text => view().locator('[data-member-request-id]').filter({ hasText: text })
  const send = async text => { await input().fill(text); await input().press('Enter') }
  const active = '请改用冷色'
  const cold = '请补一条移动端规范'
  try {
    const { sessionId, stopOneMember, stopColdMember, limitOneMember } = await state()
    const url = base + '/sessions/' + sessionId
    await page.goto(url)
    const login = page.getByRole('button', { name: '登录', exact: true })
    await login.or(page.getByRole('button', { name: /会话的工具/ })).first().waitFor()
    if (await login.isVisible()) {
      await page.locator('input').first().fill(base)
      await login.click()
      await page.waitForURL(link => link.pathname !== '/login')
      await page.goto(url)
    }
    await page.locator('.conversation-composer textarea').fill('制作HTML页面')
    await page.locator('.conversation-composer textarea').press('Enter')
    if (stopColdMember) {
      await page.getByText('主会话已收到成员结论。', { exact: true }).waitFor({ timeout: 65000 })
      assert.equal((await context.request.post(base + '/fixture/evict')).status(), 204)
      await page.reload()
      await page.locator('[data-subagent-action] button').click()
      await page.getByRole('treeitem').filter({ hasText: /designer|设计检查/ }).click()
      await send('先等我确认画布')
      await waitState(state => state.designerHeld)
      await view().getByRole('button', { name: '停止成员', exact: true }).click()
      await view().getByText('你停止了这个成员。', { exact: true }).waitFor()
      await waitState(state => state.memberModelCancelled && state.memberTurnClosed && state.receiptVerified)
      await page.reload()
      await view().getByText('你停止了这个成员。', { exact: true }).waitFor()
      assert.equal(await view().locator('[data-chat-running]').count(), 0)
      await send(cold)
      await view().getByRole('heading', { name: '移动端补充' }).waitFor({ timeout: 40000 })
      await waitState(state => state.memberTurnDone)
      assert.deepEqual(errors, [])
      return console.log('Verified independently stopping a revived member turn, terminal accounting, root recovery and same-member continuation after reopening.')
    }
    if (limitOneMember) {
      await page.getByText('主会话已收到成员结论。', { exact: true }).waitFor({ timeout: 75000 })
      await page.reload()
      await page.locator('.dsh-open-notice-member').first().click({ timeout: 15000 })
      await view().locator('[data-outcome="max-tokens"]').getByText('已达到模型输出上限').waitFor()
      await send(cold)
      await archiveMessage(cold).waitFor({ timeout: 40000 })
      await view().getByRole('heading', { name: '移动端补充' }).waitFor({ timeout: 40000 })
      assert.deepEqual(errors, [])
      return console.log('Verified archived native member failure notice, actual output limit, and a successful continuation in the same child.')
    }
    await waitState(state => state.designerHeld)
    await page.locator('[data-subagent-action] button').click()
    await page.getByRole('treeitem').filter({ hasText: /designer|设计检查/ }).click()
    await view().getByRole('button', { name: '停止成员', exact: true }).waitFor()
    if (stopOneMember) {
      await view().getByRole('button', { name: '停止成员', exact: true }).click()
      await view().getByText('你停止了这个成员。', { exact: true }).waitFor({ timeout: 30000 })
      assert.equal(await view().locator('[data-chat-running]').count(), 0)
      assert.ok(await input().isEnabled())
      await waitState(state => state.memberModelCancelled, 8000)
      await page.locator('[data-member-parent]').click()
      const parentReply = page.getByText('主会话已收到成员结论。', { exact: true })
      await parentReply.or(page.getByText('执行出错', { exact: true })).first().waitFor({ timeout: 65000 })
      assert.ok(await parentReply.isVisible(), 'the root must finish its own work after one member is stopped')
      assert.ok((await state()).receiptVerified, 'the isolated backend must verify the exact cancelled model request')
      await page.reload()
      await page.locator('[data-subagent-action] button').click()
      await page.getByRole('treeitem').filter({ hasText: /reviewer|校对检查/ }).click()
      await view().getByRole('heading', { name: '校对检查结论' }).waitFor()
      assert.deepEqual(errors, [])
      return console.log('Verified stopping only the selected running member, archived manual cause, and subsequent work by its parent and sibling.')
    }
    await send(active)
    await view().getByText('已提交，等待成员接收').or(archiveMessage(active)).first().waitFor({ timeout: 25000 })
    // While the child is busy, the user may leave the chat and its durable steering input remains addressed to this member.
    await context.request.post(base + '/goal/release')
    await archiveMessage(active).waitFor({ timeout: 40000 })
    const activeId = await archiveMessage(active).getAttribute('data-member-request-id')
    await view().getByRole('heading', { name: '配色补充' }).waitFor({ timeout: 50000 })
    assert.equal(await archiveMessage(active).count(), 1)
    await page.locator('[data-member-parent]').click()
    await page.getByText('主会话已收到成员结论。', { exact: true }).waitFor({ timeout: 45000 })
    assert.equal(await page.locator('.conversation-composer textarea').inputValue(), '')

    await page.close()
    assert.equal((await context.request.post(base + '/fixture/evict')).status(), 204)
    page = await context.newPage()
    await page.goto(url)
    const process = page.locator('[data-turn-process][aria-expanded]').first()
    if (await process.getAttribute('aria-expanded') === 'false') await process.click()
    const links = page.locator('[data-member-history-link]')
    await links.first().waitFor()
    assert.equal(await links.count(), 2, 'both old delegation rows resolve to their own descendants')
    await links.first().click()
    await view().getByRole('heading', { name: '配色补充' }).waitFor()
    await send(cold)
    await archiveMessage(cold).waitFor({ timeout: 40000 })
    await view().getByRole('heading', { name: '移动端补充' }).waitFor({ timeout: 50000 })
    const coldId = await archiveMessage(cold).getAttribute('data-member-request-id')
    assert.notEqual(coldId, activeId)
    const catalog = (await rpc('_aibertai/session/members', { sessionId })).result.members
    const designer = catalog.find(member => member.name === 'designer')
    const replay = { sessionId, memberId: designer.id, requestId: coldId, text: cold }
    const before = (await state()).modelCalls
    assert.equal((await rpc('_aibertai/session/member-prompt', replay)).result.requestId, coldId)
    assert.ok((await rpc('_aibertai/session/member-prompt', { ...replay, text: active })).error)
    assert.ok((await rpc('_aibertai/session/member-prompt', { ...replay, memberId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa' })).error)
    await page.reload()
    await archiveMessage(cold).waitFor()
    assert.equal(await archiveMessage(cold).count(), 1)
    assert.equal(await archiveMessage(active).count(), 1)
    await page.locator('[data-member-breadcrumb] button[aria-haspopup="dialog"]').click()
    await page.getByRole('treeitem').filter({ hasText: /reviewer|校对检查/ }).click()
    await view().getByRole('heading', { name: '校对检查结论' }).waitFor()
    assert.equal(await archiveMessage(cold).count(), 0)
    assert.equal(await archiveMessage(active).count(), 0)
    assert.equal((await state()).modelCalls, before)
    assert.deepEqual(errors, [])
    console.log('Verified live and cold member follow-ups, genuine history links, archived delivery, slot ownership, duplicate IDs and wrong-member rejection.')
  } catch (error) {
    console.error('Fixture state', await state())
    if (!page.isClosed()) console.error((await page.locator('body').innerText()).slice(-1200))
    throw error
  } finally {
    await context.request.post(base + '/fixture/finish').catch(() => {})
    await browser.close()
  }
}
run().catch(error => { console.error(error); process.exitCode = 1 })
