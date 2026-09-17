// openai Chat Completions 流式。
//
// 流式输出，增量文本的字段从 message.content 变成 delta.content（delta = 增量）。
// 结束标志是一条固定的 data: [DONE]，不是 JSON。
// 参考文档： https://help.aliyun.com/zh/model-studio/stream

package main

import (
	"encoding/json"
	"fmt"
)

// ---- 请求体 ----

type ChatCompletionReq struct {
	// Model 模型
	Model string `json:"model"`
	// Messages 对话历史
	Messages []ChatCompletionMsg `json:"messages"`
	// Stream 是否流式
	Stream bool `json:"stream"`
	// StreamOptions 流式下【默认不返回】token 用量，要显式打开才有
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`
}

// StreamOptions 流式的可选开关。这是 openai 的扩展字段，个别兼容实现不认，会直接 400
type StreamOptions struct {
	// IncludeUsage 是否在流的末尾额外发一片只带 usage 的分片
	IncludeUsage bool `json:"include_usage"`
}

type ChatCompletionMsg struct {
	// 角色：system / user / assistant
	Role string `json:"role"`
	// 消息内容
	Content string `json:"content"`
}

// ---- 响应体：每个分片是一个小 JSON ----

type ChatCompletionChunk struct {
	Choices []struct {
		Delta struct {
			// 这一小段新增的文本
			Content string `json:"content"`
		} `json:"delta"`
		// stop=正常结束；流式下只有最后一片才有值
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	// Usage 只在流的末尾出现，其余分片是 null。
	// 用指针是为了区分「没有这个字段」和「有这个字段但都是 0」
	Usage *ChatCompletionUsage `json:"usage"`
}

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

// cachedTokens 命中缓存的输入 token。openai 用嵌套字段，DeepSeek 用自己的顶层字段。
func (u ChatCompletionUsage) cachedTokens() int {
	if u.PromptTokensDetails.CachedTokens > 0 {
		return u.PromptTokensDetails.CachedTokens
	}
	return u.PromptCacheHitTokens
}

func streamCompletion(apiKey, baseURL, model, prompt string) error {
	reqBody := ChatCompletionReq{
		Model: model,
		Messages: []ChatCompletionMsg{
			{Role: "system", Content: "你是一个简洁的中文助手，回答尽量简短。"},
			{Role: "user", Content: prompt},
		},
		Stream:        true,
		StreamOptions: &StreamOptions{IncludeUsage: true},
	}

	var usage *ChatCompletionUsage

	// 鉴权：Authorization: Bearer <key>
	err := streamSSE(baseURL+"/v1/chat/completions", map[string]string{
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

		// 【顺序不能反】先收 usage，再看 choices。
		// 带 usage 的那一片 choices 是空数组，先判断 choices 就会把它当空分片丢掉，
		// token 用量于是永远算不出来。这是 OpenAI 兼容流式最常见的一个坑。
		if chunk.Usage != nil {
			usage = chunk.Usage
		}

		if len(chunk.Choices) == 0 {
			return nil
		}

		// 收到LLM输出内容就打印出来，而不是等生成完毕再打印，
		fmt.Print(chunk.Choices[0].Delta.Content)
		return nil
	})
	if err != nil {
		return err
	}

	if usage != nil {
		printUsage(usage.PromptTokens, usage.cachedTokens(), usage.CompletionTokens)
	}
	return nil
}
