const assert = require('node:assert/strict')
const { chromium } = require(process.env.DSH_PLAYWRIGHT_CORE || '../../AIbertAI/code/app/node_modules/playwright-core')

async function run() {
  const base = process.env.DSH_WEB_BASE
  const browser = await chromium.launch({ executablePath: process.env.DSH_CHROME, headless: true })
  const context = await browser.newContext({ locale: 'zh-CN' })
  let page = await context.newPage()
  const fixture = async () => (await context.request.get(base+'/fixture')).json()
  const until = async (test, label) => {
    let state
    const timeout = Date.now()+30000
    do {
      state=await fixture()
      if (test(state)) return state
      await new Promise(resolve=>setTimeout(resolve, 200))
    } while(Date.now()<timeout)
    assert.fail(label + ' ' + JSON.stringify(state))
  }
  const send = async text => {
    const field=page.locator('.conversation-composer textarea')
    await field.fill(text)
    await field.press('Enter')
  }
  try {
    const url=base+'/sessions/'+(await fixture()).sessionId
    await page.goto(url)
    const login=page.getByRole('button',{name:'登录',exact:true})
    await login.or(page.getByRole('button',{name:/会话的工具/})).first().waitFor()
    await page.locator('#boot-splash').waitFor({state:'hidden'})
    if(await login.isVisible()){
      await page.locator('input').first().fill(base)
      await login.click()
      await page.waitForURL(address=>address.pathname!='/login')
      await page.goto(url)
    }
    await send('测试一条请求在转发前遇到临时限流')
    await until(s=>s.ended && s.modelCalls===2 && s.answer==='限流后已继续处理。', 'safe retry')
    await page.getByText('限流后已继续处理。',{exact:true}).waitFor()
    await send('测试转发后限流')
    await until(s=>s.turnCount===2 && s.ended && s.settled && s.failure==='dsh_model_rate_limited' && !s.pending && s.modelCalls===3,'accounted rejection')
    await page.getByText(/模型请求达到频率限制/).first().waitFor()
    assert.doesNotMatch(await page.locator('body').innerText(),/private test diagnostic/)
    await send('继续处理同一个任务')
    await until(s=>s.turnCount===3 && s.ended && s.answer==='已在原会话继续。' && s.modelCalls===4,'fresh continuation')
    await send('请生成一段较长的文字')
    await until(s=>s.turnCount===4 && s.ended && s.settled && s.failure==='dsh_model_output_limit' && !s.pending && s.answer==='已保留的半句' && s.modelCalls===5,'complete limited response')
    await page.getByText(/已达到输出 token 上限/).first().waitFor()
    await send('继续')
    await until(s=>s.ended && s.answer==='已接着写完。' && s.modelCalls===6,'continue from limit')
    await send('测试结果尚未核验的模型调用')
    await until(s=>s.turnCount===6 && s.ended && s.modelCalls===7 &&
      s.failure==='dsh_model_service_unavailable' && s.reportFailures>0, 'failed call with reporting offline')
    assert.equal((await context.request.post(base+'/fixture/park-previous-invocation')).status(),204)
    assert.equal((await fixture()).pending,true)
    await page.reload()
    await page.getByText(/模型服务暂时不可用/).first().waitFor()
    await page.getByRole('button',{name:'重新核验',exact:true}).waitFor({state:'hidden'})
    const switchModel = process.env.DSH_WEB_MODEL_SWITCH === '1'
    const selectModel = async (current, next) => {
      await page.locator('.composer-model').filter({ hasText: current }).click()
      await page.getByText(next, { exact: true }).last().click()
      await page.locator('.composer-model').filter({ hasText: next }).waitFor()
    }
    if (switchModel) {
      await selectModel('验收模型', '备用验收模型')
      assert.equal((await fixture()).pending, true, 'model selection does not wait for old receipt')
    }
    await send('继续处理失败后的任务')
    await until(s=>s.turnCount===7 && s.ended && s.modelCalls===8 && !s.pending &&
      s.previousInvocationArchived && s.answer==='已从确认的失败处继续。','continue without prior receipt')
    if (switchModel) {
      assert.equal((await fixture()).model, '23232323-2323-4232-8232-232323232323')
      await selectModel('备用验收模型', '验收模型')
    }
    await send('检查中途断开的输出')
    await until(s=>s.turnCount===8 && s.ended && s.modelCalls===9 &&
      s.answer==='断开前已收到的文字' && s.failure==='dsh_model_stream_ended','partial interrupted response')
    await page.getByText('断开前已收到的文字',{exact:true}).waitFor()
    await page.getByText(/模型输出中断，已收到的内容已保留/).first().waitFor()
    await send('继续补完断流的回复')
    await until(s=>s.ended && s.modelCalls===10 && s.answer==='已补完断流的回复。','continue after partial stream')
    console.log('Verified ten provider requests in the same chat with reporting offline, including a new answer while the previous invocation receipt was unresolved.')
  } finally { await browser.close() }
}
run().catch(error=>{console.error(error);process.exitCode=1})
