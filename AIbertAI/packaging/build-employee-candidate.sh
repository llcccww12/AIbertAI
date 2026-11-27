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
dsh_resources=${AIBERTAI_DSH_RESOURCES:-}
dsh_manifest_sha256=${AIBERTAI_DSH_MANIFEST_SHA256:-}
if [[ -z "$dsh_resources" && -z "$dsh_manifest_sha256" ]]; then
  if [[ $release == true ]]; then
    echo "Set AIBERTAI_DSH_RESOURCES and AIBERTAI_DSH_MANIFEST_SHA256 from the trusted DSH build record." >&2
    exit 2
  fi
  dsh_target=darwin-arm64
  [[ "$target" == win-x64 ]] && dsh_target=windows-amd64
  pin=$(shasum -a 256 "$project/dsh-integration/patches/experimental-invocation-id.patch" | awk '{print $1}')
  # The shipped runtime is the production-only subset of the development build.
  built="$project/.local-runtime/dsh-slim/$dsh_target/$pin"
  if [[ ! -f "$built/manifest.sha256" ]]; then
    echo "Build the DSH runtime, then run dsh-integration/build-slim-runtime.sh $dsh_target, or supply a pinned DSH resource directory." >&2
    exit 2
  fi
  dsh_resources="$built/resources"
  dsh_manifest_sha256=$(cat "$built/manifest.sha256")
fi
if [[ -n "$dsh_resources" || -n "$dsh_manifest_sha256" ]]; then
  if [[ "$dsh_resources" != /* || ! -d "$dsh_resources" || ! "$dsh_manifest_sha256" =~ ^[0-9a-f]{64}$ ]]; then
    echo "Set both AIBERTAI_DSH_RESOURCES to an absolute staged resource directory and AIBERTAI_DSH_MANIFEST_SHA256 to its trusted digest." >&2
    exit 2
  fi
  python3 "$project/dsh-integration/manifest.py" verify "$dsh_resources" --manifest-sha256 "$dsh_manifest_sha256"
fi
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
    browser_host=aibertai-browser-host
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
# The version the employee sees must match the update feed for both build
# kinds; Electron's own 44.x would otherwise ship as the app version.
version = json.loads(Path(package).read_text())["version"]
d.update(CFBundleShortVersionString=version, CFBundleVersion=version)
if release == "true":
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
    icon="$project/AIbertAI.app/Contents/Resources/AIbertAI.icns"
    if [[ ! -f "$icon" ]]; then icon="$project/.local-runtime/AIbertAI.icns"; fi
    cp "$icon" "$resources/AIbertAI.icns"
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
    browser_host=aibertai-browser-host.exe
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
if [[ -n "$dsh_resources" ]]; then
  python3 - "$dsh_resources/manifest.json" "$os-$arch" <<'PYCODE'
import json
import sys
from pathlib import Path
manifest = json.loads(Path(sys.argv[1]).read_text())
if manifest["target"] != sys.argv[2]:
    raise SystemExit("DSH resource target does not match the employee package")
# The Windows distribution is a ZIP; its extraction path does not reliably
# restore symlinks. A Windows staging pipeline must materialize them first.
if manifest["target"].startswith("windows-") and any(x["type"] == "symlink" for x in manifest["files"]):
    raise SystemExit("Windows DSH resources must not contain symlinks")
PYCODE
  ditto "$dsh_resources" "$resources/dsh"
  python3 "$project/dsh-integration/manifest.py" verify "$resources/dsh" --manifest-sha256 "$dsh_manifest_sha256"
fi
ditto "$project/code/app/out" "$resources/app/out"
cp "$project/code/app/package.json" "$resources/app/"
if [[ $release != true ]]; then
  python3 - "$resources/app/package.json" <<'PY'
import json
import sys
from pathlib import Path
path = Path(sys.argv[1])
package = json.loads(path.read_text())
# Candidate packages must not reuse the installed AIbertAI profile.
package["productName"] = "AIbertAI Candidate"
path.write_text(json.dumps(package, indent=2) + "\n")
PY
fi
cp -RL "$project/code/app/node_modules" "$resources/app/node_modules"
if [[ $target == win-x64 ]]; then
  find "$resources/app/node_modules" -maxdepth 2 -name '*darwin-arm64*' -exec rm -rf {} +
elif [[ $release == true ]]; then
  find "$resources/app/node_modules" -maxdepth 2 -name '*win32-x64*' -exec rm -rf {} +
fi
ditto "$project/code/web" "$resources/web"
ditto "$project/browser-extension" "$resources/browser-extension"
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

native_ldflags=""
if [[ -n "$dsh_resources" ]]; then
  native_ldflags="-X main.dshManifestSHA256=$dsh_manifest_sha256"
fi
GOOS=$os GOARCH=$arch CGO_ENABLED=0 go -C "$project/native" build -mod=readonly -trimpath -buildvcs=false -ldflags "$native_ldflags" -o "$resources/bin/$binary" .
GOOS=$os GOARCH=$arch CGO_ENABLED=0 go -C "$project/native" build -mod=readonly -trimpath -buildvcs=false -o "$resources/bin/$browser_host" ./cmd/browser-host
test -f "$resources/web/index.html"
python3 - "$resources" <<'PY'
import sys
from pathlib import Path

resources = Path(sys.argv[1])
retired = ("baizhi" + ".cloud", "monkeycode-ai" + ".com")
patterns = tuple(host.encode(encoding) for host in retired for encoding in ("utf-8", "utf-16le"))
scripts = {".js", ".cjs", ".mjs", ".html", ".json", ".css"}
files = [resources / "employee-source.json", *(resources / "bin").iterdir()]
for root in (resources / "app/out", resources / "web", resources / "browser-extension"):
    files.extend(path for path in root.rglob("*") if path.suffix in scripts and path.is_file())
for path in files:
    data = path.read_bytes().lower()
    if any(pattern in data for pattern in patterns):
        raise SystemExit(f"Retired network host found in {path.relative_to(resources)}")
PY

if [[ $target == mac-arm64 ]]; then
  if [[ $release == true ]]; then
    electron-osx-sign "$app" "$resources/bin/$binary" "$resources/bin/$browser_host" --platform=darwin --type=distribution --hardened-runtime --identity="$mac_identity"
    codesign --verify --strict -R='anchor apple generic and certificate leaf[field.1.2.840.113635.100.6.1.13] exists and identifier "com.aibertai.desktop"' "$app"
    if ! codesign --display --verbose=2 "$app" 2>&1 | grep -E '^CodeDirectory .*\([^)]*runtime'; then
      echo "The app signature must enable the hardened runtime." >&2
      exit 2
    fi
  else
    codesign --force --deep --sign - "$app"
  fi
  codesign --verify --strict --deep "$app"
  if [[ -n "$dsh_resources" ]]; then
    # Signing may rewrite Mach-O contents; refuse a package if it changed any
    # resource after the manifest digest was embedded in the native binary.
    python3 "$project/dsh-integration/manifest.py" verify "$resources/dsh" --manifest-sha256 "$dsh_manifest_sha256"
  fi
  # 应用内更新下载此压缩包并整体替换 .app，需与下载页的 DMG 同时发布。
  update_zip="${artifact%.dmg}.zip"
  ditto -c -k --keepParent "$app" "$stage/$update_zip"
  # 安装窗口提供「应用程序」快捷方式：直接从 DMG 或下载目录运行时 macOS 会把应用隔离到只读的临时路径，应用内更新无法替换。
  image_source="$stage/image"
  mkdir "$image_source"
  ln -s /Applications "$image_source/Applications"
  mv "$app" "$image_source/"
  hdiutil create -volname "$mac_name" -srcfolder "$image_source" -format UDZO "$stage/$artifact"
  if [[ $release == true ]]; then
    codesign --sign "$mac_identity" --timestamp --identifier com.aibertai.desktop.dmg "$stage/$artifact"
    codesign --verify --strict -R='anchor apple generic and certificate leaf[field.1.2.840.113635.100.6.1.13] exists and identifier "com.aibertai.desktop.dmg"' "$stage/$artifact"
  fi
  hdiutil verify "$stage/$artifact"
else
  ditto --norsrc -c -k --keepParent "$app" "$stage/$artifact"
  unzip -tq "$stage/$artifact"
  # 员工用资源管理器解压。包内路径加上解压目录超过 260 字符会中途失败，缺文件又让
  # 运行时的资源清单校验过不去、直接起不来，且现场只能看到「反复崩溃」。
  # 在打包这一步拦下，比发出去再回滚便宜得多。
  python3 - "$stage/$artifact" <<'PY'
import sys, zipfile
limit = 200
longest = max((info.filename for info in zipfile.ZipFile(sys.argv[1]).infolist()), key=len)
if len(longest) > limit:
    raise SystemExit(f"Windows 包内最长路径 {len(longest)} 字符，超过 {limit} 的上限：\n  {longest}\n"
                     "解压到稍深目录后总长会越过 Windows 的 260 字符上限，资源管理器会中途失败。")
PY
fi

(cd "$stage" && shasum -a 256 "$artifact" > "$artifact.sha256")
mv "$stage/$artifact" "$stage/$artifact.sha256" "$output/"

# 发布 updates/ 目录到私有后端的 /downloads/updates/，员工端从登录的后端检查更新。
if [[ $target == win-x64 ]]; then
  update_zip=$artifact
  channel=latest.yml
  cp "$output/$artifact" "$stage/$update_zip"
else
  channel=latest-mac.yml
fi
mkdir -p "$output/updates"
python3 - "$stage/$update_zip" "$project/code/app/package.json" "$output/updates/$channel" "$project/packaging/release-notes.json" <<'PY'
import base64
import hashlib
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

archive, package, channel, notes_file = map(Path, sys.argv[1:])
digest = hashlib.sha512()
with archive.open("rb") as f:
    for chunk in iter(lambda: f.read(1 << 20), b""):
        digest.update(chunk)
sha512 = base64.b64encode(digest.digest()).decode()
version = json.loads(package.read_text())["version"]
size = archive.stat().st_size
# 先确认更新日志有这一条，再写 feed：feed 是原地覆盖的固定文件名，写坏了没有回退版本。
# 之前是写完才检查，漏一条就在最后一步失败，留下一个指向不存在文件的 latest*.yml。
notes = json.loads(notes_file.read_text()) if notes_file.exists() else []
if not any(entry.get("version") == version for entry in notes):
    raise SystemExit(f"packaging/release-notes.json has no entry for {version}")
channel.write_text(f"""version: {version}
files:
  - url: {archive.name}
    sha512: {sha512}
    size: {size}
path: {archive.name}
sha512: {sha512}
releaseDate: '{datetime.now(timezone.utc).isoformat()}'
""")
# 应用内更新页展示的更新日志，来自 packaging/release-notes.json；JSON 写法同样是合法 YAML。
with channel.open("a") as out:
    out.write("releaseNotes: " + json.dumps(notes, ensure_ascii=False) + "\n")
PY
mv "$stage/$update_zip" "$output/updates/"
