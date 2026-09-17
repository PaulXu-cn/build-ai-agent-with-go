// openai Chat Completions。
//
// openai 早期定的 api 标准，现在已经不推荐使用，
// 但格式已事实标准——DeepSeek、Qwen、Kimi、智谱 GLM、Ollama 全是这一套。
//
// 三种方言的主要差异在请求体形状：
//   Chat Completions：messages[].content 是【字符串】
//   Responses：没有 messages，叫 input
//   Anthropic：messages[].content 是【块数组】，system 单独放顶层

package main

import (
	"encoding/json"
	"fmt"
)

// ---- 请求体 ----

type ChatCompletionReq struct {
	// Model 模型
	Model string `json:"model"`
	// Messages 对话历史，API 无状态，每轮全量重发
	Messages []ChatCompletionMsg `json:"messages"`
	// Stream 是否流式，本章统一 false
	Stream bool `json:"stream"`
}

type ChatCompletionMsg struct {
	// Role 角色：system / user / assistant
	Role string `json:"role"`
	// Content 消息内容
	Content string `json:"content"`
}

// ---- 响应体 ----

type ChatCompletionResp struct {
	Choices []struct {
		Message      ChatCompletionMsg `json:"message"`
		FinishReason string            `json:"finish_reason"` // stop=正常结束；length=超长被截断
	} `json:"choices"`
	Usage ChatCompletionUsage `json:"usage"`
}

// ChatCompletionUsage token 用量。
//
// 缓存字段有两套写法，都要读：openai 是嵌套的 prompt_tokens_details.cached_tokens，
// DeepSeek 明明兼容 openai，却把缓存字段放在顶层自己命名。
// 这也是「事实标准」和「真的标准」的差别——兼容不等于一模一样。
type ChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	PromptTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`

	// prompt_tokens == prompt_cache_hit_tokens + prompt_cache_miss_tokens
	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens"`
}

// cachedTokens 命中缓存的输入 token。两家字段名不一样，谁有值用谁。
func (u ChatCompletionUsage) cachedTokens() int {
	if u.PromptTokensDetails.CachedTokens > 0 {
		return u.PromptTokensDetails.CachedTokens
	}
	return u.PromptCacheHitTokens
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

	printUsage(resp.Usage.PromptTokens, resp.Usage.cachedTokens(), resp.Usage.CompletionTokens)
	return resp.Choices[0].Message.Content, nil
}
