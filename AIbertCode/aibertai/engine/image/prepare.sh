#!/usr/bin/env bash
set -euo pipefail

[[ $# == 1 ]] || { printf 'Usage: bash engine/image/prepare.sh OUTPUT_DIRECTORY\n' >&2; exit 2; }
[[ ! -e "$1" && ! -L "$1" ]] || { printf 'Output already exists: %s\n' "$1" >&2; exit 1; }
source_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT

docker buildx build --platform linux/amd64,linux/arm64 --provenance=false --sbom=false \
  --output "type=oci,dest=$work/base.tar" "$source_dir"
mkdir "$work/layout"
tar -xf "$work/base.tar" -C "$work/layout"
python3 - "$work/layout" "$1" <<'IMAGE'
import hashlib
import json
from pathlib import Path
import re
import shutil
import sys

layout, output = map(Path, sys.argv[1:])
descriptor, = json.loads((layout / "index.json").read_bytes())["manifests"]
def verified(item):
    digest = item["digest"]
    assert re.fullmatch("sha256:[0-9a-f]{64}", digest)
    data = (layout / "blobs" / "sha256" / digest[7:]).read_bytes()
    assert len(data) == item["size"] and hashlib.sha256(data).hexdigest() == digest[7:]
    return data
index = json.loads(verified(descriptor))
assert sorted((m["platform"]["os"], m["platform"]["architecture"]) for m in index["manifests"]) == [("linux", "amd64"), ("linux", "arm64")]
for platform in index["manifests"]:
    manifest = json.loads(verified(platform))
    config = json.loads(verified(manifest["config"]))
    assert config["architecture"] == platform["platform"]["architecture"] and config["os"] == "linux"
    assert config["config"]["User"] == "1000:1000"
    for layer in manifest["layers"]:
        verified(layer)
reference = "local.aibertai.invalid/aibertai/cloud-base@" + descriptor["digest"]
key = hashlib.sha256(reference.encode()).hexdigest()
output.mkdir(parents=True)
shutil.move(layout, output / key)
(output / "catalog.json").write_text(json.dumps([reference]) + "\n")
print(reference)
IMAGE
