const fs = require("node:fs/promises");
const path = require("node:path");
const { O_RDONLY, O_NONBLOCK, O_NOFOLLOW } = require("node:fs").constants;
const { pathToFileURL } = require("node:url");

const textTypes = new Set([".md", ".markdown", ".txt"]);
const officeTypes = new Set([".docx", ".xlsx", ".pptx"]);
const badPath = () => new Error("invalid_path");

async function defaultApplication(resources, file) {
  if (!resources) return null;
  try {
    const entry = path.join(resources, "app", "packages", "util", "native-command", "lib", "index.js");
    const { nativeFileApplications } = await import(pathToFileURL(entry).href);
    const apps = await nativeFileApplications(file, AbortSignal.timeout(5000));
    const app = apps.find(item => item.default);
    return app ? { id: "default", name: app.name, icon: app.icon } : null;
  } catch { return null; }
}

// The caller binds this operation to a trusted local renderer, account and
// session. Accept only a workspace-relative document, including on retries.
async function workspaceDocumentAction({ cwd, relative, action, assertCurrent, preferred,
  openPath, showItemInFolder, verifyOffice }) {
  if (!["inspect", "open", "reveal"].includes(action) || typeof relative !== "string"
    || !relative || /[\\:\x00-\x1f]/.test(relative) || relative.split("/").some(part => !part || part === "." || part === "..")) throw badPath();
  const extension = path.extname(relative).toLowerCase();
  if (!textTypes.has(extension) && !officeTypes.has(extension) && extension !== ".pdf") throw badPath();
  const root = await fs.realpath(cwd);
  // The runtime maps legacy saved paths after the workspace brand migration.
  const name = relative.replace(/^\.monkeyai\//, ".aibertai/");
  const target = await fs.realpath(path.join(root, name));
  const inside = path.relative(root, target);
  if (!inside || inside === ".." || inside.startsWith(".." + path.sep) || path.isAbsolute(inside)
    || path.extname(target).toLowerCase() !== extension) throw badPath();
  const initial = await fs.stat(target);
  if (!initial.isFile() || process.platform !== "win32" && (initial.mode & 0o111)) throw badPath();
  const handle = await fs.open(target, O_RDONLY | O_NONBLOCK | O_NOFOLLOW);
  try {
    const info = await handle.stat();
    if (!info.isFile() || info.dev !== initial.dev || info.ino !== initial.ino) throw badPath();
    const head = Buffer.alloc(512);
    const { bytesRead } = await handle.read(head, 0, head.length, 0);
    const prefix = head.subarray(0, bytesRead);
    if (officeTypes.has(extension)) {
      if (!prefix.subarray(0, 4).equals(Buffer.from([80,75,3,4])) || info.size > 128 * 1024 * 1024) throw badPath();
      if (action === "open") {
        const bytes = Buffer.alloc(info.size);
        for (let offset = 0; offset < bytes.length;) {
          const read = await handle.read(bytes, offset, bytes.length - offset, offset);
          if (!read.bytesRead) throw badPath();
          offset += read.bytesRead;
        }
        await verifyOffice(bytes, extension.slice(1));
      }
    } else if (extension === ".pdf") {
      if (!prefix.subarray(0, 5).equals(Buffer.from("%PDF-"))) throw badPath();
    } else {
      let text;
      try { text = new TextDecoder("utf-8", { fatal: true }).decode(prefix, { stream: bytesRead === head.length }); }
      catch { throw badPath(); }
      if (text.startsWith("#!") || /[\x00-\x08\x0e-\x1f]/.test(text)) throw badPath();
    }
    let application;
    if (action === "inspect") application = await preferred(target);
    const current = await fs.lstat(target);
    if (!current.isFile() || current.dev !== info.dev || current.ino !== info.ino || current.mode !== info.mode
      || current.size !== info.size || current.ctimeMs !== info.ctimeMs || current.mtimeMs !== info.mtimeMs || await fs.realpath(target) !== target) throw badPath();
    assertCurrent();
    if (action === "inspect") return { application };
    if (action === "reveal") return showItemInFolder(target);
    const failure = await openPath(target);
    if (failure) throw new Error(failure);
  } finally { await handle.close(); }
}

module.exports = { defaultApplication, workspaceDocumentAction };
