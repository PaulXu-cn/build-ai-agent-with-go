// Package llm 负责「和模型说话」：把消息发出去，把流式增量收回来。
//
// 它只管 HTTP 和协议，完全不知道终端长什么样——所以既能被命令行程序用，
// 也能被 Web 服务用。这是本章拆分两个 package 的目的。
package llm

import (
	"context"
	"errors"
	"fmt"
)

// Message 一条对话消息。
// 内部统一用这个形状，发出去时再由各方言转成各自的线上格式。
type Message struct {
	Role    string // system / user / assistant
	Content string
}

// Config 连接配置。BaseURL / Model 留空就用该 provider 的默认值。
type Config struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
}

// Client 一个模型客户端
type Client struct {
	apiKey  string
	baseURL string
	model   string
	stream  streamFunc
}

// streamFunc 是每种方言的发送函数。三种方言签名一致，才能放进同一张表。
type streamFunc func(ctx context.Context, apiKey, baseURL, model string, messages []Message, onDelta func(string)) error

// errStop 由方言回调返回，表示流正常结束（比如收到 [DONE]）。
var errStop = errors.New("stream finished")

// providers 方言注册表
// 注意 deepseek / qwen / openai 三行都是 streamCompletion —— 它们是同一种方言。
var providers = map[string]struct {
	baseURL string
	model   string
	stream  streamFunc
}{
	"deepseek":  {"https://api.deepseek.com", "deepseek-chat", streamCompletion},
	"qwen":      {"https://dashscope.aliyuncs.com/compatible-mode", "qwen-plus", streamCompletion},
	"openai":    {"https://api.openai.com", "gpt-4o-mini", streamCompletion},
	"responses": {"https://api.openai.com", "gpt-4o-mini", streamResponses},
	"claude":    {"https://api.anthropic.com", "claude-opus-4-8", streamAnthropic},
}

// New 按 Provider 建客户端，并把留空的 BaseURL / Model 补成默认值。
func New(cfg Config) (*Client, error) {
	p, ok := providers[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("未知 provider：%q，可用：deepseek / qwen / openai / responses / claude", cfg.Provider)
	}
	if cfg.APIKey == "" {
		return nil, errors.New("缺少 API_KEY 环境变量")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = p.baseURL
	}
	model := cfg.Model
	if model == "" {
		model = p.model
	}

	return &Client{apiKey: cfg.APIKey, baseURL: baseURL, model: model, stream: p.stream}, nil
}

// Model 返回当前使用的模型名，界面上可以显示出来。
func (c *Client) Model() string { return c.model }

// Stream 发起一次流式对话，每收到一段文本就回调 onDelta。
// 这个方法让 *Client 满足 console.Streamer 接口，两边因此不用互相 import。
func (c *Client) Stream(ctx context.Context, messages []Message, onDelta func(string)) error {
	return c.stream(ctx, c.apiKey, c.baseURL, c.model, messages, onDelta)
}
