#!/usr/bin/env python3
"""Create and verify an externally pinned manifest for a staged DSH runtime."""

import argparse
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import stat
import sys

SOURCE_COMMIT = "3f321d32bfe382212a8ed8785c1d47a5a300aa23"
SOURCE_ARCHIVE_SHA256 = "a46ccc2890645d0a7f3b032eeadb988769893392da5dcd48a1bc4ce0e49ba3fc"
PATCH_SHA256 = "177f2f6dd9f2dd1a42f64999af5a8f89fef00b128b7b9f72c0d316e7307a6a47"
ENTRY = "app/apps/cli/lib/bin.js"
MANIFEST = "manifest.json"
HEX = re.compile(r"^[0-9a-f]{64}$")
TARGETS = {"darwin-arm64", "windows-amd64", "linux-amd64", "linux-arm64"}


def digest(path):
    h = hashlib.sha256()
    with path.open("rb") as src:
        for block in iter(lambda: src.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()


def validate_name(name):
    if not isinstance(name, str) or not name or "\\" in name or "\x00" in name:
        raise ValueError("invalid resource path")
    path = PurePosixPath(name)
    if path.is_absolute() or any(x in (".", "..") for x in path.parts) or str(path) != name:
        raise ValueError("invalid resource path")
    return path


def inventory(root):
    root = root.resolve(strict=True)
    if not root.is_dir():
        raise ValueError("resource root must be a directory")
    records = []
    for directory, names, files in os.walk(root, followlinks=False):
        names.sort()
        files.sort()
        for name in list(names) + files:
            path = Path(directory) / name
            relative = path.relative_to(root).as_posix()
            validate_name(relative)
            mode = path.lstat().st_mode
            if stat.S_ISLNK(mode):
                target = os.readlink(path)
                if not target or os.path.isabs(target) or "\\" in target:
                    raise ValueError("invalid resource symlink")
                resolved = path.resolve(strict=True)
                if not resolved.is_relative_to(root):
                    raise ValueError("resource symlink escapes root")
                records.append({"path": relative, "type": "symlink", "target": target})
            elif stat.S_ISREG(mode):
                if relative == MANIFEST:
                    continue
                if name in (".env", ".npmrc"):
                    raise ValueError("resource includes local environment configuration")
                records.append({"path": relative, "type": "file", "sha256": digest(path)})
            elif not stat.S_ISDIR(mode):
                raise ValueError("unsupported resource entry")
    return sorted(records, key=lambda entry: entry["path"])


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate manifest key")
        result[key] = value
    return result


def create(args):
    root = Path(args.root).resolve(strict=True)
    path = root / MANIFEST
    if path.exists() or path.is_symlink():
        raise ValueError("manifest already exists")
    if args.target not in TARGETS or args.archive_sha256 != SOURCE_ARCHIVE_SHA256:
        raise ValueError("invalid target or source archive digest")
    patch = Path(args.patch)
    lockfile = Path(args.lockfile)
    if digest(patch) != PATCH_SHA256:
        raise ValueError("unexpected DSH patch digest")
    node = "bin/node.exe" if args.target.startswith("windows-") else "bin/node"
    for name in (node, ENTRY):
        path_value = root / validate_name(name)
        if not path_value.is_file() or not path_value.resolve(strict=True).is_relative_to(root):
            raise ValueError("missing or invalid runtime entry")
    payload = {
        "version": 1,
        "source_commit": SOURCE_COMMIT,
        "source_archive_sha256": args.archive_sha256,
        "patch_sha256": digest(patch),
        "lockfile_sha256": digest(lockfile),
        "node_version": args.node_version,
        "target": args.target,
        "node": node,
        "entry": ENTRY,
        "files": inventory(root),
    }
    if not re.fullmatch(r"v\d+\.\d+\.\d+", args.node_version):
        raise ValueError("invalid Node version")
    encoded = (json.dumps(payload, ensure_ascii=True, sort_keys=True, separators=(",", ":")) + "\n").encode()
    with path.open("xb") as out:
        out.write(encoded)
    print(hashlib.sha256(encoded).hexdigest())


def verify(args):
    if not HEX.fullmatch(args.manifest_sha256):
        raise ValueError("a trusted manifest SHA-256 is required")
    root = Path(args.root).resolve(strict=True)
    manifest = root / MANIFEST
    if not manifest.is_file() or manifest.is_symlink() or digest(manifest) != args.manifest_sha256:
        raise ValueError("manifest digest mismatch")
    if manifest.stat().st_size > 128 * 1024 * 1024:
        raise ValueError("manifest is too large")
    payload = json.loads(manifest.read_text(encoding="utf-8"), object_pairs_hook=unique_object)
    expected = {"version", "source_commit", "source_archive_sha256", "patch_sha256", "lockfile_sha256", "node_version", "target", "node", "entry", "files"}
    if not isinstance(payload, dict) or set(payload) != expected or payload["version"] != 1 or payload["source_commit"] != SOURCE_COMMIT:
        raise ValueError("invalid manifest metadata")
    if payload["source_archive_sha256"] != SOURCE_ARCHIVE_SHA256 or payload["patch_sha256"] != PATCH_SHA256:
        raise ValueError("unexpected source or patch digest")
    if payload["target"] not in TARGETS or payload["entry"] != ENTRY:
        raise ValueError("invalid runtime target")
    node = "bin/node.exe" if payload["target"].startswith("windows-") else "bin/node"
    if payload["node"] != node or not re.fullmatch(r"v\d+\.\d+\.\d+", payload["node_version"]):
        raise ValueError("invalid runtime metadata")
    for field in ("source_archive_sha256", "patch_sha256", "lockfile_sha256"):
        if not isinstance(payload[field], str) or not HEX.fullmatch(payload[field]):
            raise ValueError("invalid provenance digest")
    if payload["files"] != inventory(root):
        raise ValueError("resource inventory mismatch")
    for name in (node, ENTRY):
        p = root / validate_name(name)
        if not p.is_file() or not p.resolve(strict=True).is_relative_to(root):
            raise ValueError("missing or invalid runtime entry")
    print("verified")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    c = commands.add_parser("create")
    c.add_argument("root")
    c.add_argument("--patch", required=True)
    c.add_argument("--lockfile", required=True)
    c.add_argument("--archive-sha256", required=True)
    c.add_argument("--node-version", required=True)
    c.add_argument("--target", required=True)
    v = commands.add_parser("verify")
    v.add_argument("root")
    v.add_argument("--manifest-sha256", required=True)
    args = parser.parse_args()
    try:
        create(args) if args.command == "create" else verify(args)
    except (OSError, ValueError, TypeError, json.JSONDecodeError) as exc:
        print(f"DSH manifest: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
