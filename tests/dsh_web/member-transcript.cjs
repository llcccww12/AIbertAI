const assert = require('node:assert/strict')
const { chromium } = require(process.env.DSH_PLAYWRIGHT_CORE || '../../AIbertAI/code/app/node_modules/playwright-core')

async function verifyStoppedMembers(page, context, base, url, memberFailure, withholdReceipt) {
  const notice = '本轮执行已停止，模型调用的用量回执待核验。重新打开会话会再次检查。'
  const readFixture = async () => (await context.request.get(base + '/fixture')).json()
  const waitFixture = async (predicate, description, timeout = 20000) => {
    let state
    const deadline = Date.now() + timeout
    while (Date.now() < deadline) {
      state = await readFixture()
      if (predicate(state)) return
      await new Promise(resolve => setTimeout(resolve, 250))
    }
    assert.fail(description + ' ' + JSON.stringify(state))
  }
  const lists = async (page, stopped) => {
    const team = page.locator('[data-team-panel]')
    const todo = page.getByTestId('todo-panel')
    const state = stopped ? '未完成' : '进行中'
    await team.getByRole('heading').getByText(new RegExp('1 ' + state)).waitFor()
    const toggle = todo.getByRole('button')
    await toggle.getByText(new RegExp('1 ' + state)).waitFor()
    if (await toggle.getAttribute('aria-expanded') !== 'true') await toggle.click()
    for (const [panel, text] of [[team, '共享设计检查'], [todo, '整理待办资料']]) {
      const item = panel.getByRole('listitem').filter({ hasText: text })
      const icon = item.getByRole('img', { name: state })
      await icon.waitFor()
      if (stopped) {
        assert.doesNotMatch(await panel.innerText(), /进行中/)
        assert.equal(await icon.locator('[data-state="ongoing"]:visible').count(), 0)
        assert.equal(await icon.evaluate(el => el.getAnimations({ subtree: true }).length), 0)
        const shape = await icon.evaluate(el => {
          const box = getComputedStyle(el, '::after')
          const edge = parseFloat(box.width)
          return box.content !== 'none' && box.display !== 'none' && edge > 0 && box.width === box.height
            && parseFloat(box.borderTopLeftRadius) < edge / 2
        })
        assert.equal(shape, true, 'an unfinished row has a static square')
      }
    }
  }
  await waitFixture(state => state.parentWait && state.memberModelHeld, 'parent waiting for live designer', 60000)
  await lists(page, false)
  await page.locator('[data-chat-running]').getByText(/正在等待子智能体回复，本轮已用/).waitFor()
  if (memberFailure) {
    assert.equal((await context.request.post(base + '/goal/release')).status(), 204)
    await waitFixture(state => state.parentEnded && state.parentFailure, 'model failure reached parent archive')
  } else {
    await page.locator('.composer-send[data-action="stop"]').click()
    await page.locator('.composer-send[data-action="send"]').waitFor()
    if (withholdReceipt) {
      await waitFixture(state => state.stopObserved && state.journalPending && state.receiptRequested && state.memberModelCancelled
        && !state.receiptVerified && !state.parentInterrupted, 'stopped execution with outstanding accounting')
      await page.getByText(notice, { exact: true }).waitFor()
    } else {
      await waitFixture(state => state.parentInterrupted && state.memberModelCancelled && state.receiptVerified, 'parent and model request stopped')
    }
  }
  await lists(page, true)
  const calls = (await readFixture()).modelCalls
  await page.close()
  const reopened = await context.newPage()
  let response
  if (withholdReceipt) reopened.on('websocket', socket => {
    if (!socket.url().endsWith('/acp')) return
    let promptID = null
    socket.on('framesent', frame => {
      const message = JSON.parse(String(frame.payload))
      if ((message.method === '_monkeyai/session/queue' || message.method === 'session/prompt')
        && JSON.stringify(message.params).includes('继续未完成的设计检查')) promptID = message.id
    })
    socket.on('framereceived', frame => {
      const message = JSON.parse(String(frame.payload))
      if (promptID !== null && message.id === promptID && !message.method) response = message
    })
  })
  await reopened.goto(url)
  await lists(reopened, true)
  if (withholdReceipt) {
    await reopened.getByText(notice, { exact: true }).waitFor()
    assert.equal(await reopened.locator('[data-chat-running]:visible').count(), 0)
    const composer = reopened.locator('.conversation-composer')
    const before = await readFixture()
    await composer.locator('textarea').fill('继续未完成的设计检查')
    const send = composer.locator('.composer-send[data-action="send"]')
    if (await send.isEnabled()) {
      await send.click()
      await waitFixture(() => response !== undefined, 'follow-up has an explicit ACP rejection')
      assert.match(JSON.stringify(response.error), /dsh_reconciliation_required/)
    }
    const after = await readFixture()
    assert.equal(after.journalPending, true)
    assert.equal(after.turnCount, before.turnCount)
    assert.equal(after.modelCalls, calls)
  }
  await reopened.locator('[data-subagent-action] button').click()
  await reopened.getByRole('treeitem').filter({ hasText: /designer|设计检查/ }).click()
  const view = reopened.locator('.dsh-member-view')
  await view.getByText('等待中的设计检查', { exact: true }).waitFor()
  await view.getByText(memberFailure ? '完成等待中的成员检查。' : '已停止', { exact: true }).waitFor()
  assert.equal(await view.locator('[data-chat-running]').count(), 0)
  assert.equal((await readFixture()).modelCalls, calls, 'opening archived progress does not restart the model')
}

// Start TestDSHWebFixture with DSH_WEB_SERVE=1; DSH_WEB_MEMBERS selects 1, stop, stop-before-headers, stop-no-receipt or error.
async function run() {
  const base = process.env.DSH_WEB_BASE || 'http://127.0.0.1:18767'
  const browser = await chromium.launch({ executablePath: process.env.DSH_CHROME, headless: true })
  const context = await browser.newContext({ locale: 'zh-CN', viewport: { width: 1440, height: 860 } })
  const errors = []
  context.on('page', page => page.on('pageerror', error => errors.push(error.message)))
  let page = await context.newPage()
  page.setDefaultTimeout(9000)
  const parentTitle = '制作HTML页面'
  const parentAnswer = '主会话已收到成员结论。'
  const view = () => page.locator('.dsh-member-view')
  const parentButton = () => page.locator('[data-member-parent]')
  const openMember = async name => {
    await page.locator('[data-subagent-action] button').click()
    await page.getByRole('treeitem').filter({ hasText: name }).click()
    await view().waitFor()
  }
  try {
    const { sessionId, memberStop, memberFailure, withholdReceipt } = await (await context.request.get(base + '/fixture')).json()
    const url = base + '/sessions/' + sessionId
    await page.goto(url)
    const login = page.getByRole('button', { name: '登录', exact: true })
    const tools = page.getByRole('button', { name: /会话的工具/ })
    await login.or(tools).first().waitFor()
    if (await login.isVisible()) {
      await page.locator('input').first().fill(base)
      await login.click()
      await page.waitForURL(address => address.pathname !== '/login')
      await page.goto(url)
    }
    const prompt = page.locator('.conversation-composer textarea')
    await prompt.fill(parentTitle)
    await prompt.press('Enter')
    if (memberStop) {
      await verifyStoppedMembers(page, context, base, url, memberFailure, withholdReceipt)
      assert.deepEqual(errors, [])
      console.log('Verified', memberFailure ? 'model failure' : 'live parent and member cancellation', 'with both unfinished task lists and archive reopen.')
      return
    }
    await page.getByText(parentAnswer, { exact: true }).waitFor({ timeout: 45000 })
    await openMember(/designer|设计检查/)
    const transcript = view().locator('.dsh-member-transcript')
    const task = transcript.locator('[data-member-event="user/message"]')
    await task.getByText('检查技能文件并总结结论。', { exact: true }).waitFor()
    assert.equal(await task.count(), 1, 'injected runtime context is not a delegated task')
    assert.doesNotMatch(await task.innerText(), /<system-reminder>|<available_skills>/)
    const beyondPreview = transcript.getByText(/技能检查项 090/).last()
    assert.equal(await beyondPreview.isVisible(), false, 'long skill output starts folded')
    const reply = transcript.locator('[data-member-event="assistant/message"]')
    await reply.getByRole('heading', { level: 2, name: '设计检查结论' }).waitFor()
    assert.equal(await reply.locator('strong').getByText('层级已核对').isVisible(), true)
    assert.equal(await reply.getByRole('listitem').count(), 2)
    const read = transcript.locator('[data-member-event="tool/call"]')
    assert.equal(await read.count(), 1, 'one native read call and its result form one card')
    const toggle = read.locator('[data-disclosure-row][aria-expanded]').first()
    assert.equal(await toggle.getAttribute('aria-expanded'), 'false')
    const folded = await read.boundingBox()
    assert.ok(folded.height < 150, 'the closed tool occupies just a few lines')
    await toggle.click()
    await beyondPreview.waitFor()
    await toggle.click()
    await beyondPreview.waitFor({ state: 'hidden' })
    assert.equal(await parentButton().innerText(), parentTitle)
    const breadcrumb = page.locator('[data-member-breadcrumb]:visible')
    assert.equal(await breadcrumb.count(), 1)
    const memberMenu = breadcrumb.locator('button[aria-haspopup="dialog"]')
    const parentBox = await parentButton().boundingBox()
    const menuBox = await memberMenu.boundingBox()
    assert.ok(menuBox.x > parentBox.x && Math.abs(menuBox.y + menuBox.height / 2 - parentBox.y - parentBox.height / 2) < 5,
      'parent title and member switch share one row')
    assert.equal(await page.locator('.dsh-root-session-heading:visible, .dsh-member-view > header:visible').count(), 0)

    await page.close()
    page = await context.newPage()
    page.setDefaultTimeout(9000)
    await page.goto(url)
    await view().getByRole('heading', { name: '设计检查结论' }).waitFor()
    await parentButton().click()
    await view().waitFor({ state: 'hidden' })
    await page.getByText(parentAnswer, { exact: true }).waitFor()
    await openMember(/reviewer|校对检查/)
    await view().getByRole('heading', { name: '校对检查结论' }).waitFor()
    assert.equal(await view().getByRole('heading', { name: '设计检查结论' }).count(), 0)
    await page.locator('[data-member-breadcrumb] button[aria-haspopup="dialog"]').click()
    await page.getByRole('treeitem').filter({ hasText: /designer|设计检查/ }).click()
    await view().getByRole('heading', { name: '设计检查结论' }).waitFor()
    assert.equal(await view().getByRole('heading', { name: '校对检查结论' }).count(), 0)
    await parentButton().click()
    await page.getByText(parentAnswer, { exact: true }).waitFor()
    await page.reload()
    const process = page.locator('[data-turn-process][aria-expanded]').first()
    if (await process.getAttribute('aria-expanded') === 'false') await process.click()
    const delegated = page.locator('[data-member-history-link]')
    await delegated.nth(1).waitFor()
    assert.equal(await delegated.count(), 2, 'the normal and fork delegation cards should both link to their archived children')
    await delegated.nth(1).click()
    await view().getByRole('heading', { name: '校对检查结论' }).waitFor()
    assert.equal(await view().getByRole('heading', { name: '设计检查结论' }).count(), 0)
    assert.deepEqual(errors, [])
    console.log('Verified two DSH members, the folded skill read and Markdown answer, session reopen and parent-title navigation.')
  } catch (error) {
    if (!page.isClosed()) console.error({ url: page.url(), visibleHeaders: await page.locator('[data-member-breadcrumb]').allTextContents() })
    throw error
  } finally {
    await context.request.post(base + '/fixture/finish').catch(() => {})
    await browser.close()
  }
}
run().catch(error => { console.error(error); process.exitCode = 1 })
