// node tests/dsh_web/native-reply.mjs /path/to/prepared-DSH/resources/app
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { resolve } from 'node:path';

const sdk = resolve(process.argv[2]);
const requireSDK = createRequire(resolve(sdk, 'package.json'));
const requireUI = createRequire(resolve(sdk, 'packages/client/ui-chat/package.json'));
const { JSDOM } = requireSDK('jsdom');
const { window } = new JSDOM('<!doctype html><html lang="zh"><body><main></main></body></html>', {
  url: 'https://fixture.invalid',
});
for (const key of ['document', 'DOMParser', 'HTMLElement', 'Element', 'Node']) globalThis[key] = window[key];
globalThis.window = window;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
const React = requireUI('react');
const { jsx } = requireUI('react/jsx-runtime');
const { bindChatFramework } = await import('../../AIbertAI/code/web/dsh/chat-framework.mjs');
bindChatFramework(React, requireUI('react/jsx-runtime'), requireUI('react-dom'));
const { default: Reply } = await import('../../AIbertAI/code/web/dsh/native-chat/reply.js');
const root = requireUI('react-dom/client').createRoot(document.querySelector('main'));
const find = selector => document.querySelector(selector);
const click = async selector => {
  assert.ok(find(selector), selector);
  await React.act(() => find(selector).click());
};
const toolCalls = {
  edit: {
    id: 'edit', name: 'edit', status: 'completed', output: 'OK',
    input: { file_path: 'a.txt', old_string: 'red', new_string: 'green' },
    dshResultMeta: { path: '/work/a.txt', diffs: [
      { path: '/work/a.txt', oldText: 'color = red', newText: 'color = green' },
    ] },
  },
  team: {
    id: 'team', name: 'team_task_get', status: 'completed', input: { id: 1 },
    output: JSON.stringify({ id: 1, subject: '画布校对', description: '核对配色', status: 'completed' }),
  },
};
const message = {
  id: 'turn1', startedAt: 1000, endedAt: 4000,
  parts: [
    { type: 'thought', text: '先检查配色。' },
    { type: 'tool_call', toolCallId: 'edit' },
    { type: 'tool_call', toolCallId: 'team' },
    { type: 'text', text: '**完成。** [查看](./a.txt)\n\n![样图](https://fixture.invalid/example.png)\n\n' },
  ],
  dshTeamBoard: { sessionId: 'team-session', enabled: true, tasks: [
    { id: 'one', revision: 2, subject: '检查布局', description: '核对画布', status: 'completed', ownerName: '设计师' },
  ] },
};
let opened;
const render = (changes = {}, streaming = false) => React.act(() => root.render(jsx(Reply, {
  message: { ...message, ...changes }, toolCalls, streaming, completed: !streaming,
  sessionId: 'lead', language: 'zh', cwd: '/work', openFile: path => { opened = path; }, renderHostPart: () => null,
  media: { render: (src, alt) => jsx('button', { 'data-controlled-image': src, children: alt }) },
})));
try {
  await render({ endedAt: undefined }, true);
  assert.ok(find('[data-controlled-image]'), 'streaming images stay user-controlled');
  await render();
  assert.ok(find('[data-controlled-image]'), 'settled images stay user-controlled');
  const link = [...document.querySelectorAll('button, a')].find(button => button.textContent === '查看');
  await React.act(() => link.click());
  assert.match(opened, /a\.txt$/);
  assert.equal(find('strong').textContent, '完成。');
  assert.equal(find('[data-turn-process]').getAttribute('aria-expanded'), 'false');
  await click('[data-turn-process]');
  assert.ok(find('[data-variant=think]'));
  await click('[data-dsh-tool=edit] [data-disclosure-row]');
  assert.match(find('[data-diff]').textContent, /color = red.*color = green/);
  assert.ok(find('[data-tool=team_task_get]').textContent.includes('画布校对'));
  await click('[data-team-task=one] button');
  assert.match(find('[data-team-task-owner]').textContent, /设计师/);
  await render({ interrupted: true });
  assert.match(find('[data-turn-process]').textContent, /停止/);
  const unfinishedBoard = { ...message.dshTeamBoard,
    tasks: [{ ...message.dshTeamBoard.tasks[0], status: 'in_progress' }] };
  toolCalls.wait = { id: 'wait', name: 'wait_agent', input: {}, status: 'failed', output: 'Error: [object Object]' };
  const waiting = { parts: [{ type: 'tool_call', toolCallId: 'wait' }], dshTeamBoard: unfinishedBoard };
  await render({ ...waiting, dshFailed: true });
  assert.match(find('[data-turn-process]').textContent, /失败/);
  assert.match(find('[data-dsh-task-snapshot] [data-team-panel]').textContent, /1 未完成/);
  assert.match(find('[data-dsh-tool=wait_agent]').textContent, /等待成员回复已中断/);
  assert.doesNotMatch(document.body.textContent, /\[object Object\]/);
  await render({ ...waiting, interrupted: true });
  assert.match(find('[data-dsh-tool=wait_agent]').textContent, /已停止等待成员回复/);
  await render({ ...waiting, endedAt: undefined }, true);
  assert.equal(find('[data-dsh-task-snapshot]'), null);
  assert.match(find('[data-team-panel]').textContent, /1 进行中/);
  const notice = { id: 'msg', memberId: 'child', afterTools: 1, time: 1800,
    kind: 'subagent-settled', stopReason: 'error', text: 'No closing message.', previousAnswer: '已交给成员。', wokeParent: true };
  await render({ dshMemberNotices: [notice] });
  assert.match(find('[data-turn-trigger]').textContent, /子智能体执行失败/);
  await click('[data-turn-trigger] button');
  assert.match(find('[data-turn-trigger]').textContent, /这条通知触发了本轮回复。.*No closing message/s);
  let address;
  window.addEventListener('aibertai:open-dsh-member', e => {address = e.detail}, {once: true});
  globalThis.CustomEvent = window.CustomEvent;
  await click('.dsh-open-notice-member');
  assert.deepEqual(address, {sessionId:'lead', memberId:'child'});
  const liveParts = [...message.parts.slice(0, 2), { type: 'text', text: notice.previousAnswer }, ...message.parts.slice(2)];
  await render({ parts: liveParts, dshMemberNotices: [{ ...notice, liveAfterParts: 3 }] });
  assert.equal(document.body.textContent.split(notice.previousAnswer).length, 2, 'the lead answer appears once during live handoff');
  console.log('Verified native DSH chat, tool diff, team board, file navigation and gated media.');
} finally {
  await React.act(() => root.unmount());
  window.close();
}
