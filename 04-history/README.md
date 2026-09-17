# 04-history

给对话加上记忆。

上一章的 `messages` 是每轮现搭的：一条 system + 一条 user，发完就扔。所以问完就忘——
你接着问"它和普通程序有什么区别"，模型不知道"它"指什么。

## 模型没有记忆

LLM 接口是**无状态**的，服务端什么都不留。

| 你以为的 | 实际发生的 |
|---|---|
| 模型记得我们刚聊过 | 服务端每次都当第一次见你 |
| 第一轮建立上下文，后面接着聊 | 每轮都要把前面全部内容重新发一遍 |

"模型有记忆"是客户端做出来的：自己把整条对话存下来，每轮全量发过去。
这一章干的就是这件事，多出来的只有一个 `history` 包。

## 结构

```
04-history/
├── main.go     组装：建 Client，交给 App 跑（几行代码）
├── history/    新：存整条对话
├── llm/        和上一章逐字相同
└── cmd/        和用户说话：读输入 + 打印
```

`llm/` 和上一章一模一样，可以直接 diff 两个文件夹。这章只改"消息从哪来"：

```go
// 03-cmd：每轮现搭，只有两条
messages := []llm.Message{
	{Role: "system", Content: systemPrompt},
	{Role: "user", Content: input},
}

// 04-history：这一句存进历史，然后把整条历史发出去
c.history.AddUser(input)
usage, err := c.streamer.Stream(ctx, c.history.Messages(), onDelta)
```

`history` 独立成包，是因为它既不属于 `llm`（发请求的人不关心有没有历史），
也不属于 `cmd`（读输入的人不关心怎么存）。后面要对历史做压缩、存盘、裁剪，
都往这个包里加。

## 三条容易踩的规矩

**1. assistant 的回答也要存。**

只存 user 的话，模型看不到自己说过什么，下一轮会把自己刚讲过的再讲一遍。
所以 `History` 上直接给了两个方法 `AddUser` / `AddAssistant`，不给你只存一半的机会。

**2. system 只放一次。**

它讲的是"你是谁"，不是"这一轮要干什么"。每轮往历史里追加一条 system，
历史里就会堆一排 system。

**3. 空回答不要塞进历史。**

请求挂了、一个字都没吐出来时（比如鉴权失败），别往历史里塞一条空的 assistant 消息，
服务端会拒绝空内容。所以收尾就一句话：

```go
if text := reply.String(); text != "" {
	a.history.AddAssistant(text)
}
```

中断（Ctrl-C）也走这条：屏幕上已经打出来的半截照记——用户下一句很可能就是接着它说的，
不记的话屏幕和历史就对不上。

## 失败的那一轮，user 为什么不撤

直觉上该把这一轮的 user 撤回，让历史回到出错前的样子。不撤有两个理由：

**用户敲的字不能悄悄丢。** 这是个单行输入的 REPL，没有撤销、没有重来，
程序把提问删了，用户只能重新打一遍。

**连续两条 user 是合法的。** Claude 的文档原话：

> Consecutive `user` or `assistant` turns in your request will be combined into a single turn.

所以历史里出现两条挨着的 user 不用管，服务端自己会合并。OpenAI / DeepSeek 同样接受。

> 例外：Claude 走 AWS Bedrock 时仍然强制 user / assistant 交替，会 400
> （`roles must alternate between "user" and "assistant"`）。直连 `api.anthropic.com` 不受影响。

## 交互

| 操作 | 行为 |
|---|---|
| 输入一行 + 回车 | 发送，回答流式打印 |
| Ctrl-C（回答进行中） | 中断这一轮，回到提示符 |
| Ctrl-C（空闲） | 退出 |
| `/clear` | 清空历史，system 留着 |
| Ctrl-D / `exit` / `quit` | 退出 |

`/clear` 值得试一下：清完再问"我刚才问了你什么"，模型一脸茫然。
记忆是我们给的，不是它自带的。

## 运行

```bash
PROVIDER=deepseek  API_KEY=sk-...     go run .
PROVIDER=qwen      API_KEY=sk-...     go run .
PROVIDER=openai    API_KEY=sk-...     go run .
PROVIDER=responses API_KEY=sk-...     go run .
PROVIDER=claude    API_KEY=sk-ant-... go run .
```

也可在根目录 `make run-04-history`。

## 预期输出

问三轮，然后清空历史再问一次。下面这段是拿一个本地假服务跑的真实输出
（回答内容是假服务编的，token 数字是真的）：

```console
04-history —— 带记忆的对话
回车发送；Ctrl-C 中断回答，空闲时再按一次退出
/clear 清空历史；Ctrl-D 或 exit 退出

> 什么是 AI Agent？
（第 1 轮）AI Agent 是一个能自主调用工具、根据结果决定下一步的程序。
token: 输入 40，输出 44，合计 84

> 它和普通程序有什么区别？
（第 2 轮）AI Agent 是一个能自主调用工具、根据结果决定下一步的程序。
token: 输入 100（缓存命中 40），输出 44，合计 144

> 我刚才问了你什么？
你一共问了 3 句：什么是 AI Agent？ / 它和普通程序有什么区别？ / 我刚才问了你什么？
token: 输入 157（缓存命中 100），输出 54，合计 211

> /clear
历史已清空（原有 7 条），再问它刚才聊了什么就不知道了

> 我刚才问了你什么？
你一共问了 1 句：我刚才问了你什么？
token: 输入 36，输出 23，合计 59
```

第三句是关键：假服务照着自己收到的历史把前三句复述了一遍，
说明这三句**确实每次都发出去了**。

## 每轮到底发了什么

`DEBUG=1` 会把原始请求体打出来。上面第二轮发出去的是这样：

```json
{
  "model": "deepseek-chat",
  "messages": [
    { "role": "system", "content": "你是一个简洁的中文助手，回答尽量简短。" },
    { "role": "user", "content": "什么是 AI Agent？" },
    { "role": "assistant", "content": "（第 1 轮）AI Agent 是……" },
    { "role": "user", "content": "它和普通程序有什么区别？" }
  ],
  "stream": true,
  "stream_options": { "include_usage": true }
}
```

四条消息全在里面，包括模型自己上一轮说的话。这就是"记忆"的全部魔法。

## token 为什么一轮比一轮多

注意看输入：`40 → 100 → 157`。历史全量重发，每轮都比上轮长，token 只会往上走。

但缓存命中也在同步涨：`40 → 100`——**上一轮的整条输入，这一轮全命中缓存**。
这就是前缀缓存的形状：不变的开头被缓存，只算新增那一段的钱。

三家的缓存策略不一样：

| 方言 | 怎么才有缓存 |
|---|---|
| Chat Completions（DeepSeek / OpenAI） | 自动，前缀一样就命中 |
| Responses | 自动，同上 |
| Messages（Anthropic） | 默认不缓存，要在消息块上显式打 `cache_control` |

所以"每轮全量重发"听着很贵，实际没有听上去那么贵——但历史终究会撑爆上下文，
这是第十三章 `13-context` 要处理的事。

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

- Anthropic Messages 的 role 交替要求：https://platform.claude.com/docs/en/api/messages
- DeepSeek 上下文硬盘缓存：https://api-docs.deepseek.com/guides/kv_cache
- OpenAI prompt caching：https://platform.openai.com/docs/guides/prompt-caching
