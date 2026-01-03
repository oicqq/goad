// Package tui 实现了终端用户界面
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/goad/internal/acp"
	"github.com/anthropics/goad/internal/agent"
	"github.com/anthropics/goad/internal/config"
	"github.com/anthropics/goad/internal/tui/styles"
)

// 视图状态
type viewState int

const (
	viewConversation viewState = iota
	viewSettings
	viewHelp
)

// Model 是TUI的主模型
type Model struct {
	// 代理
	agent      *agent.Agent
	agentReady bool

	// 配置
	appConfig   *config.AppConfig
	agentConfig *config.AgentConfig

	// UI状态
	width       int
	height      int
	view        viewState
	ready       bool
	quitting    bool
	showSidebar bool

	// 对话组件
	viewport viewport.Model
	textarea textarea.Model
	messages []ChatMessage

	// 当前状态
	thinking    bool
	statusLine  string
	currentMode string
	modes       map[string]*acp.SessionMode

	// 权限请求
	permissionRequest *acp.PermissionRequestMessage
	selectedOption    int

	// 工具调用
	toolCalls map[string]*acp.ToolCall

	// 计划
	planEntries []acp.PlanEntry

	// 错误信息
	err error
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string // "user", "agent", "system", "thinking"
	Content string
}

// AgentMessageMsg 代理消息
type AgentMessageMsg struct {
	Message acp.Message
}

// ErrorMsg 错误消息
type ErrorMsg struct {
	Err error
}

// PromptSentMsg 提示已发送
type PromptSentMsg struct {
	StopReason string
	Err        error
}

// New 创建新的TUI模型
func New(ag *agent.Agent, appConfig *config.AppConfig, agentConfig *config.AgentConfig) Model {
	ta := textarea.New()
	ta.Placeholder = "输入消息... (Ctrl+Enter 发送, Ctrl+C 退出)"
	ta.Focus()
	ta.CharLimit = 0
	ta.SetWidth(80)
	ta.SetHeight(3)

	vp := viewport.New(80, 20)
	vp.SetContent("")

	return Model{
		agent:       ag,
		appConfig:   appConfig,
		agentConfig: agentConfig,
		textarea:    ta,
		viewport:    vp,
		messages:    []ChatMessage{},
		toolCalls:   make(map[string]*acp.ToolCall),
		modes:       make(map[string]*acp.SessionMode),
		showSidebar: true,
	}
}

// Init 初始化
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.listenForAgentMessages(),
	)
}

// listenForAgentMessages 监听代理消息
func (m Model) listenForAgentMessages() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.agent.Messages()
		if !ok {
			return nil
		}
		return AgentMessageMsg{Message: msg}
	}
}

// Update 更新模型
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// 更新组件大小
		headerHeight := 1
		footerHeight := 1
		inputHeight := 5
		contentHeight := m.height - headerHeight - footerHeight - inputHeight

		sidebarWidth := 0
		if m.showSidebar {
			sidebarWidth = 30
		}
		contentWidth := m.width - sidebarWidth - 2

		m.viewport.Width = contentWidth
		m.viewport.Height = contentHeight
		m.textarea.SetWidth(contentWidth - 2)

		m.updateViewportContent()

	case AgentMessageMsg:
		m.handleAgentMessage(msg.Message)
		m.updateViewportContent()
		cmds = append(cmds, m.listenForAgentMessages())

	case PromptSentMsg:
		m.thinking = false
		if msg.Err != nil {
			m.err = msg.Err
		}

	case ErrorMsg:
		m.err = msg.Err
	}

	// 更新子组件
	if m.permissionRequest == nil {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleKeyMsg 处理按键消息
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 处理权限请求
	if m.permissionRequest != nil {
		return m.handlePermissionKey(msg)
	}

	switch msg.String() {
	case "ctrl+c", "esc":
		if m.thinking {
			m.agent.Cancel()
			m.thinking = false
			return m, nil
		}
		m.quitting = true
		m.agent.Stop()
		return m, tea.Quit

	case "ctrl+enter", "ctrl+s":
		// 发送消息
		text := strings.TrimSpace(m.textarea.Value())
		if text == "" || m.thinking {
			return m, nil
		}

		m.messages = append(m.messages, ChatMessage{
			Role:    "user",
			Content: text,
		})
		m.textarea.Reset()
		m.thinking = true
		m.updateViewportContent()
		m.viewport.GotoBottom()

		return m, m.sendPrompt(text)

	case "ctrl+l":
		// 清屏
		m.messages = []ChatMessage{}
		m.toolCalls = make(map[string]*acp.ToolCall)
		m.planEntries = nil
		m.updateViewportContent()

	case "ctrl+b":
		// 切换侧边栏
		m.showSidebar = !m.showSidebar

	case "f1":
		// 帮助
		if m.view == viewHelp {
			m.view = viewConversation
		} else {
			m.view = viewHelp
		}

	case "f2":
		// 设置
		if m.view == viewSettings {
			m.view = viewConversation
		} else {
			m.view = viewSettings
		}
	}

	return m, nil
}

// handlePermissionKey 处理权限请求按键
func (m Model) handlePermissionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedOption > 0 {
			m.selectedOption--
		}
	case "down", "j":
		if m.selectedOption < len(m.permissionRequest.Options)-1 {
			m.selectedOption++
		}
	case "enter":
		// 发送响应
		option := m.permissionRequest.Options[m.selectedOption]
		m.permissionRequest.ResponseCh <- acp.PermissionResponse{
			OptionID: option.OptionID,
			Outcome:  "selected",
		}
		m.permissionRequest = nil
		m.selectedOption = 0
	case "esc":
		// 取消
		m.permissionRequest.ResponseCh <- acp.PermissionResponse{
			Outcome: "cancelled",
		}
		m.permissionRequest = nil
		m.selectedOption = 0
	}

	return m, nil
}

// handleAgentMessage 处理代理消息
func (m *Model) handleAgentMessage(msg acp.Message) {
	switch msg := msg.(type) {
	case *acp.AgentReadyMessage:
		m.agentReady = true
		m.messages = append(m.messages, ChatMessage{
			Role:    "system",
			Content: fmt.Sprintf("已连接到 %s", m.agentConfig.Name),
		})

	case *acp.AgentFailMessage:
		m.err = fmt.Errorf("%s: %s", msg.Reason, msg.Details)

	case *acp.UpdateMessage:
		// 累积代理消息
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "agent" {
			m.messages[len(m.messages)-1].Content += msg.Text
		} else {
			m.messages = append(m.messages, ChatMessage{
				Role:    "agent",
				Content: msg.Text,
			})
		}

	case *acp.ThinkingMessage:
		// 累积思考消息
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "thinking" {
			m.messages[len(m.messages)-1].Content += msg.Text
		} else {
			m.messages = append(m.messages, ChatMessage{
				Role:    "thinking",
				Content: msg.Text,
			})
		}

	case *acp.ToolCallMessage:
		m.toolCalls[msg.ToolCall.ToolCallID] = msg.ToolCall

	case *acp.ToolCallUpdateMessage:
		if msg.ToolCall != nil {
			m.toolCalls[msg.ToolCall.ToolCallID] = msg.ToolCall
		}

	case *acp.PlanMessage:
		m.planEntries = msg.Entries

	case *acp.PermissionRequestMessage:
		m.permissionRequest = msg

	case *acp.ModeUpdateMessage:
		m.currentMode = msg.CurrentModeID

	case *acp.StatusLineMessage:
		m.statusLine = msg.Text
	}
}

// sendPrompt 发送提示
func (m Model) sendPrompt(text string) tea.Cmd {
	return func() tea.Msg {
		stopReason, err := m.agent.SendPrompt(text)
		return PromptSentMsg{StopReason: stopReason, Err: err}
	}
}

// updateViewportContent 更新视口内容
func (m *Model) updateViewportContent() {
	var content strings.Builder

	// 渲染消息
	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			content.WriteString(styles.UserMessageStyle.Render("你: " + msg.Content))
			content.WriteString("\n\n")
		case "agent":
			content.WriteString(styles.AgentMessageStyle.Render(msg.Content))
			content.WriteString("\n")
		case "thinking":
			content.WriteString(styles.ThinkingStyle.Render("思考: " + msg.Content))
			content.WriteString("\n")
		case "system":
			content.WriteString(styles.SystemMessageStyle.Render("系统: " + msg.Content))
			content.WriteString("\n\n")
		}
	}

	// 渲染工具调用
	for _, tc := range m.toolCalls {
		content.WriteString(m.renderToolCall(tc))
		content.WriteString("\n")
	}

	// 渲染计划
	if len(m.planEntries) > 0 {
		content.WriteString(m.renderPlan())
		content.WriteString("\n")
	}

	// 渲染权限请求
	if m.permissionRequest != nil {
		content.WriteString(m.renderPermissionRequest())
	}

	// 思考指示器
	if m.thinking {
		content.WriteString(styles.ThinkingStyle.Render("思考中..."))
	}

	m.viewport.SetContent(content.String())
}

// renderToolCall 渲染工具调用
func (m Model) renderToolCall(tc *acp.ToolCall) string {
	var b strings.Builder

	statusStyle := styles.GetToolCallStatusStyle(string(tc.Status))
	statusText := styles.GetToolCallStatusText(string(tc.Status))
	icon := styles.GetToolKindIcon(string(tc.Kind))

	title := fmt.Sprintf("%s %s [%s]", icon, tc.Title, statusText)
	b.WriteString(styles.ToolCallTitleStyle.Render(title))
	b.WriteString("\n")

	// 渲染内容
	for _, c := range tc.Content {
		if t, ok := c["type"].(string); ok {
			switch t {
			case "diff":
				b.WriteString(m.renderDiff(c))
			case "terminal":
				if tid, ok := c["terminalId"].(string); ok {
					b.WriteString(styles.DiffPathStyle.Render(fmt.Sprintf("终端: %s", tid)))
				}
			case "content":
				if content, ok := c["content"].(map[string]interface{}); ok {
					if text, ok := content["text"].(string); ok {
						b.WriteString(text)
					}
				}
			}
		}
	}

	return styles.ToolCallStyle.Copy().BorderForeground(statusStyle.GetForeground()).Render(b.String())
}

// renderDiff 渲染差异
func (m Model) renderDiff(content acp.ToolCallContent) string {
	var b strings.Builder

	path, _ := content["path"].(string)
	oldText, _ := content["oldText"].(string)
	newText, _ := content["newText"].(string)

	b.WriteString(styles.DiffPathStyle.Render(path))
	b.WriteString("\n")

	// 简单的差异显示
	if oldText != "" {
		oldLines := strings.Split(oldText, "\n")
		for _, line := range oldLines {
			b.WriteString(styles.DiffRemoveStyle.Render("- " + line))
			b.WriteString("\n")
		}
	}

	if newText != "" {
		newLines := strings.Split(newText, "\n")
		for _, line := range newLines {
			b.WriteString(styles.DiffAddStyle.Render("+ " + line))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderPlan 渲染计划
func (m Model) renderPlan() string {
	var b strings.Builder

	b.WriteString(styles.PlanTitleStyle.Render("计划"))
	b.WriteString("\n")

	for _, entry := range m.planEntries {
		var icon string
		switch entry.Status {
		case "pending":
			icon = styles.PlanEntryPending.String()
		case "in_progress":
			icon = styles.PlanEntryInProgress.String()
		case "completed":
			icon = styles.PlanEntryCompleted.String()
		default:
			icon = "○"
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", icon, entry.Content))
	}

	return styles.PlanStyle.Render(b.String())
}

// renderPermissionRequest 渲染权限请求
func (m Model) renderPermissionRequest() string {
	var b strings.Builder

	b.WriteString(styles.PermissionTitleStyle.Render("需要权限"))
	b.WriteString("\n\n")

	if m.permissionRequest.ToolCall != nil {
		b.WriteString(fmt.Sprintf("操作: %s\n\n", m.permissionRequest.ToolCall.Title))
	}

	for i, opt := range m.permissionRequest.Options {
		cursor := "  "
		if i == m.selectedOption {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, opt.Name))
	}

	b.WriteString("\n使用 ↑↓ 选择, Enter 确认, Esc 取消")

	return styles.PermissionStyle.Render(b.String())
}

// View 渲染视图
func (m Model) View() string {
	if !m.ready {
		return "加载中..."
	}

	if m.quitting {
		return "再见!\n"
	}

	var b strings.Builder

	// 头部
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	// 主内容
	mainContent := m.viewport.View()

	// 侧边栏
	if m.showSidebar {
		sidebar := m.renderSidebar()
		combined := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainContent)
		b.WriteString(combined)
	} else {
		b.WriteString(mainContent)
	}

	b.WriteString("\n")

	// 输入框
	b.WriteString(styles.InputStyle.Render(m.textarea.View()))
	b.WriteString("\n")

	// 状态栏
	footer := m.renderFooter()
	b.WriteString(footer)

	return b.String()
}

// renderHeader 渲染头部
func (m Model) renderHeader() string {
	title := fmt.Sprintf(" Goad - %s ", m.agentConfig.Name)
	left := styles.HeaderStyle.Render(title)

	modeText := ""
	if m.currentMode != "" {
		if mode, ok := m.modes[m.currentMode]; ok {
			modeText = fmt.Sprintf(" [%s] ", mode.Name)
		} else {
			modeText = fmt.Sprintf(" [%s] ", m.currentMode)
		}
	}

	right := styles.HeaderStyle.Render(modeText)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return left + strings.Repeat(" ", gap) + right
}

// renderSidebar 渲染侧边栏
func (m Model) renderSidebar() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("项目"))
	b.WriteString("\n")
	b.WriteString(m.agent.ProjectRoot())
	b.WriteString("\n\n")

	if len(m.planEntries) > 0 {
		b.WriteString(styles.HeaderStyle.Render("计划"))
		b.WriteString("\n")
		for _, entry := range m.planEntries {
			var icon string
			switch entry.Status {
			case "completed":
				icon = "✓"
			case "in_progress":
				icon = "●"
			default:
				icon = "○"
			}
			b.WriteString(fmt.Sprintf("%s %s\n", icon, truncate(entry.Content, 25)))
		}
	}

	return styles.SidebarStyle.Width(28).Render(b.String())
}

// renderFooter 渲染底部
func (m Model) renderFooter() string {
	status := "就绪"
	if m.thinking {
		status = "思考中..."
	}
	if m.statusLine != "" {
		status = m.statusLine
	}
	if m.err != nil {
		status = fmt.Sprintf("错误: %v", m.err)
	}

	left := styles.StatusBarStyle.Render(status)

	help := "Ctrl+Enter: 发送 | Ctrl+C: 退出 | F1: 帮助 | F2: 设置"
	right := styles.HelpStyle.Render(help)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return left + strings.Repeat(" ", gap) + right
}

// truncate 截断字符串
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
