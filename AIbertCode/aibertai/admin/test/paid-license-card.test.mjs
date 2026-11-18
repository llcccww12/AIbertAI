import test from "node:test"
import assert from "node:assert/strict"
import { readFile } from "node:fs/promises"

const root = new URL("../", import.meta.url)
const source = async (path) => readFile(new URL(path, root), "utf8")

test("paid license has no entry in admin settings or sidebar", async () => {
  const [settings, routes, app, sidebar] = await Promise.all([
    source("src/pages/other-settings-page.tsx"),
    source("src/lib/routes.ts"),
    source("src/App.tsx"),
    source("src/components/app-sidebar.tsx"),
  ])
  assert.doesNotMatch(settings, /PaidLicenseCard/)
  assert.doesNotMatch(routes, /license: "\/console\/settings\/license"/)
  assert.doesNotMatch(app, /<LicensePage \/>/)
  assert.doesNotMatch(sidebar, /pages\.license\.title/)
})

test("license page has primary locale translations", async () => {
  const [en, zh] = await Promise.all([
    source("src/i18n/locales/en-US.ts"),
    source("src/i18n/locales/zh-CN.ts"),
  ])
  for (const messages of [en, zh]) {
    assert.match(messages, /license: \{/)
    assert.match(messages, /importSuccess:/)
    assert.match(messages, /machineCodeCopied:/)
  }
})
