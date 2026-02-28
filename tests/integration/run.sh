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

  user_id="$(decode_jwt_user_id "${auth_token}")"

  export JWT_TOKEN="${auth_token}"
  export JWT_USER_TOKEN="${auth_token}"
  export LEGACY_BOOTSTRAP_AUTH_TOKEN="${auth_token}"
  if [[ -n "${access_token}" ]]; then
    export LEGACY_BOOTSTRAP_ACCESS_TOKEN="${access_token}"
  fi
  if [[ -n "${user_id}" ]]; then
    export LEGACY_USER_ID="${user_id}"
  fi

  # Bootstrap an additional throwaway user for destructive/edge parity cases.
  # This user is expected to have no linked data (orders/accounts/etc.).
  local empty_signup_response
  local empty_access_token
  local empty_auth_token
  local empty_user_id
  empty_signup_response="$(curl -fsS -X POST "${LEGACY_BASE_URL}/api/v1/user")"
  empty_access_token="$(printf '%s' "${empty_signup_response}" | jq -r '.accessToken // empty')"
  empty_auth_token="$(printf '%s' "${empty_signup_response}" | jq -r '.authToken // empty')"
  if [[ -z "${empty_auth_token}" && -n "${empty_access_token}" ]]; then
    empty_auth_token="$(curl -fsS -X POST "${LEGACY_BASE_URL}/api/v1/auth/anonymous" \
      -H "Content-Type: application/json" \
      -d "$(jq -nc --arg accessToken "${empty_access_token}" '{accessToken: $accessToken}')" \
      | jq -r '.authToken // empty')"
  fi
  if [[ -n "${empty_auth_token}" ]]; then
    export JWT_EMPTY_USER_TOKEN="${empty_auth_token}"
    empty_user_id="$(decode_jwt_user_id "${empty_auth_token}")"
    if [[ -n "${empty_user_id}" ]]; then
      export EMPTY_USER_ID="${empty_user_id}"
    fi
  fi
  if [[ -n "${empty_access_token}" ]]; then
    export EMPTY_USER_ACCESS_TOKEN="${empty_access_token}"
  fi

  # Bootstrap two accounts for transfer-balance parity checks.
  if [[ -n "${auth_token}" ]]; then
    local transfer_from_response transfer_to_response transfer_from_id transfer_to_id
    transfer_from_response="$(curl -fsS -X POST "${LEGACY_BASE_URL}/api/v1/account" \
      -H "Authorization: Bearer ${auth_token}" \
      -H "Content-Type: application/json" \
      -d '{"balance":100,"currency":"USD","name":"integration-transfer-from","platformId":null,"isExcluded":false}' 2>/dev/null || true)"
    transfer_from_id="$(printf '%s' "${transfer_from_response}" | jq -r '.id // empty' 2>/dev/null || true)"

    transfer_to_response="$(curl -fsS -X POST "${LEGACY_BASE_URL}/api/v1/account" \
      -H "Authorization: Bearer ${auth_token}" \
      -H "Content-Type: application/json" \
      -d '{"balance":0,"currency":"USD","name":"integration-transfer-to","platformId":null,"isExcluded":false}' 2>/dev/null || true)"
    transfer_to_id="$(printf '%s' "${transfer_to_response}" | jq -r '.id // empty' 2>/dev/null || true)"

    if [[ -n "${transfer_from_id}" && -n "${transfer_to_id}" ]]; then
      export TRANSFER_ACCOUNT_FROM_ID="${transfer_from_id}"
      export TRANSFER_ACCOUNT_TO_ID="${transfer_to_id}"
    fi
  fi

  # For foreign-user validation cases, we only need an ID that differs from
  # the bootstrapped user ID.
  if [[ -z "${ADMIN_USER_ID:-}" ]]; then
    local foreign_id
    foreign_id="$(uuidgen 2>/dev/null || true)"
    if [[ -z "${foreign_id}" ]]; then
      foreign_id="00000000-0000-4000-8000-000000000000"
    fi
    if [[ "${foreign_id}" == "${user_id}" ]]; then
      foreign_id="11111111-1111-4111-8111-111111111111"
    fi
    export ADMIN_USER_ID="${foreign_id}"
  fi

  # Try to bootstrap demo credentials from /info demoAuthToken.
  local info_response
  info_response="$(curl -fsS "${LEGACY_BASE_URL}/api/v1/info" || true)"
  if [[ -n "${info_response}" ]]; then
    local demo_auth_token
    demo_auth_token="$(printf '%s' "${info_response}" | jq -r '.demoAuthToken // empty' 2>/dev/null || true)"
    if [[ -n "${demo_auth_token}" ]]; then
      export JWT_DEMO_TOKEN="${demo_auth_token}"
      local demo_user_id
      demo_user_id="$(decode_jwt_user_id "${demo_auth_token}")"
      if [[ -n "${demo_user_id}" ]]; then
        export DEMO_USER_ID="${demo_user_id}"
      fi
    fi
  fi

  # Opportunistically bootstrap API key for the bootstrapped user.
  if [[ -z "${API_KEY_USER:-}" ]]; then
    local api_key_response
    api_key_response="$(curl -fsS -X POST "${LEGACY_BASE_URL}/api/v1/api-keys" \
      -H "Authorization: Bearer ${auth_token}" \
      -H "Content-Type: application/json" 2>/dev/null || true)"
    local api_key
    api_key="$(printf '%s' "${api_key_response}" | jq -r '.apiKey // empty' 2>/dev/null || true)"
    if [[ -n "${api_key}" ]]; then
      export API_KEY_USER="${api_key}"
    fi
  fi

  # Fallback: bootstrap API key via an admin user discovered from DB.
  if [[ -z "${API_KEY_USER:-}" ]]; then
    bootstrap_api_key_from_admin || true
  fi

  # Final fallback: keep API-key parity case active even when key creation is
  # forbidden in this environment. Both sides should reject this dummy key.
  if [[ -z "${API_KEY_USER:-}" ]]; then
    export API_KEY_USER="integration-dummy-api-key"
  fi

  # Fallback for demo-tag parity checks when /info lacks demoAuthToken.
  if [[ -z "${JWT_DEMO_TOKEN:-}" || -z "${DEMO_USER_ID:-}" ]]; then
    bootstrap_demo_from_db || true
  fi

  echo "bootstrapped legacy user for test run (jwt exported as JWT_USER_TOKEN/JWT_TOKEN${user_id:+, userId=${user_id}})"
}

decode_jwt_user_id() {
  local jwt_token="$1"
  if [[ -z "${jwt_token}" ]]; then
    return 0
  fi

  printf '%s' "${jwt_token}" | jq -Rr '
    def b64url_to_json:
      gsub("-"; "+")
      | gsub("_"; "/")
      | . + (["", "===", "==", "="][(length % 4)])
      | @base64d
      | fromjson;
    (split(".")[1] // "") | b64url_to_json | .id // empty
  ' 2>/dev/null || true
}

make_hs256_jwt() {
  local user_id="$1"
  local jwt_secret="$2"
  if [[ -z "${user_id}" || -z "${jwt_secret}" ]]; then
    return 1
  fi
  if ! command -v openssl >/dev/null 2>&1; then
    return 1
  fi

  local header payload encoded_header encoded_payload signing_input encoded_signature
  header='{"alg":"HS256","typ":"JWT"}'
  payload="$(jq -nc --arg id "${user_id}" '{id: $id}')"

  encoded_header="$(printf '%s' "${header}" | openssl base64 -A | tr '+/' '-_' | tr -d '=')"
  encoded_payload="$(printf '%s' "${payload}" | openssl base64 -A | tr '+/' '-_' | tr -d '=')"
  signing_input="${encoded_header}.${encoded_payload}"
  encoded_signature="$(printf '%s' "${signing_input}" \
    | openssl dgst -sha256 -hmac "${jwt_secret}" -binary \
    | openssl base64 -A | tr '+/' '-_' | tr -d '=')"
  printf '%s' "${signing_input}.${encoded_signature}"
}

bootstrap_api_key_from_admin() {
  if ! command -v psql >/dev/null 2>&1; then
    return 0
  fi
  if [[ -z "${resolved_database_url}" || -z "${resolved_jwt_secret}" ]]; then
    return 0
  fi

  local admin_user_id
  admin_user_id="$(psql "${resolved_database_url}" -Atqc \
    "SELECT id FROM \"User\" WHERE role = 'ADMIN' ORDER BY \"createdAt\" ASC LIMIT 1;" 2>/dev/null || true)"
  admin_user_id="$(printf '%s' "${admin_user_id}" | tr -d '[:space:]')"
  if [[ -z "${admin_user_id}" ]]; then
    return 0
  fi

  local admin_jwt
  admin_jwt="$(make_hs256_jwt "${admin_user_id}" "${resolved_jwt_secret}" || true)"
  if [[ -z "${admin_jwt}" ]]; then
    return 0
  fi

  local api_key_response api_key
  api_key_response="$(curl -fsS -X POST "${LEGACY_BASE_URL}/api/v1/api-keys" \
    -H "Authorization: Bearer ${admin_jwt}" \
    -H "Content-Type: application/json" 2>/dev/null || true)"
  api_key="$(printf '%s' "${api_key_response}" | jq -r '.apiKey // empty' 2>/dev/null || true)"
  if [[ -n "${api_key}" ]]; then
    export API_KEY_USER="${api_key}"
    echo "bootstrapped API_KEY_USER via admin user from database"
  fi
}

bootstrap_demo_from_db() {
  if ! command -v psql >/dev/null 2>&1; then
    return 0
  fi
  if [[ -z "${resolved_database_url}" || -z "${resolved_jwt_secret}" ]]; then
    return 0
  fi

  local demo_user_id
  demo_user_id="$(psql "${resolved_database_url}" -Atqc \
    "SELECT id FROM \"User\" WHERE role = 'DEMO' ORDER BY \"createdAt\" ASC LIMIT 1;" 2>/dev/null || true)"
  demo_user_id="$(printf '%s' "${demo_user_id}" | tr -d '[:space:]')"
  if [[ -z "${demo_user_id}" ]]; then
    demo_user_id="$(psql "${resolved_database_url}" -Atqc \
      "SELECT value FROM \"Property\" WHERE key = 'DEMO_USER_ID' LIMIT 1;" 2>/dev/null || true)"
    demo_user_id="$(printf '%s' "${demo_user_id}" | tr -d '[:space:]')"
  fi
  if [[ -z "${demo_user_id}" ]]; then
    local demo_signup_response demo_signup_token
    demo_signup_response="$(curl -fsS -X POST "${LEGACY_BASE_URL}/api/v1/user" 2>/dev/null || true)"
    demo_signup_token="$(printf '%s' "${demo_signup_response}" | jq -r '.authToken // empty' 2>/dev/null || true)"
    demo_user_id="$(decode_jwt_user_id "${demo_signup_token}")"
    if [[ -n "${demo_user_id}" ]]; then
      psql "${resolved_database_url}" -Atqc \
        "UPDATE \"User\" SET role = 'DEMO' WHERE id = '${demo_user_id}';" >/dev/null 2>&1 || true
    fi
  fi
  if [[ -z "${demo_user_id}" ]]; then
    return 0
  fi

  local demo_jwt
  demo_jwt="$(make_hs256_jwt "${demo_user_id}" "${resolved_jwt_secret}" || true)"
  if [[ -z "${demo_jwt}" ]]; then
    return 0
  fi

  export DEMO_USER_ID="${demo_user_id}"
  export JWT_DEMO_TOKEN="${demo_jwt}"
  echo "bootstrapped DEMO_USER_ID/JWT_DEMO_TOKEN via database"
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
