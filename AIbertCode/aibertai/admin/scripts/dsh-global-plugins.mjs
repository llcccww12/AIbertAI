// Generate or verify the bundled admin inventory with the packaged SDK, without booting an agent.
import { execFileSync } from "node:child_process"
import { readFileSync, writeFileSync, existsSync, mkdtempSync, rmSync } from "node:fs"
import { tmpdir } from "node:os"
import { join, resolve } from "node:path"
import { fileURLToPath } from "node:url"
import { parseDocument } from "yaml"

const admin = fileURLToPath(new URL("../", import.meta.url))
const desktop = resolve(admin, "../../../AIbertAI")
const pin = readFileSync(join(desktop, "dsh-integration/manifest.py"), "utf8")
const patch = /^PATCH_SHA256 = "([0-9a-f]{64})"/m.exec(pin)[1]
const source = /^SOURCE_COMMIT = "([0-9a-f]{40})"/m.exec(pin)[1]
const resources = join(desktop, ".local-runtime/dsh", patch, "resources")
const read = (path) => JSON.parse(readFileSync(path, "utf8"))
const manifest = read(join(resources, "manifest.json"))
if (manifest.patch_sha256 !== patch || manifest.source_commit !== source) throw Error("Wrong DSH SDK build")
const app = join(resources, "app")
const yaml = (text) => {
  const doc = parseDocument(text)
  if (doc.errors.length) throw doc.errors[0]
  return doc.toJS() // Keep unevaluated !!js expressions as strings.
}

const isolatedHome = mkdtempSync(join(tmpdir(), "dsh-inventory-"))
let rows
try {
  rows = yaml(execFileSync(join(resources, manifest.node), [
    join(resources, manifest.entry), "--profile", "sdk", "--dump-default-config",
  ], { encoding: "utf8", cwd: app, env: { PATH: process.env.PATH, DSH_HOME: isolatedHome } }))
} finally {
  rmSync(isolatedHome, { recursive: true, force: true })
}
const sessionModules = new Set()
// The SDK flattens the same packages used by the built-in agent presets into its root.
function visitPreset(rows) {
  for (const row of rows) {
    if (row.group) visitPreset(row.config)
    else sessionModules.add(row.name)
  }
}
for (const preset of ["standard", "minimal", "ptc"]) {
  const layers = yaml(readFileSync(join(app, "packages/bundle/web-app/presets", preset + ".patch.yml"), "utf8"))
  for (const layer of layers) for (const row of layer.insert) visitPreset(row.config.plugins)
}
const hostDisabled = new Set(readFileSync(join(desktop, "native/dsh_host_disabled_plugins.txt"), "utf8").trim().split(/\s+/))
hostDisabled.add("session-telemetry-otel") // dshInstalledRuntime exports DSH_TELEMETRY_DISABLED=1.
const roots = ["packages/bundle/base", "packages/bundle/sdk-app", "packages/boot/app-boot", "apps/cli"]
function describe(module) {
  const candidates = roots.flatMap(root => [join(app, root, "node_modules", module), join(app, root)])
  const dir = candidates.find(path => {
    const manifest = join(path, "package.json")
    return existsSync(manifest) && read(manifest).name === module
  })
  if (!dir) throw Error("No SDK package metadata for " + module)
  const pkg = read(join(dir, "package.json"))
  const meta = (lang) => {
    const file = join(dir, "locale", lang + ".json")
    return existsSync(file) ? read(file).meta ?? {} : {}
  }
  const zh = meta("zh"), en = meta("en")
  return { zh: zh.title, detail: zh.description, en: en.title, description: en.description || pkg.description }
}
// The loader's bootstrap include is outside the tree returned by --dump-default-config.
rows.unshift({ id: "include", name: "@deepseek-ai/cordis-plugin-include" })
function loadingPolicy(row) {
  if (hostDisabled.has(row.id)) return "hostDisabled"
  if (row.disabled === true) return "disabled"
  if (row.disabled) return "conditional"
  return "default"
}
// These SDK-only tools are exposed to agents but not part of the web presets.
const sdkSessionTools = new Set(["tool-working-directory", "tool-agent-team"])
const plugins = rows
  .filter(row => !sessionModules.has(row.name) && !sdkSessionTools.has(row.id))
  .map(row => {
    const sessionTools = rows.filter(tool => sessionModules.has(tool.name) &&
      tool.name.startsWith(row.name + "/") && tool.id.startsWith("tool-")).map(tool => tool.id)
    return { id: row.id, state: loadingPolicy(row), ...describe(row.name),
      ...(sessionTools.length ? { sessionTools } : {}) }
  })
const catalog = { version: read(join(app, "apps/cli/package.json")).version, source, patch, plugins }
const output = join(admin, "src/lib/dsh-global-plugins.json")
const data = JSON.stringify(catalog, null, 2) + "\n"
if (process.argv.includes("--check")) {
  if (readFileSync(output, "utf8") !== data) throw Error("Stale DSH global plugin catalog")
} else writeFileSync(output, data)
console.log(plugins.length + " DSH global plugins, sdk " + catalog.version)
