#!/usr/bin/env bash
# 备份本机部署的数据库和对象存储，并按天数轮换。
# 用法：bash deploy/backup.sh            每天跑一次（可挂 cron）
#      bash deploy/backup.sh --list     只看现有备份
#
# 环境变量：BACKUP_DIR（默认 /root/AIbert-backups）
#          BACKUP_KEEP_DAYS（默认 14）
#          PG_CONTAINER / S3_CONTAINER（默认 aibertai-pg / aibertai-s3）
set -euo pipefail

backup_dir=${BACKUP_DIR:-/root/AIbert-backups}
keep_days=${BACKUP_KEEP_DAYS:-14}
pg_container=${PG_CONTAINER:-aibertai-pg}
s3_container=${S3_CONTAINER:-aibertai-s3}

if [[ ${1:-} == --list ]]; then
  ls -lh "$backup_dir" 2>/dev/null || printf '还没有备份目录 %s\n' "$backup_dir"
  exit 0
fi

mkdir -p "$backup_dir"
umask 077
stamp=$(date +%Y%m%d-%H%M%S)

# 数据库：容器内自带 pg_dump，走本地 socket，不需要密码，也不会把口令写进脚本。
database=(aibertai aibertai)
if docker inspect "$pg_container" >/dev/null 2>&1; then
  user=$(docker exec "$pg_container" sh -lc 'printf %s "$POSTGRES_USER"')
  db=$(docker exec "$pg_container" sh -lc 'printf %s "$POSTGRES_DB"')
  if [[ -n $user && -n $db ]]; then
    database=("$user" "$db")
  fi
  target="$backup_dir/postgres-$stamp.dump"
  docker exec "$pg_container" pg_dump -U "${database[0]}" -d "${database[1]}" -Fc >"$target"
  printf '数据库 %s -> %s (%s)\n' "${database[1]}" "$target" "$(du -h "$target" | cut -f1)"
else
  printf '跳过数据库：容器 %s 不存在\n' "$pg_container" >&2
fi

# 对象存储：整个数据目录打一个包，权限、隐藏文件一起留着。
if docker inspect "$s3_container" >/dev/null 2>&1; then
  source=$(docker inspect "$s3_container" --format '{{range .Mounts}}{{if eq .Destination "/data"}}{{.Source}}{{end}}{{end}}')
  if [[ -n $source && -d $source ]]; then
    target="$backup_dir/storage-$stamp.tar.gz"
    tar -C "$(dirname "$source")" -czf "$target" "$(basename "$source")"
    printf '对象存储 %s -> %s (%s)\n' "$source" "$target" "$(du -h "$target" | cut -f1)"
  else
    printf '跳过对象存储：找不到 %s 的数据目录\n' "$s3_container" >&2
  fi
else
  printf '跳过对象存储：容器 %s 不存在\n' "$s3_container" >&2
fi

# 轮换：超过保留天数的备份删掉，避免把磁盘吃满。
find "$backup_dir" -maxdepth 1 -type f \( -name 'postgres-*.dump' -o -name 'storage-*.tar.gz' \) \
  -mtime +"$keep_days" -print -delete

printf '备份完成，目录 %s（保留 %s 天）\n' "$backup_dir" "$keep_days"
