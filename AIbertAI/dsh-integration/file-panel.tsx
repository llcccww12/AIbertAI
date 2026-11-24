import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { Menu } from 'sdk/client/ui-primitives/src/Menu.tsx'
import { IconChevronDownOutlineRegular, IconEyeOffOutlineRegular, IconEyeOutlineRegular } from 'sdk/client/ui-primitives/src/icons/index.tsx'
import { FilesBody } from 'sdk/client/ui-sidebar-files/src/client/FilesBody.tsx'
import { FileTree } from 'sdk/client/ui-sidebar-files/src/client/FileTree.tsx'
import { en, zh } from 'sdk/client/ui-sidebar-files/src/client/locales.ts'
import { en as previewEn, zh as previewZh } from 'sdk/client/ui-sidebar-documentpreview/src/client/locales.ts'
import previewCss from 'sdk/client/ui-sidebar-documentpreview/src/client/TextPreview.module.css'
import { EmployeeViewerTools } from './file-viewer.tsx'
export { EmployeeImage, EmployeeSpreadsheet } from './file-media.tsx'
export { EmployeeHtml } from './file-viewer.tsx'
import { PreviewFrame } from 'sdk/client/ui-sidebar-documentpreview/src/client/PreviewFrame.tsx'
import { TextBody } from 'sdk/client/ui-sidebar-documentpreview/src/client/text/TextBody.tsx'
import { installIconSprite } from 'sdk/client/ui-theme/src/client/icon-sprite.ts'
import styles from 'sdk/client/ui-sidebar-files/src/client/FilesBody.module.css'
import { EmployeeNavigatorToggle } from './file-tabs.tsx'
export { EmployeeFileTabs } from './file-tabs.tsx'
import './file-panel.css'

if (typeof document !== 'undefined' && !document.querySelector('[data-dsh-icon-sprite]')) {
  installIconSprite({ effect: mount => mount(), on: () => {} }, undefined)
}

const emptyTrigger = { getTriggerProps: () => ({}) }
const placeBeside = () => undefined

// Keep the original node with its session/device grant. Display paths alone cannot open files.
// Like DSH's navigator (search-rules isHiddenEntry, showHidden: false), dot
// entries and everything below them are left out unless hidden entries are shown.
function entriesOf(nodes, expanded, showHidden, entries = [], depth = 1, visible = true) {
  for (const node of nodes) {
    const hidden = node.name.startsWith('.')
    if (hidden && !showHidden) continue
    entries.push({
      path: node.path, name: node.name, node, depth, visible,
      meta: { kind: node.kind === 'symlink' ? 'other' : node.kind, hidden },
    })
    entriesOf(node.children, expanded, showHidden, entries, depth + 1, visible && expanded.has(node.path))
  }
  return entries
}

const showHiddenKey = 'aibertai.files.showHidden'
function readShowHidden() {
  try { return globalThis.localStorage?.getItem(showHiddenKey) === '1' } catch { return false }
}
function saveShowHidden(value) {
  try { globalThis.localStorage?.setItem(showHiddenKey, value ? '1' : '0') } catch {}
}

function moveFocus(event, entry, expanded, toggle) {
  const dir = entry.meta.kind === 'directory'
  const rows = [...event.currentTarget.closest('[role=tree]').querySelectorAll('[role=treeitem]')]
  const index = rows.indexOf(event.currentTarget)
  let next
  switch (event.key) {
    case 'ArrowDown': next = rows[index + 1]; break
    case 'ArrowUp': next = rows[index - 1]; break
    case 'Home': next = rows[0]; break
    case 'End': next = rows.at(-1); break
    case 'ArrowRight':
      if (!dir) return
      if (!expanded.has(entry.path)) toggle(entry.path)
      else if (Number(rows[index + 1]?.getAttribute('aria-level')) > entry.depth) next = rows[index + 1]
      break
    case 'ArrowLeft':
      if (dir && expanded.has(entry.path)) toggle(entry.path)
      else next = rows.slice(0, index).reverse().find(row => Number(row.getAttribute('aria-level')) < entry.depth)
      break
    default: return
  }
  event.preventDefault()
  next?.focus()
}

function Accessory({ entry }) {
  return entry.accessory
}

const treeSlot = () => Accessory

function pagination(pane, path) {
  const { directories, fileScope, query, label, onLoad } = pane
  const state = fileScope === 'all' && !query.trim() && directories[path]
  if (!state || (!state.error && state.data?.nextOffset == null)) return null
  return <button type="button" className="files-tree-more" disabled={state.loading}
    aria-label={label(state.error ? 'session.retry' : 'workspace.files.more') + ' ' + path}
    onClick={() => onLoad(path, state.error ? undefined : state.data.nextOffset)}>
    {label(state.error ? 'session.retry' : 'workspace.files.more')}
  </button>
}

function EmployeeTree({ nodes, pane, translate, showHidden }) {
  const { expanded, directories, fileScope, query, label } = pane
  const listing = fileScope === 'all' && !query.trim()
  const [focusPath, setFocus] = useState()
  const entries = useMemo(() => entriesOf(nodes, expanded, showHidden), [nodes, expanded, showHidden])
  const visible = entries.filter(entry => entry.visible && entry.meta.kind !== 'other')
  const currentRef = useCallback(element => element?.scrollIntoView?.({ block: 'nearest' }), [])
  const activePath = [focusPath, pane.selected].find(path => visible.some(entry => entry.path === path)) ?? visible[0]?.path
  const states = Object.fromEntries(Object.entries(directories).map(([path, state]) => [path, {
    loading: listing && state.loading && !state.data,
    failure: listing && state.error ? label('workspace.files.loadError') : undefined,
    omitted: listing ? undefined : 'unmatched',
  }]))
  return <>
    <div role="tree" aria-label={label('workspace.fileTree.' + fileScope)}>
      <FileTree root="" t={translate} useFactorySlot={treeSlot} directories={states}
        entries={entries.map(entry => ({ ...entry, accessory: <>
          {entry.node.operation && <span className="files-tree-operation" data-operation={entry.node.operation}
            title={label('workspace.changes.operation.' + entry.node.operation)}>{label('workspace.changes.operation.' + entry.node.operation)}</span>}
          {entry.node.staged && <span title={label('workspace.changes.staged')}>●</span>}
          {expanded.has(entry.path) && pagination(pane, entry.path)}
        </> }))}
        expanded={[...expanded]} current={pane.selected} highlight={query.trim()}
        currentRef={currentRef}
        canOpen={entry => Boolean(entry.node.file)}
        onOpen={entry => pane.onSelect(entry.node)} onToggle={entry => pane.onToggle(entry.path)}
        rowTriggerProps={entry => ({
          role: 'treeitem', 'aria-level': entry.depth, 'aria-label': entry.name, 'data-path': entry.path,
          'aria-selected': entry.meta.kind === 'file' ? entry.path === pane.selected : undefined,
          tabIndex: entry.path === activePath ? 0 : -1, onFocus: () => setFocus(entry.path),
          title: entry.path, ...pane.dragProps(entry.node),
          onKeyDown: event => moveFocus(event, entry, expanded, pane.onToggle),
        })} />
    </div>
  </>
}

// DSH owns the path header, toolbar and tree. The employee shell supplies the
// session-scoped navigator slot, including its scope picker and the git and
// session change sources.
export default function EmployeeFiles(pane) {
  const { sessionId, cwd, fileScope, onScope, query, onQuery, label } = pane
  const translate = key => (pane.language === 'en' ? en : zh)[key] ?? key
  const [searchShown, setSearchShown] = useState(Boolean(query))
  const [showHidden, setShowHidden] = useState(readShowHidden)
  const toggleHidden = () => setShowHidden(value => { saveShowHidden(!value); return !value })
  const input = useRef(null)
  const commands = useRef(null)
  const focus = () => { setSearchShown(true); input.current?.focus() }
  useLayoutEffect(() => { if (searchShown) input.current?.focus() }, [searchShown])
  const info = useMemo(() => ({ panel: { id: sessionId }, tab: {
    id: sessionId, actions: { bindCommands(value) { commands.current = value; return () => { commands.current = null } } },
  } }), [sessionId])
  const searchBarShown = (_, open) => { setSearchShown(open); if (!open) onQuery('') }
  const [scopeOpen, setScopeOpen] = useState(false)
  const scopeTrigger = useRef(null)
  const scopes = ['all', 'session', ...(pane.git ? ['git'] : [])]
  const scopePicker = <Menu open={scopeOpen} portal compact autoFocus selectedId={fileScope} anchorRef={scopeTrigger}
    className="dsh-employee-file-scope" listClassName="dsh-native-theme" onClose={() => setScopeOpen(false)}
    items={scopes.map(id => ({ id, label: label('workspace.fileTree.' + id) }))}
    onSelect={next => { onScope(next); setScopeOpen(false); scopeTrigger.current?.focus() }}
    anchor={<button type="button" ref={scopeTrigger} aria-haspopup="menu" aria-expanded={scopeOpen}
      aria-label={label('workspace.fileTree.scope')} title={cwd} data-files-scope
      onKeyDown={event => {
        if (!scopeOpen && (event.key === 'ArrowDown' || event.key === 'ArrowUp')) {
          event.preventDefault(); setScopeOpen(true)
        }
      }} onClick={() => setScopeOpen(open => !open)}>
      <span>{label('workspace.fileTree.' + fileScope)}</span><IconChevronDownOutlineRegular size={12}/>
    </button>} />
  const navigator = <div className="dsh-employee-navigator">
    <div className="dsh-employee-file-scope-row">{scopePicker}</div>
    {searchShown && <div className={styles.searchBar}><div className={styles.searchField}>
      <input type="search" ref={input} value={query} className={styles.searchInput} data-files-search
        aria-label={translate('search.placeholder')} placeholder={translate('search.placeholder')} onChange={event => onQuery(event.target.value)} />
      <button type="button" className={styles.tool + ' ' + styles.searchTool} aria-pressed={showHidden}
        aria-label={translate('search.showHidden')} title={translate('search.showHidden')} data-files-show-hidden
        onClick={toggleHidden}>
        {showHidden ? <IconEyeOutlineRegular /> : <IconEyeOffOutlineRegular />}
      </button>
    </div></div>}
    <div className="dsh-employee-file-scroll" aria-busy={pane.loading}>
      {pane.loading && !pane.directories['']?.loading && !pane.nodes.length && <p role="status">{translate('loading')}</p>}
      {pane.error && <div role="alert">{label('workspace.files.loadError')} <button type="button" onClick={pane.onRefresh}>{label('session.retry')}</button></div>}
      {(pane.nodes.length > 0 || pane.directories['']?.loading || pane.directories['']?.error) &&
        <EmployeeTree nodes={pane.nodes} pane={pane} translate={translate} showHidden={showHidden} />}
      {!showHidden && pane.nodes.length > 0 && pane.nodes.every(node => node.name.startsWith('.')) &&
        <p className="files-tree-empty" data-files-empty-hidden>{translate('emptyHidden')}</p>}
      {pagination(pane, '')}
      {pane.outsideNodes.length > 0 && <>
        <p className="files-tree-section">{label('workspace.fileTree.outside')}</p>
        <EmployeeTree nodes={pane.outsideNodes} pane={pane} translate={translate} showHidden={showHidden} />
      </>}
      {!pane.nodes.length && !pane.outsideNodes.length && !pane.loading && !pane.error && !pane.directories['']?.error &&
        <p className="files-tree-empty">{label(query ? 'workspace.fileTree.noResults' : fileScope === 'session' ? 'workspace.fileTree.noSessionChanges'
          : fileScope === 'git' ? 'workspace.fileTree.noGitChanges' : 'workspace.files.empty')}</p>}
      {pane.truncated && <p className="files-tree-empty">{label('workspace.fileTree.truncated')}</p>}
    </div>
  </div>
  const root = cwd || (pane.nodes.length || pane.outsideNodes.length ? label('workspace.fileTree.' + fileScope) : undefined)
  return <div className="dsh-native-theme dsh-employee-files" onKeyDown={event => {
    if (event.key === 'F5') { event.preventDefault(); commands.current?.refresh() }
  }}>
    <FilesBody sessionId={sessionId} useTabInfo={() => info} useContextMenu={() => emptyTrigger}
      useSessions={select => select({ byId: { [sessionId]: { cwd: root } } })}
      useStore={select => select({ byPane: { [sessionId]: { searchBarOpen: searchShown } } })}
      actions={{ searchBarShown }} refresh={pane.onRefresh} focus={focus} placeBeside={placeBeside} t={translate}
      renderFactorySlot={() => navigator} renderSlot={() => pane.reveal} />
  </div>
}

const previewTab = { tab: { actions: {} } }
export function EmployeePreview({ path, tools, viewer, handoff, children, navigator }) {
  return <div className="dsh-native-theme dsh-employee-preview">
    <PreviewFrame path={path} tools={viewer ? <EmployeeViewerTools path={path} {...viewer} /> : tools} handoff={handoff} pathTrigger={{}} useTabInfo={() => previewTab}
      renderFactorySlot={slot => navigator && slot === 'sidebar.files.navigator.toggle' ? <EmployeeNavigatorToggle {...navigator} /> : null}
      data-dsh-file-preview="true"><div className="workspace-file-view">{children}</div></PreviewFrame>
  </div>
}

export function EmployeeText({ content, position, wrap = false }) {
  const body = useRef(null)
  useEffect(() => {
    body.current?.querySelector('[data-textpreview-target]')?.scrollIntoView?.({ block: 'center' })
  }, [content, position?.line])
  const page = useMemo(() => ({ offset: 1, lines: content.split('\n').length, text: content }), [content])
  return <div className={'dsh-native-theme dsh-employee-text ' + previewCss.body + (wrap ? ' ' + previewCss.wrap : '')} data-textpreview-wrap={wrap ? '' : undefined} ref={body}>
    <TextBody content={{ kind: 'text', pages: [page] }} useTabInfo={() => ({ tab: { navigation: { params: position } } })}/>
  </div>
}

export function EmployeePageAction({ language, failed, changed, busy, next, reload }) {
  const copy = language?.startsWith('en') ? previewEn : previewZh
  return <div className="dsh-native-theme">
    {failed && <p role="alert">{changed ? copy.changed : copy.resourceUnavailable}</p>}
    <button type="button" className={previewCss.more} data-textpreview-more disabled={busy}
      onClick={failed ? reload : next}>{copy[failed ? 'reload' : busy ? 'loading' : 'loadMore']}</button>
  </div>
}
