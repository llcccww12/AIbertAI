#!/usr/bin/env bash
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")"

usage() {
  printf 'Usage: bash deploy/stack.sh init PUBLIC_URL ADMIN_EMAIL | up\n' >&2
  exit 2
}

case "${1:-}" in
  init)
    [[ $# == 3 ]] || usage
    url=${2%/}
    email=$3
    if [[ ! "$url" =~ ^https?://(\[[0-9a-fA-F:]+\]|[a-zA-Z0-9.-]+)(:[0-9]+)?$ ||
          ! "$email" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+$ ]]; then
      usage
    fi
    if [[ -e .env || -L .env ]]; then
      printf 'deploy/.env already exists. Keep it and run up.\n' >&2
      exit 1
    fi
    admin_password=$(openssl rand -hex 24)
    database_password=$(openssl rand -hex 24)
    storage_secret=$(openssl rand -hex 24)
    umask 077
    set -o noclobber
    cat >.env <<CONFIG
PUBLIC_URL=$url
INITIAL_ADMIN_EMAIL=$email
INITIAL_ADMIN_PASSWORD=$admin_password
POSTGRES_PASSWORD=$database_password
S3_ACCESS_KEY=aibertai
S3_SECRET_KEY=$storage_secret
CONFIG
    printf 'Created deploy/.env. Read the initial administrator password from that file.\n'
    ;;
  up)
    [[ $# == 1 ]] || usage
    [[ -f .env ]] || usage
    # Bind-mounted input directories remain local to this deployment.
    (umask 077; mkdir -p data/design-cache data/releases)
    compose=(docker compose --env-file .env -f compose.yaml)
    "${compose[@]}" config --quiet
    "${compose[@]}" build admin backend
    "${compose[@]}" up -d --wait --wait-timeout 90 postgres object-storage
    "${compose[@]}" stop admin backend
    "${compose[@]}" run --rm migrate
    "${compose[@]}" up -d --wait --wait-timeout 180 backend admin
    ;;
  *) usage ;;
esac
