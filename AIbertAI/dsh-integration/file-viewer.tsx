import { useMemo, useRef, useState } from 'react'
import { Menu } from 'sdk/client/ui-primitives/src/Menu.tsx'
import { Tooltip } from 'sdk/client/ui-primitives/src/Tooltip.tsx'
import { IconNowrapFillRegular, IconRefreshOutlineRegular, IconWrapFillRegular } from 'sdk/client/ui-primitives/src/icons/index.tsx'
import { HtmlBody } from 'sdk/client/ui-sidebar-documentpreview/src/client/html/HtmlBody.tsx'
import { en as htmlEn, zh as htmlZh } from 'sdk/client/ui-sidebar-documentpreview/src/client/html/locales.ts'
import { en, zh } from 'sdk/client/ui-sidebar-documentpreview/src/client/locales.ts'
import { en as codeEn, zh as codeZh } from 'sdk/client/ui-sidebar-documentpreview/src/client/code/locales.ts'
import { languageForPath } from 'sdk/util/code-language/src/index.ts'
import css from 'sdk/client/ui-sidebar-documentpreview/src/client/TextPreview.module.css'

const staticPreview = select => select(false)
const noResources = () => {}

// The DSH static viewer owns sanitization, its CSP and its opaque iframe.
export function EmployeeHtml({ content, path, language }) {
  const data = useMemo(() => new TextEncoder().encode(content), [content])
  const locale = language?.startsWith('en') ? htmlEn : htmlZh
  return <HtmlBody content={{ kind: 'bytes', data }} useTabInfo={() => ({ tab: { id: path } })}
    useInteractivePreview={staticPreview} setResources={noResources} t={key => locale[key]} />
}

// The same Menu, tool classes, icons and locale as TextPreview; choices update the employee file tab.
export function EmployeeViewerTools({ path, language, kind, modes, mode, select, wrap, toggleWrap, reload, loading, label }) {
  const english = language?.startsWith('en')
  const locale = english ? en : zh
  const code = languageForPath(path)
  const [open, setOpen] = useState(false)
  const anchor = useRef(null)
  const titles = {
    preview: kind === 'html' ? 'HTML' : kind === 'markdown' ? 'Markdown' : kind === 'spreadsheet' ? (english ? 'Spreadsheet' : '表格') : kind === 'image' ? (english ? 'Image' : '图片') : label('files.preview'),
    interactive: english ? 'Interactive preview' : '交互式预览',
    content: code ? (english ? codeEn : codeZh).title : locale['viewer.text'],
    plain: locale['viewer.text'],
    print: english ? 'Print preview' : '打印预览',
    changes: label('workspace.fileTree.diff'),
  }
  const items = modes.filter(id => id !== 'plain' || code).map(id => ({ id, label: titles[id] }))
  const selected = items.some(item => item.id === mode) ? mode : 'content'
  return <>
    {items.length > 1 && <Menu open={open} selectedId={selected} items={items} align="end" portal dense autoFocus
      anchorRef={anchor} listClassName="dsh-native-theme" onClose={() => setOpen(false)}
      onSelect={id => { select(id); setOpen(false) }}
      anchor={<button type="button" ref={anchor} className={css.tool + ' ' + css.viewerTool}
        data-document-viewer-menu aria-haspopup="menu" aria-expanded={open}
        aria-label={locale.openWith} title={titles[selected]}
        onKeyDown={event => {
          if (!open && (event.key === 'ArrowDown' || event.key === 'ArrowUp')) {
            event.preventDefault(); setOpen(true)
          }
        }} onClick={() => setOpen(value => !value)}>{titles[selected]}</button>} />}
    {(selected === 'content' || selected === 'plain') && <Tooltip side="bottom" delayMs={500}
      label={locale[wrap ? 'wrap.disable' : 'wrap.enable']}>
      <button type="button" className={css.tool} aria-label={locale['wrap.aria']} aria-pressed={wrap}
        data-textpreview-tool="wrap" onClick={toggleWrap}>{wrap ? <IconNowrapFillRegular /> : <IconWrapFillRegular />}</button>
    </Tooltip>}
    <Tooltip side="bottom" delayMs={500} label={locale.reload}>
      <button type="button" className={css.tool} data-textpreview-tool="reload" aria-label={locale.reload}
        disabled={loading} onClick={reload}><IconRefreshOutlineRegular /></button>
    </Tooltip>
  </>
}
