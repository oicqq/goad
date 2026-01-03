// Package main 是Goad的入口点
package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/anthropics/goad/internal/agent"
	"github.com/anthropics/goad/internal/config"
	"github.com/anthropics/goad/internal/tui"
)

//go:embed data/agents/*.toml
var agentsFS embed.FS

// 版本信息
var (
	Version = "0.1.0"
	Commit  = "dev"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "goad [path]",
	Short: "Goad - AI代理终端",
	Long: `Goad 是一个统一的AI代理终端界面，支持多种AI编程助手。

支持的代理:
  - Claude Code (Anthropic)
  - Codex CLI (OpenAI)
  - Gemini CLI (Google)
  - OpenHands
  - 更多...

使用示例:
  goad                    # 在当前目录启动默认代理
  goad -a claude          # 使用Claude Code
  goad -a codex ./myproject  # 在指定目录使用Codex`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMain,
}

var (
	agentFlag   string
	configFlag  string
	versionFlag bool
)

func init() {
	rootCmd.Flags().StringVarP(&agentFlag, "agent", "a", "", "指定代理 (claude, codex, gemini)")
	rootCmd.Flags().StringVarP(&configFlag, "config", "c", "", "配置文件路径")
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "显示版本信息")

	// 添加子命令
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(listCmd)
}

func runMain(cmd *cobra.Command, args []string) error {
	if versionFlag {
		fmt.Printf("Goad %s (%s)\n", Version, Commit)
		return nil
	}

	// 确定项目目录
	projectDir := "."
	if len(args) > 0 {
		projectDir = args[0]
	}

	absPath, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("无法解析路径: %w", err)
	}

	// 加载应用配置
	appConfig, err := config.LoadAppConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 加载代理注册表
	registry := config.NewAgentRegistry()
	if err := registry.LoadEmbeddedAgents(agentsFS, "data/agents"); err != nil {
		return fmt.Errorf("加载代理配置失败: %w", err)
	}

	// 加载用户自定义代理（会覆盖同名的内置代理）
	if userAgentsDir, err := config.UserAgentsDir(); err == nil {
		// 忽略错误，用户目录可能不存在
		_ = registry.LoadFromDirectory(userAgentsDir)
	}

	// 确定要使用的代理
	agentName := agentFlag
	if agentName == "" {
		agentName = appConfig.DefaultAgent
	}
	if agentName == "" {
		agentName = "claude"
	}

	// 查找代理配置
	agentConfig, ok := registry.GetByShortName(agentName)
	if !ok {
		agentConfig, ok = registry.Get(agentName)
	}
	if !ok {
		return fmt.Errorf("未找到代理: %s", agentName)
	}

	// 创建代理实例
	ag := agent.NewAgent(agentConfig, appConfig.API, absPath)

	// 启动代理
	if err := ag.Start(); err != nil {
		return fmt.Errorf("启动代理失败: %w", err)
	}
	defer ag.Stop()

	// 创建TUI
	model := tui.New(ag, appConfig, agentConfig)

	// 运行TUI
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI错误: %w", err)
	}

	return nil
}

// 配置命令
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理配置",
	Long:  `查看和编辑Goad配置，包括API密钥和默认设置。`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前配置",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadAppConfig()
		if err != nil {
			return err
		}

		fmt.Println("当前配置:")
		fmt.Printf("  默认代理: %s\n", cfg.DefaultAgent)
		fmt.Printf("  主题: %s\n", cfg.Theme)
		fmt.Printf("  语言: %s\n", cfg.Language)

		if cfg.API != nil {
			fmt.Println("\nAPI配置:")
			if cfg.API.ClaudeAPIKey != "" {
				fmt.Println("  Claude API Key: ******")
			}
			if cfg.API.ClaudeBaseURL != "" {
				fmt.Printf("  Claude Base URL: %s\n", cfg.API.ClaudeBaseURL)
			}
			if cfg.API.OpenAIAPIKey != "" {
				fmt.Println("  OpenAI API Key: ******")
			}
			if cfg.API.OpenAIBaseURL != "" {
				fmt.Printf("  OpenAI Base URL: %s\n", cfg.API.OpenAIBaseURL)
			}
			if cfg.API.GeminiAPIKey != "" {
				fmt.Println("  Gemini API Key: ******")
			}
			if cfg.API.GeminiBaseURL != "" {
				fmt.Printf("  Gemini Base URL: %s\n", cfg.API.GeminiBaseURL)
			}
		}

		configDir, _ := config.ConfigDir()
		fmt.Printf("\n配置文件位置: %s/config.toml\n", configDir)

		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "设置配置项",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		cfg, err := config.LoadAppConfig()
		if err != nil {
			cfg = config.DefaultAppConfig()
		}

		if cfg.API == nil {
			cfg.API = &config.APIConfig{}
		}

		switch key {
		case "default-agent":
			cfg.DefaultAgent = value
		case "claude-api-key":
			cfg.API.ClaudeAPIKey = value
		case "claude-base-url":
			cfg.API.ClaudeBaseURL = value
		case "openai-api-key":
			cfg.API.OpenAIAPIKey = value
		case "openai-base-url":
			cfg.API.OpenAIBaseURL = value
		case "gemini-api-key":
			cfg.API.GeminiAPIKey = value
		case "gemini-base-url":
			cfg.API.GeminiBaseURL = value
		case "theme":
			cfg.Theme = value
		case "language":
			cfg.Language = value
		default:
			return fmt.Errorf("未知配置项: %s", key)
		}

		if err := config.SaveAppConfig(cfg); err != nil {
			return fmt.Errorf("保存配置失败: %w", err)
		}

		fmt.Printf("已设置 %s\n", key)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
}

// 列出代理命令
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出可用代理",
	RunE: func(cmd *cobra.Command, args []string) error {
		registry := config.NewAgentRegistry()
		if err := registry.LoadEmbeddedAgents(agentsFS, "data/agents"); err != nil {
			return fmt.Errorf("加载代理配置失败: %w", err)
		}

		// 加载用户自定义代理
		userAgentsDir, _ := config.UserAgentsDir()
		hasUserAgents := false
		if userAgentsDir != "" {
			if err := registry.LoadFromDirectory(userAgentsDir); err == nil {
				hasUserAgents = true
			}
		}

		fmt.Println("可用代理:")
		fmt.Println()

		for _, agent := range registry.List() {
			recommended := ""
			if agent.Recommended {
				recommended = " (推荐)"
			}
			fmt.Printf("  %s (%s)%s\n", agent.Name, agent.ShortName, recommended)
			fmt.Printf("    %s\n", agent.Description)
			fmt.Printf("    命令: %s\n", agent.GetRunCommand())
			fmt.Println()
		}

		// 显示用户自定义代理目录提示
		fmt.Println("---")
		if hasUserAgents {
			fmt.Printf("用户代理目录: %s\n", userAgentsDir)
		} else {
			fmt.Printf("提示: 可在 %s 添加自定义代理配置(.toml)\n", userAgentsDir)
		}

		return nil
	},
}
