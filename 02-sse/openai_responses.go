// openai Responses 流式。
//
// 和老版的 Chat Completions 的流式的区别：
//   Chat Completions：一条 data: 行 = 一个小 JSON，靠 [DONE] 结束
//   Responses：每条事件带类型，靠 type 字段区分业务类型，结束是 response.completed
//
// 事件类型很多（response.created / response.output_text.delta / response.completed …），
// 输出内容在 response.output_text.delta 类型
// 参考： https://developers.openai.com/api/docs/guides/streaming-responses?api-mode=responses

package main

import (
	"encoding/json"
	"fmt"
)

// ---- 请求体 ----

type ResponsesReq struct {
	// Model 模型
	Model string `json:"model"`
	// Input 输入，消息数组
	Input []ResponsesMsg `json:"input"`
	// Stream 是否流式
	Stream bool `json:"stream"`
}

type ResponsesMsg struct {
	// 角色：system / user / assistant
	Role string `json:"role"`
	// 消息内容
	Content string `json:"content"`
}

// ---- 响应体：每个事件是一个 JSON，事件名放在 type 字段里 ----

type ResponsesChunk struct {
	// response.output_text.delta / response.completed / error …
	Type string `json:"type"`
	// 仅 type=response.output_text.delta 时有值
	Delta string `json:"delta"`
	// response.completed 里带整个请求的 token 用量
	Response struct {
		Usage *ResponsesUsage `json:"usage"`
	} `json:"response"`
}

type ResponsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`

	InputTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"input_tokens_details"`
}

func streamResponses(apiKey, baseURL, model, prompt string) error {
	reqBody := ResponsesReq{
		Model: model,
		Input: []ResponsesMsg{
			{Role: "system", Content: "你是一个简洁的中文助手，回答尽量简短。"},
			{Role: "user", Content: prompt},
		},
		Stream: true,
	}

	// 鉴权同样是 Bearer，但端点和事件形状都不同
	return streamSSE(baseURL+"/v1/responses", map[string]string{
		"Authorization": "Bearer " + apiKey,
	}, reqBody, func(data string) error {
		var chunk ResponsesChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("解析事件失败: %w", err)
		}

		// 事件类型
		switch chunk.Type {
		case "response.output_text.delta":
			fmt.Print(chunk.Delta)
		case "response.completed":
			// 用量在结束事件里，不用像 Chat Completions 那样额外开开关
			if u := chunk.Response.Usage; u != nil {
				printUsage(u.InputTokens, u.InputTokensDetails.CachedTokens, u.OutputTokens)
			}
			return errStop
		case "error":
			return fmt.Errorf("服务端返回 error 事件: %s", data)
		}
		return nil
	})
}
