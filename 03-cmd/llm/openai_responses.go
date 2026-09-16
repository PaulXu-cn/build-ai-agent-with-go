// openai Responses 流式。
//
// 和 Chat Completions 的流式差别很大：
//   事件带类型，靠 type 字段自己认领，结束是 response.completed
//   取文本只需要认 response.output_text.delta 一种

package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

// ---- 请求体 ----

type responsesReq struct {
	Model  string         `json:"model"`
	Input  []responsesMsg `json:"input"`
	Stream bool           `json:"stream"`
}

type responsesMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ---- 响应体：每个事件是一个 JSON，事件名放在 type 字段里 ----

type responsesChunk struct {
	// response.output_text.delta / response.completed / error …
	Type string `json:"type"`
	// 仅 type=response.output_text.delta 时有值
	Delta string `json:"delta"`
}

func streamResponses(ctx context.Context, apiKey, baseURL, model string, messages []Message, onDelta func(string)) error {
	// Responses 的 input 数组和内部 Message 形状一致，直接转
	msgs := make([]responsesMsg, 0, len(messages))
	for _, m := range messages {
		msgs = append(msgs, responsesMsg{Role: m.Role, Content: m.Content})
	}

	reqBody := responsesReq{Model: model, Input: msgs, Stream: true}

	// 鉴权同样是 Bearer，但端点和事件形状都不同
	return streamSSE(ctx, baseURL+"/v1/responses", map[string]string{
		"Authorization": "Bearer " + apiKey,
	}, reqBody, func(data string) error {
		var chunk responsesChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("解析事件失败: %w", err)
		}

		// 事件类型自己认领：要的拿着，不要的忽略
		switch chunk.Type {
		case "response.output_text.delta":
			onDelta(chunk.Delta)
		case "response.completed":
			return errStop
		case "error":
			return fmt.Errorf("服务端返回 error 事件: %s", data)
		}
		return nil
	})
}
