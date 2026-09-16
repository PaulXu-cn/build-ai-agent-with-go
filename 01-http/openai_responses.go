// openai Responses。
//
// openai 新推的接口，设计上更面向 agent：
//   · 输入叫 input，可以是字符串，也可以是消息数组（本章用数组，以便发多条消息）
//   · 输出叫 output，是 typed 数组：message / reasoning / function_call 是不同 item
//   · 多轮可用 previous_response_id 让服务端续上下文
//
// 和 Chat Completions 最直观的区别：
//   请求：messages[]                        → input
//   响应：choices[0].message.content        → output[] 里 type=message 的 content[].text
// 文档： https://developers.openai.com/api/docs/guides/migrate-to-responses?update-generation-endpoints=responses&update-multiturn=responses&structured-outputs=responses&tool-use=responses#about-the-responses-api

package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ---- 请求体 ----

type ResponsesReq struct {
	// Model 模型
	Model string `json:"model"`
	// Input 输入。也可直接写字符串，这里用消息数组以便发多条
	Input []ResponsesMsg `json:"input"`
	// Stream 是否流式，本章统一 false
	Stream bool `json:"stream"`
}

type ResponsesMsg struct {
	// 角色：system / user / assistant
	Role string `json:"role"`
	// 消息内容。也可以是 typed content 数组，这里用最简单的字符串
	Content string `json:"content"`
}

// ---- 响应体 ----

type ResponsesResp struct {
	ID     string `json:"id"`
	Output []struct {
		Type    string `json:"type"` // message / reasoning / function_call …
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"` // output_text …
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

func chatResponses(apiKey, baseURL, model, prompt string) (string, error) {
	reqBody := ResponsesReq{
		Model: model,
		Input: []ResponsesMsg{
			{Role: "system", Content: "你是一个简洁的中文助手，回答尽量简短。"},
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	// 鉴权同样是 Bearer，但端点和响应形状都不同
	body, err := postJSON(baseURL+"/v1/responses", map[string]string{
		"Authorization": "Bearer " + apiKey,
	}, reqBody)
	if err != nil {
		return "", err
	}

	var resp ResponsesResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	// output 是 typed 数组：挑出 type=message 的项，取它 content[] 里 type=output_text 的文本
	var sb strings.Builder
	for _, item := range resp.Output {
		if item.Type != "message" {
			continue
		}
		for _, c := range item.Content {
			if c.Type == "output_text" {
				sb.WriteString(c.Text)
			}
		}
	}
	if sb.Len() == 0 {
		return "", fmt.Errorf("响应 output 里没有文本")
	}
	return sb.String(), nil
}
