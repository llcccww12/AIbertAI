import assert from "node:assert/strict"
import { spawn } from "node:child_process"
import { createHash } from "node:crypto"
import { once } from "node:events"
import { copyFile, mkdir, mkdtemp, rm, writeFile } from "node:fs/promises"
import http from "node:http"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { createInterface } from "node:readline"
import test from "node:test"

test("deployment entry serves the admin UI and proxies the device bridge", { timeout: 10000 }, async (t) => {
  const dir = await mkdtemp(join(tmpdir(), "aibert-admin-proxy-"))
  t.after(() => rm(dir, { recursive: true, force: true }))
  await mkdir(join(dir, "dist"))
  await mkdir(join(dir, "scripts"))
  await writeFile(join(dir, "dist", "index.html"), "<title>Test admin</title>")
  const script = join(dir, "scripts", "serve-admin.mjs")
  await copyFile(new URL("../scripts/serve-admin.mjs", import.meta.url), script)

  const bridge = "/api/v1/endpoint-bridge?channel_kind=control&tunnel_version=1"
  const backend = http.createServer((req, res) => {
    if (req.url === "/readyz") res.end("ready")
    else res.writeHead(404).end()
  })
  backend.on("upgrade", (req, socket) => {
    if (req.headers.authorization !== "Bearer fixture") {
      socket.end("HTTP/1.1 401 Unauthorized\r\nTransfer-Encoding: chunked\r\n\r\n6\r\ndenied\r\n0\r\n\r\n")
      return
    }
    const accept = createHash("sha1")
      .update(req.headers["sec-websocket-key"] + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11")
      .digest("base64")
    socket.write("HTTP/1.1 101 Switching Protocols\r\n" +
      "Upgrade: websocket\r\nConnection: Upgrade\r\n" +
      `Sec-WebSocket-Accept: ${accept}\r\n\r\n`)
    socket.once("data", () => {
      const reply = Buffer.from(req.url)
      socket.write(Buffer.concat([Buffer.from([0x81, reply.length]), reply]))
    })
    socket.on("end", () => socket.end())
  })
  backend.listen(0, "127.0.0.1")
  await once(backend, "listening")
  t.after(() => backend.close())

  const child = spawn(process.execPath, [script], {
    env: {
      ...process.env,
      AIBERTAI_DEV_BACKEND_URL: `http://127.0.0.1:${backend.address().port}`,
      ADMIN_LISTEN_HOST: "127.0.0.1",
      ADMIN_LISTEN_PORT: "0",
    },
    stdio: ["ignore", "pipe", "inherit"],
  })
  t.after(async () => {
    if (child.exitCode === null && child.signalCode === null) {
      const exited = once(child, "exit")
      child.kill()
      await exited
    }
  })
  const lines = createInterface({ input: child.stdout })
  const [line] = await once(lines, "line", { signal: t.signal })
  lines.close()
  const base = `http://127.0.0.1:${line.match(/:(\d+)$/)[1]}`

  assert.equal(await (await fetch(base + "/console/models")).text(), "<title>Test admin</title>")
  assert.equal(await (await fetch(base + "/readyz")).text(), "ready")

  const headers = {
    Connection: "Upgrade",
    Upgrade: "websocket",
    "Sec-WebSocket-Version": "13",
    "Sec-WebSocket-Key": "dGhlIHNhbXBsZSBub25jZQ==",
  }
  const denied = await fetchRejectedUpgrade(base + bridge, headers)
  assert.deepEqual(denied, { status: 401, body: "denied" })

  const req = http.get(base + bridge, { headers: { ...headers, Authorization: "Bearer fixture" } })
  const [response, socket] = await once(req, "upgrade", { signal: t.signal })
  try {
    assert.equal(response.statusCode, 101)
    assert.equal(response.headers["sec-websocket-accept"], "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=")
    // Send a single masked WebSocket frame from the client to the upstream.
    socket.write(Buffer.from([0x81, 0x81, 1, 2, 3, 4, 0x78 ^ 1]))
    const [frame] = await once(socket, "data", { signal: t.signal })
    assert.equal(frame.subarray(2).toString(), bridge)
  } finally {
    socket.destroy()
  }
})

async function fetchRejectedUpgrade(url, headers) {
  const [response] = await once(http.get(url, { headers }), "response")
  const chunks = []
  for await (const chunk of response) chunks.push(chunk)
  return { status: response.statusCode, body: Buffer.concat(chunks).toString() }
}
