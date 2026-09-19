# 让我带你从 0 到 1 用 golang 手写一个 claude code（二）：循环与记忆

上一期我们把请求发出去了，但只能问一次，问完程序就结束。

这一篇我们继续，更近一步：**一个能一直聊下去，而且记得住你上一句说了什么的程序。**

<!-- 配图 1 ｜ 头图 -->
> **🖼 配图 1｜头图**
>
> 出图提示词：终端窗口特写，深色背景，一个简洁的命令行界面里有一问一答两段文本，问答之间用一个循环的细箭头符号连接，暗示"问—答—再问"的往复。画面侧边漂浮着一个半透明的叠层方块，表示"前面说过的内容被带着一起走"。整体冷静、克制、工程感强。横构图 16:9。画面中不要出现任何文字。

---

## 三期：把调用包进循环

### 命令行交互

"对话"这件事展开来看就是：

**读一行输入 → 发一次请求 → 把回答打出来 → 回到第一步。**

就是一个 while 循环。你发起请求，LLM 响应，你阅读他返回的内容，继续提问，经典的 cmd 交互模式

为了简单，我们先用 cmd 相关的接口编程实现，是的，这一期不是 console 编程。

### 主循环长这样

```go
// Run 主循环：打印提示 → 读一行 → 流式回答 → 再循环。
func (a *App) Run() error {
	a.watchInterrupt()
	a.banner()

	for {
		fmt.Print("\n> ")

		// Scan 读到一行返回 true；读到 EOF（Ctrl-D）返回 false
		if !a.in.Scan() {
			fmt.Println("\n再见")
			return a.in.Err()
		}

		input := strings.TrimSpace(a.in.Text())
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			fmt.Println("再见")
			return nil
		}

		a.ask(input)
	}
}
```

`bufio.Scanner` 是这期的老朋友了——上一期用它逐行读 SSE，这一期用它逐行读键盘输入。同一个东西。

在 c 语言里就是 `scan`, 捕获用户的命令行输入

### Ctrl-C 中断：比想象中麻烦一点

默认的 Ctrl-C 是直接把进程杀掉。但我们需要捕获这个信号并处理：

- 如果LLM正在生成内容 → 停掉SSE，回到提示符
- 如果已是空闲状态 → 退出程序

做法是 `signal.Notify` 接管 SIGINT，然后每次提问建一个可取消的 context：

```go
ctx, cancel := context.WithCancel(context.Background())
a.setCancel(cancel)   // 登记为「当前请求」，Ctrl-C 才知道该取消谁
defer a.setCancel(nil)
usage, err := a.streamer.Stream(ctx, messages, onDelta)
```

Ctrl-C 时查一下登记表：有请求在跑就 `cancel()`，没有就退出。

**HTTP 层有个配套要求**：必须用 `http.NewRequestWithContext`。用普通的 `http.NewRequest` 的话，context 取消了也断不掉连接，流还会继续往下收——你会看到回答还在屏幕上往外蹦，但程序已经"以为"自己中断了。

（上一期我说流式不能设 `http.Client.Timeout`，因为那管的是"发请求 + 读完 body"的总时长。现在补上后半句：**超时不由客户端计时，但取消可以由客户端发起**，靠的就是 context。）

### 运行效果

```bash
$ PROVIDER=deepseek API_KEY=sk-... go run .
03-cmd —— 流式对话控制台
回车发送；Ctrl-C 中断回答，空闲时再按一次退出；Ctrl-D 或 exit 退出

> 用一句话说明什么是 AI Agent
AI Agent 是一个能自主调用工具、分多步完成任务的程序。
token: 输入 29，输出 33，合计 62

> 用一句话说明什么是 RAG
RAG 是先检索外部资料、再让模型基于资料回答的做法。
token: 输入 30，输出 32，合计 62

> exit
再见
```

现在是能连续提问了。但它并不知道我上一轮说了啥，是的每次请求都是干净的上下文。

把 `DEBUG=1` 打开，看第二轮的请求体：

```json
{
  "model": "deepseek-chat",
  "messages": [
    { "role": "system", "content": "你是一个简洁的中文助手，回答尽量简短。" },
    { "role": "user", "content": "用一句话说明什么是 RAG" }
  ],
  "stream": true,
  "stream_options": { "include_usage": true }
}
```

**只有两条消息。** 上一句问的 AI Agent 是什么，并没有带上。接下来我们来给他带上记忆
---

## 四期：对话记忆

### 模型其实没有记忆

这是这一期唯一需要记住的一句话：

> **LLM 接口是无状态的，服务端可能会有缓存，没有通过 sees id + 最后一句话的通讯方式**

| 你以为的 | 实际发生的 |
|---|---|
| 模型记得我们刚聊过 | 服务端每次都当第一次见你 |
| 第一轮建立上下文，后面接着聊 | 每轮都要把前面全部内容重新发一遍 |

"模型有记忆"是客户端做出来的。**自己把整条对话存下来，每轮全量发过去**——这就是全部。

这一期我们要实现一个包：

```
04-history/
├── main.go     组装（和上一期一样）
├── history/    新：存整条对话
├── llm/        和上一期逐字相同
└── cmd/        多了一步：把这一轮说过的存进 history
```

其他模块没怎么动，是的这一期，你吃头 history 就行了。

```go
// 03-cmd：每轮现搭，只有两条
messages := []llm.Message{
	{Role: "system", Content: systemPrompt},
	{Role: "user", Content: input},
}

// 04-history：这一句存进历史，然后把整条历史发出去
a.history.AddUser(input)
usage, err := a.streamer.Stream(ctx, a.history.Messages(), onDelta)
```

### history 包

```go
// History 一次会话的全部消息。
type History struct {
	messages []llm.Message
}

func New(system string) *History
func (h *History) AddUser(content string)
func (h *History) AddAssistant(content string)
func (h *History) Messages() []llm.Message   // 发给模型的全量消息
func (h *History) Reset()                    // /clear
```

`AddUser` / `AddAssistant` 拆成两个方法而不是一个 `Add(role, content)`，是故意的：**不给你只存一半的机会**。

加了个 `/clear`。 执行它就会清理上下文，清完再问"我刚才问了你什么"，模型会一脸茫然。**记忆是我们给的，不是它自带的。**

### 三条容易踩的规矩

**一、assistant 的回答也要存。**

只存 user 的话，模型看不到自己说过什么，下一轮会把自己刚讲过的再讲一遍。

**二、system 只放一次。**

它讲的是"你是谁"，不是"这一轮要干什么"。每轮往历史里追加一条 system，历史里就会堆一排 system。

**三、空回答不要塞进历史。**

请求挂了、一个字都没吐出来时（比如鉴权失败），别往历史里塞一条空的 assistant 消息，服务端会拒绝空内容。所以收尾就一句话：

```go
// 把回答记进历史。
if text := reply.String(); text != "" {
	a.history.AddAssistant(text)
}
```

<!-- 配图 3 ｜ 无状态：每轮全量重发 -->
> **🖼 配图 3｜每轮全量重发**
>
> 出图提示词：极简技术示意图，白色背景。画三个从上到下排列的横向条带，代表连续的三轮请求。每一条带宽度的增长用色块长度表示：第一条最短，第二条明显更长，第三条最长。每条带子都被切成若干小段，段与段之间用细线分隔；同一个小段在三轮里始终处于相同的位置，暗示"前面说过的内容被原样重发"。细线条、几何风格、单一强调色、大量留白。横构图 3:2。画面中不要出现任何文字。

### 一个反直觉的设计：失败的那一轮，user 不撤

请求失败时，直觉上应该把这一轮的 user 从历史里撤回，让历史回到出错前的样子。

**一、用户敲的字不能悄悄丢。** 这是个单行输入的 REPL，没有撤销、没有重来。程序把提问删了，用户只能重新打一遍。

**二、连续两条 user 是合法的。** Claude 的文档原话：

> Consecutive `user` or `assistant` turns in your request will be combined into a single turn.

服务端自己会合并，不用你操心。OpenAI / DeepSeek 同样接受。

（例外：Claude 走 AWS Bedrock 时仍然强制 user / assistant 交替，会报 `roles must alternate between "user" and "assistant"`。直连 `api.anthropic.com` 不受影响。这个坑记一下就行。）

### 跑一下

```bash
$ PROVIDER=deepseek API_KEY=sk-... go run .
04-history —— 带记忆的对话
回车发送；Ctrl-C 中断回答，空闲时再按一次退出
/clear 清空历史；Ctrl-D 或 exit 退出

> 你都会什么啊？
我能回答问题、写作、翻译、编程、分析资料等。需要什么帮助？
token: 输入 22，输出 17，合计 39

> 我刚刚输入了什么啊？
你进入了和我对话的界面，可以随时提问或让我帮忙。
token: 输入 53，输出 14，合计 67

> 我的上一句话是什么？
你上句话是："我刚刚输入了什么啊？"
token: 输入 76，输出 15，合计 91
```

**重点看输入 token。**

### 证据就在输入 token 里

```
第 1 轮  输入 22
第 2 轮  输入 53  = 22（上轮输入）+ 17（上轮输出）+ 14（新问题）
第 3 轮  输入 76  = 53 + 14（上轮输出）+ 9（新问题）
```

每一轮的输入 = **上一轮输入 + 上一轮输出 + 这一句新问题**。

这个等式只可能在一个前提下成立：整条历史被原样重发了。


```bash
$ DEBUG=1 PROVIDER=deepseek API_KEY=sk-... go run .
```

第三轮发出去的请求体长这样：

```json
{
  "model": "deepseek-chat",
  "messages": [
    { "role": "system", "content": "你是一个简洁的中文助手，回答尽量简短。" },
    { "role": "user", "content": "你都会什么啊？" },
    { "role": "assistant", "content": "我能回答问题、写作、翻译、编程、分析资料等。…" },
    { "role": "user", "content": "我刚刚输入了什么啊？" },
    { "role": "assistant", "content": "你进入了和我对话的界面，可以随时提问或让我帮忙。" },
    { "role": "user", "content": "我的上一句话是什么？" }
  ],
  "stream": true,
  "stream_options": { "include_usage": true }
}
```

六条消息全在里面，**包括模型自己前两轮说的话**。这就是"记忆"的全部魔法。

### 缓存

输入一轮比一轮多，听着很贵。但实际没那么贵：**不变的开头会被缓存，只算新增那一段的钱。**

不过你上面那次跑，缓存命中全程是 0。原因是**前缀太短**——缓存有最小粒度，不足一个存储单元的前缀落不了盘。上面每轮才几十个 token，整条前缀都够不着门槛；等上下文涨到几百上千，缓存才开始真正省事。

三家的策略也不一样：

| 方言 | 怎么才有缓存 |
|---|---|
| Chat Completions（DeepSeek / OpenAI） | 自动，前缀一样就命中 |
| Responses | 自动，同上 |
| Messages（Anthropic） | 默认不缓存，要在消息块上显式打 `cache_control` |

这也埋了个坑：**历史越长，越贵，而且贵得不是线性的**。怎么在有限预算里塞下更多东西，后面专门有一期讲这个。

---

## 最后

每次代码量都不多，适合人眼 review，不会撑爆你的 context。

未完～

到这里，你已经有一个"能聊天、有记忆"的程序了。但它还只会说话，不会干活。

下一期讲 tool call —— **让模型能"要"东西**。你说"帮我看看当前目录有什么文件"，它不再瞎编一段话敷衍你，而是明确告诉你："我要调用 `ls`，参数是这样。" 我们负责执行，再把结果喂回去。

那是这门课真正开始有意思的地方。

## 参考

- Go 的 context 取消：https://pkg.go.dev/context
- Anthropic Messages API（消息交替规则）：https://platform.claude.com/docs/en/api/messages
- DeepSeek 上下文硬盘缓存：https://api-docs.deepseek.com/zh-cn/guides/kv_cache/
- OpenAI Prompt Caching：https://platform.openai.com/docs/guides/prompt-caching
- Bedrock 与直连 API 在消息交替上的行为差异：https://github.com/anthropics/anthropic-sdk-typescript/issues/565
