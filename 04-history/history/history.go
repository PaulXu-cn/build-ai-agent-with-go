// 对话历史。
//
// LLM 接口是【无状态】的：服务端不留任何东西，
// 所谓"模型有记忆"，是客户把之前的对话，在请求时一起带上了。

package history

import "github.com/PaulXu-cn/build-ai-agent-with-go/04-history/llm"

// History 一次会话的全部消息。
type History struct {
	messages []llm.Message
}

// New 建一条历史。system 提示放开头，之后一直不动它。
//
// system 只放一次，不要每轮追加：它讲的是"你是谁"，不是"这一轮要干什么"。
func New(system string) *History {
	h := &History{}
	if system != "" {
		h.messages = append(h.messages, llm.Message{Role: "system", Content: system})
	}
	return h
}

// AddUser 记下用户说的话。
func (h *History) AddUser(content string) {
	h.messages = append(h.messages, llm.Message{Role: "user", Content: content})
}

// AddAssistant 记下模型的回答。
//
// 别漏了这一步。只存 user 的话，模型看不到自己说过什么，
// 下一轮会把自己刚讲过的再讲一遍。
func (h *History) AddAssistant(content string) {
	h.messages = append(h.messages, llm.Message{Role: "assistant", Content: content})
}

// Messages 返回当前全部消息，准备发给模型。
//
// 返回的是拷贝。切片共享底层数组，直接把内部切片交出去的话，
// 调用方一 append 就可能写进我们后面还要用的那块内存。
func (h *History) Messages() []llm.Message {
	out := make([]llm.Message, len(h.messages))
	copy(out, h.messages)
	return out
}

// Len 当前消息条数，含 system。
func (h *History) Len() int { return len(h.messages) }

// Reset 清空历史，system 留着。
//
// 清完再问模型"我刚才说了什么"，它会一脸茫然——记忆是我们给的，不是它自带的。
func (h *History) Reset() {
	keep := 0
	if len(h.messages) > 0 && h.messages[0].Role == "system" {
		keep = 1
	}
	h.messages = h.messages[:keep]
}
