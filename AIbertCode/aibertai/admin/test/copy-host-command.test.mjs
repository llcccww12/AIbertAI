import assert from "node:assert/strict"
import test from "node:test"
import { copyHostCommand } from "../src/lib/copy-host-command.ts"

test("uses the Clipboard API when available", async (t) => {
  t.mock.getter(globalThis, "navigator", () => ({
    clipboard: { writeText: async (text) => assert.equal(text, "test command") },
  }))
  await copyHostCommand("test command")
})

for (const clipboard of [undefined, { writeText: async () => { throw Error("denied") } }]) {
  test(clipboard ? "falls back after Clipboard API denial" : "copies in an HTTP modal", async (t) => {
    t.mock.getter(globalThis, "navigator", () => ({ clipboard }))
    let selected = false
    let removed = false
    let restored = false
    const field = {
      style: {},
      focus() {},
      select() { selected = true },
      remove() { removed = true },
    }
    class Button {
      closest(selector) {
        assert.equal(selector, '[role="dialog"]')
        return { appendChild: (element) => assert.equal(element, field) }
      }
      focus() { restored = true }
    }
    globalThis.HTMLElement = Button
    globalThis.document = {
      activeElement: new Button(),
      createElement: () => field,
      execCommand(action) {
        assert.equal(action, "copy")
        assert.equal(field.value, "test command")
        assert.equal(selected, true)
        return true
      },
    }
    t.after(() => {
      delete globalThis.document
      delete globalThis.HTMLElement
    })
    await copyHostCommand("test command")
    assert.equal(removed, true)
    assert.equal(restored, true)
  })
}
