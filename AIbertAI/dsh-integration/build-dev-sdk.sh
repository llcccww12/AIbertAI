#!/bin/bash
set -euo pipefail
umask 077

if [[ $# != 1 || $(uname -s)-$(uname -m) != Darwin-arm64 ]]; then
  echo "Usage on Apple Silicon: $0 /path/to/dsh-checkout" >&2
  exit 2
fi
repository=$(cd -- "$1" && pwd -P)
integration=$(cd -- "$(dirname -- "$0")" && pwd -P)
patch="$integration/patches/experimental-invocation-id.patch"
pin=$(shasum -a 256 "$patch" | awk '{print $1}')
cache="$integration/../.local-runtime/dsh"
target="$cache/$pin"
if [[ -f "$target/manifest.sha256" ]]; then
  python3 "$integration/manifest.py" verify "$target/resources" --manifest-sha256 "$(cat "$target/manifest.sha256")"
  exit 0
fi
mkdir -p "$cache"
stage=$(mktemp -d "$cache/.build.XXXXXXXX")
trap 'rm -rf -- "$stage"' EXIT
resources="$stage/resources"
# Capture the recipe together. A branch edit during compilation must never label old JS with a new patch.
recipe="$stage/recipe"
mkdir -p "$recipe/patches" "$resources/bin"
cp "$integration/prepare-source.py" "$integration/manifest.py" "$recipe/"
cp "$patch" "$recipe/patches/"
patch="$recipe/patches/experimental-invocation-id.patch"
[[ $(shasum -a 256 "$patch" | awk '{print $1}') == "$pin" ]]
GIT_CEILING_DIRECTORIES="$(cd "$resources" && pwd -P)" \
  python3 "$recipe/prepare-source.py" --repository "$repository" --output "$resources/app"
cd "$resources/app"

pm=${AIBERTAI_DSH_PNPM_BIN:-pnpm}
if [[ $(node --version) != v25.9.0 || $("$pm" --version) != 11.28.5 ]]; then
  echo "Use Node v25.9.0 and pnpm 11.28.5 (AIBERTAI_DSH_PNPM_BIN can select the pnpm executable)." >&2
  exit 1
fi
# A Homebrew node executable can depend on a dylib relative to its installation.
# Bundle the pinned, independently runnable macOS binary for the SDK child.
archive=${AIBERTAI_DSH_NODE_ARCHIVE:-$stage/node.tar.gz}
if [[ -z ${AIBERTAI_DSH_NODE_ARCHIVE:-} ]]; then
  curl --fail --location --silent --show-error --output "$archive" https://nodejs.org/dist/v25.9.0/node-v25.9.0-darwin-arm64.tar.gz
fi
if [[ $(shasum -a 256 "$archive" | awk '{print $1}') != e479f3c469d3d9303a44f00a8ea37a3788395d171bb8059c48a4bbbd2e371b59 ]]; then
  echo "Node release archive digest mismatch." >&2
  exit 1
fi
tar -xzf "$archive" -C "$resources" --strip-components 1 node-v25.9.0-darwin-arm64/bin/node
node_version=$("$resources/bin/node" --version)
"$pm" install --frozen-lockfile --ignore-scripts
"$pm" run build:native-system
"$pm" exec tsx scripts/compile-referenced-projects.ts host-libraries
"$pm" exec tsc -b tsconfig.client.json
"$pm" exec tsdown --config-loader native --env.DSH_BUILD_FACE client

python3 "$recipe/manifest.py" create "$resources" --patch "$patch" --lockfile "$resources/app/pnpm-lock.yaml" \
  --archive-sha256 a46ccc2890645d0a7f3b032eeadb988769893392da5dcd48a1bc4ce0e49ba3fc --target darwin-arm64 --node-version "$node_version" > "$stage/manifest.sha256"
cd "$cache"
# Publish the complete tree in one rename. Concurrent builders cannot nest or overwrite it.
python3 -c 'import os, sys; os.rename(sys.argv[1], sys.argv[2])' "$stage" "$target"
echo "DSH development runtime ready at $target"
