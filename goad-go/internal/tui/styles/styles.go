// Package styles 定义了TUI的样式
package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// 颜色定义
var (
	// 主题色
	Primary   = lipgloss.Color("#7C3AED") // 紫色
	Secondary = lipgloss.Color("#10B981") // 绿色
	Accent    = lipgloss.Color("#F59E0B") // 橙色

	// 文本色
	TextPrimary   = lipgloss.Color("#F9FAFB")
	TextSecondary = lipgloss.Color("#9CA3AF")
	TextMuted     = lipgloss.Color("#6B7280")

	// 背景色
	BgPrimary   = lipgloss.Color("#111827")
	BgSecondary = lipgloss.Color("#1F2937")
	BgTertiary  = lipgloss.Color("#374151")

	// 状态色
	Success = lipgloss.Color("#10B981")
	Warning = lipgloss.Color("#F59E0B")
	Error   = lipgloss.Color("#EF4444")
	Info    = lipgloss.Color("#3B82F6")

	// 工具调用状态色
	Pending    = lipgloss.Color("#9CA3AF")
	InProgress = lipgloss.Color("#3B82F6")
	Completed  = lipgloss.Color("#10B981")
	Failed     = lipgloss.Color("#EF4444")
)

// 基础样式
var (
	// 应用容器
	AppStyle = lipgloss.NewStyle().
		Background(BgPrimary)

	// 标题栏
	HeaderStyle = lipgloss.NewStyle().
		Foreground(TextPrimary).
		Background(BgSecondary).
		Padding(0, 1).
		Bold(true)

	// 状态栏
	StatusBarStyle = lipgloss.NewStyle().
		Foreground(TextSecondary).
		Background(BgSecondary).
		Padding(0, 1)

	// 侧边栏
	SidebarStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BgTertiary).
		Padding(1)

	// 主内容区
	ContentStyle = lipgloss.NewStyle().
		Padding(1)
)

// 消息样式
var (
	// 用户消息
	UserMessageStyle = lipgloss.NewStyle().
		Foreground(TextPrimary).
		Background(BgTertiary).
		Padding(0, 1).
		MarginBottom(1)

	// 代理消息
	AgentMessageStyle = lipgloss.NewStyle().
		Foreground(TextPrimary).
		MarginBottom(1)

	// 思考消息
	ThinkingStyle = lipgloss.NewStyle().
		Foreground(TextMuted).
		Italic(true).
		MarginBottom(1)

	// 系统消息
	SystemMessageStyle = lipgloss.NewStyle().
		Foreground(TextSecondary).
		Italic(true).
		MarginBottom(1)
)

// 工具调用样式
var (
	// 工具调用容器
	ToolCallStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BgTertiary).
		Padding(0, 1).
		MarginBottom(1)

	// 工具调用标题
	ToolCallTitleStyle = lipgloss.NewStyle().
		Foreground(TextPrimary).
		Bold(true)

	// 工具调用状态
	ToolCallStatusPending = lipgloss.NewStyle().
		Foreground(Pending).
		SetString("待处理")

	ToolCallStatusInProgress = lipgloss.NewStyle().
		Foreground(InProgress).
		SetString("处理中")

	ToolCallStatusCompleted = lipgloss.NewStyle().
		Foreground(Completed).
		SetString("已完成")

	ToolCallStatusFailed = lipgloss.NewStyle().
		Foreground(Failed).
		SetString("失败")
)

// 差异显示样式
var (
	// 添加的行
	DiffAddStyle = lipgloss.NewStyle().
		Foreground(Success)

	// 删除的行
	DiffRemoveStyle = lipgloss.NewStyle().
		Foreground(Error)

	// 上下文行
	DiffContextStyle = lipgloss.NewStyle().
		Foreground(TextMuted)

	// 文件路径
	DiffPathStyle = lipgloss.NewStyle().
		Foreground(Info).
		Bold(true)
)

// 输入框样式
var (
	// 输入框容器
	InputStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Primary).
		Padding(0, 1)

	// 输入提示
	PromptStyle = lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true)

	// 占位符
	PlaceholderStyle = lipgloss.NewStyle().
		Foreground(TextMuted)
)

// 按钮样式
var (
	// 主要按钮
	ButtonPrimaryStyle = lipgloss.NewStyle().
		Foreground(TextPrimary).
		Background(Primary).
		Padding(0, 2)

	// 次要按钮
	ButtonSecondaryStyle = lipgloss.NewStyle().
		Foreground(TextPrimary).
		Background(BgTertiary).
		Padding(0, 2)

	// 成功按钮
	ButtonSuccessStyle = lipgloss.NewStyle().
		Foreground(TextPrimary).
		Background(Success).
		Padding(0, 2)

	// 危险按钮
	ButtonDangerStyle = lipgloss.NewStyle().
		Foreground(TextPrimary).
		Background(Error).
		Padding(0, 2)
)

// 计划样式
var (
	// 计划容器
	PlanStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Accent).
		Padding(0, 1).
		MarginBottom(1)

	// 计划标题
	PlanTitleStyle = lipgloss.NewStyle().
		Foreground(Accent).
		Bold(true)

	// 计划条目
	PlanEntryPending = lipgloss.NewStyle().
		Foreground(TextMuted).
		SetString("○")

	PlanEntryInProgress = lipgloss.NewStyle().
		Foreground(InProgress).
		SetString("●")

	PlanEntryCompleted = lipgloss.NewStyle().
		Foreground(Completed).
		SetString("✓")
)

// 权限请求样式
var (
	// 权限请求容器
	PermissionStyle = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(Warning).
		Padding(1).
		MarginBottom(1)

	// 权限请求标题
	PermissionTitleStyle = lipgloss.NewStyle().
		Foreground(Warning).
		Bold(true)
)

// 帮助文本样式
var (
	HelpStyle = lipgloss.NewStyle().
		Foreground(TextMuted)

	HelpKeyStyle = lipgloss.NewStyle().
		Foreground(TextSecondary).
		Bold(true)
)

// GetToolCallStatusStyle 根据状态返回样式
func GetToolCallStatusStyle(status string) lipgloss.Style {
	switch status {
	case "pending":
		return ToolCallStatusPending
	case "in_progress":
		return ToolCallStatusInProgress
	case "completed":
		return ToolCallStatusCompleted
	case "failed":
		return ToolCallStatusFailed
	default:
		return ToolCallStatusPending
	}
}

// GetToolCallStatusText 根据状态返回中文文本
func GetToolCallStatusText(status string) string {
	switch status {
	case "pending":
		return "待处理"
	case "in_progress":
		return "处理中"
	case "completed":
		return "已完成"
	case "failed":
		return "失败"
	default:
		return "未知"
	}
}

// GetToolKindIcon 根据工具类型返回图标
func GetToolKindIcon(kind string) string {
	switch kind {
	case "read":
		return "📖"
	case "edit":
		return "✏️"
	case "delete":
		return "🗑️"
	case "move":
		return "📦"
	case "search":
		return "🔍"
	case "execute":
		return "⚡"
	case "think":
		return "💭"
	case "fetch":
		return "🌐"
	case "switch_mode":
		return "🔄"
	default:
		return "🔧"
	}
}
