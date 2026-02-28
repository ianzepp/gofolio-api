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
# 4. Create orders via the legacy API (both APIs read same DB state)
# --------------------------------------------------------------------------
create_order() {
  local acct_id="$1" currency="$2" data_source="$3" date="$4" fee="$5"
  local quantity="$6" symbol="$7" type="$8" unit_price="$9"

  local body
  body="$(jq -nc \
    --arg accountId "${acct_id}" \
    --arg currency "${currency}" \
    --arg dataSource "${data_source}" \
    --arg date "${date}" \
    --argjson fee "${fee}" \
    --argjson quantity "${quantity}" \
    --arg symbol "${symbol}" \
    --arg type "${type}" \
    --argjson unitPrice "${unit_price}" \
    '{
      accountId: $accountId,
      currency: $currency,
      dataSource: $dataSource,
      date: $date,
      fee: $fee,
      quantity: $quantity,
      symbol: $symbol,
      type: $type,
      unitPrice: $unitPrice
    }')"

  local resp
  local order_id
  resp="$(api POST "/api/v1/order" -d "${body}")"
  order_id="$(printf '%s' "${resp}" | jq -r '.id // empty' 2>/dev/null || true)"
  if [[ -z "${order_id}" ]]; then
    fail "failed to create order ${type} ${symbol} on ${date}"
  fi
  printf '%s' "${order_id}"
}

export ROAI_BTCUSD_ORDER_ID="$(create_order "${roai_acct_id}" "USD" "YAHOO" "2021-12-12T00:00:00.000Z" 4.46 1 "BTCUSD" "BUY" 44558.42)"
export ROAI_MSFT_BUY_ORDER_ID="$(create_order "${roai_acct_id}" "USD" "YAHOO" "2021-09-16T00:00:00.000Z" 19 1 "MSFT" "BUY" 298.58)"
export ROAI_MSFT_DIV_ORDER_ID="$(create_order "${roai_acct_id}" "USD" "YAHOO" "2021-11-16T00:00:00.000Z" 0 1 "MSFT" "DIVIDEND" 0.62)"
export ROAI_NOVN_BUY_ORDER_ID="$(create_order "${roai_chf_acct_id}" "CHF" "YAHOO" "2022-03-07T00:00:00.000Z" 0 2 "NOVN.SW" "BUY" 75.8)"
export ROAI_NOVN_SELL_ORDER_ID="$(create_order "${roai_chf_acct_id}" "CHF" "YAHOO" "2022-04-08T00:00:00.000Z" 0 2 "NOVN.SW" "SELL" 85.73)"
export ROAI_BALN_BUY_ORDER_ID="$(create_order "${roai_chf_acct_id}" "CHF" "YAHOO" "2021-11-22T00:00:00.000Z" 1.55 2 "BALN.SW" "BUY" 142.9)"
export ROAI_BALN_SELL_ORDER_ID="$(create_order "${roai_chf_acct_id}" "CHF" "YAHOO" "2021-11-30T00:00:00.000Z" 1.65 2 "BALN.SW" "SELL" 136.6)"

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
