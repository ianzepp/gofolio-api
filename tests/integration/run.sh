#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
REPO_ROOT="$(cd "${ROOT_DIR}/../.." && pwd)"

LEGACY_BASE_URL="${LEGACY_BASE_URL:-http://127.0.0.1:3333}"
GO_BASE_URL="${GO_BASE_URL:-http://127.0.0.1:3334}"
GO_PORT="${GO_PORT:-3334}"
GO_HEALTH_PATH="${GO_HEALTH_PATH:-/api/v1/health}"
START_TIMEOUT_SECONDS="${START_TIMEOUT_SECONDS:-20}"
BOOTSTRAP_LEGACY_USER="${BOOTSTRAP_LEGACY_USER:-1}"
GO_BINARY_PATH="${GO_BINARY_PATH:-${ROOT_DIR}/bin/api-go}"

started_go_server=0
go_server_pid=""
go_server_log="${TMPDIR:-/tmp}/api-go-integration.log"
resolved_jwt_secret=""
resolved_database_url=""

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

resolve_jwt_secret() {
  if [[ -n "${JWT_SECRET_KEY:-}" ]]; then
    resolved_jwt_secret="${JWT_SECRET_KEY}"
    return 0
  fi

  local dotenv_file="${REPO_ROOT}/.env"
  if [[ ! -f "${dotenv_file}" ]]; then
    return 0
  fi

  local sourced_candidate
  sourced_candidate="$(bash -c '
    set -a
    # shellcheck disable=SC1090
    source "$1" >/dev/null 2>&1 || true
    set +a
    printf "%s" "${JWT_SECRET_KEY:-}"
  ' _ "${dotenv_file}" 2>/dev/null || true)"
  if [[ -n "${sourced_candidate}" ]]; then
    resolved_jwt_secret="${sourced_candidate}"
    return 0
  fi

  local candidate
  candidate="$(awk -F= '
    /^[[:space:]]*#/ { next }
    /^[[:space:]]*JWT_SECRET_KEY[[:space:]]*=/ {
      sub(/^[^=]*=/, "", $0)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", $0)
      print $0
      exit
    }
  ' "${dotenv_file}")"

  if [[ -z "${candidate}" ]]; then
    return 0
  fi

  # Strip surrounding single/double quotes if present.
  candidate="${candidate%\"}"
  candidate="${candidate#\"}"
  candidate="${candidate%\'}"
  candidate="${candidate#\'}"

  if [[ -n "${candidate}" ]]; then
    resolved_jwt_secret="${candidate}"
  fi
}

resolve_database_url() {
  if [[ -n "${DATABASE_URL:-}" ]]; then
    resolved_database_url="${DATABASE_URL}"
    return 0
  fi

  local dotenv_file="${REPO_ROOT}/.env"
  if [[ ! -f "${dotenv_file}" ]]; then
    return 0
  fi

  local sourced_candidate
  sourced_candidate="$(bash -c '
    set -a
    # shellcheck disable=SC1090
    source "$1" >/dev/null 2>&1 || true
    set +a
    printf "%s" "${DATABASE_URL:-}"
  ' _ "${dotenv_file}" 2>/dev/null || true)"
  if [[ -n "${sourced_candidate}" ]]; then
    resolved_database_url="${sourced_candidate}"
    return 0
  fi

  local candidate
  candidate="$(awk -F= '
    /^[[:space:]]*#/ { next }
    /^[[:space:]]*DATABASE_URL[[:space:]]*=/ {
      sub(/^[^=]*=/, "", $0)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", $0)
      print $0
      exit
    }
  ' "${dotenv_file}")"

  if [[ -z "${candidate}" ]]; then
    return 0
  fi

  candidate="${candidate%\"}"
  candidate="${candidate#\"}"
  candidate="${candidate%\'}"
  candidate="${candidate#\'}"

  if [[ -n "${candidate}" ]]; then
    resolved_database_url="${candidate}"
  fi
}

bootstrap_legacy_user() {
  if [[ "${BOOTSTRAP_LEGACY_USER}" != "1" ]]; then
    echo "legacy user bootstrap disabled (BOOTSTRAP_LEGACY_USER=${BOOTSTRAP_LEGACY_USER})"
    return 0
  fi

  if ! command -v jq >/dev/null 2>&1; then
    echo "error: jq is required to bootstrap legacy user credentials" >&2
    exit 1
  fi

  echo "bootstrapping test user from legacy API..."

  local signup_response
  signup_response="$(curl -fsS -X POST "${LEGACY_BASE_URL}/api/v1/user")"

  local access_token
  local auth_token
  local user_id

  access_token="$(printf '%s' "${signup_response}" | jq -r '.accessToken // empty')"
  auth_token="$(printf '%s' "${signup_response}" | jq -r '.authToken // empty')"

  if [[ -z "${auth_token}" && -n "${access_token}" ]]; then
    auth_token="$(curl -fsS -X POST "${LEGACY_BASE_URL}/api/v1/auth/anonymous" \
      -H "Content-Type: application/json" \
      -d "$(jq -nc --arg accessToken "${access_token}" '{accessToken: $accessToken}')" \
      | jq -r '.authToken // empty')"
  fi

  if [[ -z "${auth_token}" ]]; then
    echo "error: failed to bootstrap auth token from legacy API /api/v1/user or /api/v1/auth/anonymous" >&2
    echo "response: ${signup_response}" >&2
    exit 1
  fi

  user_id="$(printf '%s' "${auth_token}" | jq -Rr '
    def b64url_to_json:
      gsub("-"; "+")
      | gsub("_"; "/")
      | . + (["", "===", "==", "="][(length % 4)])
      | @base64d
      | fromjson;
    (split(".")[1] // "") | b64url_to_json | .id // empty
  ' 2>/dev/null || true)"

  export JWT_TOKEN="${auth_token}"
  export JWT_USER_TOKEN="${auth_token}"
  export LEGACY_BOOTSTRAP_AUTH_TOKEN="${auth_token}"
  if [[ -n "${access_token}" ]]; then
    export LEGACY_BOOTSTRAP_ACCESS_TOKEN="${access_token}"
  fi
  if [[ -n "${user_id}" ]]; then
    export LEGACY_USER_ID="${user_id}"
  fi

  echo "bootstrapped legacy user for test run (jwt exported as JWT_USER_TOKEN/JWT_TOKEN${user_id:+, userId=${user_id}})"
}

cleanup() {
  if [[ "$started_go_server" -eq 1 && -n "${go_server_pid}" ]]; then
    kill "${go_server_pid}" >/dev/null 2>&1 || true
    wait "${go_server_pid}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

build_go_binary() {
  local binary_dir
  binary_dir="$(dirname "${GO_BINARY_PATH}")"
  mkdir -p "${binary_dir}"

  echo "building go api binary: ${GO_BINARY_PATH}"
  (
    cd "${ROOT_DIR}"
    go build -o "${GO_BINARY_PATH}" ./cmd/server
  )
}

echo "== Differential test preflight =="
echo "legacy: ${LEGACY_BASE_URL}"
echo "go:     ${GO_BASE_URL}"

resolve_jwt_secret
resolve_database_url
if [[ -n "${resolved_jwt_secret}" ]]; then
  echo "jwt secret: configured for go api startup"
else
  echo "jwt secret: not found (auth parity cases may fail if jwt verification is required)"
fi
if [[ -n "${resolved_database_url}" ]]; then
  echo "database url: configured for go api startup"
else
  echo "database url: not found (db-backed parity cases may fail with 503)"
fi

if ! health_check "${LEGACY_BASE_URL}" "${GO_HEALTH_PATH}"; then
  echo "error: legacy API is not reachable at ${LEGACY_BASE_URL}${GO_HEALTH_PATH}" >&2
  echo "start legacy API first (expected port: 3333)" >&2
  exit 1
fi

bootstrap_legacy_user

if health_check "${GO_BASE_URL}" "${GO_HEALTH_PATH}"; then
  echo "go api already running at ${GO_BASE_URL}"
else
  build_go_binary
  echo "starting go api on port ${GO_PORT}..."
  (
    if [[ -n "${resolved_jwt_secret}" ]]; then
      if [[ -n "${resolved_database_url}" ]]; then
        env PORT="${GO_PORT}" JWT_SECRET_KEY="${resolved_jwt_secret}" DATABASE_URL="${resolved_database_url}" "${GO_BINARY_PATH}"
      else
        env PORT="${GO_PORT}" JWT_SECRET_KEY="${resolved_jwt_secret}" "${GO_BINARY_PATH}"
      fi
    else
      if [[ -n "${resolved_database_url}" ]]; then
        env PORT="${GO_PORT}" DATABASE_URL="${resolved_database_url}" "${GO_BINARY_PATH}"
      else
        env PORT="${GO_PORT}" "${GO_BINARY_PATH}"
      fi
    fi
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
