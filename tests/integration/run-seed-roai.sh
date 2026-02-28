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
#   JWT_USER_TOKEN    — JWT for the bootstrapped test user
#   DATABASE_URL      — Postgres connection string (for psql inserts)
#
# Exports on success:
#   ROAI_ACCOUNT_ID        — account used for ROAI orders
#   ROAI_BTCUSD_ORDER_ID   — BTC buy order
#   ROAI_MSFT_BUY_ORDER_ID — MSFT buy order
#   ROAI_MSFT_DIV_ORDER_ID — MSFT dividend order
#   ROAI_NOVN_BUY_ORDER_ID — NOVN buy order
#   ROAI_NOVN_SELL_ORDER_ID— NOVN sell order
#   ROAI_BALN_BUY_ORDER_ID — BALN buy order
#   ROAI_BALN_SELL_ORDER_ID— BALN sell order
#   ROAI_SEEDED=1          — flag indicating seed completed
# ---------------------------------------------------------------------------
set -euo pipefail

LEGACY_BASE_URL="${LEGACY_BASE_URL:?LEGACY_BASE_URL is required}"
JWT_USER_TOKEN="${JWT_USER_TOKEN:?JWT_USER_TOKEN is required}"
DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required}"

if ! command -v psql >/dev/null 2>&1; then
  echo "roai-seed: psql not found, skipping ROAI fixture seeding" >&2
  return 0 2>/dev/null || exit 0
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "roai-seed: jq not found, skipping ROAI fixture seeding" >&2
  return 0 2>/dev/null || exit 0
fi

api() {
  local method="$1" path="$2"
  shift 2
  curl -fsS -X "${method}" "${LEGACY_BASE_URL}${path}" \
    -H "Authorization: Bearer ${JWT_USER_TOKEN}" \
    -H "Content-Type: application/json" \
    "$@" 2>/dev/null
}

sql() {
  psql "${DATABASE_URL}" -Atqc "$1" 2>/dev/null
}

echo "roai-seed: seeding ROAI calculator fixtures..."

# --------------------------------------------------------------------------
# 1. Create a dedicated account for ROAI test orders (USD)
# --------------------------------------------------------------------------
roai_acct_response="$(api POST "/api/v1/account" \
  -d '{"balance":0,"currency":"USD","name":"roai-test","platformId":null,"isExcluded":false}' || true)"
roai_acct_id="$(printf '%s' "${roai_acct_response}" | jq -r '.id // empty' || true)"

if [[ -z "${roai_acct_id}" ]]; then
  echo "roai-seed: failed to create ROAI test account, skipping" >&2
  return 0 2>/dev/null || exit 0
fi
export ROAI_ACCOUNT_ID="${roai_acct_id}"

# Also create a CHF account for CHF-denominated orders
roai_chf_acct_response="$(api POST "/api/v1/account" \
  -d '{"balance":0,"currency":"CHF","name":"roai-test-chf","platformId":null,"isExcluded":false}' || true)"
roai_chf_acct_id="$(printf '%s' "${roai_chf_acct_response}" | jq -r '.id // empty' || true)"

# --------------------------------------------------------------------------
# 2. Seed SymbolProfiles via psql (no public API for this)
#    ON CONFLICT DO NOTHING so re-runs are safe.
# --------------------------------------------------------------------------
seed_symbol_profile() {
  local symbol="$1" currency="$2" data_source="$3" name="$4"
  sql "INSERT INTO \"SymbolProfile\" (
         id, symbol, currency, \"dataSource\", \"isActive\", \"createdAt\", \"updatedAt\"
       ) VALUES (
         gen_random_uuid(), '${symbol}', '${currency}', '${data_source}', true, NOW(), NOW()
       ) ON CONFLICT (\"dataSource\", symbol) DO NOTHING;"
}

seed_symbol_profile "BTCUSD"  "USD" "YAHOO" "Bitcoin USD"
seed_symbol_profile "MSFT"    "USD" "YAHOO" "Microsoft Corporation"
seed_symbol_profile "NOVN.SW" "CHF" "YAHOO" "Novartis AG"
seed_symbol_profile "BALN.SW" "CHF" "YAHOO" "Baloise Holding AG"

# --------------------------------------------------------------------------
# 3. Seed MarketData via psql
#    These are the exact mock prices from the ROAI spec files.
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

# BTCUSD prices (from portfolio-calculator-btcusd.spec.ts)
# System time: 2022-01-14
seed_market_data "BTCUSD" "YAHOO" "2021-12-12" 50098.3
seed_market_data "BTCUSD" "YAHOO" "2022-01-14" 43099.7

# MSFT prices (from portfolio-calculator-msft-buy-with-dividend.spec.ts)
# System time: 2023-07-10
seed_market_data "MSFT" "YAHOO" "2021-09-16" 89.12
seed_market_data "MSFT" "YAHOO" "2021-11-16" 339.51
seed_market_data "MSFT" "YAHOO" "2023-07-09" 337.22
seed_market_data "MSFT" "YAHOO" "2023-07-10" 331.83

# NOVN.SW prices (from portfolio-calculator-novn-buy-and-sell.spec.ts)
# System time: 2022-04-11
seed_market_data "NOVN.SW" "YAHOO" "2022-04-11" 87.8

# BALN.SW prices (from portfolio-calculator-baln-buy-and-sell.spec.ts)
# System time: 2021-12-18
seed_market_data "BALN.SW" "YAHOO" "2021-11-12" 146
seed_market_data "BALN.SW" "YAHOO" "2021-11-22" 142.9
seed_market_data "BALN.SW" "YAHOO" "2021-11-26" 139.9
seed_market_data "BALN.SW" "YAHOO" "2021-11-30" 136.6
seed_market_data "BALN.SW" "YAHOO" "2021-12-12" 142.0
seed_market_data "BALN.SW" "YAHOO" "2021-12-17" 143.9
seed_market_data "BALN.SW" "YAHOO" "2021-12-18" 148.9

# --------------------------------------------------------------------------
# 4. Seed exchange rates via psql (stored in MarketData as currency pairs)
#    The NestJS ExchangeRateDataService reads from MarketData with
#    synthetic pair symbols like "USDCHF".
# --------------------------------------------------------------------------

# USD→CHF rates used by CHF-base-currency tests
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
# 5. Create orders via the legacy API (ensures both APIs see them)
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
  resp="$(api POST "/api/v1/order" -d "${body}" || true)"
  printf '%s' "${resp}" | jq -r '.id // empty' 2>/dev/null || true
}

# -- BTCUSD: BUY 1 @ 44558.42, fee 4.46, 2021-12-12 --
btcusd_order_id="$(create_order "${roai_acct_id}" "USD" "YAHOO" \
  "2021-12-12T00:00:00.000Z" 4.46 1 "BTCUSD" "BUY" 44558.42)"
if [[ -n "${btcusd_order_id}" ]]; then
  export ROAI_BTCUSD_ORDER_ID="${btcusd_order_id}"
fi

# -- MSFT: BUY 1 @ 298.58, fee 19, 2021-09-16 --
msft_buy_id="$(create_order "${roai_acct_id}" "USD" "YAHOO" \
  "2021-09-16T00:00:00.000Z" 19 1 "MSFT" "BUY" 298.58)"
if [[ -n "${msft_buy_id}" ]]; then
  export ROAI_MSFT_BUY_ORDER_ID="${msft_buy_id}"
fi

# -- MSFT: DIVIDEND 1 @ 0.62, fee 0, 2021-11-16 --
msft_div_id="$(create_order "${roai_acct_id}" "USD" "YAHOO" \
  "2021-11-16T00:00:00.000Z" 0 1 "MSFT" "DIVIDEND" 0.62)"
if [[ -n "${msft_div_id}" ]]; then
  export ROAI_MSFT_DIV_ORDER_ID="${msft_div_id}"
fi

# -- NOVN.SW: BUY 2 @ 75.8, fee 0, 2022-03-07 --
chf_acct="${roai_chf_acct_id:-${roai_acct_id}}"
novn_buy_id="$(create_order "${chf_acct}" "CHF" "YAHOO" \
  "2022-03-07T00:00:00.000Z" 0 2 "NOVN.SW" "BUY" 75.8)"
if [[ -n "${novn_buy_id}" ]]; then
  export ROAI_NOVN_BUY_ORDER_ID="${novn_buy_id}"
fi

# -- NOVN.SW: SELL 2 @ 85.73, fee 0, 2022-04-08 --
novn_sell_id="$(create_order "${chf_acct}" "CHF" "YAHOO" \
  "2022-04-08T00:00:00.000Z" 0 2 "NOVN.SW" "SELL" 85.73)"
if [[ -n "${novn_sell_id}" ]]; then
  export ROAI_NOVN_SELL_ORDER_ID="${novn_sell_id}"
fi

# -- BALN.SW: BUY 2 @ 142.9, fee 1.55, 2021-11-22 --
baln_buy_id="$(create_order "${chf_acct}" "CHF" "YAHOO" \
  "2021-11-22T00:00:00.000Z" 1.55 2 "BALN.SW" "BUY" 142.9)"
if [[ -n "${baln_buy_id}" ]]; then
  export ROAI_BALN_BUY_ORDER_ID="${baln_buy_id}"
fi

# -- BALN.SW: SELL 2 @ 136.6, fee 1.65, 2021-11-30 --
baln_sell_id="$(create_order "${chf_acct}" "CHF" "YAHOO" \
  "2021-11-30T00:00:00.000Z" 1.65 2 "BALN.SW" "SELL" 136.6)"
if [[ -n "${baln_sell_id}" ]]; then
  export ROAI_BALN_SELL_ORDER_ID="${baln_sell_id}"
fi

# --------------------------------------------------------------------------
# Summary
# --------------------------------------------------------------------------
seeded=0
for var in ROAI_BTCUSD_ORDER_ID ROAI_MSFT_BUY_ORDER_ID ROAI_NOVN_BUY_ORDER_ID ROAI_BALN_BUY_ORDER_ID; do
  if [[ -n "${!var:-}" ]]; then
    seeded=$((seeded + 1))
  fi
done

echo "roai-seed: seeded ${seeded}/4 ROAI scenarios (BTCUSD=${ROAI_BTCUSD_ORDER_ID:-skip}, MSFT=${ROAI_MSFT_BUY_ORDER_ID:-skip}, NOVN=${ROAI_NOVN_BUY_ORDER_ID:-skip}, BALN=${ROAI_BALN_BUY_ORDER_ID:-skip})"

if [[ "${seeded}" -gt 0 ]]; then
  export ROAI_SEEDED=1
fi
