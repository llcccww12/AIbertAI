#!/usr/bin/env python3
"""Stage or verify the pinned FFmpeg.wasm cores used by the local video editor."""

import argparse
import hashlib
from pathlib import Path
import shutil
import tarfile
import tempfile
from urllib.request import urlopen

ROOT = Path(__file__).resolve().parents[1]
DESTINATION = ROOT / "code/web/assets/media-vendor/ffmpeg"
VERSION = "0.12.9"
ARCHIVES = {
    "core": (
        "core-st",
        "c8efbf9a7fd9bdfc9bcc9a100f25c9faed710f6e2322f3c7eb2593572b2e0e97",
        {
            "ffmpeg-core.js": "67a48f11645f85439f3fde4f2119042c16b374b910206b7a7a24f342e28dcae3",
            "ffmpeg-core.wasm": "9f57947a5bd530d8f00c5b3f2cb2a3492faa7e5d823315342d6a8656d0a6b7b7",
        },
    ),
    "core-mt": (
        "core-mt",
        "84a9da33933bcd7c8f9dee931964ba79d0b90eff7db20ce0b947edf2f749bca6",
        {
            "ffmpeg-core.js": "270a2e6ff945e173238610669a3f7132df5f9c52698a9bf708cf5c2ab6bda0de",
            "ffmpeg-core.wasm": "be2c97605366b78f3f13e21b52e81a55a79e1f29c133b03a68ec187b1a2ec41a",
            "ffmpeg-core.worker.js": "f77898d631dc010b45c29c23cb4379c611a7d7b131bf591d08a656bb729a4ca3",
        },
    ),
}


def verify(path, expected):
    with path.open("rb") as stream:
        actual = hashlib.file_digest(stream, "sha256").hexdigest()
    if actual != expected:
        raise ValueError(f"Unexpected SHA-256 for {path}")


def download(url, destination, checksum):
    with tempfile.TemporaryDirectory(dir=destination.parent) as temp:
        staging = Path(temp) / destination.name
        with urlopen(url, timeout=60) as source, staging.open("wb") as stream:
            shutil.copyfileobj(source, stream)
        verify(staging, checksum)
        staging.replace(destination)


def prepare(cache, fetch):
    cache.mkdir(parents=True, exist_ok=True)
    DESTINATION.mkdir(parents=True, exist_ok=True)
    for package, (folder, archive_checksum, files) in ARCHIVES.items():
        filename = f"{package}-{VERSION}.tgz"
        archive = cache / filename
        if not archive.exists() and fetch:
            download(f"https://registry.npmjs.org/@ffmpeg/{package}/-/{filename}", archive, archive_checksum)
        verify(archive, archive_checksum)
        dest = DESTINATION / folder
        dest.mkdir(exist_ok=True)
        with tarfile.open(archive) as source, tempfile.TemporaryDirectory(dir=dest) as temp:
            for name, checksum in files.items():
                member = source.getmember(f"package/dist/esm/{name}")
                if not member.isfile():
                    raise ValueError(f"Unexpected archive entry {member.name}")
                staging = Path(temp) / name
                with source.extractfile(member) as entry, staging.open("wb") as target:
                    shutil.copyfileobj(entry, target)
                verify(staging, checksum)
                staging.replace(dest / name)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    actions = parser.add_mutually_exclusive_group()
    actions.add_argument("--check", action="store_true")
    actions.add_argument("--download", action="store_true")
    parser.add_argument("--archives", type=Path, default=ROOT / ".local-runtime/media-dependencies")
    args = parser.parse_args()
    try:
        if not args.check:
            prepare(args.archives, args.download)
        for folder, _, files in ARCHIVES.values():
            for name, checksum in files.items():
                verify(DESTINATION / folder / name, checksum)
    except (OSError, ValueError, KeyError, tarfile.TarError) as error:
        parser.exit(1, f"Cannot verify the offline video resources: {error}\n")


if __name__ == "__main__":
    main()
