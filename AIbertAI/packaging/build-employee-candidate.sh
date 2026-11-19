#!/bin/bash
set -euo pipefail

release=false
if [[ ${1:-} == --release ]]; then
  release=true
  shift
fi
if (( $# < 3 || $# > 4 )); then
  echo "Usage: $0 [--release] mac-arm64|win-x64 APPROVED_TEST_TEXT_OR_DASH OUTPUT_DIRECTORY [ELECTRON_WIN_ZIP]" >&2
  exit 2
fi

target=$1
fixture=$2
output=$3
electron_zip=${4:-}
project=$(cd -- "$(dirname -- "$0")/.." && pwd)
mac_identity=${AIBERTAI_MAC_SIGN_IDENTITY:-}
if [[ $release == true && $fixture != - ]]; then
  echo "Distribution builds cannot contain a test fixture." >&2
  exit 2
fi
if [[ $release == true && $target == mac-arm64 ]]; then
  if [[ $mac_identity != "Developer ID Application: "* ]] ||
      ! security find-identity -v -p codesigning | awk -F '"' -v identity="$mac_identity" '
        $2 == identity { found = 1 }
        END { exit !found }
      '; then
    echo "Set AIBERTAI_MAC_SIGN_IDENTITY to a valid Developer ID Application identity in the macOS keychain." >&2
    exit 2
  fi
  if ! command -v electron-osx-sign >/dev/null; then
    echo "Install the official @electron/osx-sign build tool before creating the distribution build." >&2
    exit 2
  fi
fi
python3 - "$fixture" <<'PY'
import sys
from pathlib import Path
fixture = sys.argv[1]
if fixture != "-":
    file = Path(fixture)
    if not file.is_file() or file.stat().st_size > 128 * 1024 or "\0" in file.read_text(encoding="utf-8"):
        raise SystemExit("Use a synthetic UTF-8 text fixture of at most 128 KiB, or -")
PY

if ! python3 "$project/packaging/stage-video-codecs.py" --check; then
  echo "Stage the pinned offline video codecs with packaging/stage-video-codecs.py before packaging." >&2
  exit 2
fi

mkdir -p -- "$output"
output=$(cd -- "$output" && pwd)
stage=$(mktemp -d /tmp/aibert-employee-package.XXXXXX)
trap 'rm -rf -- "$stage"' EXIT

case "$target" in
  mac-arm64)
    os=darwin
    arch=arm64
    mac_name='AIbertAI Employee Source Candidate'
    mac_display='AIbertAI 员工端源码候选测试'
    mac_id=com.aibertai.employee-source-candidate
    artifact=AIbertAI-employee-source-candidate-mac-arm64.dmg
    if [[ $release == true ]]; then
      mac_name=AIbertAI
      mac_display=AIbertAI
      mac_id=com.aibertai.desktop
      artifact=AIbertAI-mac-arm64.dmg
    fi
    app="$stage/$mac_name.app"
    resources="$app/Contents/Resources"
    binary=aibertai-native
    ditto "$project/.local-runtime/electron/Electron.app" "$app"
    mv "$app/Contents/MacOS/Electron" "$app/Contents/MacOS/AIbertAI"
    python3 - "$app" "$mac_id" "$mac_name" "$mac_display" "$project/code/app/package.json" "$release" <<'PY'
import json
import plistlib
import sys
from pathlib import Path

app, bundle_id, name, display, package, release = sys.argv[1:]
app = Path(app)
p = app / "Contents/Info.plist"
d = plistlib.loads(p.read_bytes())
for key in ("ElectronAsarIntegrity", "CFBundleURLTypes"):
    d.pop(key, None)
d.update(CFBundleExecutable="AIbertAI", CFBundleIdentifier=bundle_id,
         CFBundleName=name, CFBundleDisplayName=display,
         CFBundleIconFile="AIbertAI.icns")
if release == "true":
    version = json.loads(Path(package).read_text())["version"]
    d.update(CFBundleShortVersionString=version, CFBundleVersion=version)
    for helper in (app / "Contents/Frameworks").glob("*.app/Contents/Info.plist"):
        info = plistlib.loads(helper.read_bytes())
        helper_name = helper.parents[1].stem
        if helper_name == "Electron Helper":
            suffix = ""
        elif helper_name.startswith("Electron Helper (") and helper_name.endswith(")"):
            role = helper_name[len("Electron Helper ("):-1]
            if not role.isascii() or not role.isalnum():
                raise SystemExit("Unexpected Electron helper role")
            suffix = "." + role
        else:
            raise SystemExit("Unexpected Electron helper executable")
        info.update(CFBundleExecutable=helper_name, CFBundleIdentifier=bundle_id + ".helper" + suffix)
        helper.write_bytes(plistlib.dumps(info))
p.write_bytes(plistlib.dumps(d))
PY
    cp "$project/AIbertAI.app/Contents/Resources/AIbertAI.icns" "$resources/"
    ;;
  win-x64)
    os=windows
    arch=amd64
    win_name='AIbertAI Employee Source Candidate'
    artifact=AIbertAI-employee-source-candidate-win-x64.zip
    if [[ $release == true ]]; then
      win_name=AIbertAI
      artifact=AIbertAI-win-x64.zip
    fi
    app="$stage/$win_name"
    resources="$app/resources"
    binary=aibertai-native.exe
    [[ -f "$electron_zip" ]]
    (
      cd -- "$(dirname -- "$electron_zip")"
      entry=${electron_zip##*/}
      awk -v f="$entry" '$2==f || $2=="*" f' SHASUMS256.txt | shasum -a 256 -c
    )
    mkdir "$app"
    ditto -x -k "$electron_zip" "$app"
    mv "$app/electron.exe" "$app/$win_name.exe"
    ;;
  *)
    echo "Unsupported target: $target" >&2
    exit 2
    ;;
esac
if [[ -e "$output/$artifact" ]]; then
  echo "Output already exists: $output/$artifact" >&2
  exit 2
fi

rm -f "$resources/default_app.asar"
mkdir -p "$resources/app" "$resources/bin"
ditto "$project/code/app/out" "$resources/app/out"
cp "$project/code/app/package.json" "$resources/app/"
cp -RL "$project/code/app/node_modules" "$resources/app/node_modules"
if [[ $target == win-x64 ]]; then
  find "$resources/app/node_modules" -maxdepth 2 -name '*darwin-arm64*' -exec rm -rf {} +
elif [[ $release == true ]]; then
  find "$resources/app/node_modules" -maxdepth 2 -name '*win32-x64*' -exec rm -rf {} +
fi
ditto "$project/code/web" "$resources/web"
mkdir -p "$resources/tray"
sips -z 18 18 "$project/brand/icon.png" --out "$resources/tray/tray.png" >/dev/null
sips -z 36 36 "$project/brand/icon.png" --out "$resources/tray/tray@2x.png" >/dev/null
if [[ "$fixture" != "-" ]]; then
  cp "$fixture" "$resources/employee-acceptance.txt"
fi
python3 - "$resources" "$os" "$arch" "$release" <<'PY'
import json
import sys
from pathlib import Path

resources = Path(sys.argv[1])
os_name, arch, release = sys.argv[2:]
package = json.loads((resources / "app/package.json").read_text())
(resources / "employee-source.json").write_text(json.dumps({"version": 1}) + "\n")
(resources / "app/out/build-info.json").write_text(json.dumps({
    "version": package["version"], "release": release == "true",
    "builder": "source-employee" if release == "true" else "source-candidate",
    "target": {"os": os_name, "arch": arch}
}) + "\n")
PY

GOOS=$os GOARCH=$arch CGO_ENABLED=0 go -C "$project/native" build -mod=readonly -trimpath -buildvcs=false -o "$resources/bin/$binary" .
test -f "$resources/web/index.html"
[[ $(find "$resources/bin" -maxdepth 1 -type f | wc -l | tr -d ' ') == 1 ]]
python3 - "$resources" "$binary" <<'PY'
import sys
from pathlib import Path

resources = Path(sys.argv[1])
retired = ("baizhi" + ".cloud", "monkeycode-ai" + ".com")
patterns = tuple(host.encode(encoding) for host in retired for encoding in ("utf-8", "utf-16le"))
scripts = {".js", ".cjs", ".mjs", ".html", ".json", ".css"}
files = [resources / "employee-source.json", resources / "bin" / sys.argv[2]]
for root in (resources / "app/out", resources / "web"):
    files.extend(path for path in root.rglob("*") if path.suffix in scripts and path.is_file())
for path in files:
    data = path.read_bytes().lower()
    if any(pattern in data for pattern in patterns):
        raise SystemExit(f"Retired network host found in {path.relative_to(resources)}")
PY

if [[ $target == mac-arm64 ]]; then
  image_source="$app"
  if [[ $release == true ]]; then
    electron-osx-sign "$app" "$resources/bin/$binary" --platform=darwin --type=distribution --hardened-runtime --identity="$mac_identity"
    codesign --verify --strict -R='anchor apple generic and certificate leaf[field.1.2.840.113635.100.6.1.13] exists and identifier "com.aibertai.desktop"' "$app"
    if ! codesign --display --verbose=2 "$app" 2>&1 | grep -E '^CodeDirectory .*\([^)]*runtime'; then
      echo "The app signature must enable the hardened runtime." >&2
      exit 2
    fi
    image_source="$stage/image"
    mkdir "$image_source"
    ln -s /Applications "$image_source/Applications"
  else
    codesign --force --deep --sign - "$app"
  fi
  codesign --verify --strict --deep "$app"
  if [[ $release == true ]]; then
    mv "$app" "$image_source/"
  fi
  hdiutil create -volname "$mac_name" -srcfolder "$image_source" -format UDZO "$stage/$artifact"
  if [[ $release == true ]]; then
    codesign --sign "$mac_identity" --timestamp --identifier com.aibertai.desktop.dmg "$stage/$artifact"
    codesign --verify --strict -R='anchor apple generic and certificate leaf[field.1.2.840.113635.100.6.1.13] exists and identifier "com.aibertai.desktop.dmg"' "$stage/$artifact"
  fi
  hdiutil verify "$stage/$artifact"
else
  ditto --norsrc -c -k --keepParent "$app" "$stage/$artifact"
  unzip -tq "$stage/$artifact"
fi

(cd "$stage" && shasum -a 256 "$artifact" > "$artifact.sha256")
mv "$stage/$artifact" "$stage/$artifact.sha256" "$output/"
