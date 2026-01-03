// Package views 实现了TUI的各个视图
package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/goad/internal/config"
	"github.com/anthropics/goad/internal/tui/styles"
)

// 输入框索引
const (
	inputClaudeAPIKey = iota
	inputClaudeBaseURL
	inputOpenAIAPIKey
	inputOpenAIBaseURL
	inputGeminiAPIKey
	inputGeminiBaseURL
)

// SettingsModel 设置视图模型
type SettingsModel struct {
	config    *config.AppConfig
	inputs    []textinput.Model
	focused   int
	width     int
	height    int
	saved     bool
	err       error
}

// NewSettingsModel 创建设置视图模型
func NewSettingsModel(cfg *config.AppConfig) SettingsModel {
	inputs := make([]textinput.Model, 6)

	// Claude API Key
	inputs[inputClaudeAPIKey] = textinput.New()
	inputs[inputClaudeAPIKey].Placeholder = "sk-ant-..."
	inputs[inputClaudeAPIKey].EchoMode = textinput.EchoPassword
	inputs[inputClaudeAPIKey].EchoCharacter = '•'
	if cfg.API != nil && cfg.API.ClaudeAPIKey != "" {
		inputs[inputClaudeAPIKey].SetValue(cfg.API.ClaudeAPIKey)
	}

	// Claude Base URL
	inputs[inputClaudeBaseURL] = textinput.New()
	inputs[inputClaudeBaseURL].Placeholder = "https://api.anthropic.com"
	if cfg.API != nil && cfg.API.ClaudeBaseURL != "" {
		inputs[inputClaudeBaseURL].SetValue(cfg.API.ClaudeBaseURL)
	}

	// OpenAI API Key
	inputs[inputOpenAIAPIKey] = textinput.New()
	inputs[inputOpenAIAPIKey].Placeholder = "sk-..."
	inputs[inputOpenAIAPIKey].EchoMode = textinput.EchoPassword
	inputs[inputOpenAIAPIKey].EchoCharacter = '•'
	if cfg.API != nil && cfg.API.OpenAIAPIKey != "" {
		inputs[inputOpenAIAPIKey].SetValue(cfg.API.OpenAIAPIKey)
	}

	// OpenAI Base URL
	inputs[inputOpenAIBaseURL] = textinput.New()
	inputs[inputOpenAIBaseURL].Placeholder = "https://api.openai.com/v1"
	if cfg.API != nil && cfg.API.OpenAIBaseURL != "" {
		inputs[inputOpenAIBaseURL].SetValue(cfg.API.OpenAIBaseURL)
	}

	// Gemini API Key
	inputs[inputGeminiAPIKey] = textinput.New()
	inputs[inputGeminiAPIKey].Placeholder = "AIza..."
	inputs[inputGeminiAPIKey].EchoMode = textinput.EchoPassword
	inputs[inputGeminiAPIKey].EchoCharacter = '•'
	if cfg.API != nil && cfg.API.GeminiAPIKey != "" {
		inputs[inputGeminiAPIKey].SetValue(cfg.API.GeminiAPIKey)
	}

	// Gemini Base URL
	inputs[inputGeminiBaseURL] = textinput.New()
	inputs[inputGeminiBaseURL].Placeholder = "https://generativelanguage.googleapis.com"
	if cfg.API != nil && cfg.API.GeminiBaseURL != "" {
		inputs[inputGeminiBaseURL].SetValue(cfg.API.GeminiBaseURL)
	}

	// 设置宽度
	for i := range inputs {
		inputs[i].Width = 50
	}

	// 聚焦第一个
	inputs[0].Focus()

	return SettingsModel{
		config: cfg,
		inputs: inputs,
	}
}

// Init 初始化
func (m SettingsModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update 更新
func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused + 1) % len(m.inputs)
			m.inputs[m.focused].Focus()

		case "shift+tab", "up":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused - 1 + len(m.inputs)) % len(m.inputs)
			m.inputs[m.focused].Focus()

		case "ctrl+s":
			// 保存配置
			if m.config.API == nil {
				m.config.API = &config.APIConfig{}
			}
			m.config.API.ClaudeAPIKey = m.inputs[inputClaudeAPIKey].Value()
			m.config.API.ClaudeBaseURL = m.inputs[inputClaudeBaseURL].Value()
			m.config.API.OpenAIAPIKey = m.inputs[inputOpenAIAPIKey].Value()
			m.config.API.OpenAIBaseURL = m.inputs[inputOpenAIBaseURL].Value()
			m.config.API.GeminiAPIKey = m.inputs[inputGeminiAPIKey].Value()
			m.config.API.GeminiBaseURL = m.inputs[inputGeminiBaseURL].Value()

			if err := config.SaveAppConfig(m.config); err != nil {
				m.err = err
			} else {
				m.saved = true
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// 更新输入框
	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View 渲染
func (m SettingsModel) View() string {
	var b strings.Builder

	title := styles.HeaderStyle.Render(" API 设置 ")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Claude 设置
	b.WriteString(styles.ToolCallTitleStyle.Render("Claude Code"))
	b.WriteString("\n")
	b.WriteString(m.renderInput("API Key:", m.inputs[inputClaudeAPIKey], m.focused == inputClaudeAPIKey))
	b.WriteString(m.renderInput("Base URL:", m.inputs[inputClaudeBaseURL], m.focused == inputClaudeBaseURL))
	b.WriteString("\n")

	// OpenAI 设置
	b.WriteString(styles.ToolCallTitleStyle.Render("Codex CLI (OpenAI)"))
	b.WriteString("\n")
	b.WriteString(m.renderInput("API Key:", m.inputs[inputOpenAIAPIKey], m.focused == inputOpenAIAPIKey))
	b.WriteString(m.renderInput("Base URL:", m.inputs[inputOpenAIBaseURL], m.focused == inputOpenAIBaseURL))
	b.WriteString("\n")

	// Gemini 设置
	b.WriteString(styles.ToolCallTitleStyle.Render("Gemini CLI"))
	b.WriteString("\n")
	b.WriteString(m.renderInput("API Key:", m.inputs[inputGeminiAPIKey], m.focused == inputGeminiAPIKey))
	b.WriteString(m.renderInput("Base URL:", m.inputs[inputGeminiBaseURL], m.focused == inputGeminiBaseURL))
	b.WriteString("\n")

	// 状态
	if m.saved {
		b.WriteString(styles.ButtonSuccessStyle.Render(" 已保存 "))
		b.WriteString("\n")
	}
	if m.err != nil {
		b.WriteString(styles.ButtonDangerStyle.Render(fmt.Sprintf(" 错误: %v ", m.err)))
		b.WriteString("\n")
	}

	// 帮助
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("Tab/↑↓: 切换 | Ctrl+S: 保存 | Esc: 返回"))

	return styles.ContentStyle.Render(b.String())
}

// renderInput 渲染输入框
func (m SettingsModel) renderInput(label string, input textinput.Model, focused bool) string {
	labelStyle := lipgloss.NewStyle().Width(12).Foreground(styles.TextSecondary)

	inputView := input.View()
	if focused {
		inputView = styles.InputStyle.Render(inputView)
	}

	return fmt.Sprintf("%s %s\n", labelStyle.Render(label), inputView)
}

// Config 返回更新后的配置
func (m SettingsModel) Config() *config.AppConfig {
	return m.config
}
