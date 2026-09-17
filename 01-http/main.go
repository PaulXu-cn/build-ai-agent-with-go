// 本章目标：用最基础的 net/http，实现「三种 LLM API」的非流式调用。

package main

import (
	"fmt"
	"os"
)

const defaultPrompt = "一句话说明什么是 AI Agent。"

// chatFunc 非流式调用函数。
type chatFunc func(apiKey, baseURL, model, prompt string) (string, error)

// providerConf 各厂商的默认端点和模型，以及用哪种方言发送。
type providerConf struct {
	baseURL string
	model   string
	chat    chatFunc
}

// providers 方言注册表
// deepseek / qwen / openai 三行都是 chatCompletion —— 它们是同一种方言。
var providers = map[string]providerConf{
	"deepseek":  {"https://api.deepseek.com", "deepseek-chat", chatCompletion},
	"qwen":      {"https://dashscope.aliyuncs.com/compatible-mode", "qwen-plus", chatCompletion},
	"openai":    {"https://api.openai.com", "gpt-4o-mini", chatCompletion},
	"responses": {"https://api.openai.com", "gpt-4o-mini", chatResponses},
	"claude":    {"https://api.anthropic.com", "claude-opus-4-8", chatAnthropic},
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

	reply, err := conf.chat(
		apiKey,
		envOr("BASE_URL", conf.baseURL),
		envOr("MODEL", conf.model),
		envOr("PROMPT", defaultPrompt),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "调用失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("===== 模型回复 =====")
	fmt.Println(reply)
}

// envOr 读环境变量，为空返回默认值。
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
