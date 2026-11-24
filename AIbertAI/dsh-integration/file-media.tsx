import { useEffect, useMemo, useState } from 'react'
import { ImageBody } from 'sdk/client/ui-sidebar-documentpreview/src/client/image/ImageBody.tsx'
import { LazyExcelBody } from 'sdk/client/ui-sidebar-documentpreview/src/client/excel/LazyExcelBody.tsx'
import { LoadingIndicator } from 'sdk/client/ui-sidebar-documentpreview/src/client/LoadingIndicator.tsx'
import { en as imageEn, zh as imageZh } from 'sdk/client/ui-sidebar-documentpreview/src/client/image/locales.ts'
import { en as excelEn, zh as excelZh } from 'sdk/client/ui-sidebar-documentpreview/src/client/excel/locales.ts'
import { en as officeEn, zh as officeZh } from 'sdk/client/ui-sidebar-documentpreview/src/client/office/locales.ts'
import { fileAddressFor } from 'sdk/util/workspace-path/src/index.ts'

const noop = () => {}
const contextMenu = () => ({ getTriggerProps: () => ({}) })
const textFor = locale => (key, values = {}) => locale[key]?.replace(/\{(\w+)\}/g, (_, name) => values[name]) ?? key

export function EmployeeImage({ imageUrl, content, path, cwd, sessionId, language, zoom, onZoom, drag, reportFailure }) {
  const address = fileAddressFor(sessionId, cwd, path)
  const lifetime = useMemo(() => new AbortController(), [address])
  useEffect(() => () => lifetime.abort(), [lifetime])
  const bytes = useMemo(() => imageUrl
    ? Uint8Array.from(atob(imageUrl.slice(imageUrl.indexOf(',') + 1)), byte => byte.charCodeAt(0))
    : new TextEncoder().encode(content), [imageUrl, content])
  const zoomActions = useMemo(() => ({ zoom: (_, value) => onZoom(value) }), [onZoom])
  const fileDrag = useMemo(() => ({
    async prepare(source, signal) {
      signal.throwIfAborted()
      const file = source.blob
      return { release: noop, start(transfer) {
        transfer.items.add(file)
        drag?.(transfer)
        return true
      } }
    }, reportFailure,
  }), [drag, reportFailure])
  return <ImageBody content={{ kind: 'bytes', data: bytes }} resourceAddress={address} sessionId={sessionId}
    useTabInfo={() => ({ tab: { id: address, signal: lifetime.signal } })}
    useStore={select => select({ byTab: { [address]: zoom } })} actions={zoomActions} retainTab={noop}
    scrollportRef={noop} useContextMenu={contextMenu} fileDrag={fileDrag}
    t={textFor(language?.startsWith('en') ? imageEn : imageZh)} />
}

// Same defaults as sidebar-documentpreview's plugin config; parsing stays in the DSH Worker.
const excelLimits = { maxBytes: 16 * 1024 * 1024, maxCells: 250_000, timeoutMs: 15_000 }
export function EmployeeSpreadsheet({ path, cwd, sessionId, language, load }) {
  const [result, setResult] = useState()
  const [attempt, setAttempt] = useState(0)
  const t = textFor(language?.startsWith('en') ? excelEn : excelZh)
  useEffect(() => {
    const controller = new AbortController()
    load(controller.signal, excelLimits.maxBytes).then(
      data => { if (!controller.signal.aborted) setResult({ load, data }) },
      error => { if (!controller.signal.aborted) setResult({ load, error: error.message === 'tooLarge' ? 'tooLarge' : 'invalid' }) },
    )
    return () => controller.abort()
  }, [load, attempt])
  if (result?.load !== load) return <LoadingIndicator label={t('loading')} />
  if (result.error) return <div className="file-panel-empty" role="alert">{t(result.error)}
    <button type="button" onClick={() => { setResult(undefined); setAttempt(n => n + 1) }}>{t('retry')}</button></div>
  return <LazyExcelBody content={{ kind: 'bytes', data: result.data }}
    resourceAddress={fileAddressFor(sessionId, cwd, path)} limits={excelLimits} t={t} />
}

// DSH's Office preview shows this indicator while the document is converted and rendered.
export function EmployeeOfficeLoading({ language }) {
  return <LoadingIndicator label={(language?.startsWith('en') ? officeEn : officeZh).loading} />
}
