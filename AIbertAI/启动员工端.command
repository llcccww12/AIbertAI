#!/bin/bash
set -euo pipefail

project=$(cd -- "$(dirname -- "$0")" && pwd)
app="$project/AIbertAI.app"
resources="$app/Contents/Resources"

if [[ ! -f "$resources/employee-source.json" || ! -x "$resources/bin/aibertai-native" ]]; then
  echo '未找到 AIbertAI 独立员工端，请先构建并放置 AIbertAI.app。' >&2
  exit 1
fi

open -a "$app"
