import { test } from 'node:test';
import assert from 'node:assert/strict';
import { contextReading, contextSegments, formatContextTokens } from '../../AIbertAI/code/web/dsh/context-meter.mjs';

test('unknown occupancy stays hidden and an observed zero is valid', () => {
  for (const input of [undefined, { window: 100 }, { used: 4 }, { used: -1, window: 100 }, { used: 1, window: 0 }]) {
    assert.equal(contextReading(input), undefined);
  }
  assert.deepEqual(contextReading({ used: 0, window: 100 }), { used: 0, size: 100 });
});

test('only complete composition estimates divide the occupied fraction of capacity', () => {
  const context = { used: 64100, window: 1000000 };
  assert.deepEqual(contextReading({ ...context, breakdown: { systemTokens: 1 } }), { used: 64100, size: 1000000 });
  const breakdown = { systemTokens: 1500, toolsTokens: 7900, messageTokens: 48100 };
  const view = contextReading({ ...context, breakdown });
  const segments = contextSegments(view);
  assert.deepEqual(segments.map(s => s.key), Object.keys(breakdown));
  assert(Math.abs(segments.reduce((sum, part) => sum + part.width, 0) - 6.41) < 0.000001);
  assert.deepEqual([context.used, context.window, ...Object.values(breakdown)].map(formatContextTokens), ['64.1K', '1M', '1.5K', '7.9K', '48.1K']);
  assert.deepEqual(contextSegments({ used: 110, size: 100 }), [{ key: 'total', width: 100 }]);
});
