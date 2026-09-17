# 01_http

用 `net/http` 手写非流式 LLM 调用。市面上的 LLM API 归为三种方言，本目录三种各实现一份。

## 三种方言

| 方言 | 厂商 | 端点 | 鉴权 | 请求体 | 响应取文本 |
|---|---|---|---|---|---|
| Chat Completions | OpenAI 旧接口、DeepSeek、Qwen、Kimi、GLM… | `/v1/chat/completions` | `Authorization: Bearer` | `messages[]`，content 为字符串 | `choices[0].message.content` |
| Responses | OpenAI 新接口 | `/v1/responses` | `Authorization: Bearer` | `input`，字符串或消息数组 | `output[]` 中 `type=message` 项的 `content[].text` |
| Messages | Anthropic | `/v1/messages` | `x-api-key` + `anthropic-version` | `messages[]`，content 为块数组，`system` 放顶层 | `content[].text` |

DeepSeek、Qwen 与 OpenAI 旧接口同属 Chat Completions 方言，共用 `chatCompletion`，只换 baseURL / model / key。

三者差异集中在端点、鉴权头、JSON 形状；传输部分（POST、超时、错误处理）一致，抽在 `http.go`。

## 文件

| 文件 | 内容 |
|---|---|
| `main.go` | 读 `PROVIDER` 查表，调用对应方言 |
| `http.go` | 共用传输层 |
| `openai_compat.go` | Chat Completions 方言 |
| `openai_responses.go` | Responses 方言 |
| `anthropic.go` | Messages 方言 |

## 运行

```bash
PROVIDER=deepseek  API_KEY=sk-...     go run .
PROVIDER=qwen      API_KEY=sk-...     go run .
PROVIDER=openai    API_KEY=sk-...     go run .
PROVIDER=responses API_KEY=sk-...     go run .
PROVIDER=claude    API_KEY=sk-ant-... go run .

# 覆盖默认端点 / 模型 / 提问
PROVIDER=deepseek API_KEY=sk-... MODEL=deepseek-reasoner PROMPT="写一首诗" go run .
```

也可在根目录 `make run-01-http`。

## 输出

打印四段：请求体 JSON、响应体 JSON、token 用量、模型回复文本。

## 环境变量

| 变量 | 说明 | 默认 |
|---|---|---|
| `PROVIDER` | `deepseek` / `qwen` / `openai` / `responses` / `claude` | 必填 |
| `API_KEY` | 密钥 | 必填 |
| `BASE_URL` | 端点根地址，`/v1/...` 由代码拼接 | 见下表 |
| `MODEL` | 模型 | 见下表 |
| `PROMPT` | 提问内容 | 一句话说明什么是 AI Agent |

| PROVIDER | BASE_URL | MODEL |
|---|---|---|
| deepseek | `https://api.deepseek.com` | `deepseek-chat` |
| qwen | `https://dashscope.aliyuncs.com/compatible-mode` | `qwen-plus` |
| openai | `https://api.openai.com` | `gpt-4o-mini` |
| responses | `https://api.openai.com` | `gpt-4o-mini` |
| claude | `https://api.anthropic.com` | `claude-opus-4-8` |


# 参考

- https://api-docs.deepseek.com/zh-cn/api/create-chat-completion
- https://www.alibabacloud.com/help/zh/model-studio/qwen-api-via-openai-chat-completions?spm=a2c63.p38356.help-menu-2400256.d_2_1_0.7877213dmKA2pY#curl-h4
- https://developers.openai.com/api/docs/guides/migrate-to-responses?update-generation-endpoints=responses&update-multiturn=responses&structured-outputs=responses&tool-use=responses#about-the-responses-api
- https://platform.claude.com/docs/en/api/http/beta/messages/create