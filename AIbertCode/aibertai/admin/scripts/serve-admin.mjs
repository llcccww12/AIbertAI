import { createReadStream } from "node:fs"
import { stat } from "node:fs/promises"
import http from "node:http"
import https from "node:https"
import { extname, resolve, sep } from "node:path"
import { fileURLToPath } from "node:url"

const dist = fileURLToPath(new URL("../dist", import.meta.url))
const backend = new URL(process.env.AIBERTAI_DEV_BACKEND_URL ?? "http://127.0.0.1:8080")
const listenHost = process.env.ADMIN_LISTEN_HOST ?? "0.0.0.0"
const listenPort = Number(process.env.ADMIN_LISTEN_PORT ?? 5173)
if (!["http:", "https:"].includes(backend.protocol) || backend.username ||
    backend.password || backend.pathname !== "/" || backend.search || backend.hash) {
  throw new Error("AIBERTAI_DEV_BACKEND_URL must be an HTTP or HTTPS origin")
}
const transport = backend.protocol === "https:" ? https : http
const proxyRoots = new Set(["api", "oauth", "mcp", "v1", "endpoints", ".well-known", "healthz", "readyz"])

const mime = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".svg": "image/svg+xml",
  ".ico": "image/x-icon",
  ".woff": "font/woff",
  ".woff2": "font/woff2",
  ".map": "application/json; charset=utf-8",
  ".txt": "text/plain; charset=utf-8",
}

function isProxyPath(path) {
  return path.startsWith("/") && proxyRoots.has(path.split(/[/?]/, 2)[1])
}

function backendRequest(req) {
  return transport.request({
    protocol: backend.protocol,
    hostname: backend.hostname.replace(/^\[|\]$/g, ""),
    port: backend.port,
    path: req.url,
    method: req.method,
    headers: { ...req.headers, host: backend.host },
  })
}

function proxyHTTP(req, res) {
  const upstream = backendRequest(req)
  upstream.on("response", (response) => {
    res.writeHead(response.statusCode, response.headers)
    response.on("error", () => res.destroy())
    response.pipe(res)
  })
  upstream.on("error", () => {
    if (res.headersSent) res.destroy()
    else res.writeHead(502).end("Bad Gateway")
  })
  res.on("close", () => upstream.destroy())
  req.pipe(upstream)
}

async function statFile(file) {
  try {
    return await stat(file)
  } catch (error) {
    if (error.code !== "ENOENT" && error.code !== "ENOTDIR") throw error
  }
}

// 更新包有数百 MB，员工端靠 Content-Length 算下载进度、靠 Range 断点续传。
// 只支持单段范围；无法满足时回 416，语法不认识时按整文件返回。
function parseRange(header, size) {
  const match = /^bytes=(\d*)-(\d*)$/.exec(String(header ?? "").trim())
  if (!match) return undefined
  const [, first, last] = match
  if (!first && !last) return undefined
  let start
  let end
  if (!first) {
    const suffix = Number(last)
    if (suffix === 0) return null
    start = Math.max(0, size - suffix)
    end = size - 1
  } else {
    start = Number(first)
    end = last ? Math.min(Number(last), size - 1) : size - 1
  }
  if (start > end || start >= size) return null
  return { start, end }
}

async function serveStatic(req, res) {
  const pathname = decodeURIComponent(req.url.split("?")[0])
  let file = resolve(dist, "." + pathname)
  if (file !== dist && !file.startsWith(dist + sep)) {
    res.writeHead(404).end()
    return
  }
  let info = await statFile(file)
  if (info?.isDirectory()) {
    file = resolve(file, "index.html")
    info = await statFile(file)
  }
  if (!info?.isFile()) {
    file = resolve(dist, "index.html")
    info = await statFile(file)
    if (!info?.isFile()) {
      res.writeHead(404).end()
      return
    }
  }
  const type = mime[extname(file)] ?? "application/octet-stream"
  const size = info.size
  const range = parseRange(req.headers.range, size)
  if (range === null) {
    res.writeHead(416, { "Content-Range": `bytes */${size}`, "Accept-Ranges": "bytes" }).end()
    return
  }
  const start = range ? range.start : 0
  const end = range ? range.end : size - 1
  const length = size === 0 ? 0 : end - start + 1
  const headers = {
    "Content-Type": type,
    "Cache-Control": "no-cache",
    "Accept-Ranges": "bytes",
    "Content-Length": length,
  }
  if (range) headers["Content-Range"] = `bytes ${start}-${end}/${size}`
  res.writeHead(range ? 206 : 200, headers)
  if (req.method === "HEAD" || length === 0) {
    res.end()
    return
  }
  const stream = createReadStream(file, { start, end })
  res.on("close", () => stream.destroy())
  stream.on("error", () => res.destroy())
  stream.pipe(res)
}

const server = http.createServer((req, res) => {
  if (isProxyPath(req.url)) {
    proxyHTTP(req, res)
  } else {
    serveStatic(req, res).catch((error) => res.writeHead(error instanceof URIError ? 400 : 500).end())
  }
})

function writeUpgradeResponse(client, response, upgrade) {
  let headers = `HTTP/1.1 ${response.statusCode} ${response.statusMessage}\r\n`
  for (let i = 0; i < response.rawHeaders.length; i += 2) {
    const [name, value] = response.rawHeaders.slice(i, i + 2)
    // Node decodes chunked HTTP responses. Rejections are framed by connection close.
    if (!upgrade && /^(connection|transfer-encoding)$/i.test(name)) continue
    headers += `${name}: ${value}\r\n`
  }
  if (!upgrade) headers += "Connection: close\r\n"
  client.write(headers + "\r\n")
}

server.on("upgrade", (req, client, head) => {
  if (!isProxyPath(req.url) || req.headers.upgrade?.toLowerCase() !== "websocket") {
    client.end("HTTP/1.1 404 Not Found\r\nConnection: close\r\n\r\n")
    return
  }
  const upstream = backendRequest(req)
  client.on("error", () => upstream.destroy())
  client.on("close", () => upstream.destroy())
  upstream.on("upgrade", (response, socket, upstreamHead) => {
    socket.setTimeout(0)
    socket.on("error", () => client.destroy())
    socket.on("close", () => client.destroy())
    client.on("close", () => socket.destroy())
    writeUpgradeResponse(client, response, true)
    if (upstreamHead.length) client.write(upstreamHead)
    if (head.length) socket.write(head)
    client.pipe(socket).pipe(client)
  })
  upstream.on("response", (response) => {
    writeUpgradeResponse(client, response, false)
    response.on("error", () => client.destroy())
    response.pipe(client)
  })
  upstream.on("error", () => {
    if (!client.destroyed) client.end("HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\n\r\n")
  })
  upstream.setTimeout(10000, () => upstream.destroy())
  upstream.end()
})

server.listen(listenPort, listenHost, () => {
  console.log(`AIbertAI admin listening on ${listenHost}:${server.address().port}`)
})
