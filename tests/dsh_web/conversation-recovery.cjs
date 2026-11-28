const assert = require('node:assert/strict')
const { chromium } = require(process.env.DSH_PLAYWRIGHT_CORE || '../../AIbertAI/code/app/node_modules/playwright-core')

async function run() {
  const base = process.env.DSH_WEB_BASE
  const browser = await chromium.launch({ executablePath: process.env.DSH_CHROME, headless: true })
  try {
    const context = await browser.newContext({ locale: 'zh-CN', viewport: { width: 1430, height: 920 } })
    let page = await context.newPage()
    let jobsOffline = false
    let historyOffline = false
    await context.routeWebSocket(url => url.hostname === new URL(base).hostname, socket => {
      const server = socket.connectToServer()
      socket.onMessage(message => {
        try {
          const { id, method, params } = JSON.parse(String(message))
          if ((jobsOffline && method === '_aibertai/session/jobs' && params.action === 'list') ||
              (historyOffline && method === '_monkeyai/session/messages')) {
            socket.send(JSON.stringify({ jsonrpc: '2.0', id, error: { code: -32603, message: 'fixture_read_unavailable' } }))
            return
          }
        } catch {}
        server.send(message)
      })
    })
    const fixture = async () => (await context.request.get(base + '/fixture')).json()
    const session = (await fixture()).sessionId
    const url = base + '/sessions/' + session
    const queueActions = []
    page.on('websocket', socket => socket.on('framesent', ({ payload }) => {
      try {
        const message = JSON.parse(String(payload))
        if (message.method === '_monkeyai/session/queue') queueActions.push(message.params.action)
      } catch {}
    }))
    await page.goto(url)
    const login = page.getByRole('button', { name: '登录', exact: true })
    const field = () => page.locator('.conversation-composer textarea')
    const send = async text => {
      await field().fill(text)
      await field().press('Enter')
    }
    await login.or(field()).first().waitFor()
    await page.locator('#boot-splash').waitFor({ state: 'hidden' })
    if (await login.isVisible()) {
      await page.locator('input').first().fill(base)
      await login.click()
      await page.waitForURL(address => address.pathname !== '/login')
      await page.goto(url)
    }
    await send('请运行首轮验收')
    await page.getByText('正在准备首轮校验。', { exact: true }).waitFor()
    await send('补充验收条件')
    const insert = page.getByRole('button', { name: '插入本轮', exact: true })
    await insert.click()
    await insert.waitFor({ state: 'hidden' })
    assert(queueActions.includes('steer') && !queueActions.includes('sendNow'), JSON.stringify(queueActions))
    assert.equal((await fixture()).modelCalls, 1)
    assert.equal((await context.request.post(base + '/fixture/release-first-step')).status(), 204)
    await page.getByText('本轮已读取补充条件，后台校验继续运行。', { exact: true }).waitFor()
    await page.getByText('补充验收条件', { exact: true }).waitFor()
    await page.getByText(/本轮已结束，用时/).first().waitFor()
    const firstTurn = await fixture()
    assert.equal(firstTurn.turnCount, 1)
    assert.equal(firstTurn.stepCancelled, false)
    await page.getByRole('status', { name: '1 个后台任务运行中', exact: true }).waitFor()
    await page.getByRole('button', { name: '1 个后台任务', exact: true }).press('ArrowDown')
    await page.getByRole('list', { name: '后台任务', exact: true }).waitFor()
    await page.keyboard.press('Escape')
    jobsOffline = true
    await page.getByText('后台状态待刷新', { exact: true }).waitFor()
    assert(await page.getByRole('button', { name: '1 个后台任务', exact: true }).isVisible())
    jobsOffline = false
    await page.getByRole('status', { name: '1 个后台任务运行中', exact: true }).waitFor()

    const planChip = () => page.locator('.conversation-composer').getByRole('button', { name: '退出计划模式', exact: true })
    const enterPlan = async () => {
      assert.equal(await page.locator('.session-page-header').getByRole('button', { name: /计划模式/ }).count(), 0)
      await field().fill('/plan')
      await page.getByRole('option', { name: /计划.*进入计划模式/ }).click()
      await planChip().waitFor()
      assert.equal(await field().inputValue(), '')
    }
    const beforePlan = (await fixture()).modelCalls
    await enterPlan()
    assert.equal((await fixture()).modelCalls, beforePlan)
    await planChip().click()
    await planChip().waitFor({ state: 'hidden' })
    await enterPlan()
    assert.equal((await fixture()).modelCalls, beforePlan)
    await send('请先制定校验计划')
    let review = page.getByRole('region', { name: '计划审阅' })
    await review.getByRole('heading', { name: '初版校验计划' }).waitFor()
    // A new window can recover the same unanswered review from the host.
    const reopened = await context.newPage()
    await reopened.goto(url)
    await page.close()
    page = reopened
    review = page.getByRole('region', { name: '计划审阅' })
    await review.getByRole('heading', { name: '初版校验计划' }).waitFor()
    await review.getByRole('textbox', { name: '计划修改意见' }).fill('覆盖第二节')
    await review.getByRole('button', { name: '要求修改', exact: true }).click()
    await review.getByRole('heading', { name: '修订校验计划' }).waitFor()
    assert.equal((await fixture()).feedbackReceived, true)
    await review.getByRole('button', { name: '批准并执行', exact: true }).click()
    await page.getByText('已按批准的计划执行。', { exact: true }).waitFor()
    await review.waitFor({ state: 'hidden' })
    await planChip().waitFor({ state: 'hidden' })

    historyOffline = true
    await page.reload()
    await page.getByText(/fixture_read_unavailable/).first().waitFor()
    await send('请在历史暂不可用时继续回复')
    await page.getByText('历史暂不可用时仍能回复。', { exact: true }).waitFor()
    historyOffline = false
    await page.reload()
    await enterPlan()
    await send('请制定一份可以暂停的计划')
    await page.getByText('正在拟定待停止的计划。', { exact: true }).waitFor()
    await page.locator('.composer-send[data-action="stop"]').click()
    await page.locator('.composer-send[data-action="stop"]').waitFor({ state: 'hidden', timeout: 8000 })
    await page.getByText('已停止', { exact: true }).first().waitFor()
    assert.equal(await page.getByText(/重新核验|暂时不能发送新消息/).count(), 0)
    await send('停止后继续')
    await page.getByText('停止后仍可在原会话继续。', { exact: true }).waitFor()
    await page.reload()
    await page.getByText('已停止', { exact: true }).first().waitFor()
    await page.getByText('停止后仍可在原会话继续。', { exact: true }).waitFor()
    console.log('Verified steering, jobs, plan menu and review, and continuing in the same session after a stop with no gateway receipt.')
  } finally {
    await browser.close()
  }
}
run().catch(error => { console.error(error); process.exitCode = 1 })
