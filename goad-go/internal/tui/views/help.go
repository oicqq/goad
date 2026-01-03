// Package views 实现了TUI的各个视图
package views

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anthropics/goad/internal/config"
	"github.com/anthropics/goad/internal/tui/markdown"
	"github.com/anthropics/goad/internal/tui/styles"
)

// HelpModel 帮助视图模型
type HelpModel struct {
	agentConfig *config.AgentConfig
	mdRenderer  *markdown.Renderer
	width       int
	height      int
	scrollY     int
	content     string
}

// NewHelpModel 创建帮助视图模型
func NewHelpModel(agentConfig *config.AgentConfig) HelpModel {
	m := HelpModel{
		agentConfig: agentConfig,
		mdRenderer:  markdown.New(80),
	}
	m.buildContent()
	return m
}

// buildContent 构建帮助内容
func (m *HelpModel) buildContent() {
	var b strings.Builder

	// 标题
	b.WriteString("# Goad 帮助\n\n")

	// 快捷键
	b.WriteString("## 快捷键\n\n")
	b.WriteString("| 按键 | 功能 |\n")
	b.WriteString("|------|------|\n")
	b.WriteString("| Ctrl+Enter | 发送消息 |\n")
	b.WriteString("| Ctrl+C / Esc | 取消/退出 |\n")
	b.WriteString("| Ctrl+L | 清屏 |\n")
	b.WriteString("| Ctrl+B | 切换侧边栏 |\n")
	b.WriteString("| PgUp / Ctrl+U | 向上翻页 |\n")
	b.WriteString("| PgDn / Ctrl+D | 向下翻页 |\n")
	b.WriteString("| Home | 滚动到顶部 |\n")
	b.WriteString("| End | 滚动到底部 |\n")
	b.WriteString("| F1 | 帮助 |\n")
	b.WriteString("| F2 | 设置 |\n")
	b.WriteString("\n")

	// 命令
	b.WriteString("## 命令行\n\n")
	b.WriteString("```\n")
	b.WriteString("goad [path]           # 在指定目录启动\n")
	b.WriteString("goad -a <agent>       # 使用指定代理\n")
	b.WriteString("goad list             # 列出可用代理\n")
	b.WriteString("goad config show      # 显示配置\n")
	b.WriteString("goad config set <k> <v>  # 设置配置\n")
	b.WriteString("goad history          # 查看会话历史\n")
	b.WriteString("```\n\n")

	// 代理信息
	if m.agentConfig != nil {
		b.WriteString("## 当前代理\n\n")
		b.WriteString("**" + m.agentConfig.Name + "**\n\n")
		b.WriteString(m.agentConfig.Description + "\n\n")

		if m.agentConfig.Help != "" {
			b.WriteString("### 代理帮助\n\n")
			b.WriteString(m.agentConfig.Help)
			b.WriteString("\n")
		}
	}

	// 关于
	b.WriteString("## 关于\n\n")
	b.WriteString("Goad 是一个统一的AI代理终端界面，支持多种AI编程助手。\n\n")
	b.WriteString("- 项目主页: https://github.com/anthropics/goad\n")
	b.WriteString("- 问题反馈: https://github.com/anthropics/goad/issues\n")

	m.content = b.String()
}

// Init 初始化
func (m HelpModel) Init() tea.Cmd {
	return nil
}

// Update 更新
func (m HelpModel) Update(msg tea.Msg) (HelpModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.scrollY > 0 {
				m.scrollY--
			}
		case "down", "j":
			m.scrollY++
		case "pgup":
			m.scrollY -= 10
			if m.scrollY < 0 {
				m.scrollY = 0
			}
		case "pgdown":
			m.scrollY += 10
		case "home":
			m.scrollY = 0
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.mdRenderer != nil {
			m.mdRenderer.SetWidth(msg.Width - 4)
		}
	}

	return m, nil
}

// View 渲染
func (m HelpModel) View() string {
	var content string
	if m.mdRenderer != nil {
		content = m.mdRenderer.Render(m.content)
	} else {
		content = m.content
	}

	// 处理滚动
	lines := strings.Split(content, "\n")
	if m.scrollY >= len(lines) {
		m.scrollY = len(lines) - 1
	}
	if m.scrollY < 0 {
		m.scrollY = 0
	}

	visibleLines := m.height - 4
	if visibleLines < 1 {
		visibleLines = 20
	}

	end := m.scrollY + visibleLines
	if end > len(lines) {
		end = len(lines)
	}

	visible := strings.Join(lines[m.scrollY:end], "\n")

	// 底部帮助
	help := styles.HelpStyle.Render("↑↓: 滚动 | PgUp/PgDn: 翻页 | Esc: 返回")

	return visible + "\n\n" + help
}

// SetAgentConfig 设置代理配置
func (m *HelpModel) SetAgentConfig(cfg *config.AgentConfig) {
	m.agentConfig = cfg
	m.buildContent()
}
