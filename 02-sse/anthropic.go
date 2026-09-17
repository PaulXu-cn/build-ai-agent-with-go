// Anthropic Messages 流式。
//
// 事件类型，一次完整对话会依次收到：
//   message_start → content_block_start → content_block_delta × N → content_block_stop → message_delta → message_stop
//
// 取文本只需要认 content_block_delta（且 delta.type=text_delta），
// 结束是 message_stop。
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
	// Content 是对象
	Content []contentBlock `json:"content"`
}

// contentBlock 使用 type 主动区分了 content 的内容：文本/图片/工具调用结果
type contentBlock struct {
	Type string `json:"type"` // text / image / tool_use / tool_result …
	Text string `json:"text"` // 仅 type=text 时有效
}

// ---- 流式响应体：每个事件是一个 JSON ----
type anthropicChunk struct {
	// message_start / content_block_delta / message_delta / message_stop / error …
	Type  string `json:"type"`
	Delta struct {
		// text_delta / input_json_delta …
		Type string `json:"type"`
		// 新增的文本
		Text string `json:"text"`
	} `json:"delta"`
	// message_start 里，输入用量嵌在 message.usage
	Message struct {
		Usage *anthropicUsage `json:"usage"`
	} `json:"message"`
	// message_delta 里，输出用量直接放在顶层 usage，而且是累计值
	Usage *anthropicUsage `json:"usage"`
}

// anthropicUsage token 用量。
//
// 注意 input_tokens 不是输入总量，它是【未缓存】的那部分，全命中缓存时可以是 0。
// 输入总量要三个相加：input_tokens + cache_creation_input_tokens + cache_read_input_tokens
type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`

	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
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

	var in, cached, out int

	// 鉴权：x-api-key + anthropic-version（版本号必填）
	err := streamSSE(baseURL+"/v1/messages", map[string]string{
		"x-api-key":         apiKey,
		"anthropic-version": "2023-06-01",
	}, reqBody, func(data string) error {
		var chunk anthropicChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("解析事件失败: %w", err)
		}

		switch chunk.Type {
		case "message_start":
			// 输入用量开头就给全了。记住三个字段是拆开的，加起来才是总量
			if u := chunk.Message.Usage; u != nil {
				in = u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
				cached = u.CacheReadInputTokens
			}
		case "content_block_delta":
			// 文本块才有 text，工具调用的参数增量走 input_json_delta，这里先不管
			fmt.Print(chunk.Delta.Text)
		case "message_delta":
			// 输出用量在这里，是累计值，直接覆盖
			if chunk.Usage != nil {
				out = chunk.Usage.OutputTokens
			}
		case "message_stop":
			printUsage(in, cached, out)
			return errStop
		case "error":
			return fmt.Errorf("服务端返回 error 事件: %s", data)
		}
		return nil
	})
	return err
}
