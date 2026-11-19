#!/bin/bash
set -euo pipefail

if [[ $# -lt 5 || $# -gt 6 ]]; then
    echo "Usage: $0 mac-arm64|win-x64 BACKEND_ORIGIN MODEL_UUID MODEL_NAME OUTPUT_DIRECTORY [ELECTRON_WIN_ZIP]" >&2
    exit 2
fi
target="$1"
server="$2"
model="$3"
name="$4"
output="$5"
electron_zip="${6:-}"
packaging="$(cd -- "$(dirname -- "$0")" && pwd)"
project="$(cd -- "$packaging/.." && pwd)"
stage="$(mktemp -d /tmp/aibert-acceptance.XXXXXX)"
trap 'rm -rf -- "$stage"' EXIT

python3 - "$server" "$model" <<'PY'
import re
import sys
from urllib.parse import urlsplit

raw, model = sys.argv[1:]
u = urlsplit(raw)
if not u.hostname or u.username is not None or any((u.path, u.query, u.fragment)) or any(c.isspace() for c in raw):
    raise SystemExit("Use an origin with no credentials, path, query, or fragment")
if u.scheme != "https" and not (u.scheme == "http" and u.hostname == "127.0.0.1"):
    raise SystemExit("Use HTTPS or loopback HTTP")
if not re.fullmatch(r"[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}",model):
    raise SystemExit("Specify the model UUID from the private backend")
PY

case "$target" in
    mac-arm64)
        GO_ARCH=arm64
        GO_OS=darwin
        app="$stage/AIbertAI Runtime Acceptance.app"
        ditto "$project/.local-runtime/electron/Electron.app" "$app"
        resources="$app/Contents/Resources"
        binary="$resources/aibertai-native"
        artifact="AIbertAI-runtime-acceptance-macOS-arm64.dmg"
        ;;
    win-x64)
        GO_ARCH=amd64
        GO_OS=windows
        if [[ ! -f "$electron_zip" ]]; then
            echo "Supply the official Electron Windows x64 ZIP" >&2
            exit 2
        fi
        electron_dir="$(cd -- "$(dirname -- "$electron_zip")" && pwd)"
        electron_file="${electron_zip##*/}"
        (cd -- "$electron_dir" && awk -v f="$electron_file" '$2==f || $2=="*" f' SHASUMS256.txt | shasum -a 256 -c)
        app="$stage/AIbertAI Runtime Acceptance"
        mkdir "$app"
        ditto -x -k "$electron_zip" "$app"
        mv "$app/electron.exe" "$app/AIbertAI Runtime Acceptance.exe"
        resources="$app/resources"
        binary="$resources/aibertai-native.exe"
        artifact="AIbertAI-runtime-acceptance-Windows-x64.zip"
        ;;
    *)
        echo "Unsupported target: $target" >&2
        exit 2
        ;;
esac

mkdir -p -- "$output" "$resources/app"
output="$(cd -- "$output" && pwd)"
artifact="$output/$artifact"
if [[ -e "$artifact" ]]; then
    echo "Output already exists: $artifact" >&2
    exit 2
fi

rm -f -- "$resources/default_app.asar"
cp "$packaging/acceptance/"* "$resources/app/"
python3 - "$resources/app/deployment.json" "$server" "$name" <<'PY'
import json
from pathlib import Path
import sys
Path(sys.argv[1]).write_text(json.dumps({"backendOrigin":sys.argv[2],"modelName":sys.argv[3]},ensure_ascii=False)+"\n")
PY
(
  cd "$project/native"
  export GOOS="$GO_OS" GOARCH="$GO_ARCH" CGO_ENABLED=0
  go build -mod=readonly -trimpath -buildvcs=false -ldflags "-X main.backendOrigin=$server -X main.modelID=$model" -o "$binary" .
)

python3 - "$resources/app" "$binary" <<'PY2'
from pathlib import Path
import sys

app, binary = map(Path, sys.argv[1:])
# Keep retired host identifiers only in this build-time deny check.
retired = (b"baizhi" + b".cloud", b"monkeycode-ai" + b".com")
for file in [*app.iterdir(), binary]:
    contents = file.read_bytes().lower()
    if any(host in contents or host.decode().encode("utf-16le") in contents for host in retired):
        raise SystemExit(f"Retired network host found in {file.name}")
PY2

if [[ "$target" == mac-arm64 ]]; then
    python3 - "$app/Contents/Info.plist" <<'PY'
import plistlib
from pathlib import Path
import sys
info = Path(sys.argv[1])
data = plistlib.loads(info.read_bytes())
data.pop("ElectronAsarIntegrity",None)
data.pop("CFBundleURLTypes",None)
data.update(CFBundleIdentifier="com.aibertai.runtime-acceptance",CFBundleName="AIbertAI Runtime Acceptance",
            CFBundleDisplayName="AIbertAI 私有部署验收",CFBundleIconFile="AIbertAI.icns")
info.write_bytes(plistlib.dumps(data))
PY
    cp "$project/AIbertAI.app/Contents/Resources/AIbertAI.icns" "$resources/"
    codesign --force --deep --sign - "$app"
    codesign --verify --strict --deep "$app"
    hdiutil create -volname "AIbertAI Runtime Acceptance" -srcfolder "$app" -format UDZO "$stage/output.dmg"
    hdiutil verify "$stage/output.dmg"
    mv "$stage/output.dmg" "$artifact"
else
    ditto --norsrc -c -k --keepParent "$app" "$stage/output.zip"
    unzip -tq "$stage/output.zip"
    mv "$stage/output.zip" "$artifact"
fi
(cd "$output" && shasum -a 256 "${artifact##*/}" > "${artifact##*/}.sha256")
