// Package components 实现TUI的UI组件
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/goad/internal/tui/styles"
)

// ThinkingBlock 思考块
type ThinkingBlock struct {
	ID        string
	Content   string
	StartTime time.Time
	EndTime   time.Time
	Collapsed bool
}

// ThinkingDisplay 思考过程显示组件
type ThinkingDisplay struct {
	blocks        []*ThinkingBlock
	currentBlock  *ThinkingBlock
	defaultExpand bool // 默认是否展开
	maxPreview    int  // 折叠时显示的最大字符数
	width         int
}

// NewThinkingDisplay 创建思考显示组件
func NewThinkingDisplay() *ThinkingDisplay {
	return &ThinkingDisplay{
		blocks:        make([]*ThinkingBlock, 0),
		defaultExpand: false, // 默认折叠
		maxPreview:    100,
		width:         80,
	}
}

// SetWidth 设置宽度
func (d *ThinkingDisplay) SetWidth(w int) {
	d.width = w
}

// SetDefaultExpand 设置默认展开状态
func (d *ThinkingDisplay) SetDefaultExpand(expand bool) {
	d.defaultExpand = expand
}

// StartBlock 开始新的思考块
func (d *ThinkingDisplay) StartBlock() {
	// 结束当前块
	if d.currentBlock != nil {
		d.currentBlock.EndTime = time.Now()
	}

	// 创建新块
	d.currentBlock = &ThinkingBlock{
		ID:        fmt.Sprintf("think-%d", len(d.blocks)+1),
		StartTime: time.Now(),
		Collapsed: !d.defaultExpand,
	}
	d.blocks = append(d.blocks, d.currentBlock)
}

// AppendContent 追加内容到当前块
func (d *ThinkingDisplay) AppendContent(text string) {
	if d.currentBlock == nil {
		d.StartBlock()
	}
	d.currentBlock.Content += text
}

// EndBlock 结束当前思考块
func (d *ThinkingDisplay) EndBlock() {
	if d.currentBlock != nil {
		d.currentBlock.EndTime = time.Now()
		d.currentBlock = nil
	}
}

// Clear 清除所有思考块
func (d *ThinkingDisplay) Clear() {
	d.blocks = make([]*ThinkingBlock, 0)
	d.currentBlock = nil
}

// ToggleBlock 切换指定块的折叠状态
func (d *ThinkingDisplay) ToggleBlock(index int) {
	if index >= 0 && index < len(d.blocks) {
		d.blocks[index].Collapsed = !d.blocks[index].Collapsed
	}
}

// ToggleAll 切换所有块的折叠状态
func (d *ThinkingDisplay) ToggleAll() {
	// 检查是否有展开的
	hasExpanded := false
	for _, block := range d.blocks {
		if !block.Collapsed {
			hasExpanded = true
			break
		}
	}

	// 如果有展开的，全部折叠；否则全部展开
	for _, block := range d.blocks {
		block.Collapsed = hasExpanded
	}
}

// ExpandAll 展开所有
func (d *ThinkingDisplay) ExpandAll() {
	for _, block := range d.blocks {
		block.Collapsed = false
	}
}

// CollapseAll 折叠所有
func (d *ThinkingDisplay) CollapseAll() {
	for _, block := range d.blocks {
		block.Collapsed = true
	}
}

// GetBlockCount 获取块数量
func (d *ThinkingDisplay) GetBlockCount() int {
	return len(d.blocks)
}

// HasContent 是否有内容
func (d *ThinkingDisplay) HasContent() bool {
	return len(d.blocks) > 0
}

// IsThinking 是否正在思考
func (d *ThinkingDisplay) IsThinking() bool {
	return d.currentBlock != nil && d.currentBlock.EndTime.IsZero()
}

// View 渲染组件
func (d *ThinkingDisplay) View() string {
	if len(d.blocks) == 0 {
		return ""
	}

	var b strings.Builder
	width := d.width
	if width < 40 {
		width = 40
	}

	for i, block := range d.blocks {
		b.WriteString(d.renderBlock(block, i))
		b.WriteString("\n")
	}

	return b.String()
}

// renderBlock 渲染单个思考块
func (d *ThinkingDisplay) renderBlock(block *ThinkingBlock, index int) string {
	var b strings.Builder
	width := d.width - 4

	// 标题行
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Italic(true)

	// 折叠/展开图标
	expandIcon := "▼"
	if block.Collapsed {
		expandIcon = "▶"
	}

	// 状态指示
	statusText := ""
	if block.EndTime.IsZero() {
		statusText = " (思考中...)"
	} else {
		duration := block.EndTime.Sub(block.StartTime)
		statusText = fmt.Sprintf(" (%.1fs)", duration.Seconds())
	}

	// 内容预览
	preview := ""
	if block.Collapsed && block.Content != "" {
		preview = getPreview(block.Content, 50)
		if preview != "" {
			preview = ": " + preview
		}
	}

	title := fmt.Sprintf("%s 思考 #%d%s%s", expandIcon, index+1, statusText, preview)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// 内容（展开时显示）
	if !block.Collapsed {
		content := block.Content
		if content == "" {
			content = "(空)"
		}

		// 处理内容换行
		lines := strings.Split(content, "\n")
		contentStyle := lipgloss.NewStyle().
			Foreground(styles.TextMuted).
			Width(width).
			PaddingLeft(2)

		for _, line := range lines {
			if len(line) > width-2 {
				// 长行换行
				for len(line) > width-2 {
					b.WriteString(contentStyle.Render(line[:width-2]))
					b.WriteString("\n")
					line = line[width-2:]
				}
				if line != "" {
					b.WriteString(contentStyle.Render(line))
					b.WriteString("\n")
				}
			} else {
				b.WriteString(contentStyle.Render(line))
				b.WriteString("\n")
			}
		}
	}

	// 包装在边框中
	blockStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.TextMuted).
		Padding(0, 1).
		Width(d.width)

	return blockStyle.Render(b.String())
}

// getPreview 获取内容预览
func getPreview(content string, maxLen int) string {
	// 移除换行符
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.TrimSpace(content)

	if len(content) <= maxLen {
		return content
	}

	return content[:maxLen-3] + "..."
}

// CompactView 紧凑视图（用于侧边栏）
func (d *ThinkingDisplay) CompactView() string {
	if len(d.blocks) == 0 {
		return ""
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(styles.TextMuted).
		Bold(true)
	b.WriteString(titleStyle.Render("思考过程"))
	b.WriteString("\n")

	// 统计
	total := len(d.blocks)
	active := 0
	if d.currentBlock != nil && d.currentBlock.EndTime.IsZero() {
		active = 1
	}

	statStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	if active > 0 {
		b.WriteString(statStyle.Render(fmt.Sprintf("  %d个块 (1个进行中)", total)))
	} else {
		b.WriteString(statStyle.Render(fmt.Sprintf("  %d个块", total)))
	}
	b.WriteString("\n")

	// 显示最后几个块的预览
	maxShow := 3
	start := len(d.blocks) - maxShow
	if start < 0 {
		start = 0
	}

	for i := start; i < len(d.blocks); i++ {
		block := d.blocks[i]
		preview := getPreview(block.Content, 25)
		if preview == "" {
			preview = "(空)"
		}

		icon := "○"
		if block.EndTime.IsZero() {
			icon = "●"
		}

		b.WriteString(fmt.Sprintf("  %s %s\n", icon, preview))
	}

	return b.String()
}
