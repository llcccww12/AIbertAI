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
    await page.getByText(/本次输出已达到模型长度上限/).first().waitFor()
    await send('继续')
    await until(s=>s.ended && s.answer==='已接着写完。' && s.modelCalls===6,'continue from limit')
    await send('测试结果尚未核验的模型调用')
    await until(s=>s.turnCount===6 && s.ended && !s.settled && s.failure==='dsh_model_outcome_unknown' && s.pending && s.modelCalls===7,'unknown outcome')
    await page.reload()
    await page.getByText(/模型调用结果仍待核验/).first().waitFor()
    const check = page.getByRole('button',{name:'重新核验',exact:true})
    await check.click()
    await page.getByText('暂未核验完成，可稍后再查。',{exact:true}).waitFor()
    assert.equal((await fixture()).modelCalls,7)
    await send('继续上一项')
    await page.waitForTimeout(800)
    const state=await fixture()
    assert.equal(state.modelCalls,7)
    assert.equal(state.turnCount,6)
    assert.equal(state.pending,true)
    assert.equal(state.queued,0)
    assert.equal(state.queuePaused,false)
    assert.equal((await context.request.post(base+'/fixture/model-receipt')).status(),204)
    await check.click()
    await until(s=>s.turnCount===6 && s.settled && s.failure==='dsh_model_service_unavailable' && !s.pending && s.modelCalls===7,'late accounting receipt')
    await check.waitFor({state:'hidden'})
    await page.getByText(/模型服务暂时不可用/).first().waitFor()
    await send('继续核验后的任务')
    await until(s=>s.turnCount===7 && s.ended && s.modelCalls===8 && s.answer==='已从确认的失败处继续。','verified continuation')
    await send('检查中途断开的输出')
    await until(s=>s.turnCount===8 && s.ended && !s.settled && s.pending && s.modelCalls===9 && s.failure==='dsh_model_response_interrupted' && s.answer==='断开前已收到的文字','partial interrupted response')
    const survivor = await context.newPage()
    await survivor.goto(base + '/')
    await survivor.locator('body').waitFor()
    assert.equal((await context.request.post(base+'/fixture/hold-stream-receipt')).status(),204)
    await page.reload()
    await until(s=>s.streamReceiptWaiting, 'first window holds the account receipt check')
    const overlappingCheck = new Promise(resolve => survivor.on('websocket', ws => {
      ws.on('framesent', frame => {
        const request = JSON.parse(String(frame.payload))
        if (request.method === '_aibertai/session/reconcile-workspace' && request.params?.sessionId === url.split('/').pop()) resolve()
      })
    }))
    await survivor.goto(url)
    await Promise.race([overlappingCheck, new Promise((_, reject) => {
      const timer = setTimeout(()=>reject(new Error('second window never joined the receipt check')), 10000)
      timer.unref()
    })])
    await page.close()
    page = survivor
    assert.equal((await context.request.post(base+'/fixture/model-receipt')).status(),204)
    await until(s=>s.turnCount===8 && s.settled && !s.pending && s.modelCalls===9 && s.failure==='dsh_model_stream_ended','overlapping windows settle without model replay')
    await page.getByRole('button',{name:'重新核验',exact:true}).waitFor({state:'hidden',timeout:10000})
    await page.getByText('断开前已收到的文字',{exact:true}).waitFor()
    await page.getByText(/用量已核验/).first().waitFor()
    await send('继续补完断流的回复')
    await until(s=>s.ended && s.modelCalls===10 && s.answer==='已补完断流的回复。','continue after partial stream')
    console.log('Verified ten provider requests through the employee workbench, including safe retry, paid rate limit, output limit, delayed accounting and recovery from a partial stream without replay.')
  } finally { await browser.close() }
}
run().catch(error=>{console.error(error);process.exitCode=1})
