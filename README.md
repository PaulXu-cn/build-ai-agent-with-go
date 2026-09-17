# build-ai-agent-with-go

用 Go 从零手写一个 AI Agent（类 Claude Code）的教学课程。

> Build an AI agent from scratch with Go — a step-by-step tutorial series that teaches you how to design, implement, and extend an LLM-powered agent. No magic, just code.

**核心目标不是复刻产品，而是讲透一个 AI Agent 到底怎么工作。**

---

## 一句话主线

> **Agent = 一个 while 循环。**
> 把消息发给 LLM → 模型返回"我要调工具 X(参数)" → 程序执行 X → 结果回传 → 再问模型 → … → 直到模型说"做完了"。

前面几期都在铺地基，一直到第六期 `06-agent-loop` 才把这条循环闭上——那期是整门课的核心。

---

## 课程特点

**每一期是一个独立能跑的文件夹，代码量不大。**

`01-http` 是 5 个文件 451 行，`02-sse` 是 5 个文件 511 行。一屏能读完，你才会真的读一遍；一屏读不完，第一反应就是丢给 AI 让它"蒸馏一下"，那就没达到效果。

**章节之间不互相 import，会有重复代码。**

`01-http` 和 `02-sse` 里都有一个 `openai_compat.go`，内容高度相似。这是故意的——让你能在 IDE 里直接 diff 两个文件夹，看清代码是怎么演进的，而不是去翻 git 的不同版本。

**只用 Go 标准库，零第三方依赖。**

五个 `go.mod` 里 `require` 加起来是 0 行。不引第三方库，是因为引入了你就得去读它的代码——本来只需要懂一个协议，变成要懂一个协议加一个库。

---

## 章节

| 期 | 文件夹 | 讲什么 | 状态 |
|---|---|---|---|
| 一 | `01-http` | 三种 LLM API 方言的非流式调用 | ✅ |
| 二 | `02-sse` | SSE 流式输出 | ✅ |
| 三 | `03-cmd` | 命令行交互程序：把调用包进循环 | ✅ |
| 四 | `04-history` | 消息历史：API 无状态，上下文每轮全量重发 | ✅ |
| 五 | `05-tool-call` | 工具调用：让模型能"要"东西（只走一次往返） | 计划中 |
| 六 | `06-agent-loop` | **Agent 主循环**：把上一次往返包进 while | 计划中 |
| 七 | `07-tools` | shell 与其余工具：超时、中断、并行执行 | 计划中 |
| 八 | `08-security` | 路径穿越、软链接逃逸、提示词注入 | 计划中 |
| 九 | `09-permission` | 危险操作先问一句：确认、白名单、记住选择 | 计划中 |
| 十 | `10-tui` | 全屏终端界面：原始模式、按键解析、中文宽度 | 已写完 * |
| 十一 | `11-mcp` | 接上外部世界：把远程工具接进循环 | 计划中 |
| 十二 | `12-skill` | 第二种扩展方式：模型"知道怎么做" | 计划中 |
| 十三 | `13-context` | 上下文是稀缺资源：token 预算与压缩 | 计划中 |
| 十四 | `14-session` | 会话持久化与 `--resume` | 计划中 |
| 十五 | `15-subagent` | 子 Agent：把任务隔离出去 | 计划中 |
| 十六 | `16-cli` | 非交互模式、输出格式、配置分层 | 计划中 |

\* `10-tui` 的代码已经写完，目前还在 `04-console/`，待重编号。

几个里程碑：

- **03 之后** —— 你能和大模型对话了
- **06 之后** —— 它自己会干活了
- **10 之后** —— 它长得像个真工具了
- **16 之后** —— 完整

---

## 目录结构

每章是一个独立 module（有自己的 `go.mod`），所以**必须在章节目录里跑**，不能在根目录 `go run ./<章节>`。

```
build-ai-agent-with-go/
├── CLAUDE.md          # 规则（只定规矩，不列章节）
├── README.md          # 本文件：课程总览
├── Makefile           # 每章一个 run-<章节> target
├── 01-http/
├── 02-sse/
├── 03-cmd/
├── 04-history/
├── 04-console/        # 待重编号为 10-tui
└── …
```

---

## 环境变量

| 变量 | 作用 | 默认 |
|---|---|---|
| `PROVIDER` | `deepseek` / `qwen` / `openai` / `responses` / `claude` | 必填 |
| `API_KEY` | 密钥 | 必填 |
| `BASE_URL` | 端点根地址，`/v1/...` 由代码拼接 | 见下表 |
| `MODEL` | 模型 | 见下表 |
| `PROMPT` | 提问内容（仅 `01-http` / `02-sse`） | 一句话说明什么是 AI Agent |
| `DEBUG` | 非空则把原始请求打到屏幕上，排障用（`03-cmd` / `04-history` / `10-tui`；后者会打乱全屏画面） | 空 |

| PROVIDER | BASE_URL | MODEL |
|---|---|---|
| deepseek | `https://api.deepseek.com` | `deepseek-chat` |
| qwen | `https://dashscope.aliyuncs.com/compatible-mode` | `qwen-plus` |
| openai | `https://api.openai.com` | `gpt-4o-mini` |
| responses | `https://api.openai.com` | `gpt-4o-mini` |
| claude | `https://api.anthropic.com` | `claude-opus-4-8` |

密钥一律走环境变量，不写死在代码里。想省钱/更快可以换便宜模型，比如 `MODEL=deepseek-chat` 或 `MODEL=claude-haiku-4-5`。

---

## 怎么跑

```bash
# 方式一：根目录
make run-01-http

# 方式二：进章节目录
cd 01-http && PROVIDER=deepseek API_KEY=sk-... go run .
```

各章的具体命令、预期输出、以及该章讲了什么，见**各章自己的 `README.md`**。
`10-tui` 必须在真终端里跑，管道或重定向下无法工作。

---

## 检查

```bash
make fmt     # gofmt
make vet     # go vet
make test    # 目前只有 10-tui 有单元测试（其余各章的逻辑都要连网络）
```

---

## 交付节奏

一期一期往下走，每期结束都保证 `make fmt && make vet` 干净、能跑通，再进入下一期。
