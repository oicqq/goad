// Package components 实现TUI的UI组件
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/goad/internal/acp"
	"github.com/anthropics/goad/internal/tui/styles"
)

// PermissionDialog 权限请求对话框
type PermissionDialog struct {
	request        *acp.PermissionRequestMessage
	selectedOption int
	width          int
	showDetails    bool
}

// NewPermissionDialog 创建权限对话框
func NewPermissionDialog() *PermissionDialog {
	return &PermissionDialog{
		width: 60,
	}
}

// SetRequest 设置权限请求
func (d *PermissionDialog) SetRequest(req *acp.PermissionRequestMessage) {
	d.request = req
	d.selectedOption = 0
	d.showDetails = false
}

// Clear 清除请求
func (d *PermissionDialog) Clear() {
	d.request = nil
	d.selectedOption = 0
}

// IsActive 是否有活跃的请求
func (d *PermissionDialog) IsActive() bool {
	return d.request != nil
}

// GetRequest 获取当前请求
func (d *PermissionDialog) GetRequest() *acp.PermissionRequestMessage {
	return d.request
}

// SetWidth 设置宽度
func (d *PermissionDialog) SetWidth(w int) {
	d.width = w
}

// MoveUp 向上移动选择
func (d *PermissionDialog) MoveUp() {
	if d.selectedOption > 0 {
		d.selectedOption--
	}
}

// MoveDown 向下移动选择
func (d *PermissionDialog) MoveDown() {
	if d.request != nil && d.selectedOption < len(d.request.Options)-1 {
		d.selectedOption++
	}
}

// ToggleDetails 切换详情显示
func (d *PermissionDialog) ToggleDetails() {
	d.showDetails = !d.showDetails
}

// GetSelectedOption 获取选中的选项
func (d *PermissionDialog) GetSelectedOption() *acp.PermissionOption {
	if d.request == nil || d.selectedOption >= len(d.request.Options) {
		return nil
	}
	return &d.request.Options[d.selectedOption]
}

// Confirm 确认选择
func (d *PermissionDialog) Confirm() *acp.PermissionResponse {
	opt := d.GetSelectedOption()
	if opt == nil {
		return nil
	}
	return &acp.PermissionResponse{
		OptionID: opt.OptionID,
		Outcome:  "selected",
	}
}

// Cancel 取消请求
func (d *PermissionDialog) Cancel() *acp.PermissionResponse {
	return &acp.PermissionResponse{
		Outcome: "cancelled",
	}
}

// View 渲染对话框
func (d *PermissionDialog) View() string {
	if d.request == nil {
		return ""
	}

	var b strings.Builder
	width := d.width
	if width < 50 {
		width = 50
	}

	// 标题
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Warning).
		Bold(true).
		Padding(0, 1)
	b.WriteString(titleStyle.Render("⚠️  权限请求"))
	b.WriteString("\n\n")

	// 工具调用信息
	if d.request.ToolCall != nil {
		tc := d.request.ToolCall

		// 操作类型和标题
		kindIcon := styles.GetToolKindIcon(string(tc.Kind))
		b.WriteString(fmt.Sprintf("操作: %s %s\n", kindIcon, tc.Title))

		// 显示详情
		if d.showDetails {
			b.WriteString("\n")
			// 显示位置
			if len(tc.Locations) > 0 {
				b.WriteString("位置:\n")
				for _, loc := range tc.Locations {
					if loc.Line != nil {
						b.WriteString(fmt.Sprintf("  - %s:%d\n", loc.Path, *loc.Line))
					} else {
						b.WriteString(fmt.Sprintf("  - %s\n", loc.Path))
					}
				}
			}
			// 显示内容预览
			if len(tc.Content) > 0 {
				b.WriteString("内容预览:\n")
				for _, content := range tc.Content {
					if t, ok := content["type"].(string); ok {
						switch t {
						case "diff":
							if path, ok := content["path"].(string); ok {
								b.WriteString(fmt.Sprintf("  [diff] %s\n", path))
							}
						case "terminal":
							if tid, ok := content["terminalId"].(string); ok {
								b.WriteString(fmt.Sprintf("  [terminal] %s\n", tid))
							}
						}
					}
				}
			}
		} else {
			b.WriteString(styles.HelpStyle.Render("按 Tab 查看详情"))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// 分隔线
	b.WriteString(strings.Repeat("─", width-4))
	b.WriteString("\n\n")

	// 选项列表
	for i, opt := range d.request.Options {
		cursor := "  "
		optStyle := lipgloss.NewStyle().Foreground(styles.TextSecondary)

		if i == d.selectedOption {
			cursor = "▶ "
			optStyle = lipgloss.NewStyle().Foreground(styles.TextPrimary).Bold(true)
		}

		// 根据选项类型添加颜色
		switch opt.Kind {
		case acp.PermissionAllowOnce, acp.PermissionAllowAlways:
			optStyle = optStyle.Foreground(styles.Success)
		case acp.PermissionRejectOnce, acp.PermissionRejectAlways:
			optStyle = optStyle.Foreground(styles.Error)
		}

		// 添加选项类型提示
		hint := ""
		switch opt.Kind {
		case acp.PermissionAllowOnce:
			hint = " (仅此次)"
		case acp.PermissionAllowAlways:
			hint = " (始终)"
		case acp.PermissionRejectOnce:
			hint = " (仅此次)"
		case acp.PermissionRejectAlways:
			hint = " (始终)"
		}

		b.WriteString(cursor)
		b.WriteString(optStyle.Render(opt.Name))
		b.WriteString(styles.HelpStyle.Render(hint))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// 快捷键提示
	helpStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	b.WriteString(helpStyle.Render("↑↓: 选择  Enter: 确认  Esc: 取消  Tab: 详情"))

	// 包装在边框中
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(styles.Warning).
		Padding(1, 2).
		Width(width)

	return dialogStyle.Render(b.String())
}

// CenteredView 渲染居中的对话框
func (d *PermissionDialog) CenteredView(screenWidth, screenHeight int) string {
	content := d.View()
	if content == "" {
		return ""
	}

	// 计算居中位置
	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	hPad := (screenWidth - contentWidth) / 2
	vPad := (screenHeight - contentHeight) / 3 // 稍微偏上

	if hPad < 0 {
		hPad = 0
	}
	if vPad < 0 {
		vPad = 0
	}

	// 添加垂直填充
	var result strings.Builder
	for i := 0; i < vPad; i++ {
		result.WriteString("\n")
	}

	// 添加水平填充
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		result.WriteString(strings.Repeat(" ", hPad))
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}
