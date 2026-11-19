#!/usr/bin/env python3
"""Assemble the two host installer architectures into a versioned release."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import sys

engine = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--output", required=True, type=Path)
parser.add_argument("--version", default="")
args = parser.parse_args()

version = args.version
if not version:
    digest = hashlib.sha256(subprocess.check_output(["go", "version"], cwd=engine).split()[2])
    sources = [*engine.rglob("*.go"), engine / "go.mod", engine / "go.sum",
               engine / "scripts/package.py", Path(__file__).resolve()]
    for path in sorted(sources):
        digest.update(path.relative_to(engine).as_posix().encode() + b"\0")
        digest.update(hashlib.sha256(path.read_bytes()).digest())
    version = "source-" + digest.hexdigest()[:16]

output = args.output.resolve()
for arch in ("amd64", "arm64"):
    subprocess.run([sys.executable, str(engine / "scripts/package.py"),
                    "--version", version, "--arch", arch, "--output", str(output)], check=True)
(output / "latest.json").write_text(json.dumps({"version": version}) + "\n")
