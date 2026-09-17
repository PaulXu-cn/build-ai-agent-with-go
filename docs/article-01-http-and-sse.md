# 让我带你从 0 到 1 用 golang 手写一个 claude code

最近不那么忙，准备沉淀一下。

之前大家炫技的方式是：手撕一个爬虫，手撕一个 IM，手撕一个数据库……

我准备手撕一个 AI Agent CLI。


<!-- 配图 1 ｜ 头图 -->
> **🖼 配图 1｜头图**
>
> 出图提示词：终端窗口特写，深色背景，一个简洁的命令行界面里正在逐字输出一行行文本，文字呈现"一行行浮现"的动态感（可用速度线或渐隐表现）。画面右侧漂浮着几个极简的几何模块，暗示模块化、可拼装的结构。整体冷静、克制、工程感强。不要出现任何真实可读的文字，文字用模糊的浅色短横线示意即可。横构图 16:9。

---

## 特点

**每一期要阅读的代码，都放在一个独立文件夹里，而且代码量不大。**

为什么非要这么干？

因为一般人看到一大段代码，第一反应是丢给 Claude "蒸馏一下"，这就没达到效果，如果一屏能看完，你就会真的读一遍。人一次能装进脑子里的东西是有限的，**代码量本身就是一种门槛**，不是内容多才叫有价值。在不撑爆你的"上下文"的情况下完成教学。

**章节之间不互相 import，会有重复代码，便于你比较。**

`01-http` 和 `02-sse` 里都有一个 `openai_compat.go`，内容高度相似。是的，这个是让你能在一个IDE进行代码对比，而不是通过 git 不同的版本进行对比。

> 后面的代码会复制之前的代码，再次基础上进行开发，便于你通过diff工具对比，看到代码是如何演进的。

**第三，只用 Go 标准库，零第三方依赖。**

四个 `go.mod` 里 `require` 加起来是 0 行。你 clone 下来不用联网下依赖，`go run .` 直接跑。

避免引入第三方库，引入了你就需要去研究它，阅读它的代码。

---

## 教学大纲

一篇一篇往下走，每一期都是一个独立能跑的文件夹：

| 期 | 文件夹 | 讲什么 |
|---|---|---|
| 一 | `01-http` | 三种 LLM API 方言的非流式调用 |
| 二 | `02-sse` | SSE 流式输出 |
| 三 | `03-cmd` | 命令行交互程序：把调用包进循环 |
| 四 | `04-history` | 消息历史：API 无状态，上下文每轮全量重发 |
| 五 | `05-tool-call` | 工具调用：让模型能"要"东西（只走一次往返） |
| 六 | `06-agent-loop` | **Agent 主循环**：把上一次往返包进 while（核心） |
| 七 | `07-tools` | shell 与其余工具：超时、中断、并行执行 |
| 八 | `08-security` | 路径穿越、软链接逃逸、提示词注入 |
| 九 | `09-permission` | 危险操作先问一句：确认、白名单、记住选择 |
| 十 | `10-tui` | 全屏终端界面：原始模式、按键解析、中文宽度 |
| 十一 | `11-mcp` | 接上外部世界：把远程工具接进循环 |
| 十二 | `12-skill` | 第二种扩展方式：模型"知道怎么做" |
| 十三 | `13-context` | 上下文是稀缺资源：token 预算与压缩 |
| 十四 | `14-session` | 会话持久化与 `--resume` |
| 十五 | `15-subagent` | 子 Agent：把任务隔离出去 |
| 十六 | `16-cli` | 非交互模式、输出格式、配置分层 |

几个里程碑：

- **03 之后** —— 你能和大模型对话了
- **06 之后** —— 它自己会干活了（这门课的核心就这一期）
- **10 之后** —— 它长得像个真工具了

配套的源码在： https://github.com/PaulXu-cn/build-ai-agent-with-go

非常建议 clone 到本地，配合文章一起学习

## 一期：完成你的一次 LLM HTTP 请求

先讲 LLM API 请求。

### 三种方言

主流的 LLM API 基本能归成三类：

| 方言 | 谁在用 | 端点 | 鉴权头 | 请求体形状 |
|---|---|---|---|---|
| Chat Completions | OpenAI 老接口、DeepSeek、Qwen、Kimi、GLM、Ollama | `/v1/chat/completions` | `Authorization: Bearer` | `messages[]`，content 是**字符串** |
| Responses | OpenAI 新接口 | `/v1/responses` | `Authorization: Bearer` | `input`（不叫 messages），字符串或消息数组 |
| Messages | Anthropic（Claude） | `/v1/messages` | `x-api-key` + `anthropic-version` | `messages[]`，content 是**块数组**，`system` 放顶层，`max_tokens` 必填 |

**Gemini，我就不在这里展开了**

你挑其中一个就能了解这个流程是怎么样的，差异只在端点、鉴权头、JSON 形状这三处。

<!-- 配图 2 ｜ 三种方言归三类 -->
> **🖼 配图 2｜三种方言，一个传输层**
>
> 出图提示词：极简技术示意图，白色背景。底部是一条水平的粗条带，标注位置留白（代表"HTTP 传输层"，用统一的纹理表示所有请求都从这条路走）。条带上方立着三根柱子/三个方块，分别用三种不同的简单几何图案区分（例如条纹、点阵、斜线），并各自向上引出一条细箭头；三个方块的形状一样大，只是图案不同，表达"骨架相同、方言不同"。几何风格、细线条、单一强调色、留白多。横构图 16:9。画面中不要出现任何文字。

### 代码结构

```
01-http/
├── main.go             读 PROVIDER 查表，调对应方言
├── http.go             共用传输层
├── openai_compat.go    Chat Completions 方言
├── openai_responses.go Responses 方言
└── anthropic.go        Messages 方言
```


### 贴一下 openai compat 的代码

精简版，去掉了注释：

```go
type ChatCompletionReq struct {
	Model    string              `json:"model"`
	Messages []ChatCompletionMsg `json:"messages"`
	Stream   bool                `json:"stream"`
}

type ChatCompletionMsg struct {
	Role    string `json:"role"`    // system / user / assistant
	Content string `json:"content"`
}

type ChatCompletionResp struct {
	Choices []struct {
		Message      ChatCompletionMsg `json:"message"`
		FinishReason string            `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func chatCompletion(apiKey, baseURL, model, prompt string) (string, error) {
	reqBody := ChatCompletionReq{
		Model: model,
		Messages: []ChatCompletionMsg{
			{Role: "system", Content: "你是一个简洁的中文助手，回答尽量简短。"},
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	// 鉴权：Authorization: Bearer <key>
	body, err := postJSON(baseURL+"/v1/chat/completions", map[string]string{
		"Authorization": "Bearer " + apiKey,
	}, reqBody)
	if err != nil {
		return "", err
	}

	var resp ChatCompletionResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("响应里没有 choices")
	}
	return resp.Choices[0].Message.Content, nil
}
```

就这么多。构造请求体 → 发出去 → 从 `choices[0].message.content` 里把文本解析出来。

`postJSON` 是三个方言共用的，也就十来行：

```go
req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBytes))
if err != nil {
	return nil, fmt.Errorf("创建请求失败: %w", err)
}
for k, v := range headers {
	req.Header.Set(k, v)
}
req.Header.Set("Content-Type", "application/json")

client := &http.Client{Timeout: 60 * time.Second}
resp, err := client.Do(req)
```

如果你自己写，要注意 `stream: false`， 非流式下模型要把整段话生成完毕才一起返回，超时设太短会被拦腰砍断。

### 跑一下

```bash
$ PROVIDER=deepseek API_KEY=sk-... go run .
===== 发送的请求体 =====
{
  "model": "deepseek-chat",
  "messages": [
    {
      "role": "system",
      "content": "你是一个简洁的中文助手，回答尽量简短。"
    },
    {
      "role": "user",
      "content": "用一句话说明什么是 AI Agent。"
    }
  ],
  "stream": false
}
========================
===== 收到的响应体 =====
{
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "AI Agent 是一个能自主调用工具、分多步完成任务的程序。…"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": { "prompt_tokens": 18, "completion_tokens": 42, "total_tokens": 60 }
}
========================
===== 模型回复 =====
AI Agent 是一个能自主调用工具、分多步完成任务的程序。…
```

看到这个 JSON，你就知道"调大模型"到底是怎么回事了。

一期工程完毕。

---

## 二期：SSE 流式

上期只是一个基础的 HTTP 请求，这期将常用的流式请求——SSE

### 先简单讲讲 SSE 协议

底层协议叫 SSE（Server-Sent Events），不好理解的小伙伴，你就把它当成：**一个不关闭的 HTTP 连接，服务端再往里一行一行往里写纯文本。** 

> 一般用到请求一下就完了，但它就喝文件下载请求一样，一点点吐内容

原始响应体长这样：

```
data: {"choices":[{"delta":{"content":"AI Age"}}]}

data: {"choices":[{"delta":{"content":"nt 是一个"}}]}

data: {"choices":[{"delta":{"content":"能自主调用工"}}]}

data: [DONE]
```

规则就三条：

1. 一行一条，格式是 `字段: 值`
2. **空行 = 一个事件结束**
3. 以 `:` 开头的是心跳注释行，跳过

事件名在 `event:` 行和 `data:` 里的 `type` 字段各有一份，所以**只读 `data:` 行就够**。

客户端要做的就一件事：**别等连接关闭，读一行处理一行。**

### 和非流式的差异

| | 非流式 | 流式 |
|---|---|---|
| 请求 | `stream: false` | `stream: true` |
| 响应 | 一整块 JSON，`io.ReadAll` 读完再解析 | 一串小 JSON，逐行读、逐段打印 |
| 文本位置 | `choices[0].message.content` | `choices[0].delta.content` |
| 客户端超时 | `http.Client.Timeout` | **不能设** |

`delta` 就是"增量"的意思——每个分片只带一小段新文字。

### 贴一下关键代码

openai compat 的流式，核心就这些：

```go
func streamCompletion(apiKey, baseURL, model, prompt string) error {
	reqBody := ChatCompletionReq{
		Model:    model,
		Messages: []ChatCompletionMsg{...},
		Stream:   true, // ← 就改这一处
	}

	return streamSSE(baseURL+"/v1/chat/completions", map[string]string{
		"Authorization": "Bearer " + apiKey,
	}, reqBody, func(data string) error {
		// 结束标志不是 JSON，先拦下来
		if data == "[DONE]" {
			return errStop
		}
		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("解析分片失败: %w", err)
		}
		if len(chunk.Choices) == 0 {
			return nil
		}
		// 收到一段就打一段
		fmt.Print(chunk.Choices[0].Delta.Content)
		return nil
	})
}
```

剩下的事全在 `streamSSE` 里，就是把响应体当纯文本逐行读：

```go
scanner := bufio.NewScanner(resp.Body) // 这里很重要，是 scanner，而不是 ReadAll！！
for scanner.Scan() {
	line := scanner.Text()

	// 空行是事件分隔符，以 : 开头的是心跳注释行，都跳过
	if line == "" || strings.HasPrefix(line, ":") {
		continue
	}
	// 事件名在 data 的 JSON 里还有一份（type 字段），所以只取 data: 就够
	if !strings.HasPrefix(line, "data:") {
		continue
	}
	data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

	if err := onData(data); err != nil {
		if errors.Is(err, errStop) {
			return nil
		}
		return err
	}
}
return scanner.Err()
```

### 注意坑：流式不能设 `http.Client.Timeout`

```go
// 如果设置了超时限制，那么回答一长就会被拦腰砍断。流式什么时候结束，由服务端的结束标志决定。
client := &http.Client{}
```

**流式的结束由服务端告诉你，不由客户端计时。**

### 三种方言的流式结束标志，也不一样

| 方言 | 增量文本在哪 | 怎么算结束 |
|---|---|---|
| Chat Completions | `choices[0].delta.content` | 收到一条 `data: [DONE]` |
| Responses | `type=response.output_text.delta` 时的 `delta` | `type=response.completed` |
| Messages | `type=content_block_delta` 时的 `delta.text` | `type=message_stop` |

注意第一行——**`[DONE]` 根本不是 JSON**，它是个哨兵字符串。你要都按 JSON 解析，会报错。后面更新后，就规范对了。

<!-- 配图 3 ｜ 非流式 vs 流式 -->
> **🖼 配图 3｜非流式 vs 流式**
>
> 出图提示词：极简技术示意图，白色背景，上下两行对比。上行（非流式）：左边一个小方块代表客户端，右边一个方块代表服务端，中间一条实线相连，线的中段有一个沙漏图标表示"要一直等"；线的右端连着一条又粗又完整的色块，表示一次性拿到全部内容。下行（流式）：同样的客户端和服务端，同样一条线，但线上有一串小箭头依次从左向右流动；线的右端连着许多个互相分离的小方块按时间顺序排开，表示内容被一段段送达。细线条、几何风格、单一强调色。横构图 3:2。画面中不要出现任何文字。

### 跑一下

```bash
$ PROVIDER=deepseek API_KEY=sk-... go run .
===== 模型回复 =====

===== 发送的请求体 =====
{
  "model": "deepseek-chat",
  "messages": [...],
  "stream": true          ← 只差这一个字
}
========================
Content-Type: text/event-stream
AI Agent 是一个能自主调用工具、分多步完成任务的程序。…    ← 这里是一个字一个字蹦出来的
```

那个 `Content-Type: text/event-stream` 是特意打出来的——它不出现，就说明这个请求根本没走流式，多半是 `stream` 忘了改或者被中间层缓冲了。

二期工程完毕。

---


## 最后

未完～

下一期讲解 cmd —— 把一次调用包进循环，变成能连续提问的命令行程序。

## 参考

- DeepSeek Chat Completion：https://api-docs.deepseek.com/zh-cn/api/create-chat-completion
- Qwen（OpenAI 兼容模式）：https://www.alibabacloud.com/help/zh/model-studio/qwen-api-via-openai-chat-completions
- OpenAI Responses API 迁移指南：https://developers.openai.com/api/docs/guides/migrate-to-responses
- Anthropic Messages API：https://platform.claude.com/docs/en/api/http/beta/messages/create
- Anthropic Streaming：https://platform.claude.com/docs/en/api/messages-streaming
- OpenAI Responses Streaming：https://platform.openai.com/docs/api-reference/responses-streaming
- SSE 规范：https://html.spec.whatwg.org/multipage/server-sent-events.html
- SSE 入门（阮一峰）：https://www.ruanyifeng.com/blog/2017/05/server-sent_events.html


