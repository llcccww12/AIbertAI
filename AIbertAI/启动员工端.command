#!/bin/bash
set -euo pipefail

project=$(cd -- "$(dirname -- "$0")" && pwd)
runtime="$project/.local-runtime"
resources="$runtime/mac-resources"
electron="$runtime/electron/Electron.app/Contents/MacOS/Electron"
web_url=http://127.0.0.1:1432
web_dir="$project/code/web"
web_pid=''

for executable in "$electron" "$resources/bin/"{monkeyai-runtime,monkeyai-runtime-login,ohmyagent,office-preview-server}; do
  if [[ ! -x "$executable" ]]; then
    printf '缺少可执行文件 %s，请恢复项目 .local-runtime 下的 macOS 依赖。\n' "$executable" >&2
    exit 1
  fi
done

cleanup() {
  if [[ -n "$web_pid" ]]; then
    kill "$web_pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

if ! curl -fsS --max-time 1 "$web_url/" > /dev/null 2>&1; then
  python3 -m http.server 1432 --bind 127.0.0.1 --directory "$web_dir" > "$runtime/web.log" 2>&1 &
  web_pid=$!
  for ((attempt=0; attempt<20; attempt++)); do
    if curl -fsS --max-time 1 "$web_url/" > /dev/null 2>&1; then break; fi
    sleep 0.2
  done
fi

if ! curl -fsS --max-time 3 "$web_url/" | cmp -s - "$web_dir/index.html"; then
  echo '无法在 1432 端口加载本项目的员工端前端。' >&2
  exit 1
fi

MONKEYAI_USER_DATA="$runtime/user-data" \
MONKEYAI_LOCAL_BRAND_ICON="$project/brand/icon.png" \
MONKEYAI_WEB_URL="$web_url" \
MONKEYAI_OHMYAGENT_BIN="$resources/bin/ohmyagent" \
AIBERTAI_LOGIN_RUNTIME_BIN="$resources/bin/monkeyai-runtime-login" \
MONKEYAI_RUNTIME_BIN="$resources/bin/monkeyai-runtime" \
MONKEYAI_OFFICE_PREVIEW_BIN="$resources/bin/office-preview-server" \
"$electron" "$project/code/app" > "$runtime/desktop.log" 2>&1
