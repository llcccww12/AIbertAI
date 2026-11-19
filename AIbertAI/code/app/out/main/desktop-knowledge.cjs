"use strict";

const { execFile } = require("node:child_process");
const path = require("node:path");

const readMethods = new Set([
  "_monkeyai/workspace/stat",
  "_monkeyai/kb/list",
  "_monkeyai/kb/get",
  "_monkeyai/kb/permissions/list",
  "_monkeyai/kb/documents/list",
  "_monkeyai/kb/documents/get",
  "_monkeyai/resources/connectors/tools",
]);
const writeMethods = new Set([
  "_monkeyai/devices/connect",
  "_monkeyai/kb/save",
  "_monkeyai/kb/delete",
  "_monkeyai/kb/permissions/update",
  "_monkeyai/kb/documents/restudy",
  "_monkeyai/kb/documents/save",
  "_monkeyai/kb/documents/move",
  "_monkeyai/kb/documents/delete",
  "_monkeyai/kb/documents/batch",
]);

function handles(method) {
  return readMethods.has(method) || writeMethods.has(method);
}

function request(executable, scope, method, params) {
  if (!path.isAbsolute(executable) || !handles(method)) return Promise.resolve(undefined);
  const allowReadFallback = readMethods.has(method);
  return new Promise((resolve, reject) => {
    const unavailable = () => {
      if (allowReadFallback) resolve(undefined);
      else reject(new Error("knowledge_request_failed"));
    };
    const child = execFile(executable, ["desktop-knowledge"], {
      timeout: 20000,
      windowsHide: true,
      maxBuffer: 5 * 1024 * 1024,
    }, (error, stdout) => {
      if (error) {
        if (error.code === "ENOENT" || error.code === "EACCES") return resolve(undefined);
        return unavailable();
      }
      let reply;
      try {
        reply = JSON.parse(stdout);
      } catch {
        return unavailable();
      }
      if (Object.hasOwn(reply, "result")) return resolve(reply.result);
      // An expired access token can be refreshed by the current account runtime.
      if (reply?.error === "desktop_credential_unavailable") return resolve(undefined);
      if (allowReadFallback && method !== "_monkeyai/resources/connectors/tools") return resolve(undefined);
      // A write may already have reached the server. Never send it again through the old runtime.
      const reason = /^[a-z][a-z0-9_]{0,79}$/.test(reply?.error) ? reply.error : "knowledge_request_failed";
      reject(Object.assign(new Error(reason), { data: { reason } }));
    });
    child.stdin.on("error", () => {});
    child.stdin.end(JSON.stringify({ ...scope, method, params }));
  });
}

module.exports = { handles, request };
