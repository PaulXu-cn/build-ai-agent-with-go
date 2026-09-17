// openai Responses 流式。
//
// 和 Chat Completions 的流式的区别：
//   Chat Completions：一条 data: 行 = 一个小 JSON，靠 [DONE] 结束
//   Responses：每条事件带类型，靠 type 字段区分业务类型，结束是 response.completed
//
// 事件类型很多，输出内容在 response.output_text.delta 类型

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
	// response.completed 里带整个请求的 token 用量
	Response struct {
		Usage *responsesUsage `json:"usage"`
	} `json:"response"`
}

type responsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`

	InputTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"input_tokens_details"`
}

func streamResponses(ctx context.Context, apiKey, baseURL, model string, messages []Message, onDelta func(string)) (Usage, error) {
	// Responses 的 input 数组和内部 Message 形状一致，直接转
	msgs := make([]responsesMsg, 0, len(messages))
	for _, m := range messages {
		msgs = append(msgs, responsesMsg{Role: m.Role, Content: m.Content})
	}

	reqBody := responsesReq{Model: model, Input: msgs, Stream: true}

	var usage Usage

	// 鉴权同样是 Bearer，但端点和事件形状都不同
	err := streamSSE(ctx, baseURL+"/v1/responses", map[string]string{
		"Authorization": "Bearer " + apiKey,
	}, reqBody, func(data string) error {
		var chunk responsesChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("解析事件失败: %w", err)
		}

		// 事件类型
		switch chunk.Type {
		case "response.output_text.delta":
			onDelta(chunk.Delta)
		case "response.completed":
			// 用量在结束事件里，不用像 Chat Completions 那样额外开开关
			if u := chunk.Response.Usage; u != nil {
				usage = Usage{
					Input:  u.InputTokens,
					Cached: u.InputTokensDetails.CachedTokens,
					Output: u.OutputTokens,
				}
			}
			return errStop
		case "error":
			return fmt.Errorf("服务端返回 error 事件: %s", data)
		}
		return nil
	})
	return usage, err
}
