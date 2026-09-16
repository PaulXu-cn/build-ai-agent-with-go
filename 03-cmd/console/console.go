// Package console 负责终端交互：读用户输入，把回答逐段打到屏幕上。
//
// 它只管输入输出，不知道 HTTP 和协议——所以把「怎么拿回答」抽象成一个 Streamer 接口。
// 接口定义在使用方（这里）而不是实现方，这是 Go 的惯例：谁用谁定。
package console

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/PaulXu-cn/build-ai-agent-with-go/03-cmd/llm"
)

// Streamer 是 console 对「模型」的全部要求：给一串消息，边收边通过 onDelta 回调。
// llm.Client 天然满足它，两者不用互相知道。
type Streamer interface {
	Stream(ctx context.Context, messages []llm.Message, onDelta func(string)) error
}

const systemPrompt = "你是一个简洁的中文助手，回答尽量简短。"

// Console 终端会话
type Console struct {
	streamer Streamer
	model    string
	in       *bufio.Scanner

	mu sync.Mutex
	// cancel 是「当前正在跑的那次请求」的取消函数，空闲时为 nil。
	// Ctrl-C 靠它决定该中断回答还是该退出程序。
	cancel context.CancelFunc
}

// New 建一个终端会话
func New(streamer Streamer, model string) *Console {
	return &Console{
		streamer: streamer,
		model:    model,
		in:       bufio.NewScanner(os.Stdin),
	}
}

// Run 主循环：打印提示 → 读一行 → 流式回答 → 再循环。
func (c *Console) Run() error {
	c.watchInterrupt()
	c.banner()

	for {
		fmt.Print("\n> ")

		// Scan 读到一行返回 true；读到 EOF（Ctrl-D）返回 false
		if !c.in.Scan() {
			fmt.Println("\n再见")
			return c.in.Err()
		}

		input := strings.TrimSpace(c.in.Text())
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			fmt.Println("再见")
			return nil
		}

		c.ask(input)
	}
}

// ask 发一轮提问，把回答边收边打出来。
func (c *Console) ask(input string) {
	// 每次提问建一个可取消的 context：Ctrl-C 时 cancel 掉，正在收的流会被立刻掐断
	ctx, cancel := context.WithCancel(context.Background())
	c.setCancel(cancel) // 登记为「当前请求」，Ctrl-C 才知道该取消谁
	defer c.setCancel(nil)

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: input},
	}

	err := c.streamer.Stream(ctx, messages, func(delta string) {
		fmt.Print(delta) // 收到一段打一段
	})
	fmt.Println()

	switch {
	case ctx.Err() == context.Canceled:
		fmt.Println("(已中断)")
	case err != nil:
		fmt.Printf("调用失败: %v\n", err)
	}
}

// watchInterrupt 接管 Ctrl-C。默认行为是直接杀进程，这里换成：
//   回答进行中 → 中断这一轮，回到输入提示
//   空闲时     → 退出
func (c *Console) watchInterrupt() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)

	go func() {
		for range sig {
			if cancel := c.currentCancel(); cancel != nil {
				cancel()
			} else {
				fmt.Println("\n再见")
				os.Exit(0)
			}
		}
	}()
}

func (c *Console) banner() {
	fmt.Println("03-cmd —— 流式对话控制台")
	fmt.Printf("model: %s\n", c.model)
	fmt.Println("回车发送；Ctrl-C 中断回答，空闲时再按一次退出；Ctrl-D 或 exit 退出")
}

func (c *Console) setCancel(cancel context.CancelFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cancel = cancel
}

func (c *Console) currentCancel() context.CancelFunc {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cancel
}
