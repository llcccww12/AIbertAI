import { useSyncExternalStore } from 'react'

// Transcript rows and the header mount separately. Share only the current, SDK-verified catalog.
const scopes = new Map<string, Map<string, string>>()
const listeners = new Set<() => void>()
const subscribe = (listener: () => void) => { listeners.add(listener); return () => listeners.delete(listener) }
const memberScope = (accountId, sessionId) => JSON.stringify([accountId, sessionId])
export function publishMemberLinks(accountId, sessionId, members) {
  const scope = memberScope(accountId, sessionId)
  const links = new Map<string, string>()
  const seen = new Set<string>()
  for (const { parentId, creationCallId, id, mode } of members) {
    if (parentId !== sessionId || mode === 'external' || !creationCallId || !id) continue
    if (seen.has(creationCallId)) links.delete(creationCallId)
    else { links.set(creationCallId, id); seen.add(creationCallId) }
  }
  const old = scopes.get(scope)
  if ((old?.size || 0) === links.size && [...links].every(([call, id]) => old?.get(call) === id)) return
  if (links.size) scopes.set(scope, links)
  else scopes.delete(scope)
  for (const listener of listeners) listener()
}
export function MemberHistoryLink({ accountId, sessionId, callId, language }) {
  const scope = memberScope(accountId, sessionId)
  const memberId = useSyncExternalStore(subscribe, () => scopes.get(scope)?.get(callId))
  if (!memberId) return null
  return <button type="button" data-member-history-link="" className="dsh-member-history-link"
    onClick={() => window.dispatchEvent(new CustomEvent('aibertai:open-dsh-member', { detail: { sessionId, memberId } }))}>
    {language?.startsWith('zh') ? '打开成员会话' : 'Open member chat'}
  </button>
}
