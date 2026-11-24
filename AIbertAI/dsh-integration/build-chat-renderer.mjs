// node build-chat-renderer.mjs <prepared DSH resources/app> [web/dsh directory] [--only=chat|browser|files|jobs|members|presets]
import { readFile, readdir, mkdir, unlink, writeFile } from 'node:fs/promises'
import { resolve, dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createRequire } from 'node:module'

const here = dirname(fileURLToPath(import.meta.url))
const [sdkPath, outputPath] = process.argv.slice(2).filter(arg => !arg.startsWith('--'))
const sdk = resolve(sdkPath)
const web = resolve(outputPath || join(here, '../code/web/dsh'))
const only = process.argv.slice(2).find(arg => arg.startsWith('--only='))?.slice(7)
const renderers = {
  chat: ['chat-renderer.tsx', 'reply'],
  browser: ['browser-panel.tsx', 'panel'],
  files: ['file-panel.tsx', 'files'],
  jobs: ['jobs-renderer.tsx', 'jobs'],
  members: ['members-renderer.tsx', 'members'],
  presets: ['preset-renderer.tsx', 'preset'],
}
if (only && !Object.hasOwn(renderers, only)) throw new Error('Unknown renderer: ' + only)
const esbuild = createRequire(join(sdk, 'packages/boot/hmr/package.json'))('esbuild')
const source = path => join(sdk, 'packages', path)
const dshTheme = source('client/ui-theme/src/styles')
for (const [name, [entryFile, entryName]] of Object.entries(renderers)) {
  if (!only || name === only) await buildRenderer(entryFile, entryName, join(web, 'native-' + name))
}

async function buildRenderer(entryFile, entryName, out) {
  await mkdir(out, { recursive: true })

  const primitiveFacade = entryFile.replace(/-(?:panel|renderer)\.tsx$/, '-primitives.ts')
  const primitive = (await readdir(here)).includes(primitiveFacade)
    ? join(here, primitiveFacade) : source('client/ui-primitives/src/index.ts')
  const embeddedInputs = new Set()
  const result = await esbuild.build({
    absWorkingDir: sdk, entryPoints: { [entryName]: join(here, entryFile) },
    outdir: out, loader: { '.png': 'file' }, bundle: true, format: 'esm', target: 'es2022', splitting: true,
    chunkNames: 'chunk-[hash]', minifyWhitespace: true, minifySyntax: true, metafile: true, jsx: 'automatic',
    legalComments: 'eof', define: { 'process.env.NODE_ENV': '"production"' },
    plugins: [{ name: 'employee-chat', setup(build) {
      build.onResolve({ filter: /^@deepseek-ai\/dsh-client-store$/ }, () => ({ path: source('client/store/src/index.ts') }))
      build.onResolve({ filter: /^sdk\// }, a => ({ path: source(a.path.slice(4)) }))
      build.onResolve({ filter: /^@deepseek-ai\/dsh-client-ui-primitives$/ }, () => ({ path: primitive }))
      build.onResolve({ filter: /^(react|react\/jsx-runtime|react-dom)$/ }, a => ({ path: a.path, namespace: 'framework' }))
      build.onLoad({ filter: /.*/, namespace: 'framework' }, a => ({
        contents: `import {framework} from 'employee-framework'; module.exports = framework.${({'react':'react','react/jsx-runtime':'jsx','react-dom':'dom'})[a.path]};`
      }))
      build.onResolve({ filter: /^employee-framework$/ }, () => ({ path: '../chat-framework.mjs', external: true }))
      build.onResolve({ filter: /@fortune-sheet\/react\/dist\/index\.css\?inline$/ }, args => ({
        path: createRequire(args.importer).resolve('@fortune-sheet/react/dist/index.css'), namespace: 'inline-excel-css',
      }))
      build.onLoad({ filter: /.*/, namespace: 'inline-excel-css' }, async ({path}) => ({ contents: await readFile(path, 'utf8'), loader: 'text' }))
      // The app allows same-origin workers. Serve the same DSH parser as a packaged asset.
      build.onLoad({ filter: /ui-sidebar-documentpreview\/src\/client\/excel\/parse\.ts$/ }, async ({path}) => {
        const original = await readFile(path, 'utf8')
        const url = "URL.createObjectURL(new Blob([workerSource], { type: 'text/javascript' }))"
        if (!original.includes(url)) throw new Error('DSH Excel worker bootstrap changed')
        return { loader: 'ts', resolveDir: dirname(path), contents: original.replace(url,
          'new URL(workerSource, import.meta.url).href').replaceAll('URL.revokeObjectURL(url)', '') }
      })
      build.onResolve({ filter: /^\.\/worker\.ts\?raw$/ }, args => ({
        path: join(args.resolveDir, 'worker.js'), namespace: 'excel-worker',
      }))
      build.onLoad({ filter: /.*/, namespace: 'excel-worker' }, async ({path}) => {
        const worker = await esbuild.build({ entryPoints: [join(dirname(path), 'worker.ts')], absWorkingDir: sdk,
          bundle: true, write: false, format: 'iife', minify: true, platform: 'browser', metafile: true,
          define: { 'process.env.NODE_ENV': '"production"' } })
        for (const input of Object.keys(worker.metafile.inputs)) embeddedInputs.add(input)
        return { contents: worker.outputFiles[0].contents, loader: 'file' }
      })
      // The employee app already ships and loads KaTeX's global stylesheet/fonts.
      build.onResolve({ filter: /^katex\/dist\/katex.min.css$/ }, () => ({ path: 'katex', namespace: 'loaded-css' }))
      build.onLoad({ filter: /.*/, namespace: 'loaded-css' }, () => ({ contents: '', loader: 'css' }))
      build.onResolve({ filter: /^employee-document-links$/ }, () => ({ path: join(here, 'document-links.tsx') }))
      build.onResolve({ filter: /^employee-media$/ }, () => ({ path: 'media', namespace: 'employee-media' }))
      build.onLoad({ filter: /.*/, namespace: 'employee-media' }, () => ({
        contents: "import {createContext} from 'react'; export const MediaContext = createContext(null);"
      }))
      // Use the shell's workspace authorization and click-to-load remote image control.
      // Keep DSH's parser, code blocks, links and all other Markdown rendering native.
      build.onLoad({ filter: /ui-primitives[\/]src[\/]markdown[\/]render\.tsx$/ }, async ({path}) => {
        const original = await readFile(path, 'utf8')
        const anchor = 'const { fileImages } = useMarkdownDelegate()\n  const [load] = useState(nextImageLoad)'
        if (original.split(anchor).length !== 2) throw new Error('DSH Markdown image adapter changed')
        const imports = "import {MediaContext} from 'employee-media'; import {useContext as useMediaContext} from 'react';"
        const link = 'function renderAnchor(url: string, children: ReactNode[], key: Key, glyph = true, streaming = false): ReactNode {'
        if (original.split(link).length !== 2) throw new Error('DSH Markdown link adapter changed')
        const headingLinks = link + `
  const file = parseFileLink(url)
  const hash = url.indexOf('#')
  const section = hash >= 0 && hash < url.length - 1 &&
    (hash === 0 || (file && !file.line && /\\.(?:md|markdown)$/i.test(file.path))) ? url.slice(hash + 1) : undefined
  const fallback = renderNativeAnchor(url, children, key, glyph, streaming)
  if (!section && !(streaming && file)) return fallback
  return <DocumentLocalLink key={key} className={clsx(css.fileMention, css.fileLink)} file={file?.path} hash={section}
    fallback={fallback} settled={!section && renderNativeAnchor(url, children, key, glyph, false)}>{children}</DocumentLocalLink>
}
` + link.replace('renderAnchor', 'renderNativeAnchor')
        return { loader: 'tsx', resolveDir: dirname(path), contents: "import {DocumentLocalLink} from 'employee-document-links';" + imports + original.replace(link, headingLinks).replace(anchor,
          anchor + '\n  const renderMedia = useMediaContext(MediaContext); if (renderMedia) return renderMedia(destination, alt)') }
      })
      build.onLoad({ filter: /(?:JobListAction|SubagentHeaderLineage)\.module\.css$/ }, async ({path}) => ({
        loader: 'local-css', resolveDir: dirname(path),
        contents: await readFile(path, 'utf8') + '\n.menu { composes: dsh-native-theme from global; }',
      }))
      // The employee shell resets SVG display and list markers. DSH Markdown
      // expects the browser defaults for its inline link glyphs and native lists.
      build.onLoad({ filter: /markdown\/MarkdownText\.module\.css$/ }, async ({path}) => ({
        loader: 'local-css', resolveDir: dirname(path),
        contents: await readFile(path, 'utf8') + '\n.linkIcon { display: inline; }\n.markdown :where(ul, ol) { list-style-type: revert; }',
      }))
      build.onLoad({ filter: /ui-primitives\/src\/MenuSurface\.module\.css$/ }, async ({path}) => ({
        loader: 'local-css', resolveDir: dirname(path),
        contents: await readFile(path, 'utf8') + '\n.surface { composes: dsh-native-theme from global; }\n.backing { composes: dsh-native-theme from global; }',
      }))
      // Portaled DSH tooltips and modal dialogs also inherit the scoped theme.
      build.onLoad({ filter: /ui-primitives\/src\/(Modal|Tooltip)\.module\.css$/ }, async ({path}) => ({
        loader: 'local-css', resolveDir: dirname(path),
        contents: await readFile(path, 'utf8') + (path.endsWith('/Modal.module.css') ? '\n.root' : '\n.bubble') + ' { composes: dsh-native-theme from global; }',
      }))
      build.onResolve({ filter: /^employee-theme$/ }, () => ({ path: 'theme', namespace: 'employee-css' }))
      build.onLoad({ filter: /.*/, namespace: 'employee-css' }, async () => {
        const sheets = await Promise.all(['base','design-platform','gradient-shadow-text','shiki']
          .map(name => readFile(join(dshTheme, name + '.css'), 'utf8')))
        const tokens = sheets.join('\n').replaceAll(/html\[data-platform='darwin'\] body\[data-ds-dark-theme\]/g, 'html.dark[data-platform=darwin] .dsh-native-theme')
          .replaceAll('body[data-ds-dark-theme]', 'html.dark .dsh-native-theme').replaceAll(/\bbody\b|:root/g, '.dsh-native-theme')
        return { loader: 'css', contents: tokens + '\n.dsh-native-reply{min-width:0;display:flex;flex-direction:column;gap:8px}' }
      })
    }}]
  })

  // Separate native entries share a document. Keep generated CSS module names
  // descriptive across builds, then minify just JS identifiers in the output.
  for (const path of Object.keys(result.metafile.outputs).filter(path => path.endsWith('.js'))) {
    const file = resolve(sdk, path)
    const js = await esbuild.transform(await readFile(file, 'utf8'), {
      format: 'esm', target: 'es2022', minify: true, legalComments: 'eof',
    })
    await writeFile(file, js.code)
  }
  const emitted = new Set(Object.keys(result.metafile.outputs).map(path => resolve(sdk, path)))
  for (const file of await readdir(out)) {
    const path = join(out, file)
    if (/^(?:chunk-[A-Z0-9]+\.(?:js|css)|worker-[A-Z0-9]+\.js)$/.test(file) && !emitted.has(path)) await unlink(path)
  }

  // Include the notices of every dependency which actually contributes code.
  const contributed = new Set(embeddedInputs)
  for (const details of Object.values(result.metafile.outputs)) {
    for (const [input, info] of Object.entries(details.inputs)) {
      if (info.bytesInOutput) contributed.add(input.replace(/^inline-excel-css:/, ''))
    }
  }
  const packageDirs = new Set()
  for (const input of contributed) {
    const match = /^(.*node_modules\/\.pnpm\/[^/]+\/node_modules\/(?:@[^/]+\/)?[^/]+)\//.exec(input)
    if (match) packageDirs.add(resolve(sdk, match[1]))
  }
  const notices = [await readFile(join(sdk, 'LICENSE'), 'utf8')]
  for (const dir of [...packageDirs].sort()) {
    const pkg = JSON.parse(await readFile(join(dir, 'package.json'), 'utf8'))
    const licenses = (await readdir(dir)).filter(file => /^(licen[sc]e|copying|notice)(\.|$)/i.test(file))
    for (const file of licenses) notices.push(pkg.name + '@' + pkg.version + '\n' + await readFile(join(dir, file), 'utf8'))
  }
  await writeFile(join(out, 'LICENSES.txt'), notices.join('\n\n'))
}
