// The host acknowledges queue admission; the DSH transcript owns displayed user messages.
import { useEffect, useMemo, useRef, useState } from 'react'
import inputStyles from 'sdk/client/ui-conversation/src/client/skeleton/InputBar.module.css'

type Draft = { text: string, requestId?: string, awaiting: string[] }
const blank = (): Draft => ({ text: '', awaiting: [] })

function readDraft(key: string): Draft {
  try {
    const saved = JSON.parse(localStorage.getItem(key) || 'null')
    if (typeof saved?.text === 'string' && Array.isArray(saved.awaiting)) return saved
  } catch {}
  return blank()
}

function saveDraft(key: string, draft: Draft) {
  try {
    if (draft.text || draft.awaiting.length) localStorage.setItem(key, JSON.stringify(draft))
    else localStorage.removeItem(key)
  } catch {}
}

export function MemberComposer({ storageKey, sessionId, member, rows, running, promptMember, interruptMember, onWatch, refresh, zh }) {
  const [draft, setDraft] = useState<Draft>(() => readDraft(storageKey))
  const current = useRef(draft)
  const busy = useRef(false)
  const mounted = useRef(false)
  const [sending, setSending] = useState(false)
  const [error, setError] = useState('')
  useEffect(() => { mounted.current = true; return () => { mounted.current = false } }, [])
  const update = (value: Draft) => {
    current.current = value
    saveDraft(storageKey, value)
    if (mounted.current) setDraft(value)
  }
  const received = useMemo(() => new Map(rows.filter(row => row.type === 'user/message' && row.requestId)
    .map(row => [row.requestId, row.seq])), [rows])
  const ended = rows.findLast(row => row.type === 'turn/end')?.seq ?? -1
  useEffect(() => {
    const delivered = draft.requestId && received.has(draft.requestId)
    const pending = delivered ? [...draft.awaiting, draft.requestId] : draft.awaiting
    const awaiting = pending.filter(id => !received.has(id) || received.get(id) >= ended)
    if (delivered || awaiting.length !== draft.awaiting.length) {
      update(delivered ? { text: '', awaiting } : { ...draft, awaiting })
      if (delivered) setError('')
    }
  }, [received, ended, draft])
  const watching = draft.awaiting.length > 0
  useEffect(() => {
    onWatch(watching)
    return () => onWatch(false)
  }, [watching, onWatch])
  async function submit() {
    const message = current.current
    if (busy.current || !message.text.trim()) return
    if (new TextEncoder().encode(message.text).length > 16384) {
      setError(zh ? '消息过长，请缩短后发送。' : 'The message is too long. Shorten it and try again.')
      return
    }
    const requestId = message.requestId || crypto.randomUUID()
    const text = message.text
    busy.current = true
    setSending(true)
    setError('')
    update({ ...message, requestId })
    try {
      const receipt = await promptMember({ sessionId, memberId: member.id, requestId, text,
        clientTimeZone: Intl.DateTimeFormat().resolvedOptions().timeZone })
      if (!receipt?.accepted || receipt.requestId !== requestId) throw new Error('invalid_member_receipt')
      const latest = mounted.current ? current.current : readDraft(storageKey)
      if (latest.requestId === requestId) update({ text: '', awaiting: [...latest.awaiting, requestId] })
    } catch (fault) {
      if (mounted.current) {
        const why = String(fault?.message || fault)
        setError(why.includes('dsh_reconciliation_required') ? (zh ? '上次调用待核验，请在主会话查看。' : 'The previous call is awaiting verification in the parent chat.')
          : why.includes('member_unavailable') ? (zh ? '这个成员暂时无法接收消息。' : 'This member cannot receive messages right now.')
          : (zh ? '发送失败。请重试。' : 'Could not send. Try again.'))
      }
    } finally {
      busy.current = false
      if (mounted.current) { setSending(false); refresh() }
    }
  }
  const stop = !!interruptMember && running && !draft.text.trim()
  async function interrupt() {
    if (busy.current || !stop) return
    busy.current = true
    setSending(true)
    setError('')
    try {
      // The host answers "not accepted" without raising when that turn already
      // ended or the runtime is gone; say so instead of doing nothing.
      const receipt = await interruptMember({ sessionId, memberId: member.id })
      if (receipt && receipt.accepted === false) throw new Error('member_not_running')
    } catch {
      if (mounted.current) setError(zh ? '未能停止成员，请重试。' : 'Could not stop the member. Try again.')
    } finally {
      busy.current = false
      if (mounted.current) { setSending(false); refresh() }
    }
  }
  const title = member.label || member.name
  const label = zh ? '发消息给' + title : 'Message ' + title
  return <form data-member-composer className={inputStyles.card} onSubmit={event => { event.preventDefault(); void submit() }}>
    <textarea className={inputStyles.input} aria-label={label} placeholder={label} rows={2}
      value={draft.text} disabled={sending} onChange={event => {
        setError('')
        update({ text: event.target.value, awaiting: current.current.awaiting })
      }} onKeyDown={event => {
        if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing && event.nativeEvent.keyCode !== 229) {
          event.preventDefault()
          void submit()
        }
      }} />
    <div className="dsh-member-send-row">
      {error ? <span role="alert">{error}</span>
        : draft.awaiting.some(id => !received.has(id)) && <span role="status">{zh ? '已提交，等待成员接收' : 'Submitted, waiting for the member'}</span>}
      <button type={stop ? 'button' : 'submit'} className={inputStyles.primary} data-action={stop ? 'stop' : 'send'}
        aria-label={stop ? (zh ? '停止成员' : 'Stop member') : error ? (zh ? '重试发送' : 'Retry sending') : (zh ? '发送' : 'Send')}
        disabled={sending || (!stop && !draft.text.trim())} onClick={stop ? interrupt : undefined}>
        {stop ? <svg viewBox="0 0 16 16" width="16" height="16" aria-hidden="true"><rect x="3" y="3" width="10" height="10" rx="2" fill="currentColor"/></svg>
          : <svg viewBox="0 0 16 16" width="16" height="16" aria-hidden="true"><path d="M8 14V2M2.5 7.5 8 2l5.5 5.5" fill="none" stroke="currentColor" strokeWidth="2"/></svg>}
      </button>
    </div>
  </form>
}
