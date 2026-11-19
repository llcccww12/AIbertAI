"use strict";
const fs = require("node:fs");
const path = require("node:path");

const markers = [".aibertai-data.json", ".aibertai-incomplete", ".aibertai-deleting"];
const legacy = (name) => name.replace(/^\.aibertai-/, ".monkeyai-");

function fileInfo(name) {
  const info = fs.lstatSync(name, { throwIfNoEntry: false });
  if (info && !info.isFile()) throw new Error("unsafe data marker: " + name);
  return info;
}

function markerPath(dir, name) {
  const current = path.join(dir, name);
  const previous = path.join(dir, legacy(name));
  if (fileInfo(current)) {
    if (fileInfo(previous)) throw new Error("conflicting data markers: " + current);
    return current;
  }
  return fileInfo(previous) ? previous : current;
}

function migrate(dir, rollback = false) {
  if (!path.isAbsolute(dir)) throw new Error("an absolute data directory is required");
  // Check every marker before moving any of them. Re-running resumes an interrupted rename.
  const moves = markers.map((name) => [
    markerPath(dir, name),
    path.join(dir, rollback ? legacy(name) : name),
  ]);
  for (const [from, to] of moves) {
    if (from !== to && fileInfo(from)) fs.renameSync(from, to);
  }
}

function isMarker(name, current) {
  return name === current || name === legacy(current);
}

module.exports = { markers, legacy, isMarker, markerPath, migrate };
if (require.main === module) {
  const [action, dir] = process.argv.slice(2);
  if (process.argv.length !== 4 || !["migrate", "rollback"].includes(action)) {
    console.error("Usage: node desktop-profile-brand.cjs migrate|rollback /absolute/data/directory");
    process.exitCode = 2;
  } else {
    migrate(dir, action === "rollback");
  }
}
