// The Electron workspace owns each page and its per-account/session tab history.
// Bind its observable state and native viewport to DSH's browser controller and UI.
import { useCallback, useLayoutEffect, useMemo, useRef, useState, useSyncExternalStore } from 'react'
import { createPortal } from 'react-dom'
import { createSnapshotStore } from 'sdk/client/store/src/index.ts'
import { BrowserBody } from 'sdk/client/ui-sidebar-browser/src/client/view/BrowserBody.tsx'
import { createBrowserControllers } from 'sdk/client/ui-sidebar-browser/src/client/browser/BrowserController.ts'
import { emptyBrowserFrame } from 'sdk/client/ui-sidebar-browser/src/client/browser/BrowserFrame.ts'
import { parseBrowserAddress } from 'sdk/client/ui-sidebar-browser/src/client/browser/url.ts'
import { zh, en } from 'sdk/client/ui-sidebar-browser/src/client/locales.ts'

const noop = () => {}
const nothing = () => undefined
const unsubscribeEmpty = () => noop
const nativeHistory = { replace: noop, forget: noop }
export { parseBrowserAddress }

export default function EmployeeBrowser({ tabId, language, state, addressRef, onToolbar, onNavigate, onAction, children }) {
  const commands = useRef({ onNavigate, onAction })
  commands.current = { onNavigate, onAction }
  const [viewport, setViewport] = useState<HTMLElement | null>(null)
  const view = useMemo(() => {
    const frame = createSnapshotStore(emptyBrowserFrame())
    const controllers = createBrowserControllers(nativeHistory, () => ({
      presentation: {
        mount(id) {
          setViewport(document.getElementById(id))
          return () => setViewport(null)
        },
      },
      frame: {
        subscribe: frame.subscribe, getSnapshot: frame.getSnapshot,
        loadUrl: target => commands.current.onNavigate(target.url),
        goBack: () => commands.current.onAction('back'),
        goForward: () => commands.current.onAction('forward'),
        reload: () => commands.current.onAction('reload'),
        dispose: async () => {},
      },
    }), () => true)
    const tab = { id: tabId, signal: new AbortController().signal, navigation: {},
      actions: { bindCommands: () => noop } }
    return { frame, controllers, useTabInfo: () => ({ tab }) }
  }, [tabId])

  const { url, title, canGoBack, canGoForward, loading, error } = state ?? {}
  useLayoutEffect(() => {
    const parsed = url ? parseBrowserAddress(url, window.location.origin) : undefined
    const target = parsed?.ok ? { ...parsed.target, title: title || parsed.target.title } : undefined
    view.frame.set({ ...emptyBrowserFrame(), target, address: target ? 'observed' : 'empty',
      loading: !!loading, canGoBack: !!canGoBack, canGoForward: !!canGoForward,
      error: error ? { description: error, code: undefined } : undefined })
  }, [view, url, title, canGoBack, canGoForward, loading, error])

  useLayoutEffect(() => () => { void view.controllers.dispose() }, [view])
  const wrapper = useRef<HTMLDivElement>(null)
  useLayoutEffect(() => {
    addressRef.current = wrapper.current?.querySelector('input') ?? null
    onToolbar(wrapper.current?.querySelector('form') ?? null)
    return () => { addressRef.current = null; onToolbar(null) }
  }, [addressRef, onToolbar])
  const useBrowserState = useCallback((id: string) => {
    const store = view.controllers.keyedHooks.browserState(id)
    return useSyncExternalStore(store?.subscribe ?? unsubscribeEmpty, store?.getSnapshot ?? nothing)
  }, [view])
  const strings = language?.startsWith('zh') ? zh : en
  return <div ref={wrapper} className="dsh-native-theme"
    style={{ display: 'flex', flex: '1 1 auto', minHeight: 0, minWidth: 0 }} data-dsh-browser>
    <BrowserBody {...view.controllers} useBrowserState={useBrowserState} useTabInfo={view.useTabInfo}
      useStore={nothing} t={key => strings[key]} />
    {viewport && createPortal(children, viewport)}
  </div>
}
