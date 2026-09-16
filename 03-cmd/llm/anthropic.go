// Anthropic Messages 流式。
//
// 一次完整对话会依次收到：
//   message_start → content_block_start → content_block_delta × N → content_block_stop → message_delta → message_stop
//
// 取文本只需要认 content_block_delta，结束是 message_stop。
//
// 转换上有一处要注意：Anthropic 的 system 提示在顶层，不在 messages 里，
// 所以要把它从消息数组里挑出来单独放。

package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

// ---- 请求体 ----

type anthropicReq struct {
	Model string `json:"model"`
	// MaxTokens 必填：最多生成多少 token
	MaxTokens int `json:"max_tokens"`
	// System 系统提示，单独放顶层
	System   string         `json:"system,omitempty"`
	Messages []anthropicMsg `json:"messages"`
	Stream   bool           `json:"stream"`
}

type anthropicMsg struct {
	Role string `json:"role"`
	// Content 是块数组，不是字符串
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"` // text / image / tool_use / tool_result …
	Text string `json:"text,omitempty"`
}

// ---- 响应体：每个事件是一个 JSON ----

type anthropicChunk struct {
	// message_start / content_block_delta / message_stop / error …
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"` // text_delta / input_json_delta …
		Text string `json:"text"`
	} `json:"delta"`
}

func streamAnthropic(ctx context.Context, apiKey, baseURL, model string, messages []Message, onDelta func(string)) error {
	// 转成 Anthropic 格式时要做两件事：
	//   1. system 消息挑出来放顶层，不留在 messages 里
	//   2. 每条消息的 content 从字符串包成块数组
	var system string
	msgs := make([]anthropicMsg, 0, len(messages))
	for _, m := range messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		msgs = append(msgs, anthropicMsg{
			Role:    m.Role,
			Content: []contentBlock{{Type: "text", Text: m.Content}},
		})
	}

	reqBody := anthropicReq{
		Model:     model,
		MaxTokens: 4096,
		System:    system,
		Messages:  msgs,
		Stream:    true,
	}

	// 鉴权：x-api-key + anthropic-version（版本号必填）
	return streamSSE(ctx, baseURL+"/v1/messages", map[string]string{
		"x-api-key":         apiKey,
		"anthropic-version": "2023-06-01",
	}, reqBody, func(data string) error {
		var chunk anthropicChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("解析事件失败: %w", err)
		}

		switch chunk.Type {
		case "content_block_delta":
			// 文本块才有 text，工具调用的参数增量走 input_json_delta，这里先不管
			onDelta(chunk.Delta.Text)
		case "message_stop":
			return errStop
		case "error":
			return fmt.Errorf("服务端返回 error 事件: %s", data)
		}
		return nil
	})
}
