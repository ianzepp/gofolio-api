#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

LEGACY_BASE_URL="${LEGACY_BASE_URL:-http://127.0.0.1:3333}"
GO_BASE_URL="${GO_BASE_URL:-http://127.0.0.1:3334}"
GO_PORT="${GO_PORT:-3334}"
GO_HEALTH_PATH="${GO_HEALTH_PATH:-/api/v1/health}"
START_TIMEOUT_SECONDS="${START_TIMEOUT_SECONDS:-20}"

started_go_server=0
go_server_pid=""
go_server_log="${TMPDIR:-/tmp}/api-go-integration.log"

health_check() {
  local base_url="$1"
  local health_path="$2"
  curl -fsS --max-time 2 "${base_url}${health_path}" >/dev/null
}

wait_for_health() {
  local base_url="$1"
  local health_path="$2"
  local timeout="$3"
  local elapsed=0

  while (( elapsed < timeout )); do
    if health_check "$base_url" "$health_path"; then
      return 0
    fi
    sleep 1
    elapsed=$((elapsed + 1))
  done

  return 1
}

cleanup() {
  if [[ "$started_go_server" -eq 1 && -n "${go_server_pid}" ]]; then
    kill "${go_server_pid}" >/dev/null 2>&1 || true
    wait "${go_server_pid}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

echo "== Differential test preflight =="
echo "legacy: ${LEGACY_BASE_URL}"
echo "go:     ${GO_BASE_URL}"

if ! health_check "${LEGACY_BASE_URL}" "${GO_HEALTH_PATH}"; then
  echo "error: legacy API is not reachable at ${LEGACY_BASE_URL}${GO_HEALTH_PATH}" >&2
  echo "start legacy API first (expected port: 3333)" >&2
  exit 1
fi

if health_check "${GO_BASE_URL}" "${GO_HEALTH_PATH}"; then
  echo "go api already running at ${GO_BASE_URL}"
else
  echo "starting go api on port ${GO_PORT}..."
  (
    cd "${ROOT_DIR}"
    env PORT="${GO_PORT}" go run ./cmd/server
  ) >"${go_server_log}" 2>&1 &
  go_server_pid="$!"
  started_go_server=1

  if ! wait_for_health "${GO_BASE_URL}" "${GO_HEALTH_PATH}" "${START_TIMEOUT_SECONDS}"; then
    echo "error: go api failed to start at ${GO_BASE_URL}${GO_HEALTH_PATH}" >&2
    echo "last log lines:" >&2
    tail -n 40 "${go_server_log}" >&2 || true
    exit 1
  fi
  echo "go api started (pid ${go_server_pid})"
fi

echo "== Running differential integration tests =="
cd "${ROOT_DIR}"
LEGACY_BASE_URL="${LEGACY_BASE_URL}" \
GO_BASE_URL="${GO_BASE_URL}" \
go test ./tests/integration -v -count=1 "$@"
