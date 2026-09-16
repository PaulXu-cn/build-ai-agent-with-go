// Anthropic Messages 流式。
//
// 事件类型比其他两家多，一次完整对话会依次收到：
//   message_start → content_block_start → content_block_delta × N → content_block_stop → message_delta → message_stop
//
// 取文本只需要认 content_block_delta（且 delta.type=text_delta），
// 结束是 message_stop。其余事件忽略即可。
// 参考： https://platform.claude.com/docs/en/build-with-claude/streaming

package main

import (
	"encoding/json"
	"fmt"
)

// ---- 请求体 ----

type anthropicReq struct {
	Model string `json:"model"`
	// MaxTokens 必填
	MaxTokens int `json:"max_tokens"`
	// System 系统提示，单独放顶层
	System   string         `json:"system"`
	Messages []anthropicMsg `json:"messages"`
	// Stream 是否流式
	Stream bool `json:"stream"`
}

type anthropicMsg struct {
	Role string `json:"role"`
	// Content 是块数组，不是字符串
	Content []contentBlock `json:"content"`
}

// contentBlock 最小内容单元，文本、图片、工具调用结果都是不同 type 的块
type contentBlock struct {
	Type string `json:"type"` // text / image / tool_use / tool_result …
	Text string `json:"text"` // 仅 type=text 时有效
}

// ---- 响应体：每个事件是一个 JSON ----

type anthropicChunk struct {
	// message_start / content_block_delta / message_stop / error …
	Type  string `json:"type"`
	Delta struct {
		// text_delta / input_json_delta …
		Type string `json:"type"`
		// 这一小段新增的文本
		Text string `json:"text"`
	} `json:"delta"`
}

func streamAnthropic(apiKey, baseURL, model, prompt string) error {
	reqBody := anthropicReq{
		Model:     model,
		MaxTokens: 1024,
		System:    "你是一个简洁的中文助手，回答尽量简短。",
		Messages: []anthropicMsg{
			{Role: "user", Content: []contentBlock{{Type: "text", Text: prompt}}},
		},
		Stream: true,
	}

	// 鉴权：x-api-key + anthropic-version（版本号必填）
	return streamSSE(baseURL+"/v1/messages", map[string]string{
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
			fmt.Print(chunk.Delta.Text)
		case "message_stop":
			return errStop
		case "error":
			return fmt.Errorf("服务端返回 error 事件: %s", data)
		}
		return nil
	})
}
