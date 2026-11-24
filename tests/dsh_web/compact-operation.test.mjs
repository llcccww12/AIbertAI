import test from 'node:test';
import assert from 'node:assert/strict';
import { beginCompactOperation, finishCompactOperation, applyDSHCompaction } from '../../AIbertAI/code/web/dsh/compact-operation.mjs';

const first = '11111111-1111-4111-8111-111111111111';
const second = '22222222-2222-4222-8222-222222222222';
function storage() {
  const data = new Map();
  return { getItem: k => data.get(k) ?? null, setItem: (k,v) => data.set(k,v), removeItem: k => data.delete(k) };
}
test('retains identity through a lost reply and clears only its own completed operation', () => {
  const state = storage();
  assert.equal(beginCompactOperation(state, 'session', () => first), first);
  assert.equal(beginCompactOperation(state, 'session', () => second), first);
  finishCompactOperation(state, 'session', second);
  assert.equal(beginCompactOperation(state, 'session', () => second), first);
  finishCompactOperation(state, 'session', first);
  assert.equal(beginCompactOperation(state, 'session', () => second), second);
  assert.equal(beginCompactOperation(state, 'another', () => first), first);
});
test('refuses an invalid identity or an unpersisted reservation', () => {
  const state = storage();
  assert.throws(() => beginCompactOperation(state, 'session', () => 'invalid'));
  assert.throws(() => beginCompactOperation({getItem: () => null, setItem: () => {}}, 'session', () => first));
  state.setItem('aibertai.compact.session', 'damaged');
  assert.throws(() => beginCompactOperation(state, 'session', () => first));
});

test('coalesces a DSH compaction by transaction through reconnect and out-of-order pushes', () => {
  const running = {type: 'compaction', kind: 'auto', status: 'in_progress'};
  const done = {type: 'compaction', kind: 'auto'};
  let message = applyDSHCompaction({parts: []}, first, running);
  message = applyDSHCompaction(message, first, running);
  message = applyDSHCompaction(message, second, running);
  message = applyDSHCompaction(message, first, done);
  message = applyDSHCompaction(JSON.parse(JSON.stringify(message)), first, running);
  message = applyDSHCompaction(message, first, done);
  message = applyDSHCompaction(message, second, {...running, status: 'failed'});
  assert.deepEqual(message.parts, [{...done, compactionId: first}, {...running, status: 'failed', compactionId: second}]);
});
