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
	"github.com/anthropics/goad/internal/tui/components"
	"github.com/anthropics/goad/internal/tui/diff"
	"github.com/anthropics/goad/internal/tui/highlight"
	"github.com/anthropics/goad/internal/tui/markdown"
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

	// 滚动状态
	autoScroll bool // 是否自动滚动到底部

	// 语法高亮
	highlighter *highlight.Highlighter

	// 差异渲染
	diffRenderer *diff.Renderer

	// Markdown渲染
	mdRenderer *markdown.Renderer

	// 当前状态
	thinking    bool
	statusLine  string
	currentMode string
	modes       map[string]*acp.SessionMode

	// 增强组件 (P0/P1)
	permissionDialog *components.PermissionDialog  // 权限对话框
	toolCallPanel    *components.ToolCallPanel     // 工具调用面板
	thinkingDisplay  *components.ThinkingDisplay   // 思考过程显示
	fuzzyFinder      *components.FuzzyFinder       // 模糊搜索
	sessionPicker    *components.SessionPicker     // 会话选择器

	// 增强组件 (P2)
	autoComplete *components.AutoComplete // 输入框自动补全
	tabBar       *components.TabBar       // 多标签会话
	agentEditor  *components.AgentEditor  // 代理参数编辑器

	// 旧的权限请求（保持兼容）
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

	// 初始化会话选择器
	sessionPicker := components.NewSessionPicker()
	sessionPicker.Initialize() // 预加载会话列表

	// 初始化自动补全
	autoComplete := components.NewAutoComplete()
	autoComplete.SetProjectRoot(ag.ProjectRoot())
	// 添加斜杠命令
	autoComplete.AddCommand("/help", "显示帮助")
	autoComplete.AddCommand("/clear", "清除对话")
	autoComplete.AddCommand("/exit", "退出程序")
	autoComplete.AddCommand("/settings", "打开设置")
	autoComplete.AddCommand("/model", "切换模型")
	autoComplete.AddCommand("/mode", "切换模式")
	autoComplete.AddCommand("/export", "导出会话")
	autoComplete.AddCommand("/history", "查看历史")
	autoComplete.AddCommand("/cancel", "取消操作")
	autoComplete.AddCommand("/compact", "压缩对话")

	// 初始化标签栏
	tabBar := components.NewTabBar()
	tabBar.AddTab("main", agentConfig.Name, agentConfig.ShortName)
	tabBar.SetActiveIndex(0)

	// 初始化代理编辑器
	agentEditor := components.NewAgentEditor()
	agentEditor.LoadConfig(agentConfig)

	return Model{
		agent:            ag,
		appConfig:        appConfig,
		agentConfig:      agentConfig,
		textarea:         ta,
		viewport:         vp,
		messages:         []ChatMessage{},
		toolCalls:        make(map[string]*acp.ToolCall),
		modes:            make(map[string]*acp.SessionMode),
		showSidebar:      true,
		autoScroll:       true,
		highlighter:      highlight.New(),
		diffRenderer:     diff.New(),
		mdRenderer:       markdown.New(80),
		// 初始化增强组件 (P0/P1)
		permissionDialog: components.NewPermissionDialog(),
		toolCallPanel:    components.NewToolCallPanel(),
		thinkingDisplay:  components.NewThinkingDisplay(),
		fuzzyFinder:      components.NewFuzzyFinder(),
		sessionPicker:    sessionPicker,
		// 初始化增强组件 (P2)
		autoComplete: autoComplete,
		tabBar:       tabBar,
		agentEditor:  agentEditor,
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

		// 更新Markdown渲染器宽度
		if m.mdRenderer != nil {
			m.mdRenderer.SetWidth(contentWidth - 4)
		}

		m.updateViewportContent()

	case AgentMessageMsg:
		m.handleAgentMessage(msg.Message)
		m.updateViewportContent()
		// 只有在自动滚动模式下才滚动到底部
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
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
	// 处理代理编辑器
	if m.agentEditor.IsActive() {
		return m.handleAgentEditorKey(msg)
	}

	// 处理自动补全
	if m.autoComplete.IsActive() {
		return m.handleAutoCompleteKey(msg)
	}

	// 处理模糊搜索
	if m.fuzzyFinder.IsActive() {
		return m.handleFuzzyFinderKey(msg)
	}

	// 处理会话选择器
	if m.sessionPicker.IsActive() {
		return m.handleSessionPickerKey(msg)
	}

	// 处理权限请求（新组件优先）
	if m.permissionDialog.IsActive() {
		return m.handlePermissionDialogKey(msg)
	}

	// 处理权限请求（旧方式兼容）
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
		m.autoScroll = true // 发送消息时恢复自动滚动
		m.updateViewportContent()
		m.viewport.GotoBottom()

		return m, m.sendPrompt(text)

	case "ctrl+l":
		// 清屏
		m.messages = []ChatMessage{}
		m.toolCalls = make(map[string]*acp.ToolCall)
		m.toolCallPanel.Clear()
		m.thinkingDisplay.Clear()
		m.planEntries = nil
		m.autoScroll = true
		m.updateViewportContent()

	case "ctrl+b":
		// 切换侧边栏
		m.showSidebar = !m.showSidebar

	case "ctrl+p":
		// 打开文件模糊搜索
		m.fuzzyFinder.LoadFilesFromDir(m.agent.ProjectRoot(), 5)
		m.fuzzyFinder.SetSize(m.width-10, m.height-10)
		m.fuzzyFinder.Activate()
		return m, nil

	case "ctrl+r":
		// 打开会话恢复
		m.sessionPicker.LoadForAgent(m.agentConfig.ShortName)
		m.sessionPicker.SetSize(m.width-10, m.height-10)
		m.sessionPicker.Activate()
		return m, nil

	case "ctrl+t":
		// 切换思考过程折叠
		m.thinkingDisplay.ToggleAll()
		m.updateViewportContent()
		return m, nil

	// 滚动控制
	case "pgup", "ctrl+u":
		// 向上翻页
		m.autoScroll = false
		m.viewport.HalfViewUp()
		return m, nil

	case "pgdown", "ctrl+d":
		// 向下翻页
		m.viewport.HalfViewDown()
		// 如果滚动到底部，恢复自动滚动
		if m.viewport.AtBottom() {
			m.autoScroll = true
		}
		return m, nil

	case "home", "ctrl+home":
		// 滚动到顶部
		m.autoScroll = false
		m.viewport.GotoTop()
		return m, nil

	case "end", "ctrl+end":
		// 滚动到底部
		m.autoScroll = true
		m.viewport.GotoBottom()
		return m, nil

	case "up":
		// 如果输入框为空，向上滚动；否则在输入框中移动
		if m.textarea.Value() == "" {
			m.autoScroll = false
			m.viewport.LineUp(1)
			return m, nil
		}

	case "down":
		// 如果输入框为空，向下滚动；否则在输入框中移动
		if m.textarea.Value() == "" {
			m.viewport.LineDown(1)
			if m.viewport.AtBottom() {
				m.autoScroll = true
			}
			return m, nil
		}

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

	case "f3":
		// 代理编辑器
		m.agentEditor.SetSize(m.width-10, m.height-10)
		m.agentEditor.Activate()
		return m, nil

	case "ctrl+n":
		// 新建标签（暂不实现多代理）
		// m.tabBar.NewTab()
		return m, nil

	case "ctrl+w":
		// 关闭当前标签
		m.tabBar.CloseActiveTab()
		return m, nil

	case "ctrl+tab":
		// 下一个标签
		m.tabBar.NextTab()
		return m, nil

	case "ctrl+shift+tab":
		// 上一个标签
		m.tabBar.PrevTab()
		return m, nil

	case "tab":
		// Tab键触发自动补全
		text := m.textarea.Value()
		if text != "" {
			m.autoComplete.Update(text)
			if m.autoComplete.IsActive() {
				return m, nil
			}
		}
	}

	return m, nil
}

// handleAutoCompleteKey 处理自动补全按键
func (m Model) handleAutoCompleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.autoComplete.Deactivate()
	case "enter", "tab":
		if text := m.autoComplete.Accept(); text != "" {
			m.textarea.SetValue(text)
			// 添加到历史
			m.autoComplete.AddHistory(text)
		}
	case "up":
		m.autoComplete.MoveUp()
	case "down":
		m.autoComplete.MoveDown()
	default:
		// 更新补全
		m.autoComplete.Deactivate()
		// 让按键继续传递到textarea
		return m, nil
	}
	return m, nil
}

// handleAgentEditorKey 处理代理编辑器按键
func (m Model) handleAgentEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.agentEditor.IsEditMode() {
		switch msg.String() {
		case "enter":
			m.agentEditor.ExitEdit(true)
		case "esc":
			m.agentEditor.ExitEdit(false)
		default:
			m.agentEditor.HandleEditKey(msg.String())
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.agentEditor.Cancel()
	case "enter":
		m.agentEditor.EnterEdit()
	case "up", "k":
		m.agentEditor.MoveUp()
	case "down", "j":
		m.agentEditor.MoveDown()
	case "tab":
		m.agentEditor.CycleOption()
	case "ctrl+s":
		m.agentEditor.Save()
		m.agentEditor.Deactivate()
	}
	return m, nil
}

// handleFuzzyFinderKey 处理模糊搜索按键
func (m Model) handleFuzzyFinderKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.fuzzyFinder.Deactivate()
	case "enter":
		if item := m.fuzzyFinder.GetSelected(); item != nil {
			// 将选中的文件路径插入到输入框
			m.textarea.SetValue(m.textarea.Value() + item.Value)
			m.fuzzyFinder.Deactivate()
		}
	case "up":
		m.fuzzyFinder.MoveUp()
	case "down":
		m.fuzzyFinder.MoveDown()
	case "backspace":
		m.fuzzyFinder.DeleteChar()
	default:
		// 追加到搜索查询
		if len(msg.String()) == 1 {
			m.fuzzyFinder.AppendQuery(rune(msg.String()[0]))
		}
	}
	return m, nil
}

// handleSessionPickerKey 处理会话选择器按键
func (m Model) handleSessionPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.sessionPicker.Deactivate()
	case "enter":
		if sess := m.sessionPicker.GetSelected(); sess != nil {
			// 恢复会话到消息列表
			for _, msg := range sess.Messages {
				m.messages = append(m.messages, ChatMessage{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
			m.sessionPicker.Deactivate()
			m.updateViewportContent()
			m.viewport.GotoBottom()
		}
	case "up":
		m.sessionPicker.MoveUp()
	case "down":
		m.sessionPicker.MoveDown()
	case "delete":
		m.sessionPicker.DeleteSelected()
	case "backspace":
		m.sessionPicker.DeleteChar()
	default:
		if len(msg.String()) == 1 {
			m.sessionPicker.AppendQuery(rune(msg.String()[0]))
		}
	}
	return m, nil
}

// handlePermissionDialogKey 处理权限对话框按键
func (m Model) handlePermissionDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.permissionDialog.MoveUp()
	case "down", "j":
		m.permissionDialog.MoveDown()
	case "tab":
		m.permissionDialog.ToggleDetails()
	case "enter":
		if resp := m.permissionDialog.Confirm(); resp != nil {
			if m.permissionRequest != nil {
				m.permissionRequest.ResponseCh <- acp.PermissionResponse{
					OptionID: resp.OptionID,
					Outcome:  resp.Outcome,
				}
				m.permissionRequest = nil
			}
			m.permissionDialog.Clear()
		}
	case "esc":
		if m.permissionRequest != nil {
			m.permissionRequest.ResponseCh <- acp.PermissionResponse{
				Outcome: "cancelled",
			}
			m.permissionRequest = nil
		}
		m.permissionDialog.Clear()
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
		// 使用思考显示组件
		m.thinkingDisplay.AppendContent(msg.Text)
		// 同时保留旧方式（兼容）
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "thinking" {
			m.messages[len(m.messages)-1].Content += msg.Text
		} else {
			m.thinkingDisplay.StartBlock()
			m.messages = append(m.messages, ChatMessage{
				Role:    "thinking",
				Content: msg.Text,
			})
		}

	case *acp.ToolCallMessage:
		m.toolCalls[msg.ToolCall.ToolCallID] = msg.ToolCall
		m.toolCallPanel.Update(msg.ToolCall)

	case *acp.ToolCallUpdateMessage:
		if msg.ToolCall != nil {
			m.toolCalls[msg.ToolCall.ToolCallID] = msg.ToolCall
			m.toolCallPanel.Update(msg.ToolCall)
		} else if msg.Update != nil {
			// 合并更新到现有的工具调用
			if existing, ok := m.toolCalls[msg.Update.ToolCallID]; ok {
				if msg.Update.Title != "" {
					existing.Title = msg.Update.Title
				}
				if msg.Update.Status != "" {
					existing.Status = msg.Update.Status
				}
				m.toolCallPanel.Update(existing)
			}
		}

	case *acp.PlanMessage:
		m.planEntries = msg.Entries

	case *acp.PermissionRequestMessage:
		m.permissionRequest = msg
		// 同时设置到新组件
		m.permissionDialog.SetRequest(msg)
		m.permissionDialog.SetWidth(m.width - 20)

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
			// 对代理消息进行Markdown渲染
			renderedContent := msg.Content
			if m.mdRenderer != nil && markdown.HasMarkdown(msg.Content) {
				renderedContent = m.mdRenderer.Render(msg.Content)
			} else if m.highlighter != nil {
				// 如果没有Markdown，使用语法高亮处理代码块
				renderedContent = m.highlighter.HighlightInline(msg.Content)
			}
			content.WriteString(styles.AgentMessageStyle.Render(renderedContent))
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

	// 使用增强的差异渲染器
	if m.diffRenderer != nil && oldText != "" && newText != "" {
		diffOutput := m.diffRenderer.RenderUnifiedDiff(oldText, newText, path)
		b.WriteString(diffOutput)
	} else {
		// 回退到简单显示
		language := highlight.DetectLanguage(path)

		if oldText != "" {
			oldLines := strings.Split(oldText, "\n")
			for _, line := range oldLines {
				highlighted := line
				if language != "" && m.highlighter != nil {
					highlighted = m.highlighter.Highlight(line, language)
				}
				b.WriteString(styles.DiffRemoveStyle.Render("- " + highlighted))
				b.WriteString("\n")
			}
		}

		if newText != "" {
			newLines := strings.Split(newText, "\n")
			for _, line := range newLines {
				highlighted := line
				if language != "" && m.highlighter != nil {
					highlighted = m.highlighter.Highlight(line, language)
				}
				b.WriteString(styles.DiffAddStyle.Render("+ " + highlighted))
				b.WriteString("\n")
			}
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

	// 渲染模态框覆盖层
	baseView := b.String()

	// 模糊搜索覆盖层
	if m.fuzzyFinder.IsActive() {
		overlay := m.renderOverlay(m.fuzzyFinder.View())
		return overlay
	}

	// 会话选择器覆盖层
	if m.sessionPicker.IsActive() {
		overlay := m.renderOverlay(m.sessionPicker.View())
		return overlay
	}

	// 权限对话框覆盖层
	if m.permissionDialog.IsActive() {
		overlay := m.renderOverlay(m.permissionDialog.View())
		return overlay
	}

	// 代理编辑器覆盖层
	if m.agentEditor.IsActive() {
		overlay := m.renderOverlay(m.agentEditor.View())
		return overlay
	}

	// 自动补全浮动层（不是覆盖层，而是在输入框上方）
	if m.autoComplete.IsActive() {
		// 在baseView上附加自动补全
		return baseView + "\n" + m.autoComplete.View()
	}

	return baseView
}

// renderOverlay 渲染覆盖层
func (m Model) renderOverlay(content string) string {
	// 计算居中位置
	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	hPad := (m.width - contentWidth) / 2
	vPad := (m.height - contentHeight) / 3

	if hPad < 0 {
		hPad = 0
	}
	if vPad < 0 {
		vPad = 0
	}

	var result strings.Builder

	// 添加垂直填充
	for i := 0; i < vPad; i++ {
		result.WriteString("\n")
	}

	// 添加水平填充并渲染内容
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		result.WriteString(strings.Repeat(" ", hPad))
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}

// renderHeader 渲染头部
func (m Model) renderHeader() string {
	var header strings.Builder

	// 标签栏（如果有多个标签）
	if m.tabBar.GetTabCount() > 1 {
		m.tabBar.SetWidth(m.width)
		header.WriteString(m.tabBar.View())
		header.WriteString("\n")
	}

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

	header.WriteString(left + strings.Repeat(" ", gap) + right)
	return header.String()
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

	// 添加滚动指示器
	scrollInfo := ""
	if m.viewport.TotalLineCount() > m.viewport.Height {
		percent := m.viewport.ScrollPercent() * 100
		if m.autoScroll {
			scrollInfo = " [底部]"
		} else {
			scrollInfo = fmt.Sprintf(" [%.0f%%]", percent)
		}
	}

	// 添加工具调用计数
	toolInfo := ""
	if m.toolCallPanel.Count() > 0 {
		active := m.toolCallPanel.ActiveCount()
		if active > 0 {
			toolInfo = fmt.Sprintf(" | 工具:%d(%d活跃)", m.toolCallPanel.Count(), active)
		} else {
			toolInfo = fmt.Sprintf(" | 工具:%d", m.toolCallPanel.Count())
		}
	}

	left := styles.StatusBarStyle.Render(status + scrollInfo + toolInfo)

	help := "Ctrl+Enter: 发送 | Ctrl+P: 搜索 | Ctrl+R: 恢复 | F3: 编辑代理"
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
