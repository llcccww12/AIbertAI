// node tests/dsh_web/native-tool-rows.mjs /path/to/prepared-DSH/resources/app
// Each tool name with a DSH keyed toolview renders through that row, not the
// generic fallback, so the employee transcript matches DSH's own conversation.
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { resolve } from 'node:path';

const sdk = resolve(process.argv[2]);
const requireSDK = createRequire(resolve(sdk, 'package.json'));
const requireUI = createRequire(resolve(sdk, 'packages/client/ui-chat/package.json'));
const { JSDOM } = requireSDK('jsdom');
const { window } = new JSDOM('<!doctype html><html lang="zh"><body><main></main></body></html>', { url: 'https://fixture.invalid' });
for (const key of ['document', 'DOMParser', 'HTMLElement', 'Element', 'Node']) globalThis[key] = window[key];
globalThis.window = window;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
const React = requireUI('react');
const { jsx } = requireUI('react/jsx-runtime');
const { bindChatFramework } = await import('../../AIbertAI/code/web/dsh/chat-framework.mjs');
bindChatFramework(React, requireUI('react/jsx-runtime'), requireUI('react-dom'));
const { default: Reply } = await import('../../AIbertAI/code/web/dsh/native-chat/reply.js');
const root = requireUI('react-dom/client').createRoot(document.querySelector('main'));
const row = name => document.querySelector(`[data-dsh-tool=${name}]`);

const content = 'name: aibert\n'.repeat(120);
const toolCalls = {
  write: { id: 'write', name: 'write', status: 'completed', output: 'Created file',
    input: { file_path: 'notes/plan.yml', content },
    dshResultMeta: { path: '/work/notes/plan.yml', operation: 'create', diffs: [] } },
  bash: { id: 'bash', name: 'bash', status: 'completed', output: 'hello\n[exit code: 0]',
    input: { command: 'echo hello', description: '打招呼' } },
  grep: { id: 'grep', name: 'grep', status: 'completed', output: 'a.txt:1:needle',
    input: { pattern: 'needle' } },
  read: { id: 'read', name: 'read', status: 'completed', output: '1\thello',
    input: { file_path: 'a.txt' }, dshResultMeta: { path: '/work/a.txt', offset: 1, lines: 1, totalLines: 1 } },
  web_fetch: { id: 'web_fetch', name: 'web_fetch', status: 'completed', output: 'page',
    input: { url: 'https://example.com/' } },
};
const message = { id: 'turn', parts: [
  ...Object.keys(toolCalls).map(toolCallId => ({ type: 'tool_call', toolCallId })),
  { type: 'text', text: '完成。' },
] };
const opened = [];
await React.act(() => root.render(jsx(Reply, {
  message, sessionId: 'session', toolCalls, streaming: false, completed: true, language: 'zh-CN',
  cwd: '/work', openFile: path => opened.push(path), renderHostPart: () => null,
})));
// Settled turns fold their process; open it like a user would.
await React.act(() => document.querySelector('[data-turn-process]').click());

for (const name of Object.keys(toolCalls)) assert.ok(row(name), `${name} row rendered`);
// FileMutationRow: path plus the decoded input size, like DSH's "编辑 path 1KB".
assert.match(row('write').textContent, /notes\/plan\.yml/);
assert.match(row('write').textContent, /\d+KB/, 'write row carries the DSH content-size suffix');
await React.act(() => row('write').querySelector('[data-disclosure-row]').click());
assert.match(row('write').querySelector('[data-diff]')?.textContent ?? '', /name: aibert/, 'write expands to its applied diff');
// BashRow uses DSH's terminal row with the streamed description.
assert.match(row('bash').textContent, /打招呼|echo hello/);
assert.match(row('grep').textContent, /needle/);
assert.match(row('read').textContent, /a\.txt/);
assert.match(row('web_fetch').textContent, /example\.com/);
assert.doesNotMatch(document.body.textContent, /\[object Object\]|undefined/);
console.log('Verified DSH keyed tool rows: write/edit size and diff, bash, grep, read and web_fetch.');
await React.act(() => root.unmount());
