#!/usr/bin/env python3
"""Reduce a staged DSH runtime to what the SDK entry loads, as DSH Desktop ships.

DSH Desktop installs its runtime with `pnpm install --prod` and copies it with
`apps/desktop/scripts/runtime-file-policy.ts`. We build DSH from a patched
workspace instead, so this reproduces that result from the built tree:

1. Walk the production dependency graph from apps/cli (dependencies plus
   installed optional and peer dependencies), resolving names as Node does.
2. Keep each workspace package's published files (package.json `files`), as
   an npm install of that package would.
3. Delete everything else, then the runtime-file-policy omissions.

The entry path is unchanged, so the host and manifest need no changes.
"""

import argparse
import fnmatch
import json
import os
import re
import shutil
import sys
from pathlib import Path

ENTRY_PACKAGE = "apps/cli"
# Paths the AIbertAI host reads directly (dshresources.go StandardSkillsDir).
HOST_PATHS = ["packages/preset/agent-preset/skills"]
METADATA = {".bin", ".modules.yaml", ".pnpm-workspace-state-v1.json"}


def read_manifest(directory):
    try:
        with open(directory / "package.json", encoding="utf-8") as src:
            return json.load(src)
    except (OSError, ValueError):
        return {}


def resolve(start, name):
    """The real directory Node loads `name` from when required in `start`."""
    directory = start
    while True:
        candidate = directory / "node_modules" / name
        if (candidate / "package.json").exists():
            return candidate.resolve()
        if directory.parent == directory:
            return None
        directory = directory.parent


def closure(app, office):
    needed, queue, missing = set(), [(app / ENTRY_PACKAGE).resolve()], set()
    while queue:
        package = queue.pop()
        if package in needed:
            continue
        needed.add(package)
        manifest = read_manifest(package)
        required = set(manifest.get("dependencies") or {})
        optional = set(manifest.get("optionalDependencies") or {}) | set(manifest.get("peerDependencies") or {})
        for name in required | optional:
            if name.startswith("@deepseek-ai/libreoffice-kit-") and name != f"@deepseek-ai/libreoffice-kit-{office}":
                continue  # DSH Desktop ships only the selected engine
            found = resolve(package, name)
            if found is not None:
                queue.append(found)
            elif name in required - optional:
                missing.add(f"{name} (from {package.relative_to(app)})")
    if missing:
        raise SystemExit("unresolved production dependencies:\n  " + "\n  ".join(sorted(missing)))
    return needed


def office_engine(app, platform, arch):
    """DSH selectOfficeEngine: the native kit when the kit declares one, else wasm."""
    kit = resolve((app / ENTRY_PACKAGE).resolve(), "@deepseek-ai/libreoffice-kit")
    for directory in [p for p in (app / "node_modules" / ".pnpm").glob("@deepseek-ai+libreoffice-kit@*/node_modules/@deepseek-ai/libreoffice-kit")] + ([kit] if kit else []):
        native = f"{platform}-{arch}"
        return native if f"@deepseek-ai/libreoffice-kit-{native}" in (read_manifest(directory).get("optionalDependencies") or {}) else "wasm"
    return "wasm"


def published(manifest, path):
    """Whether npm publishes `path` (relative to the package root)."""
    if path == "package.json" or re.match(r"(?i)^(readme|license|licence|notice|changelog)(\.|$)", path):
        return True
    patterns = manifest.get("files")
    if patterns is None:
        return True
    main = manifest.get("main")
    if isinstance(main, str) and os.path.normpath(main) == path:
        return True
    for pattern in patterns:
        pattern = pattern.removeprefix("./").rstrip("/")
        if path == pattern or path.startswith(pattern + "/") or fnmatch.fnmatchcase(path, pattern) \
                or ("**/" in pattern and fnmatch.fnmatchcase(path, pattern.replace("**/", ""))):
            return True
    return False


def excluded(relative, platform, arch, office):
    """Port of DSH apps/desktop/scripts/runtime-file-policy.ts."""
    parts = relative.split("/")
    if any(part in METADATA for part in parts):
        return "package-manager metadata"
    name = parts[-1]
    if re.search(r"\.(?:[cm]?[jt]s|css)\.map$", name):
        return "source map"
    if re.search(r"\.d\.[cm]?ts$", name):
        return "TypeScript declaration"
    if name.endswith(".tsbuildinfo"):
        return "TypeScript build cache"
    if "node_modules" not in parts:
        return None
    package_parts = parts[len(parts) - parts[::-1].index("node_modules"):]
    width = 2 if package_parts and package_parts[0].startswith("@") else 1
    package, entry = "/".join(package_parts[:width]), "/".join(package_parts[width:])
    if package.startswith("@deepseek-ai/libreoffice-kit-") and package != f"@deepseek-ai/libreoffice-kit-{office}":
        return "LibreOffice other platform"
    if package == "@mixmark-io/domino" and re.match(r"^(?:test|\.yarn)(?:/|$)", entry):
        return "Domino development files"
    if package == "node-pty" and platform != "win32" and re.match(r"^third_party/conpty(?:/|$)", entry):
        return "node-pty Windows ConPTY sources"
    if package == "node-pty" and entry.startswith("prebuilds/"):
        target = package_parts[width + 1] if len(package_parts) > width + 1 else None
        if target is not None and target != f"{platform}-{arch}":
            return "node-pty other platform"
        if name.endswith(".pdb"):
            return "node-pty debug symbols"
    if package == "@koromix/koffi-win32-x64" and entry == "win32_x64/koffi.lib":
        return "Koffi import library"
    return None


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("resources", type=Path)
    parser.add_argument("--target", required=True, choices=["darwin-arm64", "windows-amd64"])
    args = parser.parse_args()
    platform, arch = {"darwin-arm64": ("darwin", "arm64"), "windows-amd64": ("win32", "x64")}[args.target]
    app = args.resources.resolve() / "app"
    office = office_engine(app, platform, arch)
    needed = closure(app, office)
    # Workspace packages are published by their `files`. A hoisted (Windows)
    # install copies them into node_modules, so match them by name as well.
    names = {read_manifest(m.parent).get("name") for m in [*app.glob("packages/*/*/package.json"), *app.glob("apps/*/package.json"), *app.glob("vendor/*/package.json")]}
    workspace = {p for p in needed if "node_modules" not in p.relative_to(app).parts or read_manifest(p).get("name") in names}
    host_paths = [app / path for path in HOST_PATHS]
    manifests = {p: read_manifest(p) for p in workspace}
    keep_dirs = {parent for package in [*needed, *host_paths] for parent in [package, *package.parents] if parent.is_relative_to(app)}
    removed = 0
    for directory, names, files in os.walk(app, topdown=True, followlinks=False):
        current = Path(directory)
        if any(current == h or current.is_relative_to(h) for h in host_paths):
            continue
        owner = next((p for p in [current, *current.parents] if p in needed), None)
        for name in list(names) + files:
            path = current / name
            if path.is_symlink():
                continue  # dropped below if its target goes
            if path in keep_dirs or path.is_dir() and (name == "node_modules" or name.startswith("@") and current.name == "node_modules"):
                keep = True  # links inside are settled below
            elif owner is None:
                keep = False
            else:
                inner = path.relative_to(owner)
                if "node_modules" in inner.parts:
                    keep = False  # a nested install not on any needed path
                elif owner in workspace:
                    keep = path.is_dir() or published(manifests[owner], inner.as_posix())
                else:
                    keep = True
            if keep and path.is_file() and excluded(path.relative_to(app).as_posix(), platform, arch, office):
                keep = False
            if not keep:
                shutil.rmtree(path) if path.is_dir() else path.unlink()
                removed += 1
                if name in names:
                    names.remove(name)
    for directory, names, files in os.walk(app, topdown=False, followlinks=False):
        for name in names + files:
            path = Path(directory) / name
            if path.is_symlink() and (not path.exists() or excluded(path.relative_to(app).as_posix(), platform, arch, office)):
                path.unlink()
        if directory != str(app) and not os.listdir(directory):
            os.rmdir(directory)
    lost = [p for p in needed if not (p / "package.json").exists()] + [p for p in host_paths if not p.is_dir()]
    if lost:
        raise SystemExit("slimming removed needed packages: " + ", ".join(map(str, lost)))
    print(f"kept {len(needed)} packages ({len(workspace)} from the workspace); removed {removed} entries", file=sys.stderr)


if __name__ == "__main__":
    main()
