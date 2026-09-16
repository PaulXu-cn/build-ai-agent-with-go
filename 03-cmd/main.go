// 本章目标：把「和模型说话」和「和用户说话」拆成两个 package，组装成一个命令行对话程序。
//
//   llm/     —— 和模型说话：HTTP + 协议。不知道终端长什么样
//   console/ —— 和用户说话：读输入 + 打印。不知道 HTTP 长什么样
//   main.go  —— 只负责把两者接起来（几行代码）
//
// 前两章所有代码都在 package main 里，一个文件夹塞完。本章开始拆包：
// 一边是外部世界（模型），一边是用户，两边各自独立变化，中间靠一个接口对接。

package main

import (
	"fmt"
	"os"

	"github.com/PaulXu-cn/build-ai-agent-with-go/03-cmd/console"
	"github.com/PaulXu-cn/build-ai-agent-with-go/03-cmd/llm"
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

	// client 实现了 console.Streamer，直接交给它用
	if err := console.New(client, client.Model()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
