#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ATHENA_REPO="${ATHENA_REPO:-$(cd "${ROOT_DIR}/.." && pwd)/Athena}"
ATHENA_PORT="${ATHENA_PORT:-18080}"
FUND_PORT="${FUND_PORT:-18081}"
FAKE_MODEL_PORT="${FAKE_MODEL_PORT:-18083}"
SMOKE_DIR="${SMOKE_DIR:-$(mktemp -d /tmp/athena-fund-dual-smoke.XXXXXX)}"
REMOTE_TOOL_TOKEN="${ATHENA_FUND_REMOTE_TOOL_TOKEN:-}"
WRONG_REMOTE_TOOL_TOKEN=""

ATHENA_PID=""
FUND_PID=""
FAKE_MODEL_PID=""

cleanup() {
  if [[ -n "${FUND_PID}" ]]; then kill "${FUND_PID}" >/dev/null 2>&1 || true; fi
  if [[ -n "${ATHENA_PID}" ]]; then kill "${ATHENA_PID}" >/dev/null 2>&1 || true; fi
  if [[ -n "${FAKE_MODEL_PID}" ]]; then kill "${FAKE_MODEL_PID}" >/dev/null 2>&1 || true; fi
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
  for _ in $(seq 1 90); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "timeout waiting for ${label}: ${url}" >&2
  exit 1
}

require_cmd curl
require_cmd go
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

cat > "${SMOKE_DIR}/fake-openai.js" <<'NODE'
const http = require("http");
let count = 0;
const port = Number(process.env.FAKE_MODEL_PORT || "18083");
const messageText = messages => (messages || []).map(message =>
  typeof message.content === "string" ? message.content : JSON.stringify(message.content || "")
).join("\n");
const server = http.createServer((req, res) => {
  let body = "";
  req.on("data", chunk => body += chunk);
  req.on("end", () => {
    let payload = {};
    try { payload = JSON.parse(body || "{}"); } catch {}
    count += 1;
    const hasToolResult = Array.isArray(payload.messages) && payload.messages.some(message => message.role === "tool");
    res.setHeader("Content-Type", "application/json");
    if (!hasToolResult) {
      const consentMatch = messageText(payload.messages).match(/consent_grant_ref=([A-Za-z0-9_-]+)/);
      res.end(JSON.stringify({
        id: "chatcmpl-fake-tool-" + count,
        object: "chat.completion",
        created: Math.floor(Date.now() / 1000),
        model: payload.model || "fake-tool-model",
        choices: [{
          index: 0,
          finish_reason: "tool_calls",
          message: {
            role: "assistant",
            content: null,
            tool_calls: [{
              id: "call_fake_account_overview",
              type: "function",
              function: {
                name: "account_overview",
                arguments: JSON.stringify({
                  user_id: "demo-user",
                  consent_grant_ref: consentMatch ? consentMatch[1] : ""
                })
              }
            }]
          }
        }],
        usage: { prompt_tokens: 10, completion_tokens: 10, total_tokens: 20 }
      }));
      return;
    }
    res.end(JSON.stringify({
      id: "chatcmpl-fake-final-" + count,
      object: "chat.completion",
      created: Math.floor(Date.now() / 1000),
      model: payload.model || "fake-tool-model",
      choices: [{
        index: 0,
        finish_reason: "stop",
        message: {
          role: "assistant",
          content: "已读取账户概览。建议先复盘总收益、持仓集中度和近期操作，不执行自动交易。"
        }
      }],
      usage: { prompt_tokens: 10, completion_tokens: 12, total_tokens: 22 }
    }));
  });
});
server.listen(port, "127.0.0.1", () => console.log(`fake openai-compatible model listening on ${port}`));
NODE

FAKE_MODEL_PORT="${FAKE_MODEL_PORT}" node "${SMOKE_DIR}/fake-openai.js" > "${SMOKE_DIR}/fake-model.log" 2>&1 &
FAKE_MODEL_PID="$!"

(
  cd "${ATHENA_REPO}"
  HTTP_PORT="${ATHENA_PORT}" \
    SESSION_STORE_DRIVER=memory \
    MODEL_STORE_DRIVER=memory \
    CONTROL_PLANE_STORE_PATH="${SMOKE_DIR}/athena-controlplane.json" \
    CONTROL_PLANE_ALLOWED_ORIGINS="http://127.0.0.1:5173" \
    REMOTE_TOOL_ALLOWED_ORIGINS="http://127.0.0.1:${FUND_PORT}" \
    REMOTE_TOOL_MAX_RESPONSE_BYTES=1048576 \
    ATHENA_FUND_REMOTE_TOOL_TOKEN="${REMOTE_TOOL_TOKEN}" \
    ATHENA_FUND_REMOTE_TOOL_WRONG_TOKEN="${WRONG_REMOTE_TOOL_TOKEN}" \
    go run . api-server
) > "${SMOKE_DIR}/athena.log" 2>&1 &
ATHENA_PID="$!"

(
  cd "${ROOT_DIR}"
  ATHENA_FUND_API_ADDR=":${FUND_PORT}" \
    ATHENA_BASE_URL="http://127.0.0.1:${ATHENA_PORT}" \
    ATHENA_FUND_UPLOAD_DIR="${SMOKE_DIR}/uploads" \
    ATHENA_FUND_LOCAL_AUTH_SUBJECT="demo-user" \
    ATHENA_FUND_REMOTE_TOOL_TOKEN="${REMOTE_TOOL_TOKEN}" \
    go run ./cmd/api
) > "${SMOKE_DIR}/fund.log" 2>&1 &
FUND_PID="$!"

wait_for_http "http://127.0.0.1:${FAKE_MODEL_PORT}/v1/chat/completions" "fake model"
wait_for_http "http://127.0.0.1:${ATHENA_PORT}/healthz" "Athena"
wait_for_http "http://127.0.0.1:${FUND_PORT}/healthz" "fund assistant"

session_response="$(curl -fsS -X POST "http://127.0.0.1:${FUND_PORT}/api/auth/sessions" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"demo-user","ttl_seconds":3600}')"
session_token="$(printf '%s' "${session_response}" | node -pe 'JSON.parse(require("fs").readFileSync(0, "utf8")).token')"
printf '%s' "${session_response}" | node -e 'const value=JSON.parse(require("fs").readFileSync(0,"utf8"));process.stdout.write(JSON.stringify(value.session))' > "${SMOKE_DIR}/fund-session.json"
unset session_response

curl -fsS -X POST "http://127.0.0.1:${FUND_PORT}/api/consents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${session_token}" \
  -d '{"audience":"athena-runtime","scopes":["fund.account.summary.read","fund.holding.snapshot.read"],"ttl_seconds":3600}' \
  > "${SMOKE_DIR}/fund-consent.json"
consent_grant_ref="$(node -pe 'JSON.parse(require("fs").readFileSync(0, "utf8")).ref' < "${SMOKE_DIR}/fund-consent.json")"

curl -fsS -X POST "http://127.0.0.1:${ATHENA_PORT}/api/models/providers" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\":\"fake-openai-smoke\",
    \"protocol\":\"openai_compatible\",
    \"base_url\":\"http://127.0.0.1:${FAKE_MODEL_PORT}/v1\",
    \"api_key\":\"sk-fake\",
    \"request_timeout_seconds\":10,
    \"models\":[{\"model_id\":\"fake-tool-model\",\"display_name\":\"Fake Tool Model\",\"is_default\":true,\"is_fallback\":true}]
  }" > "${SMOKE_DIR}/athena-model-provider.json"

curl -fsS "http://127.0.0.1:${FUND_PORT}/internal/tools/catalog?base_url=http://127.0.0.1:${FUND_PORT}" > "${SMOKE_DIR}/fund-tool-catalog.json"
node - "${SMOKE_DIR}/fund-tool-catalog.json" "${SMOKE_DIR}" <<'NODE'
const fs = require("fs");
const catalog = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
for (const item of catalog.items) {
  fs.writeFileSync(`${process.argv[3]}/fund-tool-${item.name}.json`, JSON.stringify(item));
  if (item.name === "account_overview") {
    fs.writeFileSync(`${process.argv[3]}/fund-tool-${item.name}-wrong-auth.json`, JSON.stringify({
      ...item,
      auth: {type: "bearer", secret_ref: "env://ATHENA_FUND_REMOTE_TOOL_WRONG_TOKEN"}
    }));
  }
}
console.log(JSON.stringify({catalog_items: catalog.items.map(item => item.name)}));
NODE

curl -fsS -X PUT "http://127.0.0.1:${ATHENA_PORT}/api/control-plane/remote-tools/account_overview" \
  -H "Content-Type: application/json" \
  --data-binary "@${SMOKE_DIR}/fund-tool-account_overview-wrong-auth.json" \
  > "${SMOKE_DIR}/athena-register-account_overview-wrong-auth.json"

wrong_auth_status="$(curl -sS -o "${SMOKE_DIR}/athena-agent-run-wrong-service-auth.json" -w '%{http_code}' \
  -X POST "http://127.0.0.1:${ATHENA_PORT}/api/agent/runs" \
  -H "Content-Type: application/json" \
  -d "{
    \"goal\":\"读取账户概览；consent_grant_ref=${consent_grant_ref}\",
    \"app_instance_id\":\"athena-fund-assistant\",
    \"app_session_id\":\"dual-smoke-wrong-service-auth\",
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
  }")"
if [[ "${wrong_auth_status}" != "500" ]]; then
  echo "expected wrong service auth run to return 500, got ${wrong_auth_status}" >&2
  exit 1
fi

node - "${SMOKE_DIR}/athena-agent-run-wrong-service-auth.json" "${REMOTE_TOOL_TOKEN}" "${WRONG_REMOTE_TOOL_TOKEN}" <<'NODE'
const fs = require("fs");
const encoded = fs.readFileSync(process.argv[2], "utf8");
if (!encoded.includes("service_auth_denied")) throw new Error(`wrong service token was not denied: ${encoded}`);
if (encoded.includes(process.argv[3]) || encoded.includes(process.argv[4])) throw new Error("agent run leaked a service token");
console.log(JSON.stringify({wrong_service_auth: "denied"}));
NODE

for name in account_overview fund_market_snapshot; do
  curl -fsS -X PUT "http://127.0.0.1:${ATHENA_PORT}/api/control-plane/remote-tools/${name}" \
    -H "Content-Type: application/json" \
    --data-binary "@${SMOKE_DIR}/fund-tool-${name}.json" \
    > "${SMOKE_DIR}/athena-register-${name}.json"
done

curl -fsS -X POST "http://127.0.0.1:${ATHENA_PORT}/api/agent/runs" \
  -H "Content-Type: application/json" \
  -d "{
    \"goal\":\"请读取账户概览并给出复盘重点；consent_grant_ref=${consent_grant_ref}\",
    \"app_instance_id\":\"athena-fund-assistant\",
    \"app_session_id\":\"dual-smoke-direct\",
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
  }" > "${SMOKE_DIR}/athena-agent-run.json"

node - "${SMOKE_DIR}/athena-agent-run.json" "${consent_grant_ref}" <<'NODE'
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
  output: run.output
}, null, 2));
NODE

conversation_id="$(
  curl -fsS -X POST "http://127.0.0.1:${FUND_PORT}/api/conversations" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${session_token}" \
    -d '{"user_id":"demo-user","skill_id":"portfolio_review","title":"dual service smoke"}' \
    | node -pe 'JSON.parse(require("fs").readFileSync(0, "utf8")).session.id'
)"

curl -fsS -X POST "http://127.0.0.1:${FUND_PORT}/api/conversations/${conversation_id}/messages" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${session_token}" \
  -d "{\"role\":\"user\",\"content\":\"请读取账户概览并给我复盘重点；consent_grant_ref=${consent_grant_ref}\",\"skill_id\":\"portfolio_review\",\"consent_grant_ref\":\"${consent_grant_ref}\"}" \
  > "${SMOKE_DIR}/fund-conversation-message.json"

node - "${SMOKE_DIR}/fund-conversation-message.json" <<'NODE'
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
  output_present: accepted.metadata.output_present
}, null, 2));
NODE

curl -fsS -X POST "http://127.0.0.1:${FUND_PORT}/api/consents/${consent_grant_ref}/revoke" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${session_token}" \
  -d '{}' \
  > "${SMOKE_DIR}/fund-consent-revoked.json"

node - "${SMOKE_DIR}/fund-consent-revoked.json" <<'NODE'
const fs = require("fs");
const grant = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
if (!grant.revoked_at) throw new Error(`consent grant was not revoked: ${JSON.stringify(grant)}`);
console.log(JSON.stringify({consent_state: "revoked"}));
NODE

revoked_callback_status="$(curl -sS -o "${SMOKE_DIR}/fund-callback-revoked-consent.json" -w '%{http_code}' \
  -X POST "http://127.0.0.1:${FUND_PORT}/internal/tools/execute" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${REMOTE_TOOL_TOKEN}" \
  -d "{\"contract_version\":\"remote_tool_execution.v1\",\"request_id\":\"req_revoked_smoke\",\"tool_call_id\":\"call_revoked_smoke\",\"registration_id\":\"fund_account_overview_v1\",\"app_id\":\"athena-fund-assistant\",\"tool_name\":\"account_overview\",\"arguments\":{\"user_id\":\"demo-user\",\"consent_grant_ref\":\"${consent_grant_ref}\"},\"attempt\":1}")"
if [[ "${revoked_callback_status}" != "403" ]]; then
  echo "expected revoked callback to return 403, got ${revoked_callback_status}" >&2
  exit 1
fi

node - "${SMOKE_DIR}/fund-callback-revoked-consent.json" "${consent_grant_ref}" <<'NODE'
const fs = require("fs");
const response = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
if (response.error?.code !== "authorization_denied" || response.metadata?.authorization_code !== "grant_revoked") {
  throw new Error(`unexpected revoked callback denial: ${JSON.stringify(response)}`);
}
if (response.metadata?.consent_grant_ref !== process.argv[3] || Number(response.metadata?.consent_revision || 0) < 1) {
  throw new Error(`revoked callback evidence missing: ${JSON.stringify(response.metadata)}`);
}
console.log(JSON.stringify({revoked_callback_reason: response.metadata.authorization_code}));
NODE

revoked_status="$(curl -sS -o "${SMOKE_DIR}/athena-agent-run-revoked-consent.json" -w '%{http_code}' \
  -X POST "http://127.0.0.1:${ATHENA_PORT}/api/agent/runs" \
  -H "Content-Type: application/json" \
  -d "{
    \"goal\":\"再次读取账户概览；consent_grant_ref=${consent_grant_ref}\",
    \"app_instance_id\":\"athena-fund-assistant\",
    \"app_session_id\":\"dual-smoke-revoked-consent\",
    \"enabled_tools\":[\"account_overview\"],
    \"tools\":[{\"type\":\"function\",\"function\":{\"name\":\"account_overview\",\"description\":\"Read account overview\",\"parameters\":{\"type\":\"object\",\"properties\":{\"user_id\":{\"type\":\"string\"},\"consent_grant_ref\":{\"type\":\"string\"}},\"required\":[\"user_id\",\"consent_grant_ref\"]}}}],
    \"constraints\":{\"no_auto_trading\":true}
  }")"
if [[ "${revoked_status}" != "500" ]]; then
  echo "expected revoked consent run to return 500, got ${revoked_status}" >&2
  exit 1
fi

node - "${SMOKE_DIR}/athena-agent-run-revoked-consent.json" "${REMOTE_TOOL_TOKEN}" "${WRONG_REMOTE_TOOL_TOKEN}" <<'NODE'
const fs = require("fs");
const encoded = fs.readFileSync(process.argv[2], "utf8");
if (!encoded.includes("authorization_denied")) {
  throw new Error(`revoked consent was not denied: ${encoded}`);
}
if (encoded.includes(process.argv[3]) || encoded.includes(process.argv[4])) throw new Error("revoked run leaked a service token");
console.log(JSON.stringify({revoked_consent: "denied"}));
NODE

if grep -R -F "${REMOTE_TOOL_TOKEN}" "${SMOKE_DIR}" >/dev/null \
  || grep -R -F "${WRONG_REMOTE_TOOL_TOKEN}" "${SMOKE_DIR}" >/dev/null \
  || grep -R -F "${session_token}" "${SMOKE_DIR}" >/dev/null; then
  echo "dual-service artifacts leaked a credential" >&2
  exit 1
fi

echo "dual-service smoke passed"
echo "logs: ${SMOKE_DIR}"
