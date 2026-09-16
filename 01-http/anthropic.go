// Anthropic Messages。
//
// 和 openai 显著不同：
//   1. 鉴权：不用 Bearer，用 x-api-key + anthropic-version（版本号必填）
//   2. 请求：messages[].content 是对象，system 提示单独放顶层，max_tokens 必填
//   3. 响应：文本在 content[] 块数组里
//
// content 是块数组这点对后续工具调用很关键——工具调用结果就是和文本平级的一个块。
// 文档： https://platform.claude.com/docs/en/api/http/beta/messages/create

package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ---- 请求体 ----

type anthropicReq struct {
	Model string `json:"model"`
	// MaxTokens 必填：最多生成多少 token（openai 那边没有这个必填项）
	MaxTokens int `json:"max_tokens"`
	// System 系统提示，单独放顶层
	System   string         `json:"system"`
	Messages []anthropicMsg `json:"messages"`
	Stream   bool           `json:"stream"`
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

// ---- 响应体 ----

type anthropicResp struct {
	ID         string         `json:"id"`
	Content    []contentBlock `json:"content"`     // 输出也是块数组
	StopReason string         `json:"stop_reason"` // end_turn=正常结束
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func chatAnthropic(apiKey, baseURL, model, prompt string) (string, error) {
	reqBody := anthropicReq{
		Model:     model,
		MaxTokens: 1024,
		System:    "你是一个简洁的中文助手，回答尽量简短。",
		Messages: []anthropicMsg{
			{Role: "user", Content: []contentBlock{{Type: "text", Text: prompt}}},
		},
		Stream: false,
	}

	// 鉴权：x-api-key + anthropic-version（版本号必填，否则报错）
	body, err := postJSON(baseURL+"/v1/messages", map[string]string{
		"x-api-key":         apiKey,
		"anthropic-version": "2023-06-01",
	}, reqBody)
	if err != nil {
		return "", err
	}

	var resp anthropicResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	// content 是块数组，把 type=text 的块拼起来
	var sb strings.Builder
	for _, c := range resp.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	return sb.String(), nil
}
