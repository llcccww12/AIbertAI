import './file-document.css'
import { useEffect, useRef } from 'react'
import { DocumentHeadingContext, revealHeading } from './document-links.tsx'
import { MarkdownDelegateProvider } from '@deepseek-ai/dsh-client-ui-primitives'
import { MediaContext } from 'employee-media'
import { MarkdownBody } from 'sdk/client/ui-sidebar-documentpreview/src/client/markdown/MarkdownBody.tsx'
import { CodeBody } from 'sdk/client/ui-sidebar-documentpreview/src/client/code/CodeBody.tsx'
import { en as markdownEn, zh as markdownZh } from 'sdk/client/ui-sidebar-documentpreview/src/client/markdown/locales.ts'
import { en as codeEn, zh as codeZh } from 'sdk/client/ui-sidebar-documentpreview/src/client/code/locales.ts'
import { en } from 'sdk/client/locale/src/locales/en.ts'
import { zh } from 'sdk/client/locale/src/locales/zh.ts'
import { fileAddressFor } from 'sdk/util/workspace-path/src/index.ts'

// Document and conversation Markdown share DSH's parser, Shiki, and the host's gated image delegate.
export function EmployeeDocument({ kind, language, content: text, eof = true, path, cwd, position, heading, wrap = false, sessionId, openFile, openExternalLink, renderImage }) {
  const root = useRef(null)
  const code = useRef(null)
  useEffect(() => {
    const pane = root.current
    if (kind !== 'markdown' || !heading || !pane || revealHeading(pane, heading)) return
    // A linked heading can arrive on a later page. Navigate once, then leave scrolling to the reader.
    const observer = new MutationObserver(() => {
      if (revealHeading(pane, heading)) observer.disconnect()
    })
    observer.observe(pane, { childList: true, subtree: true })
    return () => observer.disconnect()
  }, [kind, path, heading])
  useEffect(() => {
    const node = code.current
    const line = position?.line
    if (!node || !Number.isSafeInteger(line) || line < 1) return
    let current
    function reveal() {
      const target = node.querySelector('code > .line:nth-child(' + line + ')')
      if (!target || target === current) return
      target.setAttribute('aria-current', 'location')
      if (!current) target.scrollIntoView({ block: 'nearest', inline: 'nearest' })
      current = target
    }
    // Shiki can replace the initial plain lines after the grammar loads.
    const observer = new MutationObserver(reveal)
    observer.observe(node, { childList: true, subtree: true })
    reveal()
    return () => { observer.disconnect(); current?.removeAttribute('aria-current') }
  }, [kind, path, position?.line])
  const english = language?.startsWith('en')
  const copy = kind === 'code' ? (english ? codeEn : codeZh) : (english ? markdownEn : markdownZh)
  const translate = key => copy[key] ?? (english ? en : zh)[key] ?? key
  const content = { kind: 'text', eof, text }
  const resourceAddress = fileAddressFor(sessionId, cwd, path)
  return <div ref={root} className="dsh-native-theme dsh-employee-document" data-kind={kind}>
    {kind === 'code' ? <CodeBody content={content} resourceAddress={resourceAddress} wrap={wrap} scrollportRef={code} t={translate} /> :
      <MediaContext.Provider value={renderImage}>
        <MarkdownDelegateProvider openFile={openFile} openExternalLink={openExternalLink}>
          <DocumentHeadingContext.Provider value={(target, section) => {
            if (target) openFile(target, undefined, section)
            else if (!revealHeading(root.current, section) && !eof) {
              openFile(path.split(/[\\/]/).at(-1), undefined, section)
            }
          }}>
            <MarkdownBody content={content} resourceAddress={resourceAddress} useResource={() => ({ value: { absolutePath: path } })} t={translate} />
          </DocumentHeadingContext.Provider>
        </MarkdownDelegateProvider>
      </MediaContext.Provider>}
  </div>
}
