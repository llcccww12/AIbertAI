import { mkdir } from "node:fs/promises"
import { join } from "node:path"

export async function saveTestScreenshot(target, name, options) {
  const directory = process.env.SCREENSHOT_DIR
  if (!directory) return

  await mkdir(directory, { recursive: true })
  await target.screenshot({ ...options, path: join(directory, name) })
}
