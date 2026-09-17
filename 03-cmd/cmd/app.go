// 一个简单的 cmd 程序
//
// 接受用户输入，打印 llm 输出，一问一答

package cmd

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

// Streamer 是 cmd 对「模型」的全部要求：给一串消息，边收边通过 onDelta 回调，
type Streamer interface {
	Stream(ctx context.Context, messages []llm.Message, onDelta func(string)) (llm.Usage, error)
}

const systemPrompt = "你是一个简洁的中文助手，回答尽量简短。"

// App 一次终端会话
type App struct {
	streamer Streamer
	in       *bufio.Scanner

	mu sync.Mutex
	// cancel 是当前正在跑的那次请求的取消函数，空闲时为 nil。
	// Ctrl-C 靠它决定是中断回答还是退出程序。
	cancel context.CancelFunc
}

// New 建一次终端会话
func New(streamer Streamer) *App {
	return &App{
		streamer: streamer,
		in:       bufio.NewScanner(os.Stdin),
	}
}

// Run 主循环：打印提示 → 读一行 → 流式回答 → 再循环。
func (a *App) Run() error {
	a.watchInterrupt()
	a.banner()

	for {
		fmt.Print("\n> ")

		// Scan 读到一行返回 true；读到 EOF（Ctrl-D）返回 false
		if !a.in.Scan() {
			fmt.Println("\n再见")
			return a.in.Err()
		}

		input := strings.TrimSpace(a.in.Text())
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			fmt.Println("再见")
			return nil
		}

		a.ask(input)
	}
}

// ask 发一轮提问，把回答边收边打出来。
func (a *App) ask(input string) {
	// 每次提问建一个可取消的不带超时的 context：Ctrl-C 时 cancel 掉
	ctx, cancel := context.WithCancel(context.Background())
	a.setCancel(cancel)
	defer a.setCancel(nil)

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: input},
	}

	usage, err := a.streamer.Stream(ctx, messages, func(delta string) {
		fmt.Print(delta) // LLM返回什么打印什么
	})
	fmt.Println()

	switch {
	case ctx.Err() == context.Canceled:
		fmt.Println("(已中断)")
	case err != nil:
		fmt.Printf("调用失败: %v\n", err)
	default:
		printUsage(usage)
	}
}

// printUsage 把 token 用量打一行。
func printUsage(u llm.Usage) {
	if u.Cached > 0 {
		fmt.Printf("token: 输入 %d（缓存命中 %d），输出 %d，合计 %d\n", u.Input, u.Cached, u.Output, u.Input+u.Output)
		return
	}
	fmt.Printf("token: 输入 %d，输出 %d，合计 %d\n", u.Input, u.Output, u.Input+u.Output)
}

// watchInterrupt 接管 Ctrl-C。默认行为是直接杀进程，这里换成：
//
//	回答进行中 → 中断这一轮，回到输入提示
//	空闲时     → 退出
func (a *App) watchInterrupt() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)

	go func() {
		for range sig {
			if cancel := a.currentCancel(); cancel != nil {
				cancel()
			} else {
				fmt.Println("\n再见")
				os.Exit(0)
			}
		}
	}()
}

func (a *App) banner() {
	fmt.Println("03-cmd —— 流式对话控制台")
	fmt.Println("回车发送；Ctrl-C 中断回答，空闲时再按一次退出；Ctrl-D 或 exit 退出")
}

func (a *App) setCancel(cancel context.CancelFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cancel = cancel
}

func (a *App) currentCancel() context.CancelFunc {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cancel
}
