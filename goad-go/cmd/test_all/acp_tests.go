// ACP通信测试
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// GetACPTests 返回ACP通信测试用例
func GetACPTests() []TestCase {
	return []TestCase{
		{"环境变量配置", "ACP通信", testACPEnvConfig},
		{"启动ACP进程", "ACP通信", testACPStartProcess},
		{"初始化请求", "ACP通信", testACPInitialize},
		{"创建会话", "ACP通信", testACPCreateSession},
		{"发送提示", "ACP通信", testACPSendPrompt},
	}
}

func testACPEnvConfig() error {
	// 检查环境变量是否设置
	token := os.Getenv("ANTHROPIC_AUTH_TOKEN")
	baseURL := os.Getenv("ANTHROPIC_BASE_URL")

	if token == "" {
		return fmt.Errorf("ANTHROPIC_AUTH_TOKEN 未设置")
	}
	if baseURL == "" {
		return fmt.Errorf("ANTHROPIC_BASE_URL 未设置")
	}

	return nil
}

func testACPStartProcess() error {
	// 检查 claude-code-acp 是否可执行
	cmd := exec.Command("claude-code-acp", "--help")
	if err := cmd.Run(); err != nil {
		// 尝试直接启动（有些版本可能不支持 --help）
		cmd = exec.Command("which", "claude-code-acp")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("claude-code-acp 不可用: %w", err)
		}
	}
	return nil
}

func testACPInitialize() error {
	cmd := exec.Command("claude-code-acp")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("创建stdin失败: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建stdout失败: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动进程失败: %w", err)
	}
	defer cmd.Process.Kill()

	// 读取响应
	responseCh := make(chan map[string]interface{}, 10)
	go func() {
		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 10*1024*1024)
		scanner.Buffer(buf, len(buf))
		for scanner.Scan() {
			var data map[string]interface{}
			if err := json.Unmarshal(scanner.Bytes(), &data); err == nil {
				responseCh <- data
			}
		}
	}()

	// 发送初始化请求
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialize",
		"id":      1,
		"params": map[string]interface{}{
			"protocolVersion": 1,
			"clientCapabilities": map[string]interface{}{
				"fs":       map[string]interface{}{"readTextFile": true, "writeTextFile": true},
				"terminal": true,
			},
			"clientInfo": map[string]interface{}{
				"name":    "goad-test",
				"title":   "Goad ACP Test",
				"version": "0.1.0",
			},
		},
	}
	data, _ := json.Marshal(initReq)
	stdin.Write(append(data, '\n'))

	// 等待响应
	select {
	case resp := <-responseCh:
		if _, ok := resp["result"]; ok {
			return nil
		}
		if errMsg, ok := resp["error"]; ok {
			return fmt.Errorf("初始化错误: %v", errMsg)
		}
		return nil
	case <-time.After(15 * time.Second):
		return fmt.Errorf("初始化超时")
	}
}

func testACPCreateSession() error {
	cmd := exec.Command("claude-code-acp")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("创建stdin失败: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建stdout失败: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动进程失败: %w", err)
	}
	defer cmd.Process.Kill()

	// 读取响应
	responseCh := make(chan map[string]interface{}, 10)
	go func() {
		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 10*1024*1024)
		scanner.Buffer(buf, len(buf))
		for scanner.Scan() {
			var data map[string]interface{}
			if err := json.Unmarshal(scanner.Bytes(), &data); err == nil {
				responseCh <- data
			}
		}
	}()

	// 发送初始化请求
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialize",
		"id":      1,
		"params": map[string]interface{}{
			"protocolVersion": 1,
			"clientCapabilities": map[string]interface{}{
				"fs":       map[string]interface{}{"readTextFile": true, "writeTextFile": true},
				"terminal": true,
			},
			"clientInfo": map[string]interface{}{
				"name":    "goad-test",
				"title":   "Goad ACP Test",
				"version": "0.1.0",
			},
		},
	}
	data, _ := json.Marshal(initReq)
	stdin.Write(append(data, '\n'))

	// 等待初始化响应
	select {
	case <-responseCh:
		// 继续
	case <-time.After(15 * time.Second):
		return fmt.Errorf("初始化超时")
	}

	// 发送创建会话请求
	cwd, _ := os.Getwd()
	sessionReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "session/new",
		"id":      2,
		"params": map[string]interface{}{
			"cwd":        cwd,
			"mcpServers": []interface{}{},
		},
	}
	data, _ = json.Marshal(sessionReq)
	stdin.Write(append(data, '\n'))

	// 等待会话响应
	select {
	case resp := <-responseCh:
		if result, ok := resp["result"].(map[string]interface{}); ok {
			if sessionID, ok := result["sessionId"].(string); ok && sessionID != "" {
				return nil
			}
		}
		if errMsg, ok := resp["error"]; ok {
			return fmt.Errorf("创建会话错误: %v", errMsg)
		}
		return nil
	case <-time.After(15 * time.Second):
		return fmt.Errorf("创建会话超时")
	}
}

func testACPSendPrompt() error {
	cmd := exec.Command("claude-code-acp")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("创建stdin失败: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建stdout失败: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动进程失败: %w", err)
	}
	defer cmd.Process.Kill()

	// 读取响应
	responseCh := make(chan map[string]interface{}, 100)
	go func() {
		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 10*1024*1024)
		scanner.Buffer(buf, len(buf))
		for scanner.Scan() {
			var data map[string]interface{}
			if err := json.Unmarshal(scanner.Bytes(), &data); err == nil {
				responseCh <- data
			}
		}
	}()

	// 发送初始化请求
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialize",
		"id":      1,
		"params": map[string]interface{}{
			"protocolVersion": 1,
			"clientCapabilities": map[string]interface{}{
				"fs":       map[string]interface{}{"readTextFile": true, "writeTextFile": true},
				"terminal": true,
			},
			"clientInfo": map[string]interface{}{
				"name":    "goad-test",
				"title":   "Goad ACP Test",
				"version": "0.1.0",
			},
		},
	}
	data, _ := json.Marshal(initReq)
	stdin.Write(append(data, '\n'))

	// 等待初始化响应
	select {
	case <-responseCh:
		// 继续
	case <-time.After(15 * time.Second):
		return fmt.Errorf("初始化超时")
	}

	// 发送创建会话请求
	cwd, _ := os.Getwd()
	sessionReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "session/new",
		"id":      2,
		"params": map[string]interface{}{
			"cwd":        cwd,
			"mcpServers": []interface{}{},
		},
	}
	data, _ = json.Marshal(sessionReq)
	stdin.Write(append(data, '\n'))

	// 等待会话响应
	var sessionID string
	select {
	case resp := <-responseCh:
		if result, ok := resp["result"].(map[string]interface{}); ok {
			sessionID, _ = result["sessionId"].(string)
		}
	case <-time.After(15 * time.Second):
		return fmt.Errorf("创建会话超时")
	}

	if sessionID == "" {
		return fmt.Errorf("未获取到会话ID")
	}

	// 发送提示
	promptReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "session/prompt",
		"id":      3,
		"params": map[string]interface{}{
			"sessionId": sessionID,
			"prompt": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "请回复:测试成功",
				},
			},
		},
	}
	data, _ = json.Marshal(promptReq)
	stdin.Write(append(data, '\n'))

	// 等待响应
	timeout := time.After(60 * time.Second)
	for {
		select {
		case resp := <-responseCh:
			// 检查是否是最终响应
			if id, ok := resp["id"].(float64); ok && int(id) == 3 {
				if _, hasResult := resp["result"]; hasResult {
					return nil
				}
			}
		case <-timeout:
			return fmt.Errorf("发送提示超时")
		}
	}
}
