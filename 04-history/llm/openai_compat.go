// openai Chat Completions 流式。
//
// 增量文本在 choices[0].delta.content，结束标志是一条固定的 data: [DONE]（不是 JSON）。
// DeepSeek、Qwen、OpenAI 旧接口都用这一套。

package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

// ---- 请求体 ----

type chatCompletionReq struct {
	Model    string              `json:"model"`
	Messages []chatCompletionMsg `json:"messages"`
	Stream   bool                `json:"stream"`
	// StreamOptions 流式下【默认不返回】token 用量，要显式打开才有
	StreamOptions *streamOptions `json:"stream_options,omitempty"`
}

// streamOptions 流式的可选开关。这是 openai 的扩展字段，个别兼容实现不认，会直接 400
type streamOptions struct {
	// IncludeUsage 是否在流的末尾额外发一片只带 usage 的分片
	IncludeUsage bool `json:"include_usage"`
}

type chatCompletionMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ---- 响应体：每个分片是一个小 JSON ----

type chatCompletionChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	// Usage 只在流的末尾出现，而且那一片的 choices 是空数组
	Usage *chatCompletionUsage `json:"usage"`
}

type chatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`

	PromptTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`

	// prompt_tokens == prompt_cache_hit_tokens + prompt_cache_miss_tokens
	PromptCacheHitTokens int `json:"prompt_cache_hit_tokens"`
}

// cached 命中缓存的输入 token。openai 用嵌套字段，DeepSeek 用自己的顶层字段。
func (u chatCompletionUsage) cached() int {
	if u.PromptTokensDetails.CachedTokens > 0 {
		return u.PromptTokensDetails.CachedTokens
	}
	return u.PromptCacheHitTokens
}

func streamCompletion(ctx context.Context, apiKey, baseURL, model string, messages []Message, onDelta func(string)) (Usage, error) {
	// 内部统一的 Message 转成这家方言的线上格式
	msgs := make([]chatCompletionMsg, 0, len(messages))
	for _, m := range messages {
		msgs = append(msgs, chatCompletionMsg{Role: m.Role, Content: m.Content})
	}

	reqBody := chatCompletionReq{
		Model:         model,
		Messages:      msgs,
		Stream:        true,
		StreamOptions: &streamOptions{IncludeUsage: true},
	}

	var usage Usage

	// 鉴权：Authorization: Bearer <key>
	err := streamSSE(ctx, baseURL+"/v1/chat/completions", map[string]string{
		"Authorization": "Bearer " + apiKey,
	}, reqBody, func(data string) error {
		// 结束标志不是 JSON，先拦下来
		if data == "[DONE]" {
			return errStop
		}

		var chunk chatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("解析分片失败: %w", err)
		}

		// 【顺序不能反】先收 usage，再看 choices。
		// 带 usage 的那一片 choices 是空数组，先判断 choices 就会把它当空分片丢掉。
		if chunk.Usage != nil {
			usage = Usage{
				Input:  chunk.Usage.PromptTokens,
				Cached: chunk.Usage.cached(),
				Output: chunk.Usage.CompletionTokens,
			}
		}

		if len(chunk.Choices) == 0 {
			return nil
		}
		onDelta(chunk.Choices[0].Delta.Content)
		return nil
	})
	return usage, err
}
