#!/usr/bin/env python3
"""Materialize the pinned DSH source and patch in a new isolated directory."""

import argparse
import hashlib
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile

SOURCE = "3f321d32bfe382212a8ed8785c1d47a5a300aa23"
ARCHIVE_SHA256 = "a46ccc2890645d0a7f3b032eeadb988769893392da5dcd48a1bc4ce0e49ba3fc"
PATCH_SHA256 = "177f2f6dd9f2dd1a42f64999af5a8f89fef00b128b7b9f72c0d316e7307a6a47"
PATCH = Path(__file__).parent / "patches/experimental-invocation-id.patch"


def sha256(path):
    h = hashlib.sha256()
    with path.open("rb") as f:
        for block in iter(lambda: f.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repository", required=True, help="read-only Git repository containing the fixed commit")
    parser.add_argument("--output", required=True, help="new source directory; must not exist")
    args = parser.parse_args()
    source = Path(args.repository).resolve(strict=True)
    destination = Path(args.output).absolute()
    if destination.exists() or destination.is_symlink():
        raise ValueError("output already exists")
    if not destination.parent.is_dir():
        raise ValueError("output parent does not exist")
    if destination.parent.resolve().is_relative_to(source):
        raise ValueError("output cannot be inside the source repository")
    if sha256(PATCH) != PATCH_SHA256:
        raise ValueError("patch digest mismatch")
    with tempfile.TemporaryDirectory(prefix="aibertai-dsh-source-", dir=destination.parent) as stage:
        stage = Path(stage)
        archive = stage / "source.tar"
        with archive.open("xb") as out:
            subprocess.run(["git", "-C", str(source), "archive", "--format=tar", SOURCE], stdout=out, check=True)
        if sha256(archive) != ARCHIVE_SHA256:
            raise ValueError("fixed source archive digest mismatch")
        tree = stage / "tree"
        tree.mkdir()
        with tarfile.open(archive, "r:") as tar:
            tar.extractall(tree, filter="data")
        subprocess.run(["git", "apply", "--check", str(PATCH.resolve())], cwd=tree, check=True)
        subprocess.run(["git", "apply", str(PATCH.resolve())], cwd=tree, check=True)
        # The staging directory is a sibling, so this is an atomic same-volume move.
        os.rename(tree, destination)
    print(f"prepared {destination}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, tarfile.TarError, subprocess.CalledProcessError) as exc:
        print(f"DSH source preparation failed: {exc}", file=sys.stderr)
        raise SystemExit(1)
