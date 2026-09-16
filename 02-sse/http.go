// HTTP 传输层：SSE 流式读取。
//
// 参考文档： https://www.ruanyifeng.com/blog/2017/05/server-sent_events.html
// 文档：https://help.aliyun.com/zh/oss/user-guide/sse-mode-description

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// errStop 由回调返回，表示「流可以到此为止了」（比如收到 [DONE]）。
var errStop = errors.New("stream finished")

// streamSSE 发送请求，把响应体按 SSE 逐行读出，每读到一条数据就交给 onData。
// onData 拿到的是 "data:" 后面的内容；返回 errStop 表示正常结束。
func streamSSE(url string, headers map[string]string, body any, onData func(data string) error) error {
	jsonBytes, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return fmt.Errorf("编码请求体失败: %w", err)
	}
	fmt.Println("===== 发送的请求体 =====")
	fmt.Println(string(jsonBytes))
	fmt.Println("========================")

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	// 流式【不能】设 http.Client.Timeout：它管的是「发请求 + 读完 body」的总时长，
	// 回答一长就会被拦腰砍断。流式什么时候结束，由服务端的结束标志决定。
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 正常时这里是 text/event-stream；不是的话说明请求根本没走流式
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))

	if resp.StatusCode != http.StatusOK {
		// 错误响应不是流，整块读出来打印
		errBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("返回错误 status=%d body=%s", resp.StatusCode, string(errBytes))
	}

	// SSE 是纯文本协议，一行一条。bufio.Scanner 按行读，读一行给一行。
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// 空行是事件分隔符，以 : 开头的是心跳注释行，都跳过
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		// 一个事件通常是两行：event: 事件名 / data: 数据。
		// 事件名在 data 的 JSON 里还有一份（type 字段），所以只取 data: 就够。
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

		if err := onData(data); err != nil {
			if errors.Is(err, errStop) {
				return nil
			}
			return err
		}
	}
	return scanner.Err()
}
