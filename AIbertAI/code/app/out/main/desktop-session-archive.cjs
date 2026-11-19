"use strict";

const { execFile } = require("node:child_process");
const { isAbsolute } = require("node:path");

const UUID = /^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/;
const finished = new Set(["idle", "error"]);

function read(executable, profile, sessionId, kind, parameter) {
  if (!executable || !isAbsolute(executable) || !isAbsolute(profile) || !UUID.test(sessionId)) {
    return Promise.resolve(undefined);
  }
  return new Promise((resolve) => {
    execFile(executable, ["read-session-archive", profile, sessionId], {
      timeout: 3000,
      windowsHide: true,
      maxBuffer: 5 * 1024 * 1024,
    }, (error, stdout) => {
      if (error) return resolve(undefined);
      try {
        const { session, replay } = JSON.parse(stdout);
        if (session.id !== sessionId || !finished.has(session.status) ||
            !Number.isSafeInteger(session.view_seq) || session.view_seq < 0 ||
            session.replay_applied_seq !== session.last_event_seq || !Array.isArray(replay)) {
          return resolve(undefined);
        }
        if (kind === "messages" && replay.length <= parameter) {
          const messages = replay.map((row) => row.message);
          if (messages.some((message) => typeof message?.id !== "string" || !Array.isArray(message.parts))) return resolve(undefined);
          if (new Set(messages.map((message) => message.id)).size !== messages.length) return resolve(undefined);
          const toolCalls = Object.assign(Object.create(null), ...replay.map((row) => row.toolCalls));
          const eventSeq = session.view_seq;
          for (const key of ["schema_version", "replay_version", "replay_generation", "replay_applied_seq", "replay_bytes", "last_event_seq", "events_offset", "message_count", "view_seq", "engine_session_id"]) {
            delete session[key];
          }
          return resolve({ session, messages, toolCalls, eventSeq, olderCursor: null, newerCursor: null });
        }
        if (kind === "tool_output") {
          for (let index = replay.length - 1; index >= 0; index--) {
            const calls = replay[index].toolCalls;
            if (!calls || !Object.hasOwn(calls, parameter)) continue;
            const call = calls[parameter];
            if (call.id === parameter && call.status === "completed" && typeof call.output === "string" && !call.outputTruncated &&
                (call.outputLength === undefined || call.outputLength === call.output.length)) {
              return resolve({ output: call.output, length: call.output.length });
            }
            break;
          }
        }
      } catch {
        // The active runtime still handles any format the archive reader cannot use.
      }
      resolve(undefined);
    });
  });
}

module.exports = { read };
