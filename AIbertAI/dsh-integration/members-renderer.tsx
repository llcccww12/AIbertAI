// The DSH member catalog uses DSH child IDs. Employee ACP authorizes every read
// using the owning root session, including reads from a closed SDK's archive.
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { SubagentCatalogAction, SubagentHeaderLineage } from 'sdk/client/ui-subagent/src/client/SubagentHeaderLineage.tsx'
import { zh, en } from 'sdk/client/ui-subagent/src/client/locales.ts'
import { MemberTranscript } from './member-transcript.tsx'
import { MemberComposer } from './member-composer.tsx'
import { publishMemberLinks } from './member-history-link.tsx'
export { MemberHistoryLink } from './member-history-link.tsx'
import './members.css'

const empty = []
const status = new Map()
function project(sessionId, members) {
  const children = new Map([[sessionId, []]])
  const byId = {}
  for (const member of members) {
    if (!children.has(member.parentId)) children.set(member.parentId, [])
    children.get(member.parentId).push(member)
    byId[member.id] = { origin: 'subagent', parentId: member.parentId,
      title: member.label || member.name, running: member.activity === 'running' }
  }
  const projectionsBySession = Object.fromEntries(Array.from(children, ([id, entries]) =>
    [id, { state: 'ready', values: { subagentCatalog: entries } }]))
  return { byId, projectionsBySession }
}

// The catalog answers with internal codes; say what they mean instead.
function memberErrorText(message: string, zh: boolean) {
  const text = String(message)
  if (text.includes('dsh_members_starting')) return zh ? '成员目录正在启动，请稍候。' : 'The agent catalog is starting. Try again shortly.'
  if (text.includes('dsh_runtime_exited')) return zh ? '运行时已退出，正在重启。' : 'The runtime exited and is restarting.'
  if (text.includes('dsh_member_unavailable')) return zh ? '这个成员当前不可用。' : 'This agent is not available right now.'
  if (text.includes('invalid_dsh_members_response')) return zh ? '成员目录返回了无法识别的数据。' : 'The agent catalog returned something unexpected.'
  return zh ? '读取成员目录失败，请重试。' : 'Could not load the agent catalog. Try again.'
}

export default function EmployeeMembers({ accountId, sessionId, request, promptMember, interruptMember, language, running = false, rootTitle, headerTarget }) {
  const storageKey = 'dsh.member.view.' + JSON.stringify([accountId, sessionId])
  const [selected, setSelected] = useState(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(storageKey) || 'null')
      return typeof saved?.id === 'string' ? saved : null
    } catch { return null }
  })
  const [result, setResult] = useState({ members: empty, entries: empty, memberId: null, loaded: false, caughtUp: false, live: false, error: '' })
  const [revision, refresh] = useState(0)
  const [watchSubmission, setWatchSubmission] = useState(false)
  const anchor = useRef<HTMLSpanElement>(null)
  const pane = useRef<HTMLElement>(null)
  const transcript = useRef<HTMLDivElement>(null)
  const history = useRef({ memberId: null, from: undefined, entries: [] })
  const follow = useRef(true)
  const zhCN = language?.startsWith('zh')
  const dict = zhCN ? zh : en
  const t = (key, params) => (dict[key] ?? key).replace(/\{(\w+)\}/g, (token, name) => params?.[name] ?? token)
  const select = useCallback((id, aside = false) => {
    const value = id ? { id, aside } : null
    try {
      if (value) localStorage.setItem(storageKey, JSON.stringify(value))
      else localStorage.removeItem(storageKey)
    } catch {}
    setSelected(value)
  }, [storageKey])
  const memberId = selected?.id ?? null
  useEffect(() => {
    let active = true
    let timer
    if (history.current.memberId !== memberId) {
      history.current = { memberId, from: undefined, entries: [] }
      follow.current = true
    }
    async function read() {
      let delay = 3000
      try {
        const data = await request(memberId ? { sessionId, memberId, from: history.current.from } : { sessionId })
        if (!active) return
        const batch = data.entries ?? empty
        const entries = batch.length ? history.current.entries.concat(batch) : history.current.entries
        history.current = { memberId, from: data.next, entries }
        setResult({ members: data.members, entries, memberId, loaded: true, caughtUp: !data.more, live: data.live, error: '' })
        if (!data.more && !watchSubmission && !running && !(data.live && data.members.some(member => member.activity === 'running'))) return
        delay = data.more ? 0 : 1200
      } catch (err) {
        if (!active) return
        const message = String(err?.message || err)
        setResult(old => ({ ...old, memberId, entries: history.current.entries, error: message }))
        if (memberId && message.includes('dsh_member_unavailable')) { select(null); return }
        if (!running && !watchSubmission) return
        delay = 1500
      }
      timer = setTimeout(read, delay)
    }
    void read()
    return () => { active = false; clearTimeout(timer) }
  }, [request, sessionId, memberId, revision, running, watchSubmission, select])
  useEffect(() => {
    publishMemberLinks(accountId, sessionId, result.members)
  }, [accountId, sessionId, result.members])
  useEffect(() => () => publishMemberLinks(accountId, sessionId, empty), [accountId, sessionId])
  useEffect(() => {
    let active = true
    async function open(event) {
      const detail = event.detail
      if (detail?.sessionId !== sessionId || typeof detail.memberId !== 'string') return
      let rows = result.members
      if (!rows.some(member => member.id === detail.memberId)) {
        try { rows = (await request({ sessionId })).members } catch { return }
      }
      if (active && rows.some(member => member.id === detail.memberId && member.mode !== 'external')) select(detail.memberId)
    }
    window.addEventListener('aibertai:open-dsh-member', open)
    return () => { active = false; window.removeEventListener('aibertai:open-dsh-member', open) }
  }, [request, sessionId, result.members, select])
  const sessions = useMemo(() => project(sessionId, result.members), [sessionId, result.members])
  const member = result.members.find(row => row.id === memberId)
  const title = member?.label || member?.name || (zhCN ? '成员进度' : 'Agent progress')
  const back = zhCN ? '返回主会话' : 'Return to main chat'
  const refreshCatalog = useCallback(async () => { refresh(n => n + 1) }, [])
  const rootCatalog = {
    sessionId, useSessions: selector => selector(sessions), useSession: selector => selector({ subagent: null }),
    useSessionStatus: selector => selector(status), useProjection: (_, selector) => selector(undefined),
    useFooterAvailable: selector => selector(false), renderSlot: () => null,
    prepareCatalog: async () => {}, refreshProjection: refreshCatalog,
    openChild: address => select(address.childSessionId, false),
    openChildAside: address => select(address.childSessionId, true), t,
  }
  const heading = anchor.current?.closest('.chat-page')
  const rows = result.memberId === memberId ? result.entries : empty
  useEffect(() => { if (memberId) pane.current?.focus({ preventScroll: true }) }, [memberId, heading])
  useEffect(() => {
    const box = transcript.current
    if (box && follow.current) box.scrollTop = box.scrollHeight
  }, [memberId, rows, heading])
  const error = result.memberId === memberId && result.error
  const breadcrumb = <nav aria-label={zhCN ? '会话导航' : 'Conversation navigation'}
    className="dsh-member-breadcrumb dsh-native-theme" data-member-breadcrumb=""
    data-dsh-member-lineage={selected?.aside ? undefined : ''}>
    <button type="button" data-member-parent="" title={back} onClick={() => select(null)}>{rootTitle || back}</button>
    <span aria-hidden="true">/</span>
    {member ? <SubagentHeaderLineage {...rootCatalog} lineageSessionId={memberId} displayTitle={title}
      useSession={selector => selector({ subagent: { address: { parentSessionId: member.parentId, childSessionId: memberId } } })} />
      : <span>{title}</span>}
  </nav>
  return <span ref={anchor} className="dsh-native-theme" data-dsh-members>
    {(!selected || selected.aside) && <SubagentCatalogAction {...rootCatalog} />}
    {!selected && !result.members.length && result.error && !running && !watchSubmission && <button type="button" onClick={refreshCatalog}
      title={result.error}>{zhCN ? '重试读取成员' : 'Retry loading agents'}</button>}
    {selected && heading && createPortal(<section ref={pane} tabIndex={-1} aria-label={title}
      className="dsh-member-view dsh-native-theme" data-aside={selected.aside || undefined}>
      {selected.aside || !headerTarget ? <header>{breadcrumb}</header> : createPortal(breadcrumb, headerTarget)}
      <div className="dsh-member-transcript" ref={transcript} onClickCapture={event => {
        if (event.target instanceof Element && event.target.closest('[data-disclosure-row]')) follow.current = false
      }} onKeyDownCapture={event => {
        if ((event.key === 'Enter' || event.key === ' ') && event.target instanceof Element && event.target.closest('[data-disclosure-row]')) follow.current = false
      }} onScroll={event => {
        const box = event.currentTarget
        follow.current = box.scrollHeight - box.scrollTop - box.clientHeight < 40
      }}>
        <div className="dsh-member-messages" key={memberId}>
          {error && <div role="alert">{memberErrorText(error, zhCN)} <button type="button" onClick={refreshCatalog}>{zhCN ? '重试' : 'Retry'}</button></div>}
          {!error && !rows.length && <p role="status">{result.loaded && result.memberId === memberId
            ? (zhCN ? '暂无执行记录' : 'No activity yet') : (zhCN ? '正在读取执行记录…' : 'Loading activity…')}</p>}
          <MemberTranscript rows={rows} language={language} running={result.caughtUp && result.live && member?.activity === 'running'} />
        </div>
      </div>
      {member?.mode === 'continuable' && promptMember && <MemberComposer key={memberId}
        storageKey={'dsh.member.draft.' + JSON.stringify([accountId, sessionId, memberId])}
        sessionId={sessionId} member={member} rows={rows} promptMember={promptMember}
        running={result.caughtUp && result.live && member.activity === 'running'
          && rows.findLast(row => row.type === 'turn/start' || row.type === 'turn/end')?.type === 'turn/start'}
        interruptMember={interruptMember} onWatch={setWatchSubmission} refresh={refreshCatalog} zh={zhCN} />}
    </section>, heading)}
  </span>
}
