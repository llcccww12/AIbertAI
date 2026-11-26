import { Fragment, useCallback, useEffect, useMemo, useState } from 'react'
import { DisclosureRow, IconContextInjectionOutlineRegular, MarkdownDelegateProvider, MarkdownText } from '@deepseek-ai/dsh-client-ui-primitives'
import { PartialArguments } from 'sdk/util/values/src/partial-json.ts'
import { GenericToolCard } from 'sdk/client/ui-tool/src/client/tool/toolviews/GenericToolCard.tsx'
import { RunningStatus } from 'sdk/client/ui-chat/src/client/chat/RunningStatus.tsx'
import { ReasoningRow } from 'sdk/client/ui-chat/src/client/chat/ReasoningRow.tsx'
import { TurnTriggerNodeView } from 'sdk/client/ui-chat/src/client/chat/TurnTriggerNodeView.tsx'
import { TurnProcessNodeView } from 'sdk/client/ui-chat/src/client/chat/TurnProcessNodeView.tsx'
import { ContextInjectionRow } from 'sdk/client/ui-chat/src/client/chat/ContextInjectionRow.tsx'
import { contextForm, contextProducer } from 'sdk/client/ui-chat/src/client/conversation-nodes/event-projection.ts'
import { TurnNavigator } from 'sdk/client/ui-chat/src/client/chat/TurnNavigator.tsx'
import { markdownLabels } from 'sdk/client/ui-chat/src/client/markdown-labels.ts'
import { zh as chatZh, en as chatEn } from 'sdk/client/ui-chat/src/client/locale.ts'
import { zh as toolsZh, en as toolsEn } from 'sdk/client/ui-conversation/src/client/locales.ts'
import { DetailsRow } from 'sdk/client/ui-tool/src/client/tool/toolviews/details-row.tsx'
import { FileMutationRow } from 'sdk/client/ui-tool/src/client/tool/toolviews/file-mutation-row.tsx'
import { SearchRow } from 'sdk/client/ui-tool/src/client/tool/toolviews/search-row.tsx'
import { WebRow } from 'sdk/client/ui-tool/src/client/tool/toolviews/web-row.tsx'
import { ReadRow } from 'sdk/client/ui-tool/src/client/tool/toolviews/read-row.tsx'
import { OfficeRow } from 'sdk/client/ui-tool/src/client/tool/toolviews/office-row.tsx'
import { BashRow } from 'sdk/client/ui-tool/src/client/tool/toolviews/bash-sample.tsx'
import { resolveWorkspacePath } from 'sdk/util/workspace-path/src/index.ts'
import { installIconSprite } from 'sdk/client/ui-theme/src/client/icon-sprite.ts'
export { PresentedFileCard } from 'sdk/client/ui-deliverables/src/client/PresentedFileCard.tsx'
export { default as deliverableStyles } from 'sdk/client/ui-deliverables/src/client/Deliverables.module.css'
export { zh as deliverableZh, en as deliverableEn } from 'sdk/client/ui-deliverables/src/client/locales.ts'
import { TeamModeToggle } from 'sdk/client/ui-agent-team/src/client/TeamModeToggle.tsx'
import { TeamDisableConfirmation } from 'sdk/client/ui-agent-team/src/client/TeamDisableConfirmation.tsx'
import { TeamBoard } from 'sdk/client/ui-agent-team/src/client/TeamBoard.tsx'
import { zh as teamZh, en as teamEn } from 'sdk/client/ui-agent-team/src/client/locales.ts'
import { TodoPanel } from 'sdk/client/ui-conversation/src/client/skeleton/TodoPanel.tsx'
import { zh as todoZh, en as todoEn } from 'sdk/client/ui-conversation/src/client/locales.ts'
import { MediaContext } from 'employee-media'
import { zh as commonZh } from 'sdk/client/locale/src/locales/zh.ts'
import { en as commonEn } from 'sdk/client/locale/src/locales/en.ts'
import 'employee-theme'

// The sprite's lifetime is the webview document. The employee shell supplies its theme.
if (typeof document !== 'undefined' && !document.querySelector('[data-dsh-icon-sprite]')) {
  installIconSprite({ effect: mount => mount(), on: () => {} }, undefined)
}

const detailsTools = new Set([
  'subagent', 'subagent_fork', 'spawn_teammate', 'list_agents', 'list_subagent_models',
  'send_message', 'interrupt_agent', 'wait_agent', 'team_task_create', 'team_task_get',
  'team_task_update', 'team_task_list', 'create_goal', 'get_goal', 'update_goal',
])

const labels = { zh: { ...commonZh, ...toolsZh, ...chatZh }, en: { ...commonEn, ...toolsEn, ...chatEn } }
const teamLabels = { zh: teamZh, en: teamEn }
// The board sits in the transcript; the separate subagent catalog is unmounted here.
const boardSessions = select => select({ projectionsBySession: {} })
function translate(dictionary) {
  return (key, params) => (dictionary[key] ?? key).replace(/\{(\w+)\}/g, (token, k) => params?.[k] ?? token)
}
export function EmployeeRunningStatus({ startTime, paused, language, waitingForMember = false }) {
  const locale = language?.startsWith('zh') ? 'zh' : 'en'
  const t = useMemo(() => {
    if (!waitingForMember) return translate(labels[locale])
    const label = locale === 'zh' ? '正在等待子智能体回复' : 'Waiting for a team member'
    return translate({ ...labels[locale], 'chat.deepDiving': label,
      'chat.deepDivingFor': label + (locale === 'zh' ? '，本轮已用 {duration}' : ' · {duration} elapsed this turn') })
  }, [locale, waitingForMember])
  return <div className="dsh-native-theme"><RunningStatus startTime={startTime} paused={paused} t={t} /></div>
}

// Employee conversations address root sessions. Team mode comes from the host's
// verified projection, while both controls, their labels and their behavior are DSH's.
export function EmployeeTeamMode({ enabled, locked, hasModelHistory, language, dismiss, requestDisable, save, reportFailure }) {
  const t = translate(teamLabels[language?.startsWith('zh') ? 'zh' : 'en'])
  return <div className="dsh-native-theme"><TeamModeToggle
    useSession={select => select({ subagent: null })}
    useSessions={select => select({ byId: {} })}
    useProjection={key => key === 'agentTeam' ? { enabled } : { lastUsed: hasModelHistory ? true : null }}
    locked={locked} dismiss={dismiss} t={t} selectTeamMode={save}
    requestDisableConfirmation={requestDisable} reportFailure={reportFailure}
  /></div>
}
export function EmployeeTeamConfirmation({ sessionId, language, cancel, save }) {
  return <TeamDisableConfirmation
    useDisableRequest={select => select({ sessionId })} cancelDisable={cancel} disableTeam={() => save(false)}
    t={translate(teamLabels[language?.startsWith('zh') ? 'zh' : 'en'])}
  />
}
// A closed turn keeps the last task snapshot. An unfinished item is no longer
// evidence of live execution, but its recorded status still matters on resume.
function taskLabels(dictionary, locale, terminal) {
  if (!terminal) return translate(dictionary)
  const unfinished = locale === 'zh' ? '未完成' : 'unfinished'
  return translate({ ...dictionary, 'status.in_progress': unfinished, 'todo.status.inProgress': unfinished,
    'progress.active': '{count} ' + unfinished, 'todo.progress.active': '{active} ' + unfinished })
}
export function EmployeeTodoPanel({ todos, language, terminal = false }) {
  if (!todos?.length) return null
  const locale = language?.startsWith('zh') ? 'zh' : 'en'
  const t = taskLabels(locale === 'zh' ? todoZh : todoEn, locale, terminal)
  return <div className="dsh-native-theme" data-dsh-task-snapshot={terminal || undefined} style={{
    '--dsh-composer-side-clearance': '0px', '--dsh-composer-dock-inset': '0px',
    '--dsh-composer-card-max-width': '100%', marginBottom: '8px',
  }}><TodoPanel todos={todos} t={t} /></div>
}

// DSH's turn rail over the employee transcript. The transcript owns its scroll
// area, so the host passes that area's height and the composer sits below it.
export function EmployeeTurnNavigator({ items, activeTurn, onNavigate, viewportHeight, language }) {
  const t = useMemo(() => translate(labels[language?.startsWith('zh') ? 'zh' : 'en']), [language])
  return <div className="dsh-native-theme" data-employee-turn-rail style={{
    display: 'contents', '--dsh-composer-side-clearance': '0px', '--dsh-composer-height': '0px',
    '--dsh-conversation-viewport-height': viewportHeight + 'px',
  }}><TurnNavigator items={items} activeTurn={activeTurn} busyTurn={null} onNavigate={onNavigate} t={t} /></div>
}

// DSH groups a step's injected context under "已注入上下文"; each entry is
// DSH's own context row, built from the source the host recorded.
function ContextInjections({ entries, t }) {
  const [open, setOpen] = useState(false)
  return <DisclosureRow icon={<IconContextInjectionOutlineRegular size={14} />}
    title={t('message.stepProcess.done.context')} open={open} expandable expandOnRowClick
    onToggle={() => setOpen(value => !value)}>
    {entries.map(entry => <ContextInjectionRow key={entry.id}
      content={[{ type: 'text', text: entry.truncated ? entry.text + '\n…' : entry.text }]}
      source={entry.source} producer={contextProducer(entry.source)} form={contextForm(entry.source)} t={t} />)}
  </DisclosureRow>
}

const noMenu = () => undefined
const noContextMenu = () => ({ getTriggerProps: noMenu })
const presentation = selector => selector({ settledReasoningPreview: true })
function useDisclosure() {
  const [expanded, setExpanded] = useState(false)
  return { expanded, toggle: () => setExpanded(value => !value) }
}

// DSH's own keyed toolviews (ui-tool `tool.call.toolview` registrations). The
// SDK falls back to GenericToolCard only for names without a dedicated row, so
// the employee transcript does the same. Rows that need DSH-hosted slots
// (read_image gallery, todo history, question panels) keep the generic card or
// the workbench's own surfaces.
const toolRows = {
  edit: FileMutationRow, write: FileMutationRow, apply_patch: FileMutationRow,
  grep: SearchRow, glob: SearchRow,
  web_search: WebRow, web_fetch: WebRow,
  read: ReadRow,
  office_render: OfficeRow, office_convert: OfficeRow, office_recalculate: OfficeRow,
  bash: BashRow,
}

// Adapter for the persisted and streamed ACP call records. Only the SDK's
// result metadata can describe an applied edit; the native card owns that rule.
function Tool({ call, sessionId, cwd, openFile, t, locale, interrupted }) {
  const name = call.name ?? call.title
  const block = useMemo(() => {
    const input = call.input ?? call.rawInput
    const argsRaw = typeof input === 'string' ? input : JSON.stringify(input ?? {})
    const args = PartialArguments.fromText(argsRaw)
    if (call.status !== 'completed' && call.status !== 'failed') {
      return { phase: 'start', name, argsRaw, args, callId: call.id }
    }
    let output = call.output
    if (name === 'wait_agent' && (call.status === 'failed' || call.isError) && /^Error:\s*\[object Object\]$/.test(output?.trim())) {
      output = interrupted
        ? (locale === 'zh' ? '已停止等待成员回复' : 'Stopped waiting for the member')
        : (locale === 'zh' ? '等待成员回复已中断' : 'The wait for a member reply was interrupted')
    }
    return { kind: 'tool-result', call: { name, argsRaw }, args, callId: call.id,
      content: output ? [{ type: 'text', text: output }] : [],
      meta: call.dshResultMeta, isError: call.status === 'failed' || call.isError === true }
  }, [call, name, locale, interrupted])
  const Row = toolRows[name]
  const resolveFilePath = useCallback(path => resolveWorkspacePath(cwd, path), [cwd])
  const useSessions = useCallback(select => select({ byId: { [sessionId]: { cwd } } }), [sessionId, cwd])
  if (Row) {
    return <Row toolName={name} callId={call.id} phase={'kind' in block ? 'result' : 'start'} block={block}
      cwd={cwd} openFile={openFile} sessionId={sessionId} useSessions={useSessions}
      useContextMenu={noContextMenu} resolveFilePath={resolveFilePath} useDisclosure={useDisclosure} t={t} />
  }
  const Card = detailsTools.has(name) ? DetailsRow : GenericToolCard
  return <Card toolName={name} block={block} cwd={cwd} openFile={openFile}
    getFileTriggerProps={noMenu} useDisclosure={useDisclosure} t={t} />
}

export default function NativeReply({ message, sessionId, toolCalls, streaming, completed, language,
  cwd, openFile, openExternalLink, mentions, media, renderHostPart, renderHostTool, renderToolAccessory, searchSelection, onHistoryOpenChange }) {
  const { parts } = message
  const locale = language?.startsWith('zh') ? 'zh' : 'en'
  const t = useMemo(() => translate(labels[locale]), [locale])
  const markdown = useMemo(() => markdownLabels(t), [t])
  const [expanded, setExpanded] = useState(false)
  // A failed or stopped turn opens by default so the employee can see what
  // happened, but once they fold it away it must stay folded: the default must
  // not keep flipping it back open on every render.
  const [folded, setFolded] = useState<boolean | undefined>(undefined)
  useEffect(() => { if (searchSelection) { setExpanded(true); setFolded(undefined) } }, [searchSelection])
  const interrupted = message.interrupted || message.dshStopAccountingPending
  const reason = interrupted ? 'interrupted'
    : message.dshFailed || parts.some(p => p.type === 'error') ? 'error' : 'completed'
  const closed = completed || message.endedAt !== undefined && !streaming
  const teamT = useMemo(() => taskLabels(teamLabels[locale], locale, closed), [locale, closed])
  const processOpen = folded ?? (!closed || reason !== 'completed' || expanded)
  const lastTool = parts.findLastIndex(part => part.type === 'tool_call')
  const processPart = (part, index) => part.type === 'thought' || part.type === 'tool_call'
    || part.type === 'text' && index < lastTool
  // The SDK can wake the lead during the same platform operation. Keep the
  // notice between the calls that preceded delivery and the ensuing reply.
  const memberNotices = new Map()
  const leadAnswers = new Set<number>()
  const callPositions = parts.flatMap((part, i) => part.type === 'tool_call' ? [i + 1] : [])
  for (const notice of message.dshMemberNotices ?? []) {
    const afterTools = notice.afterTools === 0 ? 0 : callPositions[notice.afterTools - 1] ?? parts.length
    const position = notice.liveAfterParts ?? afterTools
    const prior = parts.slice(afterTools, position)
    const alreadyPresent = notice.previousAnswer && prior.filter(p => p.type === 'text').map(p => p.text).join('') === notice.previousAnswer
    if (alreadyPresent) prior.forEach((part, i) => { if (part.type === 'text') leadAnswers.add(afterTools + i) })
    const rows = memberNotices.get(position) ?? []
    rows.push(<Fragment key={notice.id}>
      {notice.previousAnswer && !alreadyPresent && <MarkdownText text={notice.previousAnswer} labels={markdown} />}
      <TurnTriggerNodeView node={{ data: { time: notice.time, content: [{ type: 'text', text: notice.text }],
        source: { kind: notice.kind, senderSessionId: notice.memberId, stopReason: notice.stopReason } } }}
        t={notice.wokeParent ? t : (key, params) => key === 'message.trigger.explanation'
          ? (locale === 'zh' ? '主智能体已收到这条消息。' : 'The lead agent received this message.') : t(key, params)} />
      {sessionId && <button type="button" className="dsh-open-notice-member"
        onClick={() => window.dispatchEvent(new CustomEvent('aibertai:open-dsh-member', { detail: { sessionId, memberId: notice.memberId } }))}>
        {locale === 'zh' ? '打开成员会话' : 'Open member conversation'}
      </button>}
    </Fragment>)
    memberNotices.set(position, rows)
  }
  const injections = new Map()
  for (const entry of message.dshInjected ?? []) {
    const position = entry.liveAfterParts ?? (entry.afterTools === 0 ? 0 : callPositions[entry.afterTools - 1] ?? parts.length)
    injections.set(position, [...(injections.get(position) ?? []), entry])
  }
  const injectionRows = position => processOpen && injections.has(position)
    ? <ContextInjections key={'context-' + position} entries={injections.get(position)} t={t} /> : null
  const hasProcess = parts.some(processPart) || injections.size > 0
  useEffect(() => { onHistoryOpenChange?.(hasProcess && processOpen)
    return () => onHistoryOpenChange?.(false)
  }, [hasProcess, processOpen, onHistoryOpenChange])

  const turn = { status: closed ? 'closed' : 'open',
    start: message.startedAt === undefined ? undefined : { time: message.startedAt },
    end: closed ? { time: message.endedAt, data: { reason: { kind: reason } } } : undefined }
  function renderPart(part, index) {
    const tail = streaming && index === parts.length - 1
    if (!processOpen && processPart(part, index) && !leadAnswers.has(index)) return null
    if (part.type === 'thought') return <ReasoningRow key={index} text={part.text} running={tail}
      usePresentation={presentation} useDisclosure={useDisclosure} t={t}
      renderSlot={(_, body) => <MarkdownText text={body.text} streaming={body.running} labels={markdown} variant="compact" />} />
    if (part.type === 'text') return <div className="conversation-search-text" key={index}>
      <MarkdownText text={part.text} streaming={tail} labels={markdown} fileMentions={mentions} /></div>
    if (part.type === 'tool_call') {
      const call = toolCalls[part.toolCallId]
      return call ? <Fragment key={part.toolCallId}>
        <div data-dsh-tool={call.name ?? call.title}>{renderHostTool?.(call) ?? <Tool call={call} sessionId={sessionId} cwd={cwd} openFile={openFile} t={t} locale={locale} interrupted={interrupted} />}</div>
        {renderToolAccessory?.(call)}</Fragment> : null
    }
    return renderHostPart(part)
  }
  const partsOfType = kind => parts.filter(part => part.type === kind).length
  return <div className="dsh-native-theme dsh-native-reply">
    <MediaContext.Provider value={media?.render}><MarkdownDelegateProvider openFile={openFile} openExternalLink={openExternalLink}>
      {closed && <TurnProcessNodeView
        node={{ location: { kind: 'turn', turn }, data: { turn: message.id,
          messageCount: partsOfType('text'), toolCallCount: partsOfType('tool_call'),
          subagentCount: parts.filter(p => toolCalls[p.toolCallId]?.name?.startsWith('subagent')).length } }}
        turnProcess={{ foldable: true, hasContent: hasProcess, open: processOpen, setOpen: setFolded }} t={t} />}
      {parts.map((part, index) => <Fragment key={part.toolCallId ?? index}>
        {injectionRows(index)}{memberNotices.get(index)}{renderPart(part, index)}
      </Fragment>)}
      {injectionRows(parts.length)}
      {memberNotices.get(parts.length)}
    {message.dshTeamBoard && <div data-dsh-task-snapshot={closed || undefined}><TeamBoard sessionId={message.dshTeamBoard.sessionId}
        useProjection={() => message.dshTeamBoard} useSessions={boardSessions} t={teamT} /></div>}
    </MarkdownDelegateProvider></MediaContext.Provider>
  </div>
}

export { EmployeeOpenTarget } from './file-open-target.tsx'
export { EmployeeDocument } from './file-document.tsx'
