#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

MAGIC_BLOCK_SRC="${REPO_ROOT}/docker.local/config/b0magicBlock_4_miners_2_sharders.json"
DNS_ROOT="${REPO_ROOT}/quickstart/1s_2m_4b/config/1s_2m_4b/0dns/docker.local"
MAGIC_BLOCK_DEST="${DNS_ROOT}/config/b0magicBlock_4_miners_2_sharders.json"
CODE_ROOT="${REPO_ROOT}/quickstart/1s_2m_4b/config/1s_2m_4b/code"
ZDNS_CODE_DIR="${CODE_ROOT}/go/0dns.io"

if [ ! -f "${MAGIC_BLOCK_SRC}" ]; then
  echo "❌ Missing magic block at ${MAGIC_BLOCK_SRC}" >&2
  exit 1
fi

echo "🔍 Ensuring 0dns source code is available..."
if [ ! -d "${ZDNS_CODE_DIR}" ]; then
  if ! command -v git >/dev/null 2>&1; then
    echo "❌ Git is required to fetch the 0dns repository." >&2
    exit 1
  fi
  TMP_CLONE_DIR="${CODE_ROOT}/.0dns_tmp"
  rm -rf "${TMP_CLONE_DIR}"
  mkdir -p "${CODE_ROOT}"
  echo "📥 Cloning https://github.com/0chain/0dns.git ..."
  git clone --depth 1 https://github.com/0chain/0dns.git "${TMP_CLONE_DIR}"
  mkdir -p "${CODE_ROOT}/go"
  rm -rf "${ZDNS_CODE_DIR}"
  mv "${TMP_CLONE_DIR}/code/go/0dns.io" "${ZDNS_CODE_DIR}"
  rm -rf "${TMP_CLONE_DIR}"
  echo "✅ 0dns source synced to ${ZDNS_CODE_DIR}"
else
  echo "✅ 0dns source already present at ${ZDNS_CODE_DIR}"
fi

echo "📦 Syncing magic block to 0dns config..."
mkdir -p "${DNS_ROOT}/config"
cp "${MAGIC_BLOCK_SRC}" "${MAGIC_BLOCK_DEST}"

echo "🌐 Ensuring Docker network 'testnet0' exists..."
if ! docker network ls --format '{{.Name}}' | grep -q '^testnet0$'; then
  docker network create testnet0
fi

if docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_CMD=(docker-compose)
else
  echo "❌ Neither 'docker compose' nor 'docker-compose' is available" >&2
  exit 1
fi

echo "🚀 Starting 0dns service (detached)..."
(cd "${DNS_ROOT}" && "${COMPOSE_CMD[@]}" up -d)

echo "✅ 0dns is starting. Check status with: docker ps --filter name=0dns"

