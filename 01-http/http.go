// HTTP 传输层。

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// postJSON 把 body 序列化成 JSON，设置 header，返回响应 body
func postJSON(url string, headers map[string]string, body any) ([]byte, error) {
	// MarshalIndent 是为了打印好看
	jsonBytes, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("编码请求体失败: %w", err)
	}
	fmt.Println("===== 发送的请求体 =====")
	fmt.Println(string(jsonBytes))
	fmt.Println("========================")

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	// 非流式要等模型算完整段话，超时不宜太短
	client := &http.Client{Timeout: 60 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 非 200 就是出错：statusCode + 响应体都打出来
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("返回错误 status=%d body=%s", resp.StatusCode, string(respBytes))
	}

	fmt.Println("===== 收到的响应体 =====")
	// json.Indent 只管是不是合法 JSON，能通用地美化任何方言的响应
	var pretty bytes.Buffer
	if json.Indent(&pretty, respBytes, "", "  ") == nil {
		fmt.Println(pretty.String())
	} else {
		fmt.Println(string(respBytes))
	}
	fmt.Println("========================")
	return respBytes, nil
}

// printUsage 打印 token 用量。
// 三种方言字段名不一样，这里统一成一行；cached 是命中缓存的输入 token，没有就传 0。
func printUsage(in, cached, out int) {
	if cached > 0 {
		fmt.Printf("\ntoken 用量: 输入 %d（缓存命中 %d），输出 %d，合计 %d\n", in, cached, out, in+out)
		return
	}
	fmt.Printf("\ntoken 用量: 输入 %d，输出 %d，合计 %d\n", in, out, in+out)
}
