#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# Seed ROAI calculator test fixtures into the database.
#
# Called from run.sh after user bootstrap. Seeds SymbolProfiles, MarketData,
# Accounts, and Orders that mirror the NestJS ROAI calculator spec files so
# that differential portfolio endpoint tests can compare real output.
#
# Required env vars (passed by run.sh):
#   LEGACY_BASE_URL   — base URL for the NestJS API
#   JWT_ROAI_TOKEN    — JWT for an isolated ROAI-only test user
#   DATABASE_URL      — Postgres connection string (for psql inserts)
#
# Exports on success:
#   ROAI_ACCOUNT_ID         — USD account used for ROAI orders
#   ROAI_CHF_ACCOUNT_ID     — CHF account used for ROAI orders
#   ROAI_BTCUSD_ORDER_ID    — BTC buy order
#   ROAI_MSFT_BUY_ORDER_ID  — MSFT buy order
#   ROAI_MSFT_DIV_ORDER_ID  — MSFT dividend order
#   ROAI_NOVN_BUY_ORDER_ID  — NOVN buy order
#   ROAI_NOVN_SELL_ORDER_ID — NOVN sell order
#   ROAI_BALN_BUY_ORDER_ID  — BALN buy order
#   ROAI_BALN_SELL_ORDER_ID — BALN sell order
#   ROAI_SEEDED=1           — flag indicating full deterministic seed completed
# ---------------------------------------------------------------------------
set -euo pipefail

LEGACY_BASE_URL="${LEGACY_BASE_URL:?LEGACY_BASE_URL is required}"
JWT_ROAI_TOKEN="${JWT_ROAI_TOKEN:-${JWT_USER_TOKEN:-}}"
JWT_ROAI_TOKEN="${JWT_ROAI_TOKEN:?JWT_ROAI_TOKEN is required}"
DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required}"

if ! command -v psql >/dev/null 2>&1; then
  echo "roai-seed: psql not found, skipping ROAI fixture seeding" >&2
  return 0 2>/dev/null || exit 0
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "roai-seed: jq not found, skipping ROAI fixture seeding" >&2
  return 0 2>/dev/null || exit 0
fi

fail() {
  echo "roai-seed: $*" >&2
  return 1
}

api() {
  local method="$1" path="$2"
  shift 2
  curl -fsS -X "${method}" "${LEGACY_BASE_URL}${path}" \
    -H "Authorization: Bearer ${JWT_ROAI_TOKEN}" \
    -H "Content-Type: application/json" \
    "$@"
}

sql() {
  psql "${DATABASE_URL}" -Atqc "$1"
}

echo "roai-seed: seeding ROAI calculator fixtures..."

# --------------------------------------------------------------------------
# 1. Create dedicated accounts for ROAI test orders (isolated user)
# --------------------------------------------------------------------------
suffix="$(date +%s)-$RANDOM"

roai_acct_response="$(api POST "/api/v1/account" \
  -d "{\"balance\":0,\"currency\":\"USD\",\"name\":\"roai-test-${suffix}\",\"platformId\":null,\"isExcluded\":false}")"
roai_acct_id="$(printf '%s' "${roai_acct_response}" | jq -r '.id // empty')"
if [[ -z "${roai_acct_id}" ]]; then
  fail "failed to create ROAI USD account"
fi
export ROAI_ACCOUNT_ID="${roai_acct_id}"

roai_chf_acct_response="$(api POST "/api/v1/account" \
  -d "{\"balance\":0,\"currency\":\"CHF\",\"name\":\"roai-test-chf-${suffix}\",\"platformId\":null,\"isExcluded\":false}")"
roai_chf_acct_id="$(printf '%s' "${roai_chf_acct_response}" | jq -r '.id // empty')"
if [[ -z "${roai_chf_acct_id}" ]]; then
  fail "failed to create ROAI CHF account"
fi
export ROAI_CHF_ACCOUNT_ID="${roai_chf_acct_id}"

# --------------------------------------------------------------------------
# 2. Seed SymbolProfiles via psql (no public API for this)
# --------------------------------------------------------------------------
seed_symbol_profile() {
  local symbol="$1" currency="$2" data_source="$3"
  sql "INSERT INTO \"SymbolProfile\" (
         id, symbol, currency, \"dataSource\", \"isActive\", \"createdAt\", \"updatedAt\"
       ) VALUES (
         gen_random_uuid(), '${symbol}', '${currency}', '${data_source}', true, NOW(), NOW()
       ) ON CONFLICT (\"dataSource\", symbol) DO NOTHING;"
}

seed_symbol_profile "BTCUSD" "USD" "YAHOO"
seed_symbol_profile "MSFT" "USD" "YAHOO"
seed_symbol_profile "NOVN.SW" "CHF" "YAHOO"
seed_symbol_profile "BALN.SW" "CHF" "YAHOO"

# Ensure core ROAI symbols carry deterministic metadata used by holdings/detail
# parity checks.
sql "UPDATE \"SymbolProfile\"
     SET name = 'Bitcoin USD',
         url = 'https://bitcoin.org/',
         countries = '[]'::jsonb,
         sectors = '[]'::jsonb,
         \"assetClass\" = 'LIQUIDITY'::\"AssetClass\",
         \"assetSubClass\" = 'CRYPTOCURRENCY'::\"AssetSubClass\",
         \"updatedAt\" = NOW()
     WHERE \"dataSource\" = 'YAHOO' AND symbol = 'BTCUSD';"

sql "UPDATE \"SymbolProfile\"
     SET name = 'Microsoft Corporation',
         url = 'https://www.microsoft.com',
         countries = '[{\"code\":\"US\",\"continent\":\"North America\",\"name\":\"United States\",\"weight\":1}]'::jsonb,
         sectors = '[{\"name\":\"Technology\",\"weight\":1}]'::jsonb,
         \"assetClass\" = 'EQUITY'::\"AssetClass\",
         \"assetSubClass\" = 'STOCK'::\"AssetSubClass\",
         \"updatedAt\" = NOW()
     WHERE \"dataSource\" = 'YAHOO' AND symbol = 'MSFT';"

# --------------------------------------------------------------------------
# 3. Seed MarketData via psql
# --------------------------------------------------------------------------
seed_market_data() {
  local symbol="$1" data_source="$2" date_str="$3" price="$4"
  sql "INSERT INTO \"MarketData\" (
         id, symbol, \"dataSource\", date, \"marketPrice\", state, \"createdAt\"
       ) VALUES (
         gen_random_uuid(), '${symbol}', '${data_source}',
         '${date_str}T00:00:00Z', ${price}, 'CLOSE', NOW()
       ) ON CONFLICT (\"dataSource\", date, symbol) DO UPDATE
         SET \"marketPrice\" = EXCLUDED.\"marketPrice\";"
}

seed_market_data "BTCUSD" "YAHOO" "2021-12-12" 50098.3
seed_market_data "BTCUSD" "YAHOO" "2022-01-14" 43099.7

seed_market_data "MSFT" "YAHOO" "2021-09-16" 89.12
seed_market_data "MSFT" "YAHOO" "2021-11-16" 339.51
seed_market_data "MSFT" "YAHOO" "2023-07-09" 337.22
seed_market_data "MSFT" "YAHOO" "2023-07-10" 331.83

seed_market_data "NOVN.SW" "YAHOO" "2022-04-11" 87.8

seed_market_data "BALN.SW" "YAHOO" "2021-11-12" 146
seed_market_data "BALN.SW" "YAHOO" "2021-11-22" 142.9
seed_market_data "BALN.SW" "YAHOO" "2021-11-26" 139.9
seed_market_data "BALN.SW" "YAHOO" "2021-11-30" 136.6
seed_market_data "BALN.SW" "YAHOO" "2021-12-12" 142.0
seed_market_data "BALN.SW" "YAHOO" "2021-12-17" 143.9
seed_market_data "BALN.SW" "YAHOO" "2021-12-18" 148.9

seed_market_data "USDCHF" "YAHOO" "2015-01-01" 0.9941099999999999
seed_market_data "USDCHF" "YAHOO" "2017-12-31" 0.9787
seed_market_data "USDCHF" "YAHOO" "2018-01-01" 0.97373
seed_market_data "USDCHF" "YAHOO" "2021-11-22" 0.9238
seed_market_data "USDCHF" "YAHOO" "2021-11-30" 0.9238
seed_market_data "USDCHF" "YAHOO" "2021-12-18" 0.9238
seed_market_data "USDCHF" "YAHOO" "2022-03-07" 0.9238
seed_market_data "USDCHF" "YAHOO" "2022-04-11" 0.9238
seed_market_data "USDCHF" "YAHOO" "2023-01-03" 0.9238
seed_market_data "USDCHF" "YAHOO" "2023-07-10" 0.8854

# --------------------------------------------------------------------------
# 4. Create ROAI orders directly in DB for deterministic, provider-independent
#    seed state. API-level symbol validation is intentionally bypassed here.
# --------------------------------------------------------------------------
symbol_profile_id() {
  local data_source="$1" symbol="$2"
  sql "SELECT id FROM \"SymbolProfile\" WHERE \"dataSource\" = '${data_source}' AND symbol = '${symbol}' LIMIT 1;"
}

insert_order() {
  local account_id="$1" account_user_id="$2" user_id="$3" currency="$4"
  local date_iso="$5" fee="$6" quantity="$7" symbol_profile_id="$8" type="$9" unit_price="${10}"
  local order_id
  order_id="$(sql "WITH inserted AS (
    INSERT INTO \"Order\" (
      id, \"accountId\", \"accountUserId\", \"userId\", currency, date, fee, quantity, \"symbolProfileId\", type, \"unitPrice\", \"isDraft\", \"createdAt\", \"updatedAt\"
    ) VALUES (
      gen_random_uuid(), '${account_id}', '${account_user_id}', '${user_id}', '${currency}',
      '${date_iso}'::timestamptz, ${fee}, ${quantity}, '${symbol_profile_id}', '${type}'::\"Type\", ${unit_price}, false, NOW(), NOW()
    ) RETURNING id
  ) SELECT id FROM inserted;")"
  if [[ -z "${order_id}" ]]; then
    fail "failed to insert order ${type} (${symbol_profile_id}) on ${date_iso}"
  fi
  printf '%s' "${order_id}"
}

sp_btcusd_id="$(symbol_profile_id "YAHOO" "BTCUSD")"
sp_msft_id="$(symbol_profile_id "YAHOO" "MSFT")"
sp_novn_id="$(symbol_profile_id "YAHOO" "NOVN.SW")"
sp_baln_id="$(symbol_profile_id "YAHOO" "BALN.SW")"

for sp_var in sp_btcusd_id sp_msft_id sp_novn_id sp_baln_id; do
  if [[ -z "${!sp_var:-}" ]]; then
    fail "missing symbol profile id for ${sp_var}"
  fi
done

roai_user_id="${ROAI_USER_ID:-}"
if [[ -z "${roai_user_id}" ]]; then
  fail "ROAI_USER_ID is required for deterministic order inserts"
fi

export ROAI_BTCUSD_ORDER_ID="$(insert_order "${roai_acct_id}" "${roai_user_id}" "${roai_user_id}" "USD" "2021-12-12T00:00:00.000Z" 4.46 1 "${sp_btcusd_id}" "BUY" 44558.42)"
export ROAI_MSFT_BUY_ORDER_ID="$(insert_order "${roai_acct_id}" "${roai_user_id}" "${roai_user_id}" "USD" "2021-09-16T00:00:00.000Z" 19 1 "${sp_msft_id}" "BUY" 298.58)"
export ROAI_MSFT_DIV_ORDER_ID="$(insert_order "${roai_acct_id}" "${roai_user_id}" "${roai_user_id}" "USD" "2021-11-16T00:00:00.000Z" 0 1 "${sp_msft_id}" "DIVIDEND" 0.62)"
export ROAI_NOVN_BUY_ORDER_ID="$(insert_order "${roai_chf_acct_id}" "${roai_user_id}" "${roai_user_id}" "CHF" "2022-03-07T00:00:00.000Z" 0 2 "${sp_novn_id}" "BUY" 75.8)"
export ROAI_NOVN_SELL_ORDER_ID="$(insert_order "${roai_chf_acct_id}" "${roai_user_id}" "${roai_user_id}" "CHF" "2022-04-08T00:00:00.000Z" 0 2 "${sp_novn_id}" "SELL" 85.73)"
export ROAI_BALN_BUY_ORDER_ID="$(insert_order "${roai_chf_acct_id}" "${roai_user_id}" "${roai_user_id}" "CHF" "2021-11-22T00:00:00.000Z" 1.55 2 "${sp_baln_id}" "BUY" 142.9)"
export ROAI_BALN_SELL_ORDER_ID="$(insert_order "${roai_chf_acct_id}" "${roai_user_id}" "${roai_user_id}" "CHF" "2021-11-30T00:00:00.000Z" 1.65 2 "${sp_baln_id}" "SELL" 136.6)"

# --------------------------------------------------------------------------
# 5. Enforce full seed completeness before enabling ROAI cases
# --------------------------------------------------------------------------
required_vars=(
  ROAI_ACCOUNT_ID
  ROAI_CHF_ACCOUNT_ID
  ROAI_BTCUSD_ORDER_ID
  ROAI_MSFT_BUY_ORDER_ID
  ROAI_MSFT_DIV_ORDER_ID
  ROAI_NOVN_BUY_ORDER_ID
  ROAI_NOVN_SELL_ORDER_ID
  ROAI_BALN_BUY_ORDER_ID
  ROAI_BALN_SELL_ORDER_ID
)
for var in "${required_vars[@]}"; do
  if [[ -z "${!var:-}" ]]; then
    fail "missing required seeded value: ${var}"
  fi
done

export ROAI_SEEDED=1
echo "roai-seed: complete (ROAI_SEEDED=1, ROAI account=${ROAI_ACCOUNT_ID}, ROAI CHF account=${ROAI_CHF_ACCOUNT_ID})"
