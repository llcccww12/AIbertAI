#!/bin/bash
set -euo pipefail

project=$(cd -- "$(dirname -- "$0")" && pwd)

app="$project/AIbertAI.app"
if [[ ${1:-} == --dev || ! -d "$app" ]]; then
  if [[ ${1:-} == --dev ]]; then shift; fi
  pin=$(shasum -a 256 "$project/dsh-integration/patches/experimental-invocation-id.patch" | awk '{print $1}')
  built="$project/.local-runtime/dsh/$pin"
  if [[ -z ${AIBERTAI_DSH_RESOURCES:-} && -z ${AIBERTAI_DSH_MANIFEST_SHA256:-} && -f "$built/manifest.sha256" ]]; then
    export AIBERTAI_DSH_RESOURCES="$built/resources"
    AIBERTAI_DSH_MANIFEST_SHA256=$(cat "$built/manifest.sha256")
  fi
  if [[ ${AIBERTAI_DSH_RESOURCES:-} != /* || ! -f "$AIBERTAI_DSH_RESOURCES/manifest.json" || ! ${AIBERTAI_DSH_MANIFEST_SHA256:-} =~ ^[0-9a-f]{64}$ ]]; then
    echo '请先运行 dsh-integration/build-dev-sdk.sh，参数为本地 DSH 源码仓库路径。' >&2
    exit 1
  fi
  electron="$project/.local-runtime/electron/Electron.app/Contents/MacOS/Electron"
  if [[ ! -x "$electron" ]]; then
    echo '未找到本地 Electron 开发运行器。' >&2
    exit 1
  fi
  export AIBERTAI_USER_DATA="${AIBERTAI_USER_DATA:-$project/.local-runtime/employee-profile}"
  case "$(uname -m)" in
    arm64) arch=arm64 ;;
    x86_64) arch=x64 ;;
    *) echo '不支持当前 macOS 架构。' >&2; exit 1 ;;
  esac
  bin="$project/code/app/resources/bin/mac-$arch"
  mkdir -p "$bin"
  CGO_ENABLED=0 go -C "$project/native" build -mod=readonly -trimpath -buildvcs=false \
    -ldflags "-X main.dshManifestSHA256=$AIBERTAI_DSH_MANIFEST_SHA256 -X main.dshDevelopmentResources=enabled" -o "$bin/aibertai-native" .
  CGO_ENABLED=0 go -C "$project/native" build -mod=readonly -trimpath -buildvcs=false -o "$bin/aibertai-browser-host" ./cmd/browser-host
  exec "$electron" "$project/code/app" "$@"
fi

resources="$app/Contents/Resources"

if [[ ! -f "$resources/employee-source.json" || ! -x "$resources/bin/aibertai-native" ]]; then
  echo '未找到 AIbertAI 独立员工端，请先构建并放置 AIbertAI.app。' >&2
  exit 1
fi

open -a "$app"
