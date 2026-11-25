#!/bin/bash
# Assemble the windows-amd64 DSH resources from the macOS development build.
# The compiled JavaScript is platform-neutral; only node_modules (native addons)
# and the Node executable differ. Windows resources must not contain symlinks.
set -euo pipefail
integration=$(cd -- "$(dirname -- "$0")" && pwd -P)
patch="$integration/patches/experimental-invocation-id.patch"
pin=$(shasum -a 256 "$patch" | awk '{print $1}')
source_resources="$integration/../.local-runtime/dsh/$pin/resources"
work="$integration/../.local-runtime/dsh-win"
target="$work/$pin"
pnpm_bin=${AIBERTAI_DSH_PNPM_BIN:?Set AIBERTAI_DSH_PNPM_BIN to pnpm 11.28.5}
registry=${AIBERTAI_DSH_REGISTRY:-https://registry.npmmirror.com/}
node_zip="$work/dl/node-v25.9.0-win-x64.zip"
[[ -f "$source_resources/manifest.json" ]] || { echo "Build the macOS DSH SDK first (build-dev-sdk.sh)." >&2; exit 1; }
[[ -f "$target/manifest.sha256" ]] && { python3 "$integration/manifest.py" verify "$target/resources" --manifest-sha256 "$(cat "$target/manifest.sha256")"; exit 0; }
mkdir -p "$work/dl"
if [[ ! -f "$node_zip" ]]; then
  curl -fsSL -o "$node_zip" https://nodejs.org/dist/v25.9.0/node-v25.9.0-win-x64.zip
fi
curl -fsSL -o "$work/dl/SHASUMS256.txt" https://nodejs.org/dist/v25.9.0/SHASUMS256.txt
(cd "$work/dl" && grep ' node-v25.9.0-win-x64.zip$' SHASUMS256.txt | shasum -a 256 -c)
stage=$(mktemp -d "$work/.build.XXXXXXXX")
trap 'rm -rf -- "$stage"' EXIT
resources="$stage/resources"
mkdir -p "$resources/bin"
rsync -a --exclude node_modules "$source_resources/app/" "$resources/app/"
(cd "$resources/app" && "$pnpm_bin" install --frozen-lockfile --ignore-scripts --os win32 --cpu x64 --config.node-linker=hoisted --registry "$registry")
python3 - "$resources/app" <<'PY'
import os, shutil, sys
root = os.path.realpath(sys.argv[1])
for _ in range(10):
    links = [os.path.join(d, n) for d, dirs, files in os.walk(sys.argv[1]) for n in dirs + files if os.path.islink(os.path.join(d, n))]
    if not links:
        break
    for link in links:
        real = os.path.realpath(link)
        os.unlink(link)
        if not os.path.exists(real) or not real.startswith(root + os.sep):
            continue
        if os.path.isdir(real):
            shutil.copytree(real, link, symlinks=True)
        else:
            shutil.copy2(real, link)
else:
    raise SystemExit("symlinks remain after dereferencing")
PY
unzip -q -j "$node_zip" node-v25.9.0-win-x64/node.exe -d "$resources/bin"
python3 "$integration/manifest.py" create "$resources" --patch "$patch" --lockfile "$resources/app/pnpm-lock.yaml" \
  --archive-sha256 a46ccc2890645d0a7f3b032eeadb988769893392da5dcd48a1bc4ce0e49ba3fc --target windows-amd64 --node-version v25.9.0 > "$stage/manifest.sha256"
python3 -c 'import os, sys; os.rename(sys.argv[1], sys.argv[2])' "$stage" "$target"
echo "Windows DSH runtime ready at $target"
