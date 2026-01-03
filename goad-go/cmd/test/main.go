// Package main 提供一个简单的ACP测试程序
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

func main() {
	fmt.Println("=== Goad ACP 测试 ===")
	fmt.Println()

	// 设置环境变量
	os.Setenv("ANTHROPIC_AUTH_TOKEN", "sk-1xZLD7dGf8g8IgQWkwxwMmpUSnbhUhfyIT9F1bugnmK1f5hB")
	os.Setenv("ANTHROPIC_BASE_URL", "https://pmpjfbhq.cn-nb1.rainapp.top")

	// 启动 claude-code-acp
	fmt.Println("启动 claude-code-acp...")
	cmd := exec.Command("claude-code-acp")
	cmd.Dir, _ = os.Getwd()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Printf("创建stdin失败: %v\n", err)
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("创建stdout失败: %v\n", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("创建stderr失败: %v\n", err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("启动进程失败: %v\n", err)
		return
	}

	// 读取stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Printf("[stderr] %s\n", scanner.Text())
		}
	}()

	// 读取stdout的goroutine
	responseCh := make(chan map[string]interface{}, 10)
	go func() {
		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 10*1024*1024)
		scanner.Buffer(buf, len(buf))
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("[收到] %s\n", truncate(line, 200))
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(line), &data); err == nil {
				responseCh <- data
			}
		}
	}()

	// 发送初始化请求
	fmt.Println("\n--- 发送 initialize 请求 ---")
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialize",
		"id":      1,
		"params": map[string]interface{}{
			"protocolVersion": 1,
			"clientCapabilities": map[string]interface{}{
				"fs": map[string]interface{}{
					"readTextFile":  true,
					"writeTextFile": true,
				},
				"terminal": true,
			},
			"clientInfo": map[string]interface{}{
				"name":    "goad-test",
				"title":   "Goad ACP Test",
				"version": "0.1.0",
			},
		},
	}

	sendRequest(stdin, initReq)

	// 等待响应
	select {
	case resp := <-responseCh:
		fmt.Printf("\n初始化响应: %v\n", formatJSON(resp))
	case <-time.After(10 * time.Second):
		fmt.Println("初始化超时")
		cmd.Process.Kill()
		return
	}

	// 发送创建会话请求
	fmt.Println("\n--- 发送 session/new 请求 ---")
	sessionReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "session/new",
		"id":      2,
		"params": map[string]interface{}{
			"cwd":        cmd.Dir,
			"mcpServers": []interface{}{},
		},
	}

	sendRequest(stdin, sessionReq)

	var sessionID string
	select {
	case resp := <-responseCh:
		fmt.Printf("\n会话响应: %v\n", formatJSON(resp))
		if result, ok := resp["result"].(map[string]interface{}); ok {
			sessionID, _ = result["sessionId"].(string)
		}
	case <-time.After(10 * time.Second):
		fmt.Println("创建会话超时")
		cmd.Process.Kill()
		return
	}

	if sessionID == "" {
		fmt.Println("未获取到会话ID")
		cmd.Process.Kill()
		return
	}

	fmt.Printf("\n会话ID: %s\n", sessionID)

	// 发送一个简单的提示
	fmt.Println("\n--- 发送 session/prompt 请求 ---")
	promptReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "session/prompt",
		"id":      3,
		"params": map[string]interface{}{
			"sessionId": sessionID,
			"prompt": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "请用一句话介绍你自己",
				},
			},
		},
	}

	sendRequest(stdin, promptReq)

	// 收集响应
	fmt.Println("\n--- 等待代理响应 ---")
	timeout := time.After(30 * time.Second)
	for {
		select {
		case resp := <-responseCh:
			// 检查是否是最终响应
			if _, hasResult := resp["result"]; hasResult {
				if id, ok := resp["id"].(float64); ok && int(id) == 3 {
					fmt.Printf("\n提示响应完成: %v\n", formatJSON(resp))
					goto done
				}
			}
			// 检查session/update通知
			if method, ok := resp["method"].(string); ok && method == "session/update" {
				if params, ok := resp["params"].(map[string]interface{}); ok {
					if update, ok := params["update"].(map[string]interface{}); ok {
						handleUpdate(update)
					}
				}
			}
		case <-timeout:
			fmt.Println("\n等待响应超时")
			goto done
		}
	}

done:
	fmt.Println("\n=== 测试完成 ===")
	cmd.Process.Kill()
	cmd.Wait()
}

func sendRequest(stdin io.Writer, req map[string]interface{}) {
	data, _ := json.Marshal(req)
	fmt.Printf("[发送] %s\n", string(data))
	stdin.Write(append(data, '\n'))
}

func handleUpdate(update map[string]interface{}) {
	updateType, _ := update["sessionUpdate"].(string)
	switch updateType {
	case "agent_message_chunk":
		if content, ok := update["content"].(map[string]interface{}); ok {
			if text, ok := content["text"].(string); ok {
				fmt.Print(text)
			}
		}
	case "agent_thought_chunk":
		if content, ok := update["content"].(map[string]interface{}); ok {
			if text, ok := content["text"].(string); ok {
				fmt.Printf("[思考] %s", text)
			}
		}
	case "tool_call":
		title, _ := update["title"].(string)
		fmt.Printf("\n[工具调用] %s\n", title)
	case "tool_call_update":
		status, _ := update["status"].(string)
		fmt.Printf("[工具更新] 状态: %s\n", status)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func formatJSON(v interface{}) string {
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data)
}
