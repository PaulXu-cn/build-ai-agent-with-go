// openai Chat Completions。
//
// 它是 openai 早期定的 api 标准，目前 openai 不推荐使用，
// 但 Chat Completions 的请求/响应格式已事实标准，被其他厂商照着实现——
// DeepSeek、Qwen（通义千问）、Kimi、智谱 GLM、Ollama…… 全是这一套。
//
// 三种方言的主要差异在请求体形状：
//   Chat Completions：messages[].content 是【字符串】
//   Responses：没有 messages，叫 input（见 openai_responses.go）
//   Anthropic：messages[].content 是【块数组】，system 单独放顶层（见 anthropic.go）

package main

import (
	"encoding/json"
	"fmt"
)

// ---- 请求体 ----

type ChatCompletionReq struct {
	// Model 模型，deepseek-chat / qwen-plus / gpt-4o-mini …
	Model string `json:"model"`
	// Messages 对话历史。API 无状态：每轮都要把整个上下文全量重发。
	Messages []ChatCompletionMsg `json:"messages"`
	// Stream 是否流式，本章统一 false
	Stream bool `json:"stream"`
}

type ChatCompletionMsg struct {
	// 角色：system / user / assistant
	Role string `json:"role"`
	// 消息内容
	Content string `json:"content"`
}

// ---- 响应体 ----

type ChatCompletionResp struct {
	Choices []struct {
		Message      ChatCompletionMsg `json:"message"`
		FinishReason string            `json:"finish_reason"` // stop=正常结束；length=超长被截断
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
