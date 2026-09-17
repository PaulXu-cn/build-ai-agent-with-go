// 一个简单的 cmd 程序
//
// 接受用户输入，打印 llm 输出，一问一答。
// 和上一章的区别：整条对话存下来了，每轮全量发给模型。

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

	"github.com/PaulXu-cn/build-ai-agent-with-go/04-history/history"
	"github.com/PaulXu-cn/build-ai-agent-with-go/04-history/llm"
)

// Streamer 是 cmd 对「模型」的全部要求：给一串消息，边收边通过 onDelta 回调，
// 结束后把 token 用量交回来。
// llm.Client 天然满足它，两者不用互相知道。
type Streamer interface {
	Stream(ctx context.Context, messages []llm.Message, onDelta func(string)) (llm.Usage, error)
}

const systemPrompt = "你是一个简洁的中文助手，回答尽量简短。"

// App 一次终端会话
type App struct {
	streamer Streamer
	in       *bufio.Scanner
	// history 这条会话的全部消息。每轮把它整条发给模型——
	// 模型能记住上一句，全靠它。
	history *history.History

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
		history:  history.New(systemPrompt),
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

		switch input {
		case "exit", "quit":
			fmt.Println("再见")
			return nil
		case "clear", "/clear":
			n := a.history.Len()
			a.history.Reset()
			fmt.Printf("历史已清空（原有 %d 条），再问它刚才聊了什么就不知道了\n", n)
			continue
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

	// 为历史消息加上刚刚的用户输入
	a.history.AddUser(input)

	// 发送消息并带上历史
	var reply strings.Builder
	usage, err := a.streamer.Stream(ctx, a.history.Messages(), func(delta string) {
		reply.WriteString(delta)
		fmt.Print(delta) // LLM返回什么打印什么
	})
	fmt.Println()

	// 把回答记进历史。
	// 一个字都没出来（比如鉴权失败）就什么都不记：空的 assistant 消息服务端会拒绝。
	// 已经吐出来的半截照记——它打在屏幕上了，用户下一句很可能就是接着它说的。
	//
	// 这一轮的 user 不撤。用户敲的字不该被程序悄悄丢掉，
	// 而且连续两条 user 是合法的，服务端自己会合并成一条
	if text := reply.String(); text != "" {
		a.history.AddAssistant(text)
	}

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
	fmt.Println("04-history —— 带记忆的对话")
	fmt.Println("回车发送；Ctrl-C 中断回答，空闲时再按一次退出")
	fmt.Println("/clear 清空历史；Ctrl-D 或 exit 退出")
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
