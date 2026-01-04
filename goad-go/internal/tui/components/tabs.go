// Package components 实现TUI的UI组件
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/goad/internal/tui/styles"
)

// TabState 标签状态
type TabState int

const (
	TabStateIdle TabState = iota
	TabStateActive
	TabStateThinking
	TabStateError
)

// Tab 单个标签
type Tab struct {
	ID       string
	Title    string
	State    TabState
	Modified bool   // 是否有未保存的修改
	AgentID  string // 关联的代理ID
}

// TabBar 标签栏组件
type TabBar struct {
	tabs          []*Tab
	activeIndex   int
	width         int
	maxTabs       int
	onTabChange   func(int)        // 标签切换回调
	onTabClose    func(int)        // 标签关闭回调
	onNewTab      func()           // 新建标签回调
}

// NewTabBar 创建标签栏
func NewTabBar() *TabBar {
	return &TabBar{
		tabs:    make([]*Tab, 0),
		maxTabs: 10,
		width:   80,
	}
}

// SetWidth 设置宽度
func (t *TabBar) SetWidth(w int) {
	t.width = w
}

// SetMaxTabs 设置最大标签数
func (t *TabBar) SetMaxTabs(max int) {
	t.maxTabs = max
}

// SetOnTabChange 设置标签切换回调
func (t *TabBar) SetOnTabChange(fn func(int)) {
	t.onTabChange = fn
}

// SetOnTabClose 设置标签关闭回调
func (t *TabBar) SetOnTabClose(fn func(int)) {
	t.onTabClose = fn
}

// SetOnNewTab 设置新建标签回调
func (t *TabBar) SetOnNewTab(fn func()) {
	t.onNewTab = fn
}

// AddTab 添加标签
func (t *TabBar) AddTab(id, title, agentID string) int {
	if len(t.tabs) >= t.maxTabs {
		return -1
	}

	tab := &Tab{
		ID:      id,
		Title:   title,
		State:   TabStateIdle,
		AgentID: agentID,
	}
	t.tabs = append(t.tabs, tab)
	return len(t.tabs) - 1
}

// RemoveTab 移除标签
func (t *TabBar) RemoveTab(index int) bool {
	if index < 0 || index >= len(t.tabs) {
		return false
	}

	// 不允许关闭最后一个标签
	if len(t.tabs) <= 1 {
		return false
	}

	t.tabs = append(t.tabs[:index], t.tabs[index+1:]...)

	// 调整活动索引
	if t.activeIndex >= len(t.tabs) {
		t.activeIndex = len(t.tabs) - 1
	} else if t.activeIndex > index {
		t.activeIndex--
	}

	return true
}

// GetTab 获取标签
func (t *TabBar) GetTab(index int) *Tab {
	if index >= 0 && index < len(t.tabs) {
		return t.tabs[index]
	}
	return nil
}

// GetActiveTab 获取活动标签
func (t *TabBar) GetActiveTab() *Tab {
	return t.GetTab(t.activeIndex)
}

// GetActiveIndex 获取活动索引
func (t *TabBar) GetActiveIndex() int {
	return t.activeIndex
}

// SetActiveIndex 设置活动索引
func (t *TabBar) SetActiveIndex(index int) {
	if index >= 0 && index < len(t.tabs) {
		t.activeIndex = index
		if t.onTabChange != nil {
			t.onTabChange(index)
		}
	}
}

// NextTab 切换到下一个标签
func (t *TabBar) NextTab() {
	if len(t.tabs) <= 1 {
		return
	}
	newIndex := (t.activeIndex + 1) % len(t.tabs)
	t.SetActiveIndex(newIndex)
}

// PrevTab 切换到上一个标签
func (t *TabBar) PrevTab() {
	if len(t.tabs) <= 1 {
		return
	}
	newIndex := t.activeIndex - 1
	if newIndex < 0 {
		newIndex = len(t.tabs) - 1
	}
	t.SetActiveIndex(newIndex)
}

// SetTabState 设置标签状态
func (t *TabBar) SetTabState(index int, state TabState) {
	if tab := t.GetTab(index); tab != nil {
		tab.State = state
	}
}

// SetTabTitle 设置标签标题
func (t *TabBar) SetTabTitle(index int, title string) {
	if tab := t.GetTab(index); tab != nil {
		tab.Title = title
	}
}

// SetTabModified 设置标签修改状态
func (t *TabBar) SetTabModified(index int, modified bool) {
	if tab := t.GetTab(index); tab != nil {
		tab.Modified = modified
	}
}

// GetTabCount 获取标签数量
func (t *TabBar) GetTabCount() int {
	return len(t.tabs)
}

// FindTabByID 根据ID查找标签索引
func (t *TabBar) FindTabByID(id string) int {
	for i, tab := range t.tabs {
		if tab.ID == id {
			return i
		}
	}
	return -1
}

// CloseActiveTab 关闭活动标签
func (t *TabBar) CloseActiveTab() bool {
	if t.onTabClose != nil {
		t.onTabClose(t.activeIndex)
	}
	return t.RemoveTab(t.activeIndex)
}

// NewTab 新建标签
func (t *TabBar) NewTab() {
	if t.onNewTab != nil {
		t.onNewTab()
	}
}

// View 渲染标签栏
func (t *TabBar) View() string {
	if len(t.tabs) == 0 {
		return ""
	}

	var b strings.Builder
	width := t.width
	tabWidth := (width - 4) / len(t.tabs)
	if tabWidth > 25 {
		tabWidth = 25
	}
	if tabWidth < 10 {
		tabWidth = 10
	}

	for i, tab := range t.tabs {
		isActive := i == t.activeIndex

		// 状态图标
		stateIcon := ""
		switch tab.State {
		case TabStateThinking:
			stateIcon = "..."
		case TabStateError:
			stateIcon = "!"
		case TabStateActive:
			stateIcon = "*"
		}

		// 修改标记
		modifiedMark := ""
		if tab.Modified {
			modifiedMark = "*"
		}

		// 标题截断
		title := tab.Title
		maxTitleLen := tabWidth - 6
		if len(title) > maxTitleLen {
			title = title[:maxTitleLen-2] + ".."
		}

		// 构建标签文本
		tabText := fmt.Sprintf(" %s%s%s ", stateIcon, title, modifiedMark)

		// 样式
		var tabStyle lipgloss.Style
		if isActive {
			tabStyle = lipgloss.NewStyle().
				Foreground(styles.TextPrimary).
				Background(styles.Primary).
				Bold(true).
				Padding(0, 1)
		} else {
			tabStyle = lipgloss.NewStyle().
				Foreground(styles.TextSecondary).
				Background(styles.BgSecondary).
				Padding(0, 1)
		}

		b.WriteString(tabStyle.Render(tabText))

		// 分隔符
		if i < len(t.tabs)-1 {
			sepStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
			b.WriteString(sepStyle.Render("|"))
		}
	}

	// 新建标签按钮
	if len(t.tabs) < t.maxTabs {
		addStyle := lipgloss.NewStyle().
			Foreground(styles.TextMuted).
			Padding(0, 1)
		b.WriteString(addStyle.Render(" + "))
	}

	return b.String()
}

// CompactView 紧凑视图（显示索引）
func (t *TabBar) CompactView() string {
	if len(t.tabs) <= 1 {
		return ""
	}

	var indicators []string
	for i := range t.tabs {
		if i == t.activeIndex {
			indicators = append(indicators, fmt.Sprintf("[%d]", i+1))
		} else {
			indicators = append(indicators, fmt.Sprintf(" %d ", i+1))
		}
	}

	style := lipgloss.NewStyle().Foreground(styles.TextMuted)
	return style.Render(strings.Join(indicators, ""))
}

// HelpText 帮助文本
func (t *TabBar) HelpText() string {
	return "Ctrl+T: 新标签  Ctrl+W: 关闭  Ctrl+Tab: 下一个  Ctrl+Shift+Tab: 上一个"
}
