#!/bin/bash
set -euo pipefail

project=$(cd -- "$(dirname -- "$0")" && pwd)

if [[ ${1:-} == --dev ]]; then
  shift
  electron="$project/.local-runtime/electron/Electron.app/Contents/MacOS/Electron"
  if [[ ! -x "$electron" ]]; then
    echo '未找到本地 Electron 开发运行器。' >&2
    exit 1
  fi
  case "$(uname -m)" in
    arm64) arch=arm64 ;;
    x86_64) arch=x64 ;;
    *) echo '不支持当前 macOS 架构。' >&2; exit 1 ;;
  esac
  bin="$project/code/app/resources/bin/mac-$arch"
  mkdir -p "$bin"
  CGO_ENABLED=0 go -C "$project/native" build -mod=readonly -trimpath -buildvcs=false -o "$bin/aibertai-native" .
  CGO_ENABLED=0 go -C "$project/native" build -mod=readonly -trimpath -buildvcs=false -o "$bin/aibertai-browser-host" ./cmd/browser-host
  exec "$electron" "$project/code/app" "$@"
fi

app="$project/AIbertAI.app"
resources="$app/Contents/Resources"

if [[ ! -f "$resources/employee-source.json" || ! -x "$resources/bin/aibertai-native" ]]; then
  echo '未找到 AIbertAI 独立员工端，请先构建并放置 AIbertAI.app。' >&2
  exit 1
fi

open -a "$app"
