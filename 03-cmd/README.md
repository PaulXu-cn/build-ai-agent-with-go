# 03-cmd

一个命令行对话程序：终端里输入问题，回答流式打出来。

## 结构

前两章所有代码都在 `package main` 里，一个文件夹塞完。本章开始拆包：

```
03-cmd/
├── main.go     组装：建 Client，交给 App 跑（几行代码）
├── llm/        和模型说话：HTTP + 协议。不知道终端长什么样
└── cmd/        和用户说话：读输入 + 打印。不知道 HTTP 长什么样
```

两边靠一个接口对接。接口定义在**使用方**（cmd）而不是实现方，这是 Go 的惯例：

```go
// cmd/cmd.go
type Streamer interface {
	Stream(ctx context.Context, messages []llm.Message, onDelta func(string)) error
}
```

`*llm.Client` 天然满足它，所以两个包不用互相 import，`main.go` 里直接传进去就行。

接口上有个细节：`Stream` 除了回调，还会**返回** token 用量。

```go
usage, err := c.streamer.Stream(ctx, messages, func(delta string) { fmt.Print(delta) })
```

为什么不干脆让 `llm` 包自己 `fmt.Print`：那就等于让模型包往终端写东西，
「llm 不知道终端长什么样」这句话就破了。要显示什么、怎么显示，是 `cmd` 包的事。

用量本身三种方言算法不同，`llm` 包负责把它们统一成 `Usage{Input, Cached, Output}`：

| 方言 | 输入总量 | 命中缓存 |
|---|---|---|
| Chat Completions | `prompt_tokens` | `cached_tokens`，DeepSeek 则是 `prompt_cache_hit_tokens` |
| Responses | `input_tokens` | `input_tokens_details.cached_tokens` |
| Messages | `input_tokens + cache_creation + cache_read` | `cache_read_input_tokens` |

流式下还有个坑：Chat Completions **默认不返回用量**，要在请求里加
`stream_options: {"include_usage": true}`；开启后 `[DONE]` 之前会多一片
`choices` 为空数组的分片，用量在里面——**先判断 `choices` 就会把它丢掉**。

## 交互

| 操作 | 行为 |
|---|---|
| 输入一行 + 回车 | 发送，回答流式打印 |
| Ctrl-C（回答进行中） | 中断这一轮，回到提示符 |
| Ctrl-C（空闲） | 退出 |
| Ctrl-D / `exit` / `quit` | 退出 |

## Ctrl-C 中断是怎么做的

默认 Ctrl-C 直接杀进程。这里用 `signal.Notify` 接管，每次提问再建一个可取消的 context：

```go
ctx, cancel := context.WithCancel(context.Background())
c.setCancel(cancel)   // 登记为「当前请求」，Ctrl-C 才知道该取消谁
defer c.setCancel(nil)
err := c.streamer.Stream(ctx, messages, onDelta)
```

Ctrl-C 时查登记表：有请求在跑就 `cancel()`，没有就退出。

HTTP 层有个配套要求：必须用 `http.NewRequestWithContext`。用普通的 `http.NewRequest` 的话，context 取消了也断不掉连接，流会继续收。

## 运行

```bash
PROVIDER=deepseek  API_KEY=sk-...     go run .
PROVIDER=qwen      API_KEY=sk-...     go run .
PROVIDER=openai    API_KEY=sk-...     go run .
PROVIDER=responses API_KEY=sk-...     go run .
PROVIDER=claude    API_KEY=sk-ant-... go run .
```

也可在根目录 `make run-03-cmd`。

## 环境变量

| 变量 | 说明 | 默认 |
|---|---|---|
| `PROVIDER` | `deepseek` / `qwen` / `openai` / `responses` / `claude` | 必填 |
| `API_KEY` | 密钥 | 必填 |
| `BASE_URL` | 端点根地址，`/v1/...` 由代码拼接 | 见下表 |
| `MODEL` | 模型 | 见下表 |
| `DEBUG` | 非空则打印原始请求体 | 空 |

| PROVIDER | BASE_URL | MODEL |
|---|---|---|
| deepseek | `https://api.deepseek.com` | `deepseek-chat` |
| qwen | `https://dashscope.aliyuncs.com/compatible-mode` | `qwen-plus` |
| openai | `https://api.openai.com` | `gpt-4o-mini` |
| responses | `https://api.openai.com` | `gpt-4o-mini` |
| claude | `https://api.anthropic.com` | `claude-opus-4-8` |

# 参考

- SSE 规范：https://html.spec.whatwg.org/multipage/server-sent-events.html
- Go 的 context 取消：https://pkg.go.dev/context
