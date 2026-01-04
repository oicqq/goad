// 配置管理测试
package main

import (
	"fmt"
	"sync"

	"github.com/anthropics/goad/internal/config"
)

// GetConfigTests 返回配置管理测试用例
func GetConfigTests() []TestCase {
	return []TestCase{
		{"默认配置创建", "配置管理", testConfigDefault},
		{"代理注册表", "配置管理", testAgentRegistry},
		{"API环境变量", "配置管理", testAPIConfig},
		{"MCP服务器配置", "配置管理", testMCPConfig},
		{"配置目录路径", "配置管理", testConfigDir},
		{"并发配置读取", "配置管理", testConfigConcurrent},
	}
}

func testConfigDefault() error {
	cfg := config.DefaultAppConfig()
	if cfg == nil {
		return fmt.Errorf("默认配置为空")
	}
	if cfg.DefaultAgent != "claude.com" {
		return fmt.Errorf("默认代理不正确: %s", cfg.DefaultAgent)
	}
	if cfg.API == nil {
		return fmt.Errorf("API配置为空")
	}
	if cfg.Theme != "default" {
		return fmt.Errorf("默认主题不正确: %s", cfg.Theme)
	}
	if cfg.Language != "zh-CN" {
		return fmt.Errorf("默认语言不正确: %s", cfg.Language)
	}
	return nil
}

func testAgentRegistry() error {
	registry := config.NewAgentRegistry()
	if registry == nil {
		return fmt.Errorf("注册表为空")
	}

	// 测试列表方法
	agents := registry.List()
	if agents == nil {
		return fmt.Errorf("代理列表为nil")
	}

	// 测试获取不存在的代理
	_, ok := registry.Get("non-existent")
	if ok {
		return fmt.Errorf("不应该找到不存在的代理")
	}

	// 测试按短名称获取
	_, ok = registry.GetByShortName("non-existent")
	if ok {
		return fmt.Errorf("不应该找到不存在的代理")
	}

	return nil
}

func testAPIConfig() error {
	apiCfg := &config.APIConfig{
		ClaudeAPIKey:  "claude-key",
		ClaudeBaseURL: "https://claude.url",
		OpenAIAPIKey:  "openai-key",
		OpenAIBaseURL: "https://openai.url",
		GeminiAPIKey:  "gemini-key",
		GeminiBaseURL: "https://gemini.url",
	}

	// 测试 Claude 环境变量
	env := apiCfg.GetEnvVars("claude.com")
	if env["ANTHROPIC_API_KEY"] != "claude-key" {
		return fmt.Errorf("Claude API Key未正确设置")
	}
	if env["ANTHROPIC_BASE_URL"] != "https://claude.url" {
		return fmt.Errorf("Claude Base URL未正确设置")
	}

	// 测试 OpenAI 环境变量
	env = apiCfg.GetEnvVars("openai.com")
	if env["OPENAI_API_KEY"] != "openai-key" {
		return fmt.Errorf("OpenAI API Key未正确设置")
	}

	// 测试 Gemini 环境变量
	env = apiCfg.GetEnvVars("geminicli.com")
	if env["GOOGLE_API_KEY"] != "gemini-key" {
		return fmt.Errorf("Gemini API Key未正确设置")
	}

	return nil
}

func testMCPConfig() error {
	cfg := &config.AppConfig{
		McpServers: map[string]*config.McpServerCfg{
			"server1": {
				Name:    "Server 1",
				Type:    "stdio",
				Command: "test-cmd",
				Enabled: true,
			},
			"server2": {
				Name:    "Server 2",
				Type:    "http",
				URL:     "http://localhost:8080",
				Enabled: false,
			},
		},
	}

	enabled := cfg.GetEnabledMcpServers()
	if len(enabled) != 1 {
		return fmt.Errorf("启用的MCP服务器数量不正确: %d", len(enabled))
	}
	if enabled[0].Name != "Server 1" {
		return fmt.Errorf("启用的服务器名称不正确")
	}

	return nil
}

func testConfigDir() error {
	dir, err := config.ConfigDir()
	if err != nil {
		return fmt.Errorf("获取配置目录失败: %w", err)
	}
	if dir == "" {
		return fmt.Errorf("配置目录为空")
	}

	agentsDir, err := config.UserAgentsDir()
	if err != nil {
		return fmt.Errorf("获取用户代理目录失败: %w", err)
	}
	if agentsDir == "" {
		return fmt.Errorf("用户代理目录为空")
	}

	return nil
}

func testConfigConcurrent() error {
	var wg sync.WaitGroup
	errCh := make(chan error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg := config.DefaultAppConfig()
			if cfg == nil {
				errCh <- fmt.Errorf("并发读取配置失败")
				return
			}
			registry := config.NewAgentRegistry()
			if registry == nil {
				errCh <- fmt.Errorf("并发创建注册表失败")
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		return err
	}
	return nil
}
