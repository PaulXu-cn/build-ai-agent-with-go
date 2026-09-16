# build-ai-agent-with-go

用 Go 从零手写一个 AI Agent（类 Claude Code）的教学课程。

> Build an AI agent from scratch with Go — a step-by-step tutorial series that teaches you how to design, implement, and extend an LLM-powered agent. No magic, just code.

**核心目标不是复刻产品，而是讲透一个 AI Agent 到底怎么工作。**

---

## 一句话主线

> **Agent = 一个 while 循环。**
> 把消息发给 LLM → 模型返回"我要调工具 X(参数)" → 程序执行 X → 结果回传 → 再问模型 → … → 直到模型说"做完了"。

整门课围绕这条主线，每一章揭开一层黑盒。学完你能自己写一个"迷你 Claude Code"，更关键的是理解它为什么这样设计。

---

## 课程计划

按四个阶段递进。每章一个**独立可执行文件夹**（`go run ./<章节>` 直接跑），代码不互相 import，每章都能单独读懂。

### 阶段一：基础篇 —— 会跑的最小 Agent

先把"和模型说话"这件事跑通，并把它包装成一个能交互的命令行程序。

| 章 | 主题 | 教会你 |
|---|---|---|
| `00-hello` | 最小 HTTP 调用 | LLM = 一个 HTTP 函数；Messages API 的请求/响应结构（`role`/`content`） |
| `01-streaming` | SSE 流式输出 + 配置读取 | 为什么流式（长输出/超时/体验）；配置从环境变量读，不写死 |
| `02-console` | 命令行程序 | console app 基础：读输入 → 循环 → 打印，把前两章包成可交互程序 |
| `03-multi-turn` | 多轮对话 | 消息历史管理；API 无状态，上下文 = 每次全量重发 |

### 阶段二：工具篇 —— 给 Agent 装上"手"

让模型从"只会说话"变成"能干活"。

| 章 | 主题 | 教会你 | 对标真 Claude Code |
|---|---|---|---|
| `04-tool-call` | 工具调用 | function calling：模型只发指令（`tool_use`），程序执行 | — |
| `05-agent-loop` | **Agent 主循环** | **ReAct 模型**（思考→行动→观察）；agent = while 循环 ⭐核心 | `query/` 主循环 |
| `06-file-tools` | 文件工具 | read / write / edit，agent 的眼睛和手 | FileRead/Write/Edit |
| `07-bash-tool` | 命令执行 | shell tool + 安全边界（目录/超时/路径校验） | BashTool |
| `08-search-tools` | 搜索工具 | glob / grep，在代码库里定位信息 | GlobTool/GrepTool |

### 阶段三：进阶篇 —— 更像 Claude Code

补上让它"能用、敢用"的关键能力。

| 章 | 主题 | 教会你 | 对标真 Claude Code |
|---|---|---|---|
| `09-permission` | 权限系统 | agent 不可信，危险操作要人批准 | 权限/审批门 |
| `10-context` | 上下文 / Token 管理 | 上下文窗口是稀缺资源 | compact / contextCollapse |
| `11-cli` | 完整交互式 REPL | 打字机渲染 + Esc 取消 + 历史，把前面全串起来 | 终端入口 |

### 阶段四：高级篇 —— 生产级（后续，暂不编号）

学完前三阶段后再扩展的方向：

- **子 Agent 调度**：把大任务拆给子 agent 并行处理
- **记忆系统**：跨会话记住用户偏好与项目背景
- **并行工具调用**：一次性发出多个工具调用
- **生产级部署**：日志、配置分层、多 provider 接入

---

## 目录结构

```
build-ai-agent-with-go/
├── CLAUDE.md          # 规则（只定规矩，不列章节）
├── README.md          # 本文件：课程总览 + 学习路线
├── Makefile           # 每章一个 run target + mock/vet/fmt 等
├── go.mod             # module 根，go 1.18
├── mock/              # 共享的假模型服务器（离线演示用，本身也是教学材料）
├── 00-hello/          # 最小 HTTP 调用
├── 01-streaming/      # SSE 流式 + 配置读取
├── 02-console/        # 命令行程序
├── 03-multi-turn/     # 多轮对话
├── 04-tool-call/      # 工具调用 / function calling
├── 05-agent-loop/     # Agent 主循环（核心）
├── 06-file-tools/     # 文件读写工具
├── 07-bash-tool/      # 命令执行工具
├── 08-search-tools/   # 搜索工具（glob/grep）
├── 09-permission/     # 权限系统
├── 10-context/        # 上下文 / Token 管理
└── 11-cli/            # 交互式 REPL
```

---

## 环境变量

| 变量 | 作用 | 默认值 |
|---|---|---|
| `ANTHROPIC_API_KEY` | 鉴权（mock 模式可不设） | 无 |
| `ANTHROPIC_BASE_URL` | API 地址，指向 mock 即可离线跑 | `https://api.anthropic.com` |
| `ANTHROPIC_MODEL` | 模型 ID | `claude-opus-4-8` |

- 想省钱/更快：`ANTHROPIC_MODEL=claude-haiku-4-5`（最快最便宜）或 `claude-sonnet-4-6`。

---

## 怎么跑

每章都支持两种跑法：

```bash
# 真实模式：连 Anthropic API
export ANTHROPIC_API_KEY=sk-ant-...
go run ./00-hello

# mock 模式：离线，不花钱，先跑通流程
go run ./mock &                       # 起假模型服务器
ANTHROPIC_BASE_URL=http://localhost:8080 go run ./00-hello
```

> 各章的具体命令和预期输出，见各章自己的 `README.md`。

---

## 交付节奏

- 阶段一 + 阶段二（`00`–`08`）：讲透"agent 怎么工作"的核心闭环。
- 阶段三（`09`–`11`）：产出可用的"迷你 Claude Code"。
- 每个阶段结束都要 `make` 全量能编译、能跑通，再进入下一阶段。
