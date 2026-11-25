export async function copyHostCommand(command: string): Promise<void> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(command)
      return
    } catch {
      // The browser can deny Clipboard API access even on HTTPS.
    }
  }

  const previous = document.activeElement
  const field = document.createElement("textarea")
  field.value = command
  field.readOnly = true
  field.tabIndex = -1
  field.style.cssText = "position:fixed;inset:0 auto auto 0;opacity:0;pointer-events:none"
  // Keep focus inside the current Radix modal on HTTP deployments.
  const container = previous?.closest('[role="dialog"]') ?? document.body
  container.appendChild(field)
  try {
    field.focus({ preventScroll: true })
    field.select()
    if (!document.execCommand("copy")) throw new Error("clipboard_copy_failed")
  } finally {
    field.remove()
    if (previous instanceof HTMLElement) previous.focus({ preventScroll: true })
  }
}

// Works on plain-HTTP deployments where navigator.clipboard is unavailable.
export const copyText = copyHostCommand
