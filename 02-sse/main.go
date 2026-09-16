// 本章目标：用最基础的 net/http，实现「三种 LLM API」的流式(SSE)调用。
//
// 非流式是「等服务端算完，一次性拿回整段话」；流式是「服务端算一段发一段」。
// 底层协议是 SSE（Server-Sent Events）：一个长连接 + 纯文本，一行一条数据。
//
// 流式与非流式的差异只有两处：
//   请求：stream 改成 true
//   响应：不是一整块 JSON，而是一串小 JSON；读法从 io.ReadAll 变成逐行读
//
// 三种方言的差异依旧在端点、鉴权、事件形状；前两者与非流式相同，
// 事件形状的差异见各自文件。

package main

import (
	"fmt"
	"os"
)

const defaultPrompt = "用一句话说明什么是 AI Agent。"

// chatFunc 是所有方言流式函数的统一签名。文本由各自函数边收边打印，所以不返回内容。
type chatFunc func(apiKey, baseURL, model, prompt string) error

// providerConf 一家厂商的默认端点和模型，以及用哪种方言发送。
type providerConf struct {
	baseURL string
	model   string
	chat    chatFunc
}

// providers 方言注册表
// 注意 deepseek / qwen / openai 三行都是 streamCompletion —— 它们是同一种方言。
var providers = map[string]providerConf{
	"deepseek":  {"https://api.deepseek.com", "deepseek-chat", streamCompletion},
	"qwen":      {"https://dashscope.aliyuncs.com/compatible-mode", "qwen-plus", streamCompletion},
	"openai":    {"https://api.openai.com", "gpt-4o-mini", streamCompletion},
	"responses": {"https://api.openai.com", "gpt-4o-mini", streamResponses},
	"claude":    {"https://api.anthropic.com", "claude-opus-4-8", streamAnthropic},
}

func main() {
	conf, ok := providers[os.Getenv("PROVIDER")]
	if !ok {
		fmt.Fprintln(os.Stderr, "用法：PROVIDER=<deepseek|qwen|openai|responses|claude> API_KEY=... [BASE_URL=...] [MODEL=...] [PROMPT=...] go run .")
		os.Exit(1)
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "缺少 API_KEY")
		os.Exit(1)
	}

	fmt.Println("===== 模型回复 =====")
	err := conf.chat(
		apiKey,
		envOr("BASE_URL", conf.baseURL),
		envOr("MODEL", conf.model),
		envOr("PROMPT", defaultPrompt),
	)
	if err != nil {
		// 流式已经打了一半，报错前先换行，免得糊在一起
		fmt.Fprintf(os.Stderr, "\n调用失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()
}

// envOr 读环境变量，为空返回默认值。
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
