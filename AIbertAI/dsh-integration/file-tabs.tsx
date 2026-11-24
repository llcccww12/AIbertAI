import { TabStrip } from 'sdk/client/ui-dockkit/src/components/TabPanel.tsx'
import { TextTitle } from 'sdk/client/ui-sidebar-documentpreview/src/client/TextTitle.tsx'
import { FilesTitle } from 'sdk/client/ui-sidebar-files/src/client/FilesTitle.tsx'
import { NavigatorToggle } from 'sdk/client/ui-sidebar-files/src/client/NavigatorToggle.tsx'
import { zh, en } from 'sdk/client/ui-sidebar-files/src/client/locales.ts'

const directoryId = 'directory'

/** The employee shell owns file locators and focus; DockKit handles chip sizing, scroll, keys and closing. */
export function EmployeeFileTabs({ items, active, onSelect, onClose, onBrowse, language, directoryLabel }) {
  const directory = { id: directoryId, title: directoryLabel }
  const records = Object.fromEntries(items.map(item => [item.id, item]))
  if (!active) records[directoryId] = directory
  const tabs = [...(!active ? [directoryId] : []), ...items.map(item => item.id)]
  const pane = { id: 'documents', tabs, activeTabId: active ?? directoryId }
  const state = { tabs: records, activePaneId: pane.id, floats: [] }
  const isEnglish = language?.startsWith('en')
  return <div className="dsh-native-theme dsh-employee-file-tabs">
    <TabStrip pane={pane} state={state} callbacks={{
      labels: { closeTab: isEnglish ? 'Close tab' : '关闭标签页', addTab: isEnglish ? 'Browse workspace files' : '浏览工作区文件' },
      onFocusTab: id => id === directoryId ? onBrowse() : onSelect(id),
      canCloseTab: id => id !== directoryId, onCloseTab: onClose,
      canAddTab: () => Boolean(active), onAddTab: onBrowse,
      splitBlock: () => 'budget', hideSplitWhenBlocked: true, onTabPressed: () => {},
      renderTabTitle: record => {
        const useTabInfo = () => ({ tab: record })
        return <span className="dsh-file-tab-label" title={record.path ?? directoryLabel}>
          {record.id === directoryId ? <FilesTitle useTabInfo={useTabInfo}/> : <TextTitle useTabInfo={useTabInfo}/>}
        </span>
      },
    }} />
  </div>
}

export function EmployeeNavigatorToggle({ open, onOpen, buttonRef, language, sessionId, cwd }) {
  const locale = language?.startsWith('en') ? en : zh
  return <NavigatorToggle sessionId={sessionId} useTabInfo={() => ({ panel: { id: 'files' } })}
    useSessions={select => select({ byId: { [sessionId]: { cwd } } })}
    useStore={select => select({ byPane: { files: { open } } })}
    toggleDrawn={(_, element) => { buttonRef.current = element }}
    actions={{ shown: (_, next) => onOpen(next) }} t={key => locale[key]} />
}
