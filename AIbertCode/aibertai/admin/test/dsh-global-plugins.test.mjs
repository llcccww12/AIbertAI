import assert from "node:assert/strict"
import { readFileSync } from "node:fs"
import test from "node:test"

const read = relative => readFileSync(new URL(relative, import.meta.url), "utf8")
const catalog = JSON.parse(read("../src/lib/dsh-global-plugins.json"))

test("global plugin catalog matches the desktop SDK pin and host policy", () => {
  const desktop = "../../../../AIbertAI/"
  const pins = read(desktop + "dsh-integration/manifest.py")
  assert.match(pins, new RegExp('^SOURCE_COMMIT = "' + catalog.source + '"$', "m"))
  assert.match(pins, new RegExp('^PATCH_SHA256 = "' + catalog.patch + '"$', "m"))
  const rows = new Map(catalog.plugins.map(plugin => [plugin.id, plugin.state]))
  assert.equal(rows.size, catalog.plugins.length)
  const hostDisabled = read(desktop + "native/dsh_host_disabled_plugins.txt").trim().split(/\s+/)
  for (const id of hostDisabled.filter(id => !id.startsWith("tool-"))) {
    assert.equal(rows.get(id), "hostDisabled", id)
  }
  assert.equal(rows.get("timer"), "default")
  assert.equal(rows.get("session-telemetry-otel"), "hostDisabled")
  assert.deepEqual(catalog.plugins.find(plugin => plugin.id === "plugin-manager").sessionTools, ["tool-plugin-manager"])
  assert.ok(!rows.has("tool-plugin-manager"))
  assert.ok(!rows.has("tool-fs"))
})
