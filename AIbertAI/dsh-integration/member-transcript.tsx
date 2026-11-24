// The archive is paged in event order. Pair only SDK-issued call IDs within their turn.
import { lazy, memo, Suspense, useMemo, useState } from 'react'
import { DisclosureRow } from 'sdk/client/ui-primitives/src/DisclosureRow.tsx'
import { UserMessageBubble } from 'sdk/client/ui-primitives/src/UserMessageBubble.tsx'
import { IconSparkleRegular } from 'sdk/client/ui-primitives/src/icons/index.tsx'
import messageStyle from 'sdk/client/ui-chat/src/client/chat/MessageItem.module.css'

const loadChat = () => import(new URL('../native-chat/reply.js', import.meta.url).href)
const Reply = lazy(loadChat)
const RunningStatus = lazy(async () => ({ default: (await loadChat()).EmployeeRunningStatus }))
const noTools = {}
// Member archives do not grant read access to paths or images in the root workspace.
const media = { render: (_destination, alt) => alt || null }

type Entry = { seq: number, type: string, text: string, time?: number, turn?: number, step?: number,
  source?: string, requestId?: string, tool?: string, callId?: string, failed?: boolean, errorCode?: string, abortReason?: string,
  truncated?: boolean, textLength?: number, attachments?: { type: string, name?: string, bytes?: number }[] }
type Item = { entry: Entry, result?: Entry }

function projectEntries(rows: Entry[]) {
  const items: Item[] = []
  const calls = new Map<string, Item>()
  let start: Entry | undefined
  for (const entry of rows) {
    if (entry.type === 'turn/start') { calls.clear(); start = entry; continue }
    if (entry.type === 'turn/end') {
      calls.clear()
      if (start?.turn === entry.turn) start = undefined
      if (entry.text === 'completed') continue
    }
    const key = entry.callId && JSON.stringify([entry.turn, entry.callId])
    if (key && entry.type === 'tool/result' && calls.has(key)) {
      calls.get(key)!.result = entry
      calls.delete(key)
      continue
    }
    const item = { entry }
    items.push(item)
    if (key && entry.type === 'tool/call') calls.set(key, item)
  }
  return { items, start }
}

function Cutoff({ entry, zh }) {
  if (!entry.truncated) return null
  return <small data-member-truncation>{entry.textLength && zh ? `本条内容已截断，原文共 ${entry.textLength} 个字符`
    : entry.textLength ? `Preview truncated from ${entry.textLength} characters`
      : zh ? '本条内容已截断' : 'Preview truncated'}</small>
}

function ToolEvent({ entry, result, zh }: { entry: Entry, result?: Entry, zh: boolean }) {
  const [open, setOpen] = useState(false)
  const call = entry.type === 'tool/call' ? entry : undefined
  const output = call ? result : entry
  const name = call?.tool || output?.tool
  const failed = output?.failed === true
  const title = failed ? (zh ? '工具执行失败' : 'Tool failed') : !call ? (zh ? '工具结果' : 'Tool result')
    : output ? (zh ? '已调用工具' : 'Called tool') : (zh ? '调用工具' : 'Tool call')
  return <div data-member-event={entry.type} data-call-id={entry.callId} data-failed={failed || undefined}>
    <DisclosureRow icon={<IconSparkleRegular size={14} />} title={name ? title + ' · ' + name : title}
      expandable expandOnRowClick open={open} onToggle={() => setOpen(value => !value)}>
      <div className="dsh-member-tool-details">
        {call && <section aria-label={zh ? '调用参数' : 'Arguments'}><small>{zh ? '调用参数' : 'Arguments'}</small>
          {call.text && <pre>{call.text}</pre>}<Cutoff entry={call} zh={zh} /></section>}
        {output && <section aria-label={zh ? '工具结果' : 'Tool result'}><small>{failed ? (zh ? '错误' : 'Error') : (zh ? '工具结果' : 'Result')}</small>
          {output.text && <pre>{output.text}</pre>}
          {output.attachments?.map((file, index) => <div key={index}>{file.name ||
            (file.type === 'image' ? (zh ? '图片' : 'Image') : (zh ? '文件' : 'File'))}{file.bytes !== undefined && ` · ${file.bytes} B`}</div>)}
          <Cutoff entry={output} zh={zh} /></section>}
      </div>
    </DisclosureRow>
  </div>
}

const MessageEvent = memo(function MessageEvent({ entry, language, zh }: { entry: Entry, language: string, zh: boolean }) {
  if (entry.type === 'user/message') {
    const relay = entry.source === 'agent-message'
    const label = relay ? (zh ? '团队消息' : 'Team message') : entry.requestId ? (zh ? '你的消息' : 'Your message') : (zh ? '任务' : 'Task')
    return <div data-member-event={entry.type} data-member-request-id={entry.requestId} className={messageStyle.userRow} aria-label={label}>
      <div className={messageStyle.userStack}>
        {relay && <small>{label}</small>}
        <UserMessageBubble data-message-bubble="">{entry.text}</UserMessageBubble><Cutoff entry={entry} zh={zh} />
      </div>
    </div>
  }
  if (entry.type === 'assistant/message') return <div data-member-event={entry.type}>
    <Suspense fallback={<div className="dsh-member-text">{entry.text}</div>}>
      <Reply message={{ id: entry.seq, parts: [{ type: 'text', text: entry.text }] }}
        toolCalls={noTools} language={language} media={media} />
    </Suspense><Cutoff entry={entry} zh={zh} />
  </div>
  if (entry.type !== 'turn/end') return null
  const label = {
    aborted: zh ? '已停止' : 'Stopped',
    interrupted: zh ? '本轮执行中断' : 'Turn interrupted',
    'max-tokens': zh ? '已达到模型输出上限' : 'Model output limit reached',
    blocked: zh ? '本轮执行受阻' : 'Turn blocked',
    error: zh ? '处理失败' : 'Run failed',
    forked: zh ? '已从本轮派生新会话' : 'Turn forked',
  }[entry.text] || (zh ? '本轮结束' : 'Turn ended')
  const cause = entry.text === 'aborted' && {
    user: zh ? '你停止了这个成员。' : 'You stopped this member.',
    parent: zh ? '主会话停止了这个成员。' : 'The parent chat stopped this member.',
  }[entry.abortReason || '']
  return <div data-member-event={entry.type} data-outcome={entry.text} data-failed={entry.text === 'error' || undefined}
    role={entry.text === 'error' ? 'alert' : 'status'}>
    <span>{label}</span>{entry.errorCode && ' · ' + entry.errorCode}{cause && <span> {cause}</span>}
  </div>
})

export function MemberTranscript({ rows, language, running }) {
  const { items, start } = useMemo(() => projectEntries(rows), [rows])
  const zh = language?.startsWith('zh')
  return <>
    {items.map(({ entry, result }) => entry.type === 'tool/call' || entry.type === 'tool/result'
      ? <ToolEvent key={entry.seq} entry={entry} result={result} zh={zh} />
      : <MessageEvent key={entry.seq} entry={entry} language={language} zh={zh} />)}
    {running && start && <Suspense fallback={null}>
      <RunningStatus startTime={start.time} language={language} paused={false} />
    </Suspense>}
  </>
}
