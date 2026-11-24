import { test } from 'node:test';
import { strict as assert } from 'node:assert';
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';
import { dshToolActions, dshStructuredAction, isDSHTool, displayOtherApproval, displayDSHStructuredOutput, displayReadOffset, dshErrorKey, displayDSHRead, displayDSHWrite, normalizeDSHToolCalls } from '../../AIbertAI/code/web/dsh/tool-actions.mjs';

test('native background job actions have readable labels', () => {
  assert.equal(dshStructuredAction('job_list'), 'jobList');
  assert.equal(dshStructuredAction('job_output'), 'jobOutput');
  assert.equal(dshStructuredAction('job_kill'), 'jobKill');
  assert.equal(displayDSHStructuredOutput('synthetic-web-job\n[status: running]'), 'synthetic-web-job\n[status: running]');
});

test('native web, skill, and subagent names use existing readable actions', () => {
  assert.equal(dshToolActions.web_search, 'webSearch');
  assert.equal(dshToolActions.web_fetch, 'webFetch');
  assert.equal(dshToolActions.subagent, 'agent');
  assert.equal(dshToolActions.skill, 'skill');
});

test('DSH file calls use the existing readable workbench actions', () => {
  assert.deepEqual([dshToolActions.read, dshToolActions.write, dshToolActions.edit], ['read', 'write', 'edit']);
  assert.equal(displayReadOffset('read', 1), 1);
  assert.equal(displayReadOffset('Read', 0), 1);
  assert.equal(dshErrorKey('dsh_write_outcome_unknown'), 'errors.codes.dsh_write_outcome_unknown');
  assert.equal(dshErrorKey('dsh_write_outcome_unknown: details'), 'errors.codes.dsh_write_outcome_unknown');
  assert.equal(dshErrorKey('session_turn_failed'), 'errors.codes.session_turn_failed');
  assert.equal(dshErrorKey('model_vision_unavailable'), 'errors.codes.model_vision_unavailable');
  assert.equal(dshErrorKey('attachment_image_invalid'), 'errors.codes.attachment_image_invalid');
  assert.equal(dshErrorKey('attachment_unavailable_or_too_large'), 'errors.codes.attachment_unavailable_or_too_large');
  assert.equal(dshErrorKey('owned_model_index_unavailable'), 'errors.codes.owned_model_index_unavailable');
  assert.equal(dshErrorKey('dsh_model_request_too_large'), 'errors.codes.dsh_model_request_too_large');
  assert.equal(dshErrorKey('dsh_compaction_unverified'), 'errors.codes.dsh_compaction_unverified');
  assert.equal(dshErrorKey('compaction_result_unverified'), 'errors.codes.dsh_compaction_unverified');
  assert.equal(dshErrorKey('dsh_compaction_usage_invalid'), 'errors.codes.dsh_compaction_usage_invalid');
  assert.equal(dshErrorKey('unrelated'), undefined);
  const raw = '<path>/a.txt</path>\n<type>file</type>\n<content>\n1: hello\n\n(End of file - total 1 lines)\n</content>';
  assert.equal(displayDSHRead(raw), '1: hello\n\n(End of file - total 1 lines)');
  assert.equal(displayDSHRead('other output'), 'other output');
  const calls = { a: { name: 'read', rawInput: { file_path: 'brief.txt' } }, b: { name: 'Read', input: { path: 'original' }, rawInput: { path: 'ignored' } } };
  assert.deepEqual(normalizeDSHToolCalls(calls).a.input, { file_path: 'brief.txt' });
  assert.deepEqual(normalizeDSHToolCalls(calls).b, calls.b);
  assert.deepEqual(normalizeDSHToolCalls({ custom: { name: 'custom_native_tool', rawInput: { query: '保留参数' }, status: 'failed' } }).custom.input, { query: '保留参数' });
  assert.deepEqual(normalizeDSHToolCalls({ custom: { name: 'custom_native_tool', input: { query: '现有参数' }, rawInput: { query: '旧参数' } } }).custom.input, { query: '现有参数' });
});

test('write result removes only the known SDK success envelope', () => {
  const raw = '<path>/workspace/result.txt</path>\n<type>file</type>\n<content>\nCreated file\n</content>';
  assert.equal(displayDSHWrite(raw), '文件已创建');
  for (const value of [
    '普通正文 <path>hello</path>',
    '&lt;path&gt;/workspace/result.txt&lt;/path&gt;',
    raw.replace('Created file', 'Unexpected <error>'),
    raw + '\nextra',
    raw.replace('<type>file</type>', '<type>directory</type>'),
  ]) assert.equal(displayDSHWrite(value), value);
});

test('structured DSH tools keep readable results and diagnostic inputs', () => {
  const connector = 'connector_0123456789abcdef0123456789abcdef';
  assert.equal(dshStructuredAction('knowledge_search'), 'knowledgeSearch');
  assert.equal(dshStructuredAction('canvas_create_text'), 'canvasCreateText');
  assert.equal(dshStructuredAction(connector), 'connectorCall');
  assert.equal(isDSHTool('unrelated_tool'), false);
  assert.deepEqual(normalizeDSHToolCalls({ a: { name: 'knowledge_search', rawInput: { query: '设计规范' } } }).a.input, { query: '设计规范' });
  const result = JSON.stringify({ items: [{ content: '资料内容', citation: { heading_path: '章节' } }] });
  assert.equal(displayDSHStructuredOutput(result), '资料内容\n\n章节');
  assert.equal(displayDSHStructuredOutput(JSON.stringify({ output: JSON.stringify({ content: [{ type: 'text', text: '连接器结果' }] }) })), '连接器结果\n\ntype: text');
  assert.equal(displayDSHStructuredOutput('{"opaque_id":"secret"}'), 'opaque_id: secret');
  assert.equal(displayDSHStructuredOutput('{"error":{"message":"无权访问"}}'), '无权访问');
  assert.equal(displayDSHStructuredOutput('普通文本'), '普通文本');
  const detailed = JSON.stringify({ items: [{ text: '结果', score: 0.87, metadata: { matched: true }, count: 0 }], error: { message: '无权', code: 403 } });
  assert.equal(displayDSHStructuredOutput(detailed), '结果\n\nscore: 0.87\n\nmetadata.matched: true\n\ncount: 0\n\n无权\n\ncode: 403');
  assert.equal(displayDSHStructuredOutput('{"items":[{"text":"相同"},{"text":"相同"}]}'), '相同\n\n相同');
  assert.equal(displayDSHStructuredOutput('{"count":0,"available":false,"extra":null,"items_extra":[],"metadata":{}}'), 'count: 0\n\navailable: false\n\nextra: null\n\nitems_extra: []\n\nmetadata: {}');
});

test('unrecognized tool approval retains its reason alongside the tool name', () => {
  assert.equal(displayOtherApproval('custom_tool', '需要访问测试资源', 'custom_tool', '工具调用'), 'custom_tool · 需要访问测试资源');
  assert.equal(displayOtherApproval('custom_tool', 'custom_tool', 'custom_tool', '工具调用'), 'custom_tool');
  assert.equal(displayOtherApproval('', '', '', '工具调用'), '工具调用');
  assert.equal(displayOtherApproval('bash', '需要运行测试命令', '运行命令', '工具调用'), '运行命令 · 需要运行测试命令');
  assert.equal(displayOtherApproval('bash', '需要运行测试命令', '运行命令 · 需要运行测试命令', '工具调用'), '运行命令 · 需要运行测试命令');
  assert.equal(displayOtherApproval('bash', '需要运行测试命令', undefined, '工具调用'), '需要运行测试命令');
});

test('connector summaries retain the trusted display title and requested object', async () => {
  const bundle = await readFile(new URL('../../AIbertAI/code/web/assets/index-zDHqw1Zn.js', import.meta.url), 'utf8');
  const code = bundle.match(/function ih\(e, t = "", n, r\) \{[\s\S]*?\n\}\nfunction O9t/)?.[0].replace(/\nfunction O9t$/, '');
  assert.ok(code);
  const context = vm.createContext({
    dshStructuredAction,
    ehe: {},
    rf: value => value && typeof value === 'object' ? value : {},
    Bo: (value, ...keys) => keys.map(key => value[key]).find(item => typeof item === 'string' && item) || '',
  });
  const summary = vm.runInContext(`(${code})`, context);
  const alias = 'connector_0123456789abcdef0123456789abcdef';
  assert.equal(summary({ name: alias, title: '模拟连接器 / 查询记录', input: { name: '示例记录' } }).summary, '模拟连接器 / 查询记录 · 示例记录');
  assert.equal(summary({ name: alias, title: alias, input: { name: '示例记录' } }).summary, '示例记录');
});
