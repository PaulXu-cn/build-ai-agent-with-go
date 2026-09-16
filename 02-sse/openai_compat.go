// openai Chat Completions 流式。
//
// 流式和上一章非流式的差别只在两处：
//   请求：stream 改成 true
//   响应：不再是「一个完整 JSON」，而是「一串小 JSON」，每个只带一小段文本
//
// 增量文本的位置也从 message.content 变成 delta.content（delta = 增量）。
// 结束标志是一条固定的 data: [DONE]，不是 JSON。
// 参考文档： https://help.aliyun.com/zh/model-studio/stream

package main

import (
	"encoding/json"
	"fmt"
)

// ---- 请求体 ----

type ChatCompletionReq struct {
	// Model 模型，deepseek-chat / qwen-plus / gpt-4o-mini …
	Model string `json:"model"`
	// Messages 对话历史
	Messages []ChatCompletionMsg `json:"messages"`
	// Stream 是否流式
	Stream bool `json:"stream"`
}

type ChatCompletionMsg struct {
	// 角色：system / user / assistant
	Role string `json:"role"`
	// 消息内容
	Content string `json:"content"`
}

// ---- 响应体：每个分片是一个小 JSON ----

type ChatCompletionChunk struct {
	Choices []struct {
		Delta struct {
			// 这一小段新增的文本
			Content string `json:"content"`
		} `json:"delta"`
		// stop=正常结束；流式下只有最后一片才有值
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func streamCompletion(apiKey, baseURL, model, prompt string) error {
	reqBody := ChatCompletionReq{
		Model: model,
		Messages: []ChatCompletionMsg{
			{Role: "system", Content: "你是一个简洁的中文助手，回答尽量简短。"},
			{Role: "user", Content: prompt},
		},
		Stream: true,
	}

	// 鉴权：Authorization: Bearer <key>
	return streamSSE(baseURL+"/v1/chat/completions", map[string]string{
		"Authorization": "Bearer " + apiKey,
	}, reqBody, func(data string) error {
		// 结束标志不是 JSON，先拦下来
		if data == "[DONE]" {
			return errStop
		}

		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("解析分片失败: %w", err)
		}
		if len(chunk.Choices) == 0 {
			return nil
		}

		// 收到一段就打一段，产出一边生成一边显示，这就是流式的意义
		fmt.Print(chunk.Choices[0].Delta.Content)
		return nil
	})
}
