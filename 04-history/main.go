// 本章目标：给对话加上记忆。
//
//   history/ —— 存整条对话。接口是无状态的，"模型记得"全靠它
//   llm/     —— 和模型说话：HTTP + 协议。和上一章逐字相同
//   cmd/     —— 和用户说话：读输入 + 打印
//   main.go  —— 只负责把它们接起来（几行代码）
//
// 上一章的 messages 是每轮现搭的，只有一条 system + 一条 user，所以问完就忘。
// 这一章把每轮说过的话都存进 history，每轮把整条发出去。

package main

import (
	"fmt"
	"os"

	"github.com/PaulXu-cn/build-ai-agent-with-go/04-history/cmd"
	"github.com/PaulXu-cn/build-ai-agent-with-go/04-history/llm"
)

func main() {
	// 配置从环境变量读，这里只做「取值」，默认值由 llm.New 负责补
	client, err := llm.New(llm.Config{
		Provider: os.Getenv("PROVIDER"),
		APIKey:   os.Getenv("API_KEY"),
		BaseURL:  os.Getenv("BASE_URL"),
		Model:    os.Getenv("MODEL"),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// client 实现了 cmd.Streamer，直接交给它用
	if err := cmd.New(client).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
