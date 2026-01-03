// Package components 实现TUI的UI组件
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/goad/internal/session"
	"github.com/anthropics/goad/internal/tui/styles"
)

// SessionPicker 会话选择器
type SessionPicker struct {
	sessions      []*session.Session
	filtered      []*session.Session
	query         string
	selectedIndex int
	width         int
	height        int
	maxDisplay    int
	active        bool
	manager       *session.Manager
}

// NewSessionPicker 创建会话选择器
func NewSessionPicker() *SessionPicker {
	return &SessionPicker{
		sessions:   make([]*session.Session, 0),
		filtered:   make([]*session.Session, 0),
		maxDisplay: 8,
	}
}

// Initialize 初始化加载会话列表
func (p *SessionPicker) Initialize() error {
	var err error
	p.manager, err = session.NewManager()
	if err != nil {
		return err
	}

	p.sessions, err = p.manager.List()
	if err != nil {
		return err
	}

	p.filter()
	return nil
}

// LoadForProject 加载指定项目的会话
func (p *SessionPicker) LoadForProject(projectRoot string) error {
	if p.manager == nil {
		if err := p.Initialize(); err != nil {
			return err
		}
	}

	var err error
	p.sessions, err = p.manager.ListByProject(projectRoot)
	if err != nil {
		return err
	}

	p.filter()
	return nil
}

// LoadForAgent 加载指定代理的会话
func (p *SessionPicker) LoadForAgent(agentID string) error {
	if p.manager == nil {
		if err := p.Initialize(); err != nil {
			return err
		}
	}

	var err error
	p.sessions, err = p.manager.ListByAgent(agentID)
	if err != nil {
		return err
	}

	p.filter()
	return nil
}

// SetSize 设置大小
func (p *SessionPicker) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.maxDisplay = height - 6
	if p.maxDisplay < 3 {
		p.maxDisplay = 3
	}
}

// Activate 激活组件
func (p *SessionPicker) Activate() {
	p.active = true
	p.selectedIndex = 0
	p.query = ""
	p.filter()
}

// Deactivate 停用组件
func (p *SessionPicker) Deactivate() {
	p.active = false
	p.query = ""
}

// IsActive 是否激活
func (p *SessionPicker) IsActive() bool {
	return p.active
}

// SetQuery 设置搜索查询
func (p *SessionPicker) SetQuery(query string) {
	p.query = query
	p.selectedIndex = 0
	p.filter()
}

// AppendQuery 追加查询字符
func (p *SessionPicker) AppendQuery(ch rune) {
	p.query += string(ch)
	p.selectedIndex = 0
	p.filter()
}

// DeleteChar 删除查询字符
func (p *SessionPicker) DeleteChar() {
	if len(p.query) > 0 {
		p.query = p.query[:len(p.query)-1]
		p.selectedIndex = 0
		p.filter()
	}
}

// MoveUp 向上移动
func (p *SessionPicker) MoveUp() {
	if p.selectedIndex > 0 {
		p.selectedIndex--
	}
}

// MoveDown 向下移动
func (p *SessionPicker) MoveDown() {
	if p.selectedIndex < len(p.filtered)-1 {
		p.selectedIndex++
	}
}

// GetSelected 获取选中的会话
func (p *SessionPicker) GetSelected() *session.Session {
	if p.selectedIndex >= 0 && p.selectedIndex < len(p.filtered) {
		return p.filtered[p.selectedIndex]
	}
	return nil
}

// HasSessions 是否有会话
func (p *SessionPicker) HasSessions() bool {
	return len(p.sessions) > 0
}

// GetSessionCount 获取会话数量
func (p *SessionPicker) GetSessionCount() int {
	return len(p.sessions)
}

// filter 筛选会话
func (p *SessionPicker) filter() {
	if p.query == "" {
		p.filtered = p.sessions
		return
	}

	queryLower := strings.ToLower(p.query)
	p.filtered = make([]*session.Session, 0)

	for _, s := range p.sessions {
		// 搜索标题和ID
		if strings.Contains(strings.ToLower(s.Title), queryLower) ||
			strings.Contains(strings.ToLower(s.ID), queryLower) ||
			strings.Contains(strings.ToLower(s.AgentID), queryLower) {
			p.filtered = append(p.filtered, s)
		}
	}
}

// View 渲染组件
func (p *SessionPicker) View() string {
	if !p.active {
		return ""
	}

	var b strings.Builder
	width := p.width
	if width < 50 {
		width = 50
	}

	// 标题
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true)
	b.WriteString(titleStyle.Render("恢复会话"))
	b.WriteString("\n\n")

	// 搜索框
	promptStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	queryStyle := lipgloss.NewStyle().Foreground(styles.TextPrimary)

	b.WriteString(promptStyle.Render("搜索: "))
	b.WriteString(queryStyle.Render(p.query))
	b.WriteString("█") // 光标
	b.WriteString("\n")

	// 结果数量
	countStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	b.WriteString(countStyle.Render(fmt.Sprintf("  %d/%d 会话", len(p.filtered), len(p.sessions))))
	b.WriteString("\n\n")

	// 分隔线
	b.WriteString(strings.Repeat("─", width-4))
	b.WriteString("\n\n")

	// 会话列表
	if len(p.filtered) == 0 {
		if p.query != "" {
			b.WriteString(styles.HelpStyle.Render("  无匹配的会话"))
		} else {
			b.WriteString(styles.HelpStyle.Render("  暂无历史会话"))
		}
		b.WriteString("\n")
	} else {
		displayCount := p.maxDisplay
		if displayCount > len(p.filtered) {
			displayCount = len(p.filtered)
		}

		for i := 0; i < displayCount; i++ {
			s := p.filtered[i]
			b.WriteString(p.renderSession(s, i == p.selectedIndex))
			b.WriteString("\n")
		}

		if len(p.filtered) > p.maxDisplay {
			b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("  ... 还有 %d 个会话", len(p.filtered)-p.maxDisplay)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// 帮助
	b.WriteString(styles.HelpStyle.Render("↑↓: 选择  Enter: 恢复  Esc: 取消  Del: 删除"))

	// 边框
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Width(width)

	return boxStyle.Render(b.String())
}

// renderSession 渲染单个会话
func (p *SessionPicker) renderSession(s *session.Session, selected bool) string {
	var b strings.Builder

	cursor := "  "
	titleStyle := lipgloss.NewStyle().Foreground(styles.TextSecondary)
	metaStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)

	if selected {
		cursor = "▶ "
		titleStyle = lipgloss.NewStyle().Foreground(styles.TextPrimary).Bold(true)
		metaStyle = lipgloss.NewStyle().Foreground(styles.TextSecondary)
	}

	// 标题
	title := s.Title
	if title == "" {
		title = "(无标题)"
	}
	maxTitleLen := p.width - 20
	if len(title) > maxTitleLen {
		title = title[:maxTitleLen-3] + "..."
	}

	b.WriteString(cursor)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// 元信息
	meta := fmt.Sprintf("    %s | %d条消息 | %s",
		s.AgentID,
		len(s.Messages),
		s.UpdatedAt.Format("01-02 15:04"))
	b.WriteString(metaStyle.Render(meta))

	return b.String()
}

// DeleteSelected 删除选中的会话
func (p *SessionPicker) DeleteSelected() error {
	s := p.GetSelected()
	if s == nil {
		return nil
	}

	if p.manager == nil {
		return fmt.Errorf("会话管理器未初始化")
	}

	err := p.manager.Delete(s.ID)
	if err != nil {
		return err
	}

	// 从列表中移除
	newSessions := make([]*session.Session, 0, len(p.sessions)-1)
	for _, sess := range p.sessions {
		if sess.ID != s.ID {
			newSessions = append(newSessions, sess)
		}
	}
	p.sessions = newSessions
	p.filter()

	// 调整选择索引
	if p.selectedIndex >= len(p.filtered) {
		p.selectedIndex = len(p.filtered) - 1
	}
	if p.selectedIndex < 0 {
		p.selectedIndex = 0
	}

	return nil
}
