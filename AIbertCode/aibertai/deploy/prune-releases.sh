#!/usr/bin/env bash
# 清理发布目录里的构建残留：stage-* 是打包时的中间目录，发布完成后就没用了。
# 默认只打印将要删除的内容，加 --apply 才真的删。
# 用法：bash deploy/prune-releases.sh [发布目录] [--apply]
#      发布目录默认 /root/AIbert-releases/client-downloads
set -euo pipefail

root=${1:-/root/AIbert-releases/client-downloads}
apply=false
for arg in "$@"; do
  [[ $arg == --apply ]] && apply=true
done
if [[ ! -d $root ]]; then
  printf '发布目录 %s 不存在\n' "$root" >&2
  exit 1
fi

# 中间目录：打包残留，删掉不影响下载和自动更新。
mapfile -t stages < <(find "$root" -maxdepth 1 -mindepth 1 -type d -name 'stage-*' | sort)
if (( ${#stages[@]} )); then
  total=$(du -sc "${stages[@]}" 2>/dev/null | tail -1 | cut -f1)
  printf '中间目录 %d 个，共 %s MiB：\n' "${#stages[@]}" "$(( total / 1024 ))"
  printf '  %s\n' "${stages[@]##*/}"
  if $apply; then
    rm -rf -- "${stages[@]}"
    printf '已删除。\n'
  fi
else
  printf '没有中间目录需要清理。\n'
fi

# 只保留最近三个版本的安装包，更早的留着只会占磁盘。
mapfile -t versions < <(find "$root" -maxdepth 1 -type f -name 'AIbertAI-[0-9]*' |
  sed -E 's|.*/AIbertAI-([0-9]+\.[0-9]+\.[0-9]+)-.*|\1|' | sort -uV)
if (( ${#versions[@]} > 3 )); then
  drop=("${versions[@]:0:${#versions[@]}-3}")
  printf '安装包版本共 %d 个，保留最近三个（%s），将删除：%s\n' \
    "${#versions[@]}" "${versions[*]: -3}" "${drop[*]}"
  if $apply; then
    for version in "${drop[@]}"; do
      find "$root" -maxdepth 1 -type f -name "AIbertAI-$version-*" -print -delete
    done
  fi
else
  printf '安装包版本 %d 个，不需要清理。\n' "${#versions[@]}"
fi
