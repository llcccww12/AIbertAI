import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { pathToFileURL } from 'node:url'

const sdk = path.resolve(process.argv[2])
const require = createRequire(path.join(sdk, 'packages/sdk/server/package.json'))
// Exercise the same SDK services as the employee transport, using isolated journals and a scripted model.
const dep = spec => import(pathToFileURL(require.resolve(spec)))
const source = name => import(pathToFileURL(path.join(sdk, name)))
const { Context } = await dep('@deepseek-ai/cordis')
const { default: AgentLoop } = await dep('@deepseek-ai/dsh-agent-loop')
const { mountAgentLoopTestDependencies } = await dep('@deepseek-ai/dsh-agent-loop-testkit')
const { default: Subagents, NO_START_CAPABILITIES } = await dep('@deepseek-ai/dsh-subagent')
const Persistence = await dep('@deepseek-ai/dsh-session-persistence-jsonl')
const { mountWorkingDirectoryFixture } = await source('packages/subagent/subagent/tests/working-directory-fixture.ts')
const { MockAdapter, textResponse, toolCallResponse } = await source('packages/core/agent-loop/tests/mock-adapter.ts')
const { HarnessSdkJsonRpcServer } = await source('packages/sdk/server/src/server.ts')
const { registerWaitAgent } = await source('packages/subagent/tool-subagent-control/src/wait-agent.ts')

const home = await mkdtemp(path.join(tmpdir(), 'dsh-member-prompts-'))
const ctx = new Context()
const root = '11111111-1111-4111-8111-111111111111'
const parentOp = '22222222-2222-4222-8222-222222222222'
const followId = '33333333-3333-4333-8333-333333333333'
const coldId = '44444444-4444-4444-8444-444444444444'
const parentWaiting = Promise.withResolvers(), childWaiting = Promise.withResolvers()
let server
const settlements = []
const followingStarted = Promise.withResolvers()
let followingMember
const guard = setTimeout(() => { console.error('member control timed out'); process.exit(1) }, 20000)
try {
  await mountAgentLoopTestDependencies(ctx)
  await ctx.plugin(AgentLoop, { agents: [] })
  await mountWorkingDirectoryFixture(ctx)
  await ctx.plugin(Subagents)
  await ctx.plugin(Persistence, { root: home })
  const { TestSessionQueryEngine } = await source('packages/session-query/session-query/tests/test-service.ts')
  await ctx.plugin(TestSessionQueryEngine)
  registerWaitAgent(ctx)
  const script = [
    toolCallResponse('wait-parent', 'wait_agent', { timeout_ms: 600000 }),
    toolCallResponse('wait-child', 'wait_agent', { timeout_ms: 600000 }),
    'hang',
    textResponse('archived member answered'),
    textResponse('root received the result'),
  ]
  ctx.llm.registerAdapter(['aibertai'], new MockAdapter(script))
  server = new HarnessSdkJsonRpcServer(ctx, {
    request: async method => { throw new Error('unexpected host request ' + method) },
    notify(method, msg) {
      if (method === 'aibertai/model-settled' && msg.memberInterrupt) settlements.push(msg)
      if (method === 'session.event' && msg.sessionId === followingMember && msg.event.type === 'aibertai/invocation') followingStarted.resolve()
      if (method === 'session.event' && msg.event.type === 'tool/call' && msg.event.data.name === 'wait_agent')
        (msg.sessionId === root ? parentWaiting : childWaiting).resolve()
    },
  })
  await server.initialize({ cwd: home, provider: 'aibertai', model: 'mock', aibertaiLeaseSessionId: root })
  ctx.subagents.registerProvider({ name: 'local-test', capabilities: NO_START_CAPABILITIES, inheritsParentContext: false, prepareContinuable: async () => ({}) })
  const rpc = (method, args) => server.handleRequest(method, { sessionId: root, ...args })
  const bind = id => rpc('aibertai/turn-bind', {
    operationId: id, hostTools: [], expertInstructions: '', workspaceBridge: false,
  })
  await bind(parentOp)
  await rpc('session/prompt', {
    contentBlocks: [{ type: 'text', text: 'manage a real team' }], aibertaiOperationId: parentOp,
  })
  await parentWaiting.promise
  const child = await ctx.subagents.startActivation({
    provider: 'local-test', parent: ctx.agents.get(root), label: 'designer', name: 'designer',
    signal: new AbortController().signal, delivery: 'caller', prompt: [{ type: 'text', text: 'work on the plan' }],
  })
  await childWaiting.promise

  const send = args => rpc('aibertai/member-prompt', { memberId: child.childId, ...args })
  const interrupt = (operationId, memberId = child.childId) => rpc('aibertai/member-interrupt', { memberId, operationId })
  const input = { requestId: followId, operationId: parentOp, text: 'use blue' }
  const [first, retry] = await Promise.all([send(input), send(input)])
  assert.ok(first.messageId)
  assert.deepEqual(first, retry)
  await assert.rejects(send({ ...input, text: 'conflict' }), /member_request_conflict/)
  await assert.rejects(send({ ...input, memberId: 'foreign' }), /member_unavailable/)
  await assert.rejects(interrupt(parentOp, 'foreign'), /member_unavailable/)

  assert.deepEqual(await interrupt(parentOp), { accepted: true })
  await child.result
  assert.equal(ctx.agents.get(root).status, 'running')
  assert.ok((await rpc('aibertai/session-cancel', {})).accepted)
  await rpc('session/wait', {})
  await rpc('aibertai/turn-release', { operationId: parentOp })

  await bind(coldId)
  const admission = await send({ requestId: coldId, operationId: coldId, text: 'one more change', standalone: true })
  assert.ok(admission.messageId)
  const result = await rpc('session/wait', {})
  assert.equal(result.reason, 'completed')
  assert.equal(result.answer, 'archived member answered')
  assert.equal(result.memberStopped, false)
  let count = 0
  for (let from = 0; ;) {
    const page = await rpc('aibertai/members', { memberId: child.childId, from })
    count += page.entries.filter(entry => entry.requestId === coldId).length
    if (!page.more) break
    from = page.next
  }
  assert.equal(count, 1)
  await rpc('aibertai/turn-release', { operationId: coldId })

  const stoppedId = '55555555-5555-4555-8555-555555555555'
  await bind(stoppedId)
  followingMember = child.childId
  script.splice(0, script.length, 'hang', textResponse('root acknowledged the interruption'))
  await send({ requestId: stoppedId, operationId: stoppedId, text: 'work until stopped', standalone: true })
  await followingStarted.promise
  assert.deepEqual(await interrupt(stoppedId), { accepted: true })
  const stopped = await rpc('session/wait', {})
  assert.equal(stopped.memberStopped, true)
  assert.equal(stopped.abortReason, 'user')
  assert.equal(stopped.reason, 'aborted')
  await rpc('aibertai/turn-release', { operationId: stoppedId })
  for (const operationId of [parentOp, stoppedId]) {
    const matches = settlements.filter(r => r.memberInterrupt.operationId === operationId)
    assert.equal(matches.length, 1, 'missing or duplicate native member interruption settlement')
    assert.equal(matches[0].sessionId, child.childId)
    assert.equal(matches[0].state, 'failed')
    assert.ok(matches[0].seq > 0 && matches[0].seq < matches[0].memberInterrupt.turnEndSeq)
  }
  console.log('Verified member admission, retry, scope, isolated stop and same-ID continuation.')
} finally {
  clearTimeout(guard)
  try {
    await server?.shutdown()
  } finally {
    await ctx.fiber.dispose()
    await rm(home, { recursive: true, force: true })
  }
}
