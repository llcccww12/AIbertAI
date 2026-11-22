import { test } from 'node:test';
import { strict as assert } from 'node:assert';
import { applyDSHStream, preserveDSHLiveMessage } from '../../AIbertAI/code/web/dsh/tool-actions.mjs';

test('keeps an active DSH step visible while the first model output is pending', () => {
  const user = { id: 'user', role: 'user', parts: [] };
  const live = { id: 'answer', role: 'assistant', dshStep: 1, parts: [] };
  assert.deepEqual(preserveDSHLiveMessage([user, live], [user], true), [user, live]);
  assert.deepEqual(preserveDSHLiveMessage([user, live], [user], false), [user]);
  const completed = { ...live, endedAt: 4000 };
  assert.deepEqual(preserveDSHLiveMessage([user, live], [user, completed], true), [user, completed]);
});

test('history refresh keeps an active stream until authoritative settlement', () => {
  const user = { id: 'user', role: 'user', parts: [{ type: 'text', text: 'question' }] };
  let live = applyDSHStream({ id: 'answer', role: 'assistant', parts: [] }, {
    sessionUpdate: 'agent_message_start', streamAttemptId: 'a', streamRevision: 1,
  });
  let messages = preserveDSHLiveMessage([user, live], [user], true);
  live = applyDSHStream(messages[1], {
    sessionUpdate: 'agent_thought_chunk', streamAttemptId: 'a', streamRevision: 2,
    streamIndex: 0, content: { type: 'text', text: 'thinking' },
  });
  messages = preserveDSHLiveMessage([user, live], [user, { id: 'answer', role: 'assistant', parts: [] }], true);
  assert.equal(messages[1].parts[0].text, 'thinking');
  live = applyDSHStream(messages[1], {
    sessionUpdate: 'agent_message_settled', streamAttemptId: 'a', content: { type: 'text', text: 'done' },
  });
  const authoritative = { id: 'answer', role: 'assistant', parts: [{ type: 'text', text: 'done' }] };
  assert.deepEqual(preserveDSHLiveMessage([user, live], [user, authoritative], true), [user, authoritative]);
  assert.deepEqual(preserveDSHLiveMessage([user, messages[1]], [user, authoritative], false), [user, authoritative]);
});

test('settles only the matching attempt and ignores late or duplicate events', () => {
  let m = { parts: [{ type: 'tool_call', toolCallId: 'one' }] };
  const update = (value) => { m = applyDSHStream(m, value); };
  update({ sessionUpdate: 'agent_message_start', streamAttemptId: 'a', streamRevision: 1 });
  update({ sessionUpdate: 'agent_message_chunk', streamAttemptId: 'a', streamRevision: 2, streamIndex: 0, content: { type: 'text', text: 'temporary' } });
  update({ sessionUpdate: 'agent_message_chunk', streamAttemptId: 'a', streamRevision: 2, streamIndex: 0, content: { type: 'text', text: 'duplicate' } });
  update({ sessionUpdate: 'agent_message_settled', streamAttemptId: 'a', content: { type: 'text', text: 'committed' } });
  update({ sessionUpdate: 'agent_message_chunk', streamAttemptId: 'a', streamRevision: 3, streamIndex: 1, content: { type: 'text', text: 'late' } });
  assert.deepEqual(m.parts.map(p => p.text || p.toolCallId), ['one', 'committed']);
  update({ sessionUpdate: 'agent_message_start', streamAttemptId: 'b', streamRevision: 4 });
  update({ sessionUpdate: 'agent_message_chunk', streamAttemptId: 'b', streamRevision: 5, streamIndex: 0, content: { type: 'text', text: 'discard' } });
  update({ sessionUpdate: 'agent_message_reset', streamAttemptId: 'b' });
  update({ sessionUpdate: 'agent_message_settled', streamAttemptId: 'b', content: { type: 'text', text: 'late' } });
  assert.deepEqual(m.parts.map(p => p.text || p.toolCallId), ['one', 'committed']);
});

test('authoritative settlement replaces a partial or empty stream', () => {
  for (const partial of [false, true]) {
    let m = { parts: [] };
    m = applyDSHStream(m, { sessionUpdate: 'agent_message_start', streamAttemptId: 'a', streamRevision: 10 });
    if (partial) m = applyDSHStream(m, { sessionUpdate: 'agent_message_chunk', streamAttemptId: 'a', streamRevision: 11, streamIndex: 4, content: { type: 'text', text: 'partial' } });
    m = applyDSHStream(m, { sessionUpdate: 'agent_message_settled', streamAttemptId: 'a', content: { type: 'text', text: 'authoritative' } });
    assert.deepEqual(m.parts.map(part => part.text), ['authoritative']);
  }
});

test('keeps native reasoning in order and replaces an abandoned attempt', () => {
  let m = { parts: [] };
  const update = value => { m = applyDSHStream(m, value); };
  update({ sessionUpdate: 'agent_message_start', streamAttemptId: 'a', streamRevision: 1 });
  update({ sessionUpdate: 'agent_thought_chunk', streamAttemptId: 'a', streamRevision: 2, streamIndex: 0, content: { type: 'text', text: 'consider' } });
  update({ sessionUpdate: 'agent_message_chunk', streamAttemptId: 'a', streamRevision: 3, streamIndex: 1, content: { type: 'text', text: 'answer' } });
  update({ sessionUpdate: 'agent_thought_chunk', streamAttemptId: 'a', streamRevision: 4, streamIndex: 2, content: { type: 'text', text: 'next' } });
  assert.deepEqual(m.parts.map(p => [p.type, p.text]), [['thought','consider'], ['text','answer'], ['thought','next']]);
  update({ sessionUpdate: 'agent_message_settled', streamAttemptId: 'a', content: { type: 'text', text: 'final' }, _meta: { dshParts: [{type:'thought',text:'verified'},{type:'text',text:'final'}] } });
  assert.deepEqual(m.parts.map(p => [p.type, p.text]), [['thought','verified'], ['text','final']]);
  update({ sessionUpdate: 'agent_message_start', streamAttemptId: 'b', streamRevision: 5 });
  update({ sessionUpdate: 'agent_thought_chunk', streamAttemptId: 'b', streamRevision: 6, streamIndex: 0, content: {type:'text',text:'abandoned'} });
  update({ sessionUpdate: 'agent_message_reset', streamAttemptId: 'b' });
  assert.deepEqual(m.parts.map(p => [p.type, p.text]), [['thought','verified'], ['text','final']]);
});
test('provisional thought stays live until receipt, then becomes settled', () => {
  let message = { id: 'r', role: 'assistant', parts: [] };
  message = applyDSHStream(message, { sessionUpdate: 'agent_message_start', streamAttemptId: 'a', streamRevision: 1 });
  message = applyDSHStream(message, { sessionUpdate: 'agent_thought_chunk', streamAttemptId: 'a', streamRevision: 2, streamIndex: 0, content: { type: 'text', text: 'live' } });
  message = applyDSHStream(message, { sessionUpdate: 'agent_message_settled', streamAttemptId: 'a', content: { type: 'text', text: '' }, _meta: { dshProvisional: true } });
  assert.equal(message.dshStream.closed, false);
  assert.equal(preserveDSHLiveMessage([message], [{ id: 'r', role: 'assistant', parts: [] }], true)[0].parts[0].text, 'live');
  message = applyDSHStream(message, { sessionUpdate: 'agent_message_settled', streamAttemptId: 'a', content: { type: 'text', text: '' }, _meta: { dshParts: [{ type: 'thought', text: 'verified' }] } });
  assert.equal(message.dshStream.closed, true);
  assert.equal(message.parts[0].text, 'verified');
});

test('late reset removes only the failed attempt', () => {
  const message = { parts: [{ type: 'thought', text: 'failed', dshAttemptId: 'a' }, { type: 'thought', text: 'current', dshAttemptId: 'b' }], dshStream: { id: 'b', closed: false, revision: 3 } };
  const next = applyDSHStream(message, { sessionUpdate: 'agent_message_reset', streamAttemptId: 'a' });
  assert.deepEqual(next.parts.map(part => part.text), ['current']);
  assert.equal(next.dshStream.id, 'b');
  assert.equal(next.dshStream.closed, false);
});

test('renders an SDK message that settled without streaming once and in order', () => {
  const before = { parts: [{ type: 'tool_call', toolCallId: 'read-template' }] };
  const update = {
    sessionUpdate: 'agent_message_settled', streamAttemptId: '',
    _meta: { dshMessageSeq: 45, dshParts: [
      { type: 'thought', text: 'Use the selected template' },
      { type: 'text', text: 'Template loaded' },
    ] },
  };
  const message = applyDSHStream(before, update);
  assert.deepEqual(message.parts, [...before.parts, ...update._meta.dshParts]);
  assert.equal(applyDSHStream(message, update), message);
  assert.equal(applyDSHStream(message, { ...update, _meta: { ...update._meta, dshMessageSeq: 44 } }), message);
  assert.equal(applyDSHStream(message, { ...update, _meta: { dshParts: update._meta.dshParts } }), message);
});
