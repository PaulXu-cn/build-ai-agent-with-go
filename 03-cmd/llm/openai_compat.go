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
}

func streamCompletion(ctx context.Context, apiKey, baseURL, model string, messages []Message, onDelta func(string)) error {
	// 内部统一的 Message 转成这家方言的线上格式
	msgs := make([]chatCompletionMsg, 0, len(messages))
	for _, m := range messages {
		msgs = append(msgs, chatCompletionMsg{Role: m.Role, Content: m.Content})
	}

	reqBody := chatCompletionReq{Model: model, Messages: msgs, Stream: true}

	// 鉴权：Authorization: Bearer <key>
	return streamSSE(ctx, baseURL+"/v1/chat/completions", map[string]string{
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
		if len(chunk.Choices) == 0 {
			return nil
		}
		onDelta(chunk.Choices[0].Delta.Content)
		return nil
	})
}
