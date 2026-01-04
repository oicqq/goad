// Package components 实现TUI的UI组件
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/goad/internal/config"
	"github.com/anthropics/goad/internal/tui/styles"
)

// FieldType 字段类型
type FieldType int

const (
	FieldTypeString FieldType = iota
	FieldTypeInt
	FieldTypeBool
	FieldTypeSelect
	FieldTypeText // 多行文本
)

// EditField 可编辑字段
type EditField struct {
	Key         string
	Label       string
	Value       string
	Type        FieldType
	Options     []string // 用于Select类型
	Description string
	ReadOnly    bool
}

// AgentEditor 代理参数编辑器
type AgentEditor struct {
	fields        []EditField
	selectedIndex int
	editMode      bool
	editBuffer    string
	cursorPos     int
	width         int
	height        int
	scrollOffset  int
	active        bool

	// 原始配置
	agentConfig *config.AgentConfig

	// 回调
	onSave   func(*config.AgentConfig)
	onCancel func()
}

// NewAgentEditor 创建代理编辑器
func NewAgentEditor() *AgentEditor {
	return &AgentEditor{
		fields: make([]EditField, 0),
		width:  60,
		height: 20,
	}
}

// SetSize 设置大小
func (e *AgentEditor) SetSize(width, height int) {
	e.width = width
	e.height = height
}

// SetOnSave 设置保存回调
func (e *AgentEditor) SetOnSave(fn func(*config.AgentConfig)) {
	e.onSave = fn
}

// SetOnCancel 设置取消回调
func (e *AgentEditor) SetOnCancel(fn func()) {
	e.onCancel = fn
}

// LoadConfig 加载代理配置
func (e *AgentEditor) LoadConfig(cfg *config.AgentConfig) {
	e.agentConfig = cfg
	e.fields = make([]EditField, 0)

	// 基本信息
	e.fields = append(e.fields, EditField{
		Key:         "name",
		Label:       "名称",
		Value:       cfg.Name,
		Type:        FieldTypeString,
		Description: "代理显示名称",
	})

	e.fields = append(e.fields, EditField{
		Key:         "shortName",
		Label:       "短名称",
		Value:       cfg.ShortName,
		Type:        FieldTypeString,
		Description: "用于命令行的短名称",
		ReadOnly:    true,
	})

	e.fields = append(e.fields, EditField{
		Key:         "description",
		Label:       "描述",
		Value:       cfg.Description,
		Type:        FieldTypeString,
		Description: "代理描述",
	})

	e.fields = append(e.fields, EditField{
		Key:         "identity",
		Label:       "标识",
		Value:       cfg.Identity,
		Type:        FieldTypeString,
		Description: "代理唯一标识",
		ReadOnly:    true,
	})

	// 运行命令配置
	runCmd := cfg.GetRunCommand()
	e.fields = append(e.fields, EditField{
		Key:         "runCommand",
		Label:       "运行命令",
		Value:       runCmd,
		Type:        FieldTypeString,
		Description: "执行代理的命令",
		ReadOnly:    true,
	})

	// URL
	e.fields = append(e.fields, EditField{
		Key:         "url",
		Label:       "URL",
		Value:       cfg.URL,
		Type:        FieldTypeString,
		Description: "代理URL",
	})

	// 协议
	e.fields = append(e.fields, EditField{
		Key:         "protocol",
		Label:       "协议",
		Value:       string(cfg.Protocol),
		Type:        FieldTypeSelect,
		Options:     []string{"acp"},
		Description: "通信协议",
		ReadOnly:    true,
	})

	// 类型
	e.fields = append(e.fields, EditField{
		Key:         "type",
		Label:       "类型",
		Value:       string(cfg.Type),
		Type:        FieldTypeSelect,
		Options:     []string{"coding", "chat"},
		Description: "代理类型",
	})

	// 欢迎语
	e.fields = append(e.fields, EditField{
		Key:         "welcome",
		Label:       "欢迎语",
		Value:       cfg.Welcome,
		Type:        FieldTypeString,
		Description: "启动时显示的欢迎信息",
	})

	e.selectedIndex = 0
	e.scrollOffset = 0
}

// Activate 激活编辑器
func (e *AgentEditor) Activate() {
	e.active = true
	e.editMode = false
	e.selectedIndex = 0
}

// Deactivate 停用编辑器
func (e *AgentEditor) Deactivate() {
	e.active = false
	e.editMode = false
}

// IsActive 是否激活
func (e *AgentEditor) IsActive() bool {
	return e.active
}

// IsEditMode 是否在编辑模式
func (e *AgentEditor) IsEditMode() bool {
	return e.editMode
}

// MoveUp 向上移动
func (e *AgentEditor) MoveUp() {
	if e.editMode {
		return
	}
	if e.selectedIndex > 0 {
		e.selectedIndex--
		e.adjustScroll()
	}
}

// MoveDown 向下移动
func (e *AgentEditor) MoveDown() {
	if e.editMode {
		return
	}
	if e.selectedIndex < len(e.fields)-1 {
		e.selectedIndex++
		e.adjustScroll()
	}
}

// adjustScroll 调整滚动
func (e *AgentEditor) adjustScroll() {
	visibleLines := e.height - 8
	if e.selectedIndex < e.scrollOffset {
		e.scrollOffset = e.selectedIndex
	} else if e.selectedIndex >= e.scrollOffset+visibleLines {
		e.scrollOffset = e.selectedIndex - visibleLines + 1
	}
}

// EnterEdit 进入编辑模式
func (e *AgentEditor) EnterEdit() {
	if e.selectedIndex >= len(e.fields) {
		return
	}
	field := e.fields[e.selectedIndex]
	if field.ReadOnly {
		return
	}

	e.editMode = true
	e.editBuffer = field.Value
	e.cursorPos = len(e.editBuffer)
}

// ExitEdit 退出编辑模式
func (e *AgentEditor) ExitEdit(save bool) {
	if !e.editMode {
		return
	}

	if save && e.selectedIndex < len(e.fields) {
		e.fields[e.selectedIndex].Value = e.editBuffer
	}

	e.editMode = false
	e.editBuffer = ""
}

// HandleEditKey 处理编辑模式按键
func (e *AgentEditor) HandleEditKey(key string) {
	if !e.editMode {
		return
	}

	switch key {
	case "backspace":
		if e.cursorPos > 0 {
			e.editBuffer = e.editBuffer[:e.cursorPos-1] + e.editBuffer[e.cursorPos:]
			e.cursorPos--
		}
	case "delete":
		if e.cursorPos < len(e.editBuffer) {
			e.editBuffer = e.editBuffer[:e.cursorPos] + e.editBuffer[e.cursorPos+1:]
		}
	case "left":
		if e.cursorPos > 0 {
			e.cursorPos--
		}
	case "right":
		if e.cursorPos < len(e.editBuffer) {
			e.cursorPos++
		}
	case "home":
		e.cursorPos = 0
	case "end":
		e.cursorPos = len(e.editBuffer)
	default:
		// 插入字符
		if len(key) == 1 {
			e.editBuffer = e.editBuffer[:e.cursorPos] + key + e.editBuffer[e.cursorPos:]
			e.cursorPos++
		}
	}
}

// CycleOption 循环选择选项（用于Select类型）
func (e *AgentEditor) CycleOption() {
	if e.selectedIndex >= len(e.fields) {
		return
	}
	field := &e.fields[e.selectedIndex]
	if field.Type != FieldTypeSelect || len(field.Options) == 0 {
		return
	}

	// 找到当前选项索引
	currentIdx := -1
	for i, opt := range field.Options {
		if opt == field.Value {
			currentIdx = i
			break
		}
	}

	// 循环到下一个
	nextIdx := (currentIdx + 1) % len(field.Options)
	field.Value = field.Options[nextIdx]
}

// Save 保存配置
func (e *AgentEditor) Save() *config.AgentConfig {
	if e.agentConfig == nil {
		return nil
	}

	// 更新配置
	for _, field := range e.fields {
		switch field.Key {
		case "name":
			e.agentConfig.Name = field.Value
		case "description":
			e.agentConfig.Description = field.Value
		case "url":
			e.agentConfig.URL = field.Value
		case "type":
			e.agentConfig.Type = config.AgentType(field.Value)
		case "welcome":
			e.agentConfig.Welcome = field.Value
		}
	}

	if e.onSave != nil {
		e.onSave(e.agentConfig)
	}

	return e.agentConfig
}

// Cancel 取消编辑
func (e *AgentEditor) Cancel() {
	if e.onCancel != nil {
		e.onCancel()
	}
	e.Deactivate()
}

// GetSelectedField 获取选中字段
func (e *AgentEditor) GetSelectedField() *EditField {
	if e.selectedIndex >= 0 && e.selectedIndex < len(e.fields) {
		return &e.fields[e.selectedIndex]
	}
	return nil
}

// View 渲染编辑器
func (e *AgentEditor) View() string {
	if !e.active {
		return ""
	}

	var b strings.Builder
	width := e.width
	if width < 50 {
		width = 50
	}

	// 标题
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true)
	b.WriteString(titleStyle.Render("代理配置编辑器"))
	b.WriteString("\n\n")

	// 代理名称
	if e.agentConfig != nil {
		nameStyle := lipgloss.NewStyle().Foreground(styles.TextSecondary)
		b.WriteString(nameStyle.Render(fmt.Sprintf("编辑: %s", e.agentConfig.ShortName)))
		b.WriteString("\n\n")
	}

	// 分隔线
	b.WriteString(strings.Repeat("─", width-4))
	b.WriteString("\n\n")

	// 字段列表
	visibleLines := e.height - 12
	if visibleLines < 5 {
		visibleLines = 5
	}

	for i := e.scrollOffset; i < len(e.fields) && i < e.scrollOffset+visibleLines; i++ {
		field := e.fields[i]
		isSelected := i == e.selectedIndex

		// 光标
		cursor := "  "
		if isSelected {
			cursor = "> "
		}

		// 标签样式
		labelStyle := lipgloss.NewStyle().Foreground(styles.TextMuted).Width(12)
		if isSelected {
			labelStyle = labelStyle.Foreground(styles.TextPrimary).Bold(true)
		}

		// 值样式
		valueStyle := lipgloss.NewStyle().Foreground(styles.TextSecondary)
		if field.ReadOnly {
			valueStyle = valueStyle.Foreground(styles.TextMuted).Italic(true)
		} else if isSelected {
			valueStyle = valueStyle.Foreground(styles.Primary)
		}

		// 构建行
		b.WriteString(cursor)
		b.WriteString(labelStyle.Render(field.Label + ":"))

		// 显示值或编辑缓冲区
		if isSelected && e.editMode {
			// 编辑模式：显示带光标的缓冲区
			before := e.editBuffer[:e.cursorPos]
			after := e.editBuffer[e.cursorPos:]
			cursorStyle := lipgloss.NewStyle().Background(styles.Primary)
			b.WriteString(before)
			b.WriteString(cursorStyle.Render(" "))
			b.WriteString(after)
		} else {
			// 正常显示
			displayValue := field.Value
			if displayValue == "" {
				displayValue = "(未设置)"
			}
			maxValueLen := width - 20
			if len(displayValue) > maxValueLen {
				displayValue = displayValue[:maxValueLen-3] + "..."
			}
			b.WriteString(valueStyle.Render(displayValue))
		}

		// 类型标记
		if field.Type == FieldTypeSelect && !field.ReadOnly {
			b.WriteString(" ")
			tagStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
			b.WriteString(tagStyle.Render("[Tab切换]"))
		}

		b.WriteString("\n")

		// 显示描述（选中时）
		if isSelected && field.Description != "" {
			descStyle := lipgloss.NewStyle().Foreground(styles.TextMuted).Italic(true)
			b.WriteString("    ")
			b.WriteString(descStyle.Render(field.Description))
			b.WriteString("\n")
		}
	}

	// 滚动指示
	if len(e.fields) > visibleLines {
		scrollInfo := fmt.Sprintf("\n  [%d/%d]", e.selectedIndex+1, len(e.fields))
		b.WriteString(styles.HelpStyle.Render(scrollInfo))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// 帮助
	helpText := "↑↓: 选择  Enter: 编辑  Tab: 切换选项  Ctrl+S: 保存  Esc: 取消"
	if e.editMode {
		helpText = "Enter: 确认  Esc: 取消编辑  ←→: 移动光标"
	}
	b.WriteString(styles.HelpStyle.Render(helpText))

	// 边框
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Width(width)

	return boxStyle.Render(b.String())
}

// HasChanges 是否有修改
func (e *AgentEditor) HasChanges() bool {
	if e.agentConfig == nil {
		return false
	}

	for _, field := range e.fields {
		switch field.Key {
		case "name":
			if field.Value != e.agentConfig.Name {
				return true
			}
		case "description":
			if field.Value != e.agentConfig.Description {
				return true
			}
		case "url":
			if field.Value != e.agentConfig.URL {
				return true
			}
		case "type":
			if field.Value != string(e.agentConfig.Type) {
				return true
			}
		case "welcome":
			if field.Value != e.agentConfig.Welcome {
				return true
			}
		}
	}

	return false
}
