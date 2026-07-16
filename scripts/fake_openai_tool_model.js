const http = require("http");

let count = 0;
const port = Number(process.env.FAKE_MODEL_PORT || "18083");

function messageText(messages) {
  return (messages || []).map(message => {
    if (typeof message.content === "string") return message.content;
    return JSON.stringify(message.content || "");
  }).join("\n");
}

const server = http.createServer((req, res) => {
  let body = "";
  req.on("data", chunk => {
    body += chunk;
  });
  req.on("end", () => {
    let payload = {};
    try {
      payload = JSON.parse(body || "{}");
    } catch {
      payload = {};
    }

    count += 1;
    const hasToolResult = Array.isArray(payload.messages) && payload.messages.some(message => message.role === "tool");
    res.setHeader("Content-Type", "application/json");

    if (!hasToolResult) {
      const consentMatch = messageText(payload.messages).match(/consent_grant_ref=([A-Za-z0-9_-]+)/);
      res.end(JSON.stringify({
        id: `chatcmpl-fake-tool-${count}`,
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
                  consent_grant_ref: consentMatch ? consentMatch[1] : "",
                }),
              },
            }],
          },
        }],
        usage: { prompt_tokens: 10, completion_tokens: 10, total_tokens: 20 },
      }));
      return;
    }

    res.end(JSON.stringify({
      id: `chatcmpl-fake-final-${count}`,
      object: "chat.completion",
      created: Math.floor(Date.now() / 1000),
      model: payload.model || "fake-tool-model",
      choices: [{
        index: 0,
        finish_reason: "stop",
        message: {
          role: "assistant",
          content: "已读取账户概览。建议先复盘总收益、持仓集中度和近期操作，不执行自动交易。",
        },
      }],
      usage: { prompt_tokens: 10, completion_tokens: 12, total_tokens: 22 },
    }));
  });
});

server.listen(port, "0.0.0.0", () => {
  console.log(`fake openai-compatible model listening on ${port}`);
});
