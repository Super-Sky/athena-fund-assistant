#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_NAME="${PROJECT_NAME:-athena-fund-dual-smoke}"
ATHENA_REPO="${ATHENA_REPO:-$(cd "${ROOT_DIR}/.." && pwd)/Athena}"
ATHENA_PORT="${ATHENA_PORT:-18080}"
FUND_PORT="${FUND_PORT:-18081}"
WEB_PORT="${WEB_PORT:-18082}"
FAKE_MODEL_PORT="${FAKE_MODEL_PORT:-18083}"
POSTGRES_PORT="${POSTGRES_PORT:-15433}"
REDIS_PORT="${REDIS_PORT:-16380}"
KEEP_DOCKER_SMOKE="${KEEP_DOCKER_SMOKE:-0}"
REMOTE_TOOL_TOKEN="${ATHENA_FUND_REMOTE_TOOL_TOKEN:-}"
WRONG_REMOTE_TOOL_TOKEN=""

compose() {
  ATHENA_REPO="${ATHENA_REPO}" \
    ATHENA_FUND_REMOTE_TOOL_TOKEN="${REMOTE_TOOL_TOKEN}" \
    ATHENA_FUND_REMOTE_TOOL_WRONG_TOKEN="${WRONG_REMOTE_TOOL_TOKEN}" \
    ATHENA_DUAL_API_PORT="${ATHENA_PORT}" \
    ATHENA_FUND_API_PORT="${FUND_PORT}" \
    ATHENA_FUND_WEB_PORT="${WEB_PORT}" \
    ATHENA_FAKE_MODEL_PORT="${FAKE_MODEL_PORT}" \
    ATHENA_FUND_POSTGRES_PORT="${POSTGRES_PORT}" \
    ATHENA_FUND_REDIS_PORT="${REDIS_PORT}" \
    docker compose \
      -p "${PROJECT_NAME}" \
      -f "${ROOT_DIR}/docker-compose.yml" \
      -f "${ROOT_DIR}/docker-compose.dual.yml" \
      "$@"
}

cleanup() {
  if [[ "${KEEP_DOCKER_SMOKE}" != "1" ]]; then
    compose down -v --remove-orphans >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

wait_for_http() {
  local url="$1"
  local label="$2"
  for _ in $(seq 1 120); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "timeout waiting for ${label}: ${url}" >&2
  compose ps >&2 || true
  exit 1
}

require_cmd curl
require_cmd docker
require_cmd node
require_cmd openssl

if [[ -z "${REMOTE_TOOL_TOKEN}" ]]; then
  REMOTE_TOOL_TOKEN="$(openssl rand -hex 32)"
fi
WRONG_REMOTE_TOOL_TOKEN="${REMOTE_TOOL_TOKEN}-wrong"

if [[ ! -d "${ATHENA_REPO}" ]]; then
  echo "ATHENA_REPO does not exist: ${ATHENA_REPO}" >&2
  exit 1
fi

compose down -v --remove-orphans >/dev/null 2>&1 || true
compose up -d --build

wait_for_http "http://127.0.0.1:${FAKE_MODEL_PORT}/v1/chat/completions" "fake model"
wait_for_http "http://127.0.0.1:${ATHENA_PORT}/healthz" "Athena"
wait_for_http "http://127.0.0.1:${FUND_PORT}/healthz" "fund assistant"
wait_for_http "http://127.0.0.1:${WEB_PORT}/healthz" "fund assistant web proxy"

smoke_dir="$(mktemp -d /tmp/athena-fund-dual-docker.XXXXXX)"

session_response="$(curl -fsS -X POST "http://127.0.0.1:${FUND_PORT}/api/auth/sessions" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"demo-user","ttl_seconds":3600}')"
session_token="$(printf '%s' "${session_response}" | node -pe 'JSON.parse(require("fs").readFileSync(0, "utf8")).token')"
printf '%s' "${session_response}" | node -e 'const value=JSON.parse(require("fs").readFileSync(0,"utf8"));process.stdout.write(JSON.stringify(value.session))' > "${smoke_dir}/fund-session.json"
unset session_response

curl -fsS -X POST "http://127.0.0.1:${FUND_PORT}/api/consents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${session_token}" \
  -d '{"audience":"athena-runtime","scopes":["fund.account.summary.read","fund.holding.snapshot.read"],"ttl_seconds":3600}' \
  > "${smoke_dir}/fund-consent.json"
consent_grant_ref="$(node -pe 'JSON.parse(require("fs").readFileSync(0, "utf8")).ref' < "${smoke_dir}/fund-consent.json")"

curl -fsS -X POST "http://127.0.0.1:${ATHENA_PORT}/api/models/providers" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"fake-openai-docker-smoke",
    "protocol":"openai_compatible",
    "base_url":"http://fake-openai:18083/v1",
    "api_key":"sk-fake",
    "request_timeout_seconds":10,
    "models":[{"model_id":"fake-tool-model","display_name":"Fake Tool Model","is_default":true,"is_fallback":true}]
  }' > "${smoke_dir}/athena-model-provider.json"

curl -fsS "http://127.0.0.1:${FUND_PORT}/internal/tools/catalog?base_url=http://api:8081" > "${smoke_dir}/fund-tool-catalog.json"
node - "${smoke_dir}/fund-tool-catalog.json" "${smoke_dir}" <<'NODE'
const fs = require("fs");
const catalog = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
for (const item of catalog.items) {
  fs.writeFileSync(`${process.argv[3]}/fund-tool-${item.name}.json`, JSON.stringify(item));
  if (item.name === "account_overview") {
    fs.writeFileSync(`${process.argv[3]}/fund-tool-${item.name}-wrong-auth.json`, JSON.stringify({
      ...item,
      auth: { type: "bearer", secret_ref: "env://ATHENA_FUND_REMOTE_TOOL_WRONG_TOKEN" },
    }));
  }
}
console.log(JSON.stringify({ catalog_items: catalog.items.map(item => item.name) }));
NODE

curl -fsS -X PUT "http://127.0.0.1:${ATHENA_PORT}/api/control-plane/remote-tools/account_overview" \
  -H "Content-Type: application/json" \
  --data-binary "@${smoke_dir}/fund-tool-account_overview-wrong-auth.json" \
  > "${smoke_dir}/athena-register-account_overview-wrong-auth.json"

wrong_auth_status="$(curl -sS -o "${smoke_dir}/athena-agent-run-wrong-service-auth.json" -w '%{http_code}' \
  -X POST "http://127.0.0.1:${ATHENA_PORT}/api/agent/runs" \
  -H "Content-Type: application/json" \
  -d "{
    \"goal\":\"读取账户概览；consent_grant_ref=${consent_grant_ref}\",
    \"app_instance_id\":\"athena-fund-assistant\",
    \"app_session_id\":\"dual-docker-wrong-service-auth\",
    \"enabled_tools\":[\"account_overview\"],
    \"tools\":[{\"type\":\"function\",\"function\":{\"name\":\"account_overview\",\"description\":\"Read account overview\",\"parameters\":{\"type\":\"object\",\"properties\":{\"user_id\":{\"type\":\"string\"},\"consent_grant_ref\":{\"type\":\"string\"}},\"required\":[\"user_id\",\"consent_grant_ref\"]}}}],
    \"constraints\":{\"no_auto_trading\":true}
  }")"
if [[ "${wrong_auth_status}" != "500" ]]; then
  echo "expected wrong service auth run to return 500, got ${wrong_auth_status}" >&2
  exit 1
fi

node - "${smoke_dir}/athena-agent-run-wrong-service-auth.json" "${REMOTE_TOOL_TOKEN}" "${WRONG_REMOTE_TOOL_TOKEN}" <<'NODE'
const fs = require("fs");
const encoded = fs.readFileSync(process.argv[2], "utf8");
if (!encoded.includes("service_auth_denied")) throw new Error(`wrong service token was not denied: ${encoded}`);
if (encoded.includes(process.argv[3]) || encoded.includes(process.argv[4])) throw new Error("agent run leaked a service token");
console.log(JSON.stringify({ wrong_service_auth: "denied" }));
NODE

for name in account_overview fund_market_snapshot; do
  curl -fsS -X PUT "http://127.0.0.1:${ATHENA_PORT}/api/control-plane/remote-tools/${name}" \
    -H "Content-Type: application/json" \
    --data-binary "@${smoke_dir}/fund-tool-${name}.json" \
    > "${smoke_dir}/athena-register-${name}.json"
done

curl -fsS -X POST "http://127.0.0.1:${ATHENA_PORT}/api/agent/runs" \
  -H "Content-Type: application/json" \
  -d "{
    \"goal\":\"请读取账户概览并给出复盘重点；consent_grant_ref=${consent_grant_ref}\",
    \"app_instance_id\":\"athena-fund-assistant\",
    \"app_session_id\":\"dual-docker-direct\",
    \"enabled_tools\":[\"account_overview\"],
    \"tools\":[{
      \"type\":\"function\",
      \"function\":{
        \"name\":\"account_overview\",
        \"description\":\"Read account overview\",
        \"parameters\":{\"type\":\"object\",\"properties\":{\"user_id\":{\"type\":\"string\"},\"consent_grant_ref\":{\"type\":\"string\"}},\"required\":[\"user_id\",\"consent_grant_ref\"]}
      }
    }],
    \"constraints\":{\"no_auto_trading\":true}
  }" > "${smoke_dir}/athena-agent-run.json"

node - "${smoke_dir}/athena-agent-run.json" "${consent_grant_ref}" <<'NODE'
const fs = require("fs");
const run = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
if (run.status !== "completed") throw new Error(`Athena run status ${run.status}`);
if (!Array.isArray(run.tool_calls) || run.tool_calls.length < 1) throw new Error("Athena run did not return tool_calls");
const toolMessage = (run.messages || []).find(message => message.role === "tool" && message.name === "account_overview");
if (!toolMessage || toolMessage.status !== "completed") throw new Error("Athena run did not complete account_overview");
const toolContent = JSON.parse(toolMessage.content || "{}");
if (toolContent.tool !== "account_overview" || toolContent.overview?.account?.user_id !== "demo-user") {
  throw new Error(`unexpected account overview content: ${toolMessage.content}`);
}
if (toolContent.safety?.read_only !== true || toolContent.safety?.consent_grant_ref !== process.argv[3] || Number(toolContent.safety?.consent_revision || 0) < 1) {
  throw new Error(`account overview safety evidence missing: ${JSON.stringify(toolContent.safety)}`);
}
console.log(JSON.stringify({
  athena_status: run.status,
  tool_calls: run.tool_calls.map(call => call.function?.name || call.name),
  output: run.output,
}, null, 2));
NODE

conversation_id="$(
  curl -fsS -X POST "http://127.0.0.1:${FUND_PORT}/api/conversations" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${session_token}" \
    -d '{"user_id":"demo-user","skill_id":"portfolio_review","title":"dual docker smoke"}' \
    | node -pe 'JSON.parse(require("fs").readFileSync(0, "utf8")).session.id'
)"

curl -fsS -X POST "http://127.0.0.1:${FUND_PORT}/api/conversations/${conversation_id}/messages" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${session_token}" \
  -d "{\"role\":\"user\",\"content\":\"请读取账户概览并给我复盘重点；consent_grant_ref=${consent_grant_ref}\",\"skill_id\":\"portfolio_review\",\"consent_grant_ref\":\"${consent_grant_ref}\"}" \
  > "${smoke_dir}/fund-conversation-message.json"

node - "${smoke_dir}/fund-conversation-message.json" <<'NODE'
const fs = require("fs");
const detail = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
const runs = (detail.trace || []).filter(event => event.kind === "athena_agent_run");
const accepted = runs.find(event => event.status === "ok" && event.metadata?.run_status === "completed");
if (!accepted) throw new Error(`fund conversation missing completed Athena trace: ${JSON.stringify(runs)}`);
if (Number(accepted.metadata?.tool_call_count || "0") < 1) throw new Error(`fund trace missing tool_call_count: ${JSON.stringify(accepted)}`);
console.log(JSON.stringify({
  conversation: detail.session.id,
  athena_run_status: accepted.metadata.run_status,
  tool_call_count: accepted.metadata.tool_call_count,
  output_present: accepted.metadata.output_present,
}, null, 2));
NODE

curl -fsS -X POST "http://127.0.0.1:${FUND_PORT}/api/consents/${consent_grant_ref}/revoke" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${session_token}" \
  -d '{}' \
  > "${smoke_dir}/fund-consent-revoked.json"

node - "${smoke_dir}/fund-consent-revoked.json" <<'NODE'
const fs = require("fs");
const grant = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
if (!grant.revoked_at) throw new Error(`consent grant was not revoked: ${JSON.stringify(grant)}`);
console.log(JSON.stringify({ consent_state: "revoked" }));
NODE

revoked_callback_status="$(curl -sS -o "${smoke_dir}/fund-callback-revoked-consent.json" -w '%{http_code}' \
  -X POST "http://127.0.0.1:${FUND_PORT}/internal/tools/execute" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${REMOTE_TOOL_TOKEN}" \
  -d "{\"contract_version\":\"remote_tool_execution.v1\",\"request_id\":\"req_revoked_docker_smoke\",\"tool_call_id\":\"call_revoked_docker_smoke\",\"registration_id\":\"fund_account_overview_v1\",\"app_id\":\"athena-fund-assistant\",\"tool_name\":\"account_overview\",\"arguments\":{\"user_id\":\"demo-user\",\"consent_grant_ref\":\"${consent_grant_ref}\"},\"attempt\":1}")"
if [[ "${revoked_callback_status}" != "403" ]]; then
  echo "expected revoked callback to return 403, got ${revoked_callback_status}" >&2
  exit 1
fi

node - "${smoke_dir}/fund-callback-revoked-consent.json" "${consent_grant_ref}" <<'NODE'
const fs = require("fs");
const response = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
if (response.error?.code !== "authorization_denied" || response.metadata?.authorization_code !== "grant_revoked") {
  throw new Error(`unexpected revoked callback denial: ${JSON.stringify(response)}`);
}
if (response.metadata?.consent_grant_ref !== process.argv[3] || Number(response.metadata?.consent_revision || 0) < 1) {
  throw new Error(`revoked callback evidence missing: ${JSON.stringify(response.metadata)}`);
}
console.log(JSON.stringify({ revoked_callback_reason: response.metadata.authorization_code }));
NODE

revoked_status="$(curl -sS -o "${smoke_dir}/athena-agent-run-revoked-consent.json" -w '%{http_code}' \
  -X POST "http://127.0.0.1:${ATHENA_PORT}/api/agent/runs" \
  -H "Content-Type: application/json" \
  -d "{
    \"goal\":\"再次读取账户概览；consent_grant_ref=${consent_grant_ref}\",
    \"app_instance_id\":\"athena-fund-assistant\",
    \"app_session_id\":\"dual-docker-revoked-consent\",
    \"enabled_tools\":[\"account_overview\"],
    \"tools\":[{\"type\":\"function\",\"function\":{\"name\":\"account_overview\",\"description\":\"Read account overview\",\"parameters\":{\"type\":\"object\",\"properties\":{\"user_id\":{\"type\":\"string\"},\"consent_grant_ref\":{\"type\":\"string\"}},\"required\":[\"user_id\",\"consent_grant_ref\"]}}}],
    \"constraints\":{\"no_auto_trading\":true}
  }")"
if [[ "${revoked_status}" != "500" ]]; then
  echo "expected revoked consent run to return 500, got ${revoked_status}" >&2
  exit 1
fi

node - "${smoke_dir}/athena-agent-run-revoked-consent.json" "${REMOTE_TOOL_TOKEN}" "${WRONG_REMOTE_TOOL_TOKEN}" <<'NODE'
const fs = require("fs");
const encoded = fs.readFileSync(process.argv[2], "utf8");
if (!encoded.includes("authorization_denied")) throw new Error(`revoked consent was not denied: ${encoded}`);
if (encoded.includes(process.argv[3]) || encoded.includes(process.argv[4])) throw new Error("revoked run leaked a service token");
console.log(JSON.stringify({ revoked_consent: "denied" }));
NODE

curl -fsS -X POST "http://127.0.0.1:${FUND_PORT}/api/analysis/fund" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${session_token}" \
  -d '{
    "instrument_code":"510300",
    "profile":{
      "risk_preference":"balanced",
      "investment_horizon_months":24,
      "max_acceptable_drawdown_pct":25,
      "single_instrument_max_allocation_pct":20,
      "cash_preference_pct":8,
      "default_decision_style":"three_options"
    },
    "portfolio":{"holdings":[{
      "instrument_code":"510300",
      "instrument_name":"Sample CSI 300 ETF From CSV",
      "market":"CN",
      "currency":"CNY",
      "holding_amount":50000,
      "cost_basis":4.2,
      "allocation_pct":22,
      "user_thesis":"broad China equity beta"
    }]}
  }' > "${smoke_dir}/fund-analysis.json"

node - "${smoke_dir}/fund-analysis.json" <<'NODE'
const fs = require("fs");
const data = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
const trace = data.decision_matrix?.trace || {};
const styles = (data.decision_matrix?.options || []).map(option => option.style).sort();
if (trace.data_provider !== "csv_provider") throw new Error(`expected csv_provider trace, got ${trace.data_provider}`);
if (trace.temporary_data !== true) throw new Error(`expected temporary_data=true, got ${trace.temporary_data}`);
if (styles.join(",") !== "aggressive,balanced,conservative") throw new Error(`unexpected option styles: ${styles.join(",")}`);
console.log(JSON.stringify({
  fund_provider: trace.data_provider,
  data_boundary: trace.data_boundary,
  temporary_data: trace.temporary_data,
  option_styles: styles,
}, null, 2));
NODE

compose logs --no-color > "${smoke_dir}/compose.log"
compose exec -T athena-api sh -lc 'cat /app/config/controlplane/dual-compose.json' > "${smoke_dir}/athena-controlplane.json"

if grep -R -F "${REMOTE_TOOL_TOKEN}" "${smoke_dir}" >/dev/null \
  || grep -R -F "${WRONG_REMOTE_TOOL_TOKEN}" "${smoke_dir}" >/dev/null \
  || grep -R -F "${session_token}" "${smoke_dir}" >/dev/null; then
  echo "dual-service Docker artifacts leaked a credential" >&2
  exit 1
fi

echo "dual-service Docker smoke passed"
echo "artifacts: ${smoke_dir}"
