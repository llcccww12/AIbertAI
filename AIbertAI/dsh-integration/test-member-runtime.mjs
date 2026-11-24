// Run with the packaged SDK's tsx loader and pass its resources/app path.
// Only the model is scripted. Both long waits, the hierarchy, cancellation and its journals are native DSH.
import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { pathToFileURL } from 'node:url'
import { performance } from 'node:perf_hooks'

const sdk = path.resolve(process.argv[2])
const require = createRequire(path.join(sdk, 'packages/sdk/server/package.json'))
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

const root = '11111111-1111-4111-8111-111111111111', parentOp = '22222222-2222-4222-8222-222222222222'
const home = await mkdtemp(path.join(tmpdir(), 'dsh-team-cancel-'))
const ctx = new Context()
let server
const parentWaiting = Promise.withResolvers(), childWaiting = Promise.withResolvers()
const guard = setTimeout(() => { console.error('native cancellation timed out'); process.exit(1) }, 20000)
try {
  await mountAgentLoopTestDependencies(ctx)
  await ctx.plugin(AgentLoop, { agents: [] })
  await mountWorkingDirectoryFixture(ctx)
  await ctx.plugin(Subagents)
  await ctx.plugin(Persistence, { root: home })
  registerWaitAgent(ctx)
  ctx.llm.registerAdapter(['mock'], new MockAdapter([
    toolCallResponse('wait-parent', 'wait_agent', { timeout_ms: 600000 }),
    toolCallResponse('wait-child', 'wait_agent', { timeout_ms: 600000 }),
    textResponse('next child completed'),
  ]))
  server = new HarnessSdkJsonRpcServer(ctx, {
    request: async (method) => { throw new Error('unexpected host request ' + method) },
    notify(method, msg) {
      if (method === 'session.event' && msg.event.type === 'tool/call' && msg.event.data.name === 'wait_agent') {
        (msg.sessionId === root ? parentWaiting : childWaiting).resolve()
      }
    },
  })
  await server.initialize({ cwd: home, provider: 'mock', model: 'mock', aibertaiHostSessionId: root })
  ctx.subagents.registerProvider({ name: 'local-test', capabilities: NO_START_CAPABILITIES, inheritsParentContext: false, prepareContinuable: async () => ({}) })
  await server.prompt({ sessionId: root, contentBlocks: [{ type: 'text', text: 'wait for the team' }], aibertaiOperationId: parentOp })
  await parentWaiting.promise
  const parent = ctx.agents.get(root)
  const delegate = label => ctx.subagents.startActivation({
    provider: 'local-test', parent, label, signal: new AbortController().signal,
    delivery: 'caller', prompt: [{ type: 'text', text: label }],
  })
  const first = await delegate('first delegate')
  await childWaiting.promise
  const startedAt = performance.now()
  const stopped = await server.handleRequest('aibertai/session-cancel', { sessionId: root })
  assert.deepEqual(stopped, { accepted: true })
  await server.wait({ sessionId: root })
  const elapsed = Math.round(performance.now() - startedAt)
  assert.ok(elapsed < 5000, 'did not stop before host graceful-stop timeout')
  assert.equal(ctx.agents.get(first.childId), undefined)
  assert.ok(await ctx.sessions.flush(parent.session))
  for (const [id, cause] of [[root,'user'],[first.childId,'parent']]) {
    const handle = await ctx.sessionPersistence.open(id, 'read')
    try {
      const { events } = await handle.read()
      const end = events.findLast(e => e.type === 'turn/end')
      assert.ok(end, 'missing durable turn/end for '+cause+' after '+events.at(-1)?.type)
      assert.deepEqual(end.data.reason, {kind:'aborted',reason:{kind:cause}})
    } finally { await handle.close() }
  }
  assert.deepEqual(await server.handleRequest('aibertai/session-cancel', { sessionId: root }), { accepted: false })
  const second = await delegate('second delegate')
  const outcome = await second.result
  assert.equal(outcome.stopReason, 'completed')
  assert.ok(outcome.output.some(b => b.type === 'text' && b.text === 'next child completed'))
  console.log('Stopped two live wait_agent calls in ' + elapsed + ' ms; same root completed a new delegation.')
} finally {
  clearTimeout(guard)
  try { await server?.shutdown() } finally {
    await ctx.fiber.dispose()
    await rm(home, { recursive: true, force: true })
  }
}

const { memberHistoryEntry } = await source('packages/sdk/server/src/aibertai-member-history.ts')
const result = {
  type: 'tool/result', seq: 46, time: 123,
  data: { turn: 1, step: 2, message: { source: { kind: 'tool', callId: 'read-evidence' }, content: [
    { type: 'text', text: 'a'.repeat(4095) + '😀 more' },
    { type: 'image', attachment: { name: 'page.png', bytes: 250, attachmentId: 'private-reference' } },
  ] }, meta: { raw: 'private metadata' } },
}
const preview = memberHistoryEntry(result)
assert.equal(preview.callId, 'read-evidence')
assert.equal(preview.text.length, 4095)
assert.equal(preview.textLength, 4102)
assert.equal(preview.truncated, true)
assert.deepEqual(preview.attachments, [{ type: 'image', name: 'page.png', bytes: 250 }])
assert.ok(!JSON.stringify(preview).includes('private'))
assert.equal(memberHistoryEntry({ ...result, surfaceOp: { op: 'replace', startSeq: 2, endSeq: 2 } }), undefined)
assert.equal(memberHistoryEntry({ type: 'user/message', data: { source: { kind: 'time-context' }, content: [] } }), undefined)

// A cold catalog marks every member inactive, even if the previous process
// crashed inside one of its turns. A sibling's durable journal must agree.
const { teamCancelProof } = await source('packages/sdk/server/src/aibertai-team-cancel-proof.ts')
const event = (seq, type, data) => ({ seq, type, time: seq + 100, data })
const childId = 'child', siblingId = 'sibling', pending = 'pending'
const begin = event(0, 'turn/start', { turn: 1 })
const start = event(1, 'step/start', { turn: 1, step: 1 })
const stepEnd = event(4, 'step/end', { turn: 1, step: 1 })
const end = cause => event(5, 'turn/end', { turn: 1, reason: { kind: 'aborted', reason: { kind: cause } } })
const archives = new Map([
  [root, [begin, event(1, 'user/message', { source: { kind: 'user', aibertaiOperationId: parentOp } }),
    event(2, 'subagent/catalog', { childId }), event(3, 'subagent/catalog', { childId: siblingId }), end('user')]],
  [childId, [begin, start, event(2, 'aibertai/invocation', { purpose: 'conversation', invocationId: pending }),
    event(3, 'assistant/message', { turn: 1, step: 1, invocationId: pending, interrupted: true }), stepEnd, end('parent')]],
  [siblingId, [begin, start, stepEnd, end('parent')]],
])
const journal = {
  sessions: { flush: async () => true },
  sessionPersistence: { open: async id => ({
    header: id === root ? {} : { origin: 'subagent', parentSession: root }, inheritedEventCount: 0,
    read: async () => ({ events: archives.get(id) }), close: async () => {},
  }) },
  get: () => ({ listDescendants: async () => [childId, siblingId].map(id => ({
    id, parentId: root, kind: 'child', mode: 'continuable', activity: 'inactive', depth: 1,
  })) }),
}
const proof = () => teamCancelProof(journal, {id:root}, parentOp, pending)
assert.equal((await proof()).state, 'cancelled')
archives.set(siblingId, [begin, start])
assert.deepEqual(await proof(), {state:'unknown'})
archives.set(siblingId, [begin, start, end('parent')])
assert.deepEqual(await proof(), {state:'unknown'})
archives.set(siblingId, archives.get(childId))
assert.deepEqual(await proof(), {state:'unknown'})

archives.set(siblingId, [begin, start, stepEnd, end('parent')])
const marker = event(2, 'aibertai/invocation', { purpose: 'conversation', invocationId: pending })
const attempt = { turn: 1, step: 1, invocationId: pending, stream: [] }
function beforeTokenCase(data, suffix = [stepEnd, end('parent')]) {
  archives.set(childId, [begin, start, marker, event(3, 'assistant/attempt', data), ...suffix])
}
beforeTokenCase(attempt)
assert.deepEqual(await proof(), {state:'cancelled', memberId:childId, seq:3, turn:1, step:1,
  rootEndSeq:5, childEndSeq:5, beforeFirstToken:true})
for (const invalid of [
  {...attempt, stream:[{type:'text-delta', text:'partial'}]}, {...attempt, usage:{}},
  {...attempt, invocationFinish:'error'}, {...attempt, invocationId:'different'}, {...attempt, step:2},
]) {
  beforeTokenCase(invalid)
  assert.deepEqual(await proof(), {state:'unknown'})
}
beforeTokenCase(attempt, [event(4, 'tool/call', { turn:1, step:1 }),
  event(5, 'step/end', {turn:1,step:1}), event(6, 'turn/end', {turn:1, reason:{kind:'aborted',reason:{kind:'parent'}}})])
assert.deepEqual(await proof(), {state:'unknown'})
beforeTokenCase(attempt, [stepEnd, event(5, 'turn/end', { turn:1, reason:{kind:'error'} })])
assert.deepEqual(await proof(), {state:'unknown'})
beforeTokenCase(attempt)
archives.set(root, [...archives.get(root).slice(0,-1), end('parent')])
assert.deepEqual(await proof(), {state:'unknown'})

const { memberInterruptEvidence, pendingMemberInvocation } = await source('packages/sdk/server/src/aibertai-team-cancel-proof.ts')
const target = [begin, start, marker, event(3, 'assistant/attempt', attempt), stepEnd, end('user')]
assert.deepEqual(pendingMemberInvocation(target.slice(0,3)), {id: pending, turn: 1})
assert.equal(pendingMemberInvocation(target), undefined)
assert.deepEqual(memberInterruptEvidence(target, pending), {seq: 3, step: 1, turn: 1, childEndSeq: 5, beforeFirstToken: true})
assert.equal(memberInterruptEvidence([...target.slice(0,-1),end('parent')], pending), undefined)
assert.equal(memberInterruptEvidence(target, 'different-invocation'), undefined)
assert.equal(memberInterruptEvidence([...target, marker], pending), undefined)
assert.equal(memberInterruptEvidence([begin,start,marker,event(3,'assistant/attempt',{...attempt,stream:['partial']}),stepEnd,end('user')], pending), undefined)
