// Package components 实现TUI的UI组件
package components

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/goad/internal/acp"
	"github.com/anthropics/goad/internal/tui/styles"
)

// ContentType 内容类型
type ContentType int

const (
	ContentTypePlain ContentType = iota
	ContentTypeANSI
	ContentTypeMarkdown
)

// ToolCallPanel 工具调用状态面板
type ToolCallPanel struct {
	toolCalls   map[string]*ToolCallState
	order       []string // 保持顺序
	width       int
	collapsed   bool
	maxDisplay  int // 最多显示的工具调用数
}

// ToolCallState 工具调用状态
type ToolCallState struct {
	ToolCall  *acp.ToolCall
	StartTime time.Time
	EndTime   time.Time
	Expanded  bool // 是否展开显示内容
}

// NewToolCallPanel 创建工具调用面板
func NewToolCallPanel() *ToolCallPanel {
	return &ToolCallPanel{
		toolCalls:  make(map[string]*ToolCallState),
		order:      make([]string, 0),
		maxDisplay: 5,
	}
}

// Update 更新工具调用
func (p *ToolCallPanel) Update(tc *acp.ToolCall) {
	if tc == nil {
		return
	}

	if existing, ok := p.toolCalls[tc.ToolCallID]; ok {
		// 更新现有的
		existing.ToolCall = tc
		if tc.Status == acp.ToolStatusCompleted || tc.Status == acp.ToolStatusFailed {
			existing.EndTime = time.Now()
			// 自动展开非 read 类型的已完成/失败调用
			if tc.Kind != acp.ToolKindRead && !existing.Expanded {
				existing.Expanded = shouldAutoExpand(tc)
			}
		}
	} else {
		// 新增
		state := &ToolCallState{
			ToolCall:  tc,
			StartTime: time.Now(),
			Expanded:  false, // 默认不展开
		}
		p.toolCalls[tc.ToolCallID] = state
		p.order = append(p.order, tc.ToolCallID)
	}
}

// shouldAutoExpand 判断是否应该自动展开
func shouldAutoExpand(tc *acp.ToolCall) bool {
	// read 类型默认不自动展开（避免噪音）
	if tc.Kind == acp.ToolKindRead {
		return false
	}
	// 失败的调用自动展开
	if tc.Status == acp.ToolStatusFailed {
		return true
	}
	// 有内容且不是 read 类型的调用可以展开
	if len(tc.Content) > 0 {
		return true
	}
	return false
}

// ToggleExpand 切换指定工具调用的展开状态
func (p *ToolCallPanel) ToggleExpand(toolCallID string) {
	if state, ok := p.toolCalls[toolCallID]; ok {
		state.Expanded = !state.Expanded
	}
}

// ExpandAll 展开所有
func (p *ToolCallPanel) ExpandAll() {
	for _, state := range p.toolCalls {
		if hasContent(state.ToolCall) {
			state.Expanded = true
		}
	}
}

// CollapseAll 折叠所有
func (p *ToolCallPanel) CollapseAll() {
	for _, state := range p.toolCalls {
		state.Expanded = false
	}
}

// hasContent 检查是否有内容
func hasContent(tc *acp.ToolCall) bool {
	return len(tc.Content) > 0
}

// Clear 清除所有工具调用
func (p *ToolCallPanel) Clear() {
	p.toolCalls = make(map[string]*ToolCallState)
	p.order = make([]string, 0)
}

// SetWidth 设置宽度
func (p *ToolCallPanel) SetWidth(w int) {
	p.width = w
}

// Toggle 切换折叠状态
func (p *ToolCallPanel) Toggle() {
	p.collapsed = !p.collapsed
}

// IsCollapsed 是否折叠
func (p *ToolCallPanel) IsCollapsed() bool {
	return p.collapsed
}

// Count 返回工具调用数量
func (p *ToolCallPanel) Count() int {
	return len(p.toolCalls)
}

// ActiveCount 返回活跃的工具调用数量
func (p *ToolCallPanel) ActiveCount() int {
	count := 0
	for _, state := range p.toolCalls {
		if state.ToolCall.Status == acp.ToolStatusPending ||
			state.ToolCall.Status == acp.ToolStatusInProgress {
			count++
		}
	}
	return count
}

// View 渲染面板
func (p *ToolCallPanel) View() string {
	if len(p.toolCalls) == 0 {
		return ""
	}

	var b strings.Builder
	width := p.width
	if width < 40 {
		width = 40
	}

	// 标题
	activeCount := p.ActiveCount()
	totalCount := len(p.toolCalls)

	titleText := fmt.Sprintf("工具调用 (%d/%d)", activeCount, totalCount)
	if p.collapsed {
		titleText += " [+]"
	} else {
		titleText += " [-]"
	}
	title := styles.HeaderStyle.Render(titleText)
	b.WriteString(title)
	b.WriteString("\n")

	if p.collapsed {
		// 折叠状态只显示摘要
		summary := p.renderSummary()
		b.WriteString(summary)
	} else {
		// 展开状态显示详细列表
		displayed := 0
		for i := len(p.order) - 1; i >= 0 && displayed < p.maxDisplay; i-- {
			id := p.order[i]
			state, ok := p.toolCalls[id]
			if !ok {
				continue
			}
			b.WriteString(p.renderToolCall(state, width-4))
			b.WriteString("\n")
			displayed++
		}

		if len(p.order) > p.maxDisplay {
			more := len(p.order) - p.maxDisplay
			b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("  ... 还有 %d 个工具调用", more)))
			b.WriteString("\n")
		}
	}

	return panelStyle.Width(width).Render(b.String())
}

// renderSummary 渲染摘要
func (p *ToolCallPanel) renderSummary() string {
	pending := 0
	inProgress := 0
	completed := 0
	failed := 0

	for _, state := range p.toolCalls {
		switch state.ToolCall.Status {
		case acp.ToolStatusPending:
			pending++
		case acp.ToolStatusInProgress:
			inProgress++
		case acp.ToolStatusCompleted:
			completed++
		case acp.ToolStatusFailed:
			failed++
		}
	}

	var parts []string
	if pending > 0 {
		parts = append(parts, styles.ToolCallStatusPending.Render(fmt.Sprintf("待处理:%d", pending)))
	}
	if inProgress > 0 {
		parts = append(parts, styles.ToolCallStatusInProgress.Render(fmt.Sprintf("处理中:%d", inProgress)))
	}
	if completed > 0 {
		parts = append(parts, styles.ToolCallStatusCompleted.Render(fmt.Sprintf("完成:%d", completed)))
	}
	if failed > 0 {
		parts = append(parts, styles.ToolCallStatusFailed.Render(fmt.Sprintf("失败:%d", failed)))
	}

	return "  " + strings.Join(parts, " | ")
}

// renderToolCall 渲染单个工具调用
func (p *ToolCallPanel) renderToolCall(state *ToolCallState, maxWidth int) string {
	tc := state.ToolCall
	var b strings.Builder

	// 展开/折叠图标
	expandIcon := "▶"
	if state.Expanded {
		expandIcon = "▼"
	}
	if !hasContent(tc) {
		expandIcon = " " // 无内容时不显示展开图标
	}

	// 状态图标（参考 Python 版本）
	var statusIcon string
	var statusStyle lipgloss.Style
	switch tc.Status {
	case acp.ToolStatusPending:
		statusIcon = "⏲"
		statusStyle = lipgloss.NewStyle().Foreground(styles.Pending)
	case acp.ToolStatusInProgress:
		statusIcon = "◐"
		statusStyle = lipgloss.NewStyle().Foreground(styles.InProgress)
	case acp.ToolStatusCompleted:
		statusIcon = "✔"
		statusStyle = lipgloss.NewStyle().Foreground(styles.Completed)
	case acp.ToolStatusFailed:
		statusIcon = "✘"
		statusStyle = lipgloss.NewStyle().Foreground(styles.Failed)
	default:
		statusIcon = "?"
		statusStyle = lipgloss.NewStyle().Foreground(styles.TextMuted)
	}

	// 类型图标
	kindIcon := styles.GetToolKindIcon(string(tc.Kind))

	// 耗时
	duration := ""
	if !state.StartTime.IsZero() {
		if state.EndTime.IsZero() {
			duration = fmt.Sprintf(" (%.1fs)", time.Since(state.StartTime).Seconds())
		} else {
			duration = fmt.Sprintf(" (%.1fs)", state.EndTime.Sub(state.StartTime).Seconds())
		}
	}

	// 标题 (截断)
	title := tc.Title
	maxLen := maxWidth - 15
	if maxLen < 20 {
		maxLen = 20
	}
	if len(title) > maxLen {
		title = title[:maxLen-3] + "..."
	}

	// 渲染标题行
	titleLine := fmt.Sprintf("  %s %s %s %s%s",
		expandIcon,
		statusStyle.Render(statusIcon),
		kindIcon,
		title,
		styles.HelpStyle.Render(duration),
	)
	b.WriteString(titleLine)

	// 如果展开，渲染内容
	if state.Expanded && hasContent(tc) {
		b.WriteString("\n")
		b.WriteString(p.renderToolCallContent(tc, maxWidth-4))
	}

	return b.String()
}

// renderToolCallContent 渲染工具调用内容
func (p *ToolCallPanel) renderToolCallContent(tc *acp.ToolCall, maxWidth int) string {
	var b strings.Builder
	contentStyle := lipgloss.NewStyle().
		Foreground(styles.TextSecondary).
		PaddingLeft(4)

	for _, content := range tc.Content {
		contentType, _ := content["type"].(string)
		switch contentType {
		case "content":
			if subContent, ok := content["content"].(map[string]interface{}); ok {
				if text, ok := subContent["text"].(string); ok {
					rendered := renderTextContent(text, maxWidth)
					b.WriteString(contentStyle.Render(rendered))
					b.WriteString("\n")
				}
			}
		case "diff":
			path, _ := content["path"].(string)
			oldText, _ := content["oldText"].(string)
			newText, _ := content["newText"].(string)
			diffContent := renderDiffContent(path, oldText, newText, maxWidth)
			b.WriteString(contentStyle.Render(diffContent))
			b.WriteString("\n")
		case "terminal":
			if terminalID, ok := content["terminalId"].(string); ok {
				b.WriteString(contentStyle.Render(fmt.Sprintf("终端: %s", terminalID)))
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

// DetectContentType 检测内容类型
func DetectContentType(text string) ContentType {
	// 检测 ANSI 转义码
	if strings.Contains(text, "\x1b[") {
		return ContentTypeANSI
	}
	// 检测 Markdown（代码块或标题）
	if strings.Contains(text, "```") {
		return ContentTypeMarkdown
	}
	// 检测 Markdown 标题
	if matched, _ := regexp.MatchString(`(?m)^#{1,6}\s+.+$`, text); matched {
		return ContentTypeMarkdown
	}
	return ContentTypePlain
}

// renderTextContent 渲染文本内容（智能检测类型）
func renderTextContent(text string, maxWidth int) string {
	contentType := DetectContentType(text)

	// 限制行数
	lines := strings.Split(text, "\n")
	maxLines := 10
	truncated := false
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		truncated = true
	}

	// 限制每行长度
	for i, line := range lines {
		if len(line) > maxWidth {
			lines[i] = line[:maxWidth-3] + "..."
		}
	}

	result := strings.Join(lines, "\n")
	if truncated {
		result += "\n" + styles.HelpStyle.Render(fmt.Sprintf("... (共 %d 行)", len(strings.Split(text, "\n"))))
	}

	switch contentType {
	case ContentTypeANSI:
		// ANSI 内容直接返回（终端会解析）
		return result
	case ContentTypeMarkdown:
		// Markdown 内容可以后续处理
		return result
	default:
		return result
	}
}

// renderDiffContent 渲染差异内容
func renderDiffContent(path, oldText, newText string, maxWidth int) string {
	var b strings.Builder

	// 文件路径
	b.WriteString(styles.DiffPathStyle.Render(path))
	b.WriteString("\n")

	// 简化显示差异
	if oldText == "" && newText != "" {
		// 新文件
		b.WriteString(styles.DiffAddStyle.Render("+ 新建文件"))
		lines := strings.Split(newText, "\n")
		if len(lines) > 5 {
			b.WriteString(styles.HelpStyle.Render(fmt.Sprintf(" (%d 行)", len(lines))))
		}
	} else if oldText != "" && newText == "" {
		// 删除文件
		b.WriteString(styles.DiffRemoveStyle.Render("- 删除文件"))
	} else {
		// 修改文件
		oldLines := len(strings.Split(oldText, "\n"))
		newLines := len(strings.Split(newText, "\n"))
		added := 0
		removed := 0
		if newLines > oldLines {
			added = newLines - oldLines
		} else {
			removed = oldLines - newLines
		}
		b.WriteString(fmt.Sprintf("修改: +%d -%d 行", added, removed))
	}

	return b.String()
}

// 面板样式
var panelStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(styles.BgTertiary).
	Padding(0, 1)
