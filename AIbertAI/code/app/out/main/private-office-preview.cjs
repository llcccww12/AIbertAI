const { createServer } = require("node:http");
const { execFile } = require("node:child_process");
const fs = require("node:fs/promises");
const path = require("node:path");
const os = require("node:os");
const { pathToFileURL } = require("node:url");
const { timingSafeEqual } = require("node:crypto");
const { promisify } = require("node:util");

const run = promisify(execFile);
const LIMIT = 128 * 1024 * 1024;
const POLICY = "default-src 'none'; script-src 'self'; worker-src 'self'; connect-src 'self'; style-src 'unsafe-inline'; img-src data: blob:; frame-ancestors 'none'; base-uri 'none'; form-action 'none'";
const VIEW = '<!doctype html><meta charset="utf-8"><title>Office 预览</title><style>body{margin:16px;background:#e8eaed;font:15px system-ui;color:#202124}canvas{display:block;max-width:100%;height:auto;margin:12px auto;box-shadow:0 1px 4px #0003}</style><main>正在加载文档…</main><script type="module" src="/vendor/viewer.mjs"></script>';
const OFFICE_FONTS = `<?xml version="1.0"?>
<fontconfig>
  <dir>/System/Library/Fonts</dir>
  <dir>/Library/Fonts</dir>
  <cachedir prefix="xdg">fontconfig</cachedir>
  <match target="pattern">
    <test name="lang" compare="contains"><string>zh</string></test>
    <edit name="family" mode="prepend"><string>Heiti SC</string></edit>
  </match>
  <alias><family>Arial</family><prefer><family>Arial</family><family>Heiti SC</family></prefer></alias>
</fontconfig>`;

async function renderOffice(bytes, format, converter, temp, signal, kitRoot) {
  if (process.platform !== "darwin" || (converter && !path.isAbsolute(converter))) throw new Error("office_converter_unavailable");
  const directory = await fs.mkdtemp(path.join(temp, "aibert-office-"));
  try {
    const home = await fs.realpath(os.homedir());
    const privateRoots = ["Documents", "Desktop", "Downloads", ".ssh", ".aws", ".gnupg"].map((name) => path.join(home, name));
    const denyOtherFiles = privateRoots.map((root) => `(deny file-read-data (subpath ${JSON.stringify(root)}))`).join("\n");
    const source = path.join(directory, "document." + format);
    const output = path.join(directory, "document.pdf");
    await fs.writeFile(source, bytes, { mode: 0o600, flag: "wx" });
    const sandbox = "(version 1) (allow default) (deny network*)\n" + denyOtherFiles;
    if (kitRoot && !converter) {
      const kit = await fs.realpath(kitRoot);
      const cli = path.join(kit, "app", "packages", "document", "office-kit", "node_modules", "@deepseek-ai", "libreoffice-kit", "lib", "cli.js");
      await run("/usr/bin/sandbox-exec", [
        "-p", sandbox + "\n(allow file-read-data (subpath " + JSON.stringify(kit) + "))",
        path.join(kit, "bin", "node"), cli, "convert", "--input", source, "--output", output
      ], { cwd: directory, timeout: 90_000, maxBuffer: 64 * 1024, signal,
        env: { PATH: "/usr/bin:/bin", HOME: directory, TMPDIR: directory } });
    } else {
      const app = await fs.realpath(converter || "/Applications/LibreOffice.app/Contents/MacOS/soffice");
      const profile = path.join(directory, "profile");
      const fontConfig = path.join(directory, "fonts.conf");
      await fs.mkdir(path.join(profile, "user"), { recursive: true });
      await fs.writeFile(path.join(profile, "user", "registrymodifications.xcu"),
        '<?xml version="1.0" encoding="UTF-8"?><oor:items xmlns:oor="http://openoffice.org/2001/registry"><item oor:path="/org.openoffice.Office.Common/Security/Scripting"><prop oor:name="MacroSecurityLevel" oor:op="fuse"><value>3</value></prop></item></oor:items>');
      await fs.writeFile(fontConfig, OFFICE_FONTS, { mode: 0o600, flag: "wx" });
      await run("/usr/bin/sandbox-exec", [
        "-p", sandbox,
        app, `-env:UserInstallation=${pathToFileURL(profile).href}`,
        "--headless", "--norestore", "--convert-to", "pdf", "--outdir", directory, source
      ], {
        cwd: directory, timeout: 90_000, maxBuffer: 64 * 1024, signal,
        env: { ...process.env, FONTCONFIG_FILE: fontConfig, XDG_CACHE_HOME: directory }
      });
    }
    const stat = await fs.stat(output);
    if (!stat.isFile() || stat.size < 5 || stat.size > LIMIT) throw new Error("invalid_office_preview");
    const pdf = await fs.readFile(output);
    if (pdf.subarray(0, 5).toString("ascii") !== "%PDF-") throw new Error("invalid_office_preview");
    return pdf;
  } finally {
    await fs.rm(directory, { recursive: true, force: true }).catch(() => {});
  }
}

async function startPrivateOfficePreview({ token, converter, temp, vendor, validate, dshResources }) {
  const key = Buffer.from(token);
  const docs = new Map();
  const assets = new Map([
    ["/vendor/pdf.js", "pdf.mjs"], ["/vendor/pdf.worker.js", "pdf.worker.mjs"], ["/vendor/viewer.mjs", "viewer.mjs"]
  ]);
  function session(id) {
    let entry = docs.get(id);
    if (!entry && docs.size < 8) {
      let resolve;
      const ready = new Promise((finish) => { resolve = finish; });
      const controller = new AbortController();
      const expire = setTimeout(() => remove(id), 15 * 60_000);
      expire.unref();
      entry = { ready, resolve, controller, expire, uploaded: false };
      docs.set(id, entry);
    }
    return entry;
  }
  function remove(id) {
    const entry = docs.get(id);
    if (!entry) return;
    docs.delete(id);
    clearTimeout(entry.expire);
    entry.controller.abort();
    entry.resolve(null);
  }
  const service = createServer(async (request, response) => {
    const address = `127.0.0.1:${service.address().port}`;
    response.setHeader("Content-Security-Policy", POLICY);
    response.setHeader("Cache-Control", "no-store");
    response.setHeader("X-Content-Type-Options", "nosniff");
    response.setHeader("Referrer-Policy", "no-referrer");
    const send = (status, type = "text/plain; charset=utf-8", body = "") => response.writeHead(status, { "Content-Type": type }).end(body);
    if (request.headers.host !== address) return send(403);
    try {
      const url = new URL(request.url, `http://${address}`);
      if (url.host !== address) return send(403);
      const requestedKey = Buffer.from(url.searchParams.get("t") ?? "");
      if (request.method === "GET" && assets.has(url.pathname)) {
        return send(200, "text/javascript; charset=utf-8", await fs.readFile(path.join(vendor, assets.get(url.pathname))));
      }
      if (requestedKey.length !== key.length || !timingSafeEqual(requestedKey, key)) return send(403);
      if (request.method === "GET" && url.pathname === "/healthz") return send(200, "text/html; charset=utf-8", "<!doctype html><title>Office preview</title>");
      const match = /^\/p\/([0-9a-f]{32})(?:\/(events|doc))?$/.exec(url.pathname);
      if (!match) return send(404);
      const [, id, kind] = match;
      if (request.method === "DELETE" && !kind) {
        remove(id);
        return send(204);
      }
      const entry = session(id);
      if (!entry) return send(429);
      if (request.method === "POST" && !kind) {
        if (entry.uploaded) return send(409);
        entry.uploaded = true;
        try {
          const name = request.headers["x-filename"];
          const format = typeof name === "string" ? path.extname(name).slice(1).toLowerCase() : "";
          if (!["pptx", "docx", "xlsx", "pdf"].includes(format)) {
            entry.resolve(null);
            return send(415);
          }
          const chunks = [];
          let size = 0;
          for await (const chunk of request) {
            size += chunk.length;
            if (size > LIMIT) throw new Error("office_preview_too_large");
            chunks.push(chunk);
          }
          const bytes = Buffer.concat(chunks, size);
          const pdf = format === "pdf" ? bytes : await (async () => {
            await validate(bytes, format);
            return renderOffice(bytes, format, converter, temp, entry.controller.signal, dshResources?.());
          })();
          if (pdf.subarray(0, 5).toString("ascii") !== "%PDF-") throw new Error("invalid_office_preview");
          entry.resolve(pdf);
          return send(200);
        } catch {
          entry.resolve(null);
          return send(422);
        }
      }
      if (request.method !== "GET") return send(405);
      if (!kind) return send(200, "text/html; charset=utf-8", VIEW);
      const pdf = await entry.ready;
      if (kind === "doc") return pdf ? send(200, "application/pdf", pdf) : send(422);
      const event = pdf
        ? `event: ready\ndata: ${JSON.stringify({ mime: "application/pdf", docUrl: `/p/${id}/doc?t=${encodeURIComponent(token)}` })}\n\n`
        : "event: failed\ndata: {}\n\n";
      return send(200, "text/event-stream", event);
    } catch {
      if (!response.headersSent) send(500);
      else response.destroy();
    }
  });
  await new Promise((resolve, reject) => {
    service.once("error", reject);
    service.listen(0, "127.0.0.1", resolve);
  });
  return {
    baseUrl: `http://127.0.0.1:${service.address().port}`, token,
    close() {
      for (const id of docs.keys()) remove(id);
      service.close();
      service.closeAllConnections();
    }
  };
}

module.exports = { startPrivateOfficePreview };
