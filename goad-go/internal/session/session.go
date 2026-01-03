// Package session 提供会话管理功能
package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Message 会话消息
type Message struct {
	Role      string    `json:"role"`      // "user", "agent", "system", "thinking"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Session 会话信息
type Session struct {
	ID          string    `json:"id"`
	AgentID     string    `json:"agent_id"`
	ProjectRoot string    `json:"project_root"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Messages    []Message `json:"messages"`
	Title       string    `json:"title"` // 会话标题（从第一条消息生成）
}

// Manager 会话管理器
type Manager struct {
	dataDir string
}

// NewManager 创建会话管理器
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dataDir := filepath.Join(homeDir, ".config", "goad", "sessions")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	return &Manager{dataDir: dataDir}, nil
}

// Create 创建新会话
func (m *Manager) Create(agentID, projectRoot string) *Session {
	return &Session{
		ID:          generateID(),
		AgentID:     agentID,
		ProjectRoot: projectRoot,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Messages:    []Message{},
	}
}

// Save 保存会话
func (m *Manager) Save(session *Session) error {
	session.UpdatedAt = time.Now()

	// 生成标题
	if session.Title == "" && len(session.Messages) > 0 {
		for _, msg := range session.Messages {
			if msg.Role == "user" {
				session.Title = truncateTitle(msg.Content, 50)
				break
			}
		}
	}

	path := m.sessionPath(session.ID)
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Load 加载会话
func (m *Manager) Load(id string) (*Session, error) {
	path := m.sessionPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// Delete 删除会话
func (m *Manager) Delete(id string) error {
	path := m.sessionPath(id)
	return os.Remove(path)
}

// List 列出所有会话
func (m *Manager) List() ([]*Session, error) {
	entries, err := os.ReadDir(m.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Session{}, nil
		}
		return nil, err
	}

	var sessions []*Session
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5] // 去掉.json后缀
		session, err := m.Load(id)
		if err != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	// 按更新时间排序（最新的在前）
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// ListByAgent 列出指定代理的会话
func (m *Manager) ListByAgent(agentID string) ([]*Session, error) {
	all, err := m.List()
	if err != nil {
		return nil, err
	}

	var filtered []*Session
	for _, session := range all {
		if session.AgentID == agentID {
			filtered = append(filtered, session)
		}
	}

	return filtered, nil
}

// ListByProject 列出指定项目的会话
func (m *Manager) ListByProject(projectRoot string) ([]*Session, error) {
	all, err := m.List()
	if err != nil {
		return nil, err
	}

	var filtered []*Session
	for _, session := range all {
		if session.ProjectRoot == projectRoot {
			filtered = append(filtered, session)
		}
	}

	return filtered, nil
}

// AddMessage 添加消息到会话
func (s *Session) AddMessage(role, content string) {
	s.Messages = append(s.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	s.UpdatedAt = time.Now()
}

// sessionPath 获取会话文件路径
func (m *Manager) sessionPath(id string) string {
	return filepath.Join(m.dataDir, id+".json")
}

// generateID 生成唯一ID
func generateID() string {
	return time.Now().Format("20060102-150405") + "-" + randomString(6)
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

// truncateTitle 截断标题
func truncateTitle(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// ExportFormat 导出格式
type ExportFormat int

const (
	FormatMarkdown ExportFormat = iota
	FormatJSON
	FormatText
)

// Export 导出会话
func (s *Session) Export(format ExportFormat) (string, error) {
	switch format {
	case FormatMarkdown:
		return s.exportMarkdown(), nil
	case FormatJSON:
		return s.exportJSON()
	case FormatText:
		return s.exportText(), nil
	default:
		return s.exportMarkdown(), nil
	}
}

// exportMarkdown 导出为Markdown格式
func (s *Session) exportMarkdown() string {
	var b strings.Builder

	// 标题
	b.WriteString("# ")
	if s.Title != "" {
		b.WriteString(s.Title)
	} else {
		b.WriteString("会话记录")
	}
	b.WriteString("\n\n")

	// 元信息
	b.WriteString("## 信息\n\n")
	b.WriteString("- **会话ID**: ")
	b.WriteString(s.ID)
	b.WriteString("\n")
	b.WriteString("- **代理**: ")
	b.WriteString(s.AgentID)
	b.WriteString("\n")
	b.WriteString("- **项目**: ")
	b.WriteString(s.ProjectRoot)
	b.WriteString("\n")
	b.WriteString("- **创建时间**: ")
	b.WriteString(s.CreatedAt.Format("2006-01-02 15:04:05"))
	b.WriteString("\n")
	b.WriteString("- **更新时间**: ")
	b.WriteString(s.UpdatedAt.Format("2006-01-02 15:04:05"))
	b.WriteString("\n\n")

	// 对话内容
	b.WriteString("## 对话\n\n")
	for _, msg := range s.Messages {
		switch msg.Role {
		case "user":
			b.WriteString("### 用户\n\n")
		case "agent":
			b.WriteString("### 助手\n\n")
		case "system":
			b.WriteString("### 系统\n\n")
		case "thinking":
			b.WriteString("### 思考\n\n")
		default:
			b.WriteString("### ")
			b.WriteString(msg.Role)
			b.WriteString("\n\n")
		}
		b.WriteString(msg.Content)
		b.WriteString("\n\n")
		b.WriteString("*")
		b.WriteString(msg.Timestamp.Format("15:04:05"))
		b.WriteString("*\n\n---\n\n")
	}

	return b.String()
}

// exportJSON 导出为JSON格式
func (s *Session) exportJSON() (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// exportText 导出为纯文本格式
func (s *Session) exportText() string {
	var b strings.Builder

	// 标题
	if s.Title != "" {
		b.WriteString(s.Title)
	} else {
		b.WriteString("会话记录")
	}
	b.WriteString("\n")
	b.WriteString(strings.Repeat("=", 50))
	b.WriteString("\n\n")

	// 元信息
	b.WriteString("会话ID: ")
	b.WriteString(s.ID)
	b.WriteString("\n")
	b.WriteString("代理: ")
	b.WriteString(s.AgentID)
	b.WriteString("\n")
	b.WriteString("项目: ")
	b.WriteString(s.ProjectRoot)
	b.WriteString("\n")
	b.WriteString("时间: ")
	b.WriteString(s.CreatedAt.Format("2006-01-02 15:04:05"))
	b.WriteString(" - ")
	b.WriteString(s.UpdatedAt.Format("2006-01-02 15:04:05"))
	b.WriteString("\n\n")
	b.WriteString(strings.Repeat("-", 50))
	b.WriteString("\n\n")

	// 对话内容
	for _, msg := range s.Messages {
		role := msg.Role
		switch role {
		case "user":
			role = "用户"
		case "agent":
			role = "助手"
		case "system":
			role = "系统"
		case "thinking":
			role = "思考"
		}
		b.WriteString("[")
		b.WriteString(msg.Timestamp.Format("15:04:05"))
		b.WriteString("] ")
		b.WriteString(role)
		b.WriteString(":\n")
		b.WriteString(msg.Content)
		b.WriteString("\n\n")
	}

	return b.String()
}

// ExportToFile 导出会话到文件
func (s *Session) ExportToFile(path string, format ExportFormat) error {
	content, err := s.Export(format)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
