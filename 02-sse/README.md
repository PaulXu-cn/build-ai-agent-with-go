# 02_sse

用 `net/http` 手写流式调用。

非流式是「等服务端算完，一次性拿回整段话」；流式是「服务端算一段、发一段」。
底层协议是 SSE（Server-Sent Events）：一个长连接 + 纯文本，一行一条数据。

## 和非流式的差异

| | 非流式 | 流式 |
|---|---|---|
| 请求 | `stream: false` | `stream: true` |
| 响应 | 一整块 JSON，`io.ReadAll` 读完再解析 | 一串小 JSON，逐行读、逐段打印 |
| 客户端超时 | `http.Client.Timeout` | 不能设。它管的是「发请求 + 读完 body」的总时长，长回答会被拦腰砍断 |

## SSE 格式

响应体是纯文本，`Content-Type: text/event-stream`：

```
event: content_block_delta
data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"你"}}

event: message_stop
data: {"type":"message_stop"}
```

- 一行一条，格式是 `字段: 值`
- 空行 = 一个事件结束
- 以 `:` 开头的是心跳注释行，跳过
- 事件名在 `event:` 行和数据里的 `type` 字段各有一份，所以只读 `data:` 行就够

## 三种方言的流式事件形状

| 方言 | 数据行 | 增量文本位置 | 结束标志 |
|---|---|---|---|
| Chat Completions | 一个小 JSON | `choices[0].delta.content` | `data: [DONE]`（不是 JSON） |
| Responses | 一个事件 JSON | `delta`，当 `type=response.output_text.delta` | `type=response.completed` |
| Messages | 一个事件 JSON | `delta.text`，当 `type=content_block_delta` | `type=message_stop` |

Chat Completions 用非 JSON 的哨兵值结束，另两家用事件类型结束，这是最容易踩的一处。

## 文件

| 文件 | 内容 |
|---|---|
| `main.go` | 读 `PROVIDER` 查表，调用对应方言 |
| `http.go` | 共用传输层：发请求 + 逐行读 SSE |
| `openai_compat.go` | Chat Completions 流式 |
| `openai_responses.go` | Responses 流式 |
| `anthropic.go` | Messages 流式 |

## 运行

```bash
PROVIDER=deepseek  API_KEY=sk-...     go run .
PROVIDER=qwen      API_KEY=sk-...     go run .
PROVIDER=openai    API_KEY=sk-...     go run .
PROVIDER=responses API_KEY=sk-...     go run .
PROVIDER=claude    API_KEY=sk-ant-... go run .

# 覆盖默认端点 / 模型 / 提问
PROVIDER=deepseek API_KEY=sk-... PROMPT="用三句话解释 AI Agent" go run .
```

也可在根目录 `make run-02-sse`。

## 输出

四段：请求体 JSON、响应的 `Content-Type`（流式应为 `text/event-stream`）、逐段打印的回答、token 用量。

## 环境变量

| 变量 | 说明 | 默认 |
|---|---|---|
| `PROVIDER` | `deepseek` / `qwen` / `openai` / `responses` / `claude` | 必填 |
| `API_KEY` | 密钥 | 必填 |
| `BASE_URL` | 端点根地址，`/v1/...` 由代码拼接 | 见下表 |
| `MODEL` | 模型 | 见下表 |
| `PROMPT` | 提问内容 | 用一句话说明什么是 AI Agent |

| PROVIDER | BASE_URL | MODEL |
|---|---|---|
| deepseek | `https://api.deepseek.com` | `deepseek-chat` |
| qwen | `https://dashscope.aliyuncs.com/compatible-mode` | `qwen-plus` |
| openai | `https://api.openai.com` | `gpt-4o-mini` |
| responses | `https://api.openai.com` | `gpt-4o-mini` |
| claude | `https://api.anthropic.com` | `claude-opus-4-8` |

# 参考

- SSE 规范：https://html.spec.whatwg.org/multipage/server-sent-events.html
- Anthropic streaming：https://platform.claude.com/docs/en/api/messages-streaming
- OpenAI Responses streaming：https://platform.openai.com/docs/api-reference/responses-streaming
