// Package components 实现TUI的UI组件
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/goad/internal/acp"
	"github.com/anthropics/goad/internal/tui/styles"
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
		}
	} else {
		// 新增
		p.toolCalls[tc.ToolCallID] = &ToolCallState{
			ToolCall:  tc,
			StartTime: time.Now(),
		}
		p.order = append(p.order, tc.ToolCallID)
	}
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
			b.WriteString(p.renderToolCall(state))
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
func (p *ToolCallPanel) renderToolCall(state *ToolCallState) string {
	tc := state.ToolCall

	// 状态图标
	var statusIcon string
	var statusStyle lipgloss.Style
	switch tc.Status {
	case acp.ToolStatusPending:
		statusIcon = "○"
		statusStyle = lipgloss.NewStyle().Foreground(styles.Pending)
	case acp.ToolStatusInProgress:
		statusIcon = "●"
		statusStyle = lipgloss.NewStyle().Foreground(styles.InProgress)
	case acp.ToolStatusCompleted:
		statusIcon = "✓"
		statusStyle = lipgloss.NewStyle().Foreground(styles.Completed)
	case acp.ToolStatusFailed:
		statusIcon = "✗"
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
	maxLen := p.width - 15
	if maxLen < 20 {
		maxLen = 20
	}
	if len(title) > maxLen {
		title = title[:maxLen-3] + "..."
	}

	return fmt.Sprintf("  %s %s %s%s",
		statusStyle.Render(statusIcon),
		kindIcon,
		title,
		styles.HelpStyle.Render(duration),
	)
}

// 面板样式
var panelStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(styles.BgTertiary).
	Padding(0, 1)
