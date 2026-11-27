#!/bin/bash
# Derive the shipped DSH runtime from a verified development build, keeping
# only what the SDK entry loads (see slim-runtime.py). The development build
# stays intact for tests and for deriving the Windows build.
set -euo pipefail
umask 077

if [[ $# != 1 || ! $1 =~ ^(darwin-arm64|windows-amd64)$ ]]; then
  echo "Usage: $0 darwin-arm64|windows-amd64" >&2
  exit 2
fi
target=$1
integration=$(cd -- "$(dirname -- "$0")" && pwd -P)
patch="$integration/patches/experimental-invocation-id.patch"
pin=$(shasum -a 256 "$patch" | awk '{print $1}')
if [[ $target == darwin-arm64 ]]; then
  source="$integration/../.local-runtime/dsh/$pin"
  hint="dsh-integration/build-dev-sdk.sh"
else
  source="$integration/../.local-runtime/dsh-win/$pin"
  hint="dsh-integration/build-windows-sdk.sh"
fi
cache="$integration/../.local-runtime/dsh-slim/$target"
destination="$cache/$pin"
if [[ ! -f "$source/manifest.sha256" ]]; then
  echo "Build the full runtime first with $hint." >&2
  exit 1
fi
python3 "$integration/manifest.py" verify "$source/resources" --manifest-sha256 "$(cat "$source/manifest.sha256")"
# 缓存键只有补丁摘要，所以重建过 DSH 之后这里会误命中，把旧运行时当新运行时发出去。
# 再绑一份源清单摘要：重建出了不同内容就必须重新派生。
source_digest=$(cat "$source/manifest.sha256")
if [[ -f "$destination/manifest.sha256" && -f "$destination/source-manifest.sha256" ]] &&
    [[ "$(cat "$destination/source-manifest.sha256")" == "$source_digest" ]]; then
  python3 "$integration/manifest.py" verify "$destination/resources" --manifest-sha256 "$(cat "$destination/manifest.sha256")"
  exit 0
fi
if [[ -e "$destination" ]]; then
  echo "Source runtime changed since the cached slim build; deriving it again." >&2
  rm -rf -- "$destination"
fi
mkdir -p "$cache"
stage=$(mktemp -d "$cache/.build.XXXXXXXX")
trap 'rm -rf -- "$stage"' EXIT
# APFS clones keep the copy cheap; symlinks are copied as links.
cp -cR "$source/resources" "$stage/resources" 2>/dev/null || cp -R "$source/resources" "$stage/resources"
cp "$source/resources/app/pnpm-lock.yaml" "$stage/pnpm-lock.yaml"
node_version=$(python3 -c 'import json, sys; print(json.load(open(sys.argv[1]))["node_version"])' "$source/resources/manifest.json")
rm "$stage/resources/manifest.json"
python3 "$integration/slim-runtime.py" "$stage/resources" --target "$target"
python3 "$integration/manifest.py" create "$stage/resources" --patch "$patch" --lockfile "$stage/pnpm-lock.yaml" \
  --archive-sha256 a46ccc2890645d0a7f3b032eeadb988769893392da5dcd48a1bc4ce0e49ba3fc --target "$target" --node-version "$node_version" > "$stage/manifest.sha256"
rm "$stage/pnpm-lock.yaml"
printf '%s\n' "$source_digest" > "$stage/source-manifest.sha256"
python3 -c 'import os, sys; os.rename(sys.argv[1], sys.argv[2])' "$stage" "$destination"
echo "Slim $target DSH runtime ready at $destination ($(du -sh "$destination/resources" | cut -f1))"
