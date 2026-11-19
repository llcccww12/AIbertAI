"use strict";

const { spawn } = require("node:child_process");
const { isAbsolute } = require("node:path");
const { Readable } = require("node:stream");

async function request(executable, scope, address, options) {
  if (!isAbsolute(executable)) throw new Error("file_transfer_failed");
  const headers = new Headers(options.headers);
  const match = /^Bearer ([A-Za-z0-9_-]{32,256})$/.exec(headers.get("Authorization") ?? "");
  if (!match) throw new Error("invalid_file_transfer");
  options.signal?.throwIfAborted();
  const child = spawn(executable, ["desktop-file-transfer"], { windowsHide: true, stdio: ["pipe", "pipe", "ignore"] });
  const done = new Promise((resolve) => {
    child.once("error", () => resolve(false));
    child.once("close", (code) => resolve(code === 0));
  });
  const abort = () => child.kill();
  options.signal?.addEventListener("abort", abort, { once: true });
  void done.then(() => options.signal?.removeEventListener("abort", abort));
  if (options.signal?.aborted) abort();
  child.stdin.on("error", () => {});
  child.stdin.end(JSON.stringify({
    server: scope.serverUrl,
    machineId: scope.machineId,
    filesOrigin: scope.filesOrigin,
    url: address,
    method: options.method,
    ticket: match[1],
    range: headers.get("Range") ?? undefined,
    sessionId: headers.get("Upload-Session-ID") ?? undefined,
    offset: Number(headers.get("Upload-Offset") ?? 0),
    chunk: options.body ? Buffer.from(options.body).toString("base64") : undefined,
  }));

  const iterator = child.stdout[Symbol.asyncIterator]();
  let prefix = Buffer.alloc(0);
  try {
    for (;;) {
      const chunk = await iterator.next();
      if (chunk.done) throw new Error("file_transfer_failed");
      const newline = chunk.value.indexOf(10);
      if (prefix.length + (newline < 0 ? chunk.value.length : newline) > 32768) throw new Error("invalid_file_transfer");
      if (newline < 0) {
        prefix = Buffer.concat([prefix, chunk.value]);
        continue;
      }
      const meta = JSON.parse(Buffer.concat([prefix, chunk.value.subarray(0, newline)]));
      const head = new Headers();
      for (const [key, values] of Object.entries(meta.headers)) {
        for (const value of values) head.append(key, value);
      }
      if (options.method !== "GET" || ![200, 206].includes(meta.status)) {
        for (;;) {
          if ((await iterator.next()).done) break;
        }
        if (!(await done)) throw new Error("file_transfer_failed");
        return new Response(null, { status: meta.status, headers: head });
      }
      const remainder = chunk.value.subarray(newline + 1);
      const stream = Readable.from((async function* () {
        try {
          if (remainder.length) yield remainder;
          for (;;) {
            const next = await iterator.next();
            if (next.done) break;
            yield next.value;
          }
          if (!(await done)) throw new Error("file_transfer_failed");
        } finally {
          abort();
          await iterator.return();
        }
      })(), { objectMode: false });
      stream.once("close", abort);
      return new Response(Readable.toWeb(stream), { status: meta.status, headers: head });
    }
  } catch (error) {
    abort();
    throw error;
  }
}

module.exports = { request };
