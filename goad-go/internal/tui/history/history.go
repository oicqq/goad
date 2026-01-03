// Package history 实现命令历史记录
package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry 历史记录条目
type Entry struct {
	Input     string  `json:"input"`
	Timestamp float64 `json:"timestamp"`
}

// History 历史记录管理器
type History struct {
	path     string
	entries  []Entry
	current  string // 当前输入 (未提交)
	position int    // 浏览位置 (0=最新, 负数=历史)
	opened   bool
	mu       sync.RWMutex
}

// New 创建历史记录管理器
func New(path string) *History {
	return &History{
		path:     path,
		entries:  make([]Entry, 0),
		position: 0,
	}
}

// DefaultPath 获取默认历史文件路径
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "goad")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.jsonl"), nil
}

// Open 打开历史文件
func (h *History) Open() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.opened {
		return nil
	}

	// 确保目录存在
	dir := filepath.Dir(h.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 创建文件(如果不存在)
	file, err := os.OpenFile(h.path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 读取历史记录
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		h.entries = append(h.entries, entry)
	}

	h.opened = true
	return nil
}

// Append 添加历史记录
func (h *History) Append(input string) error {
	if input == "" {
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// 确保已打开
	if !h.opened {
		h.mu.Unlock()
		if err := h.Open(); err != nil {
			h.mu.Lock()
			return err
		}
		h.mu.Lock()
	}

	entry := Entry{
		Input:     input,
		Timestamp: float64(time.Now().Unix()),
	}

	// 添加到内存
	h.entries = append(h.entries, entry)

	// 写入文件
	file, err := os.OpenFile(h.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = file.WriteString(string(data) + "\n")

	// 重置位置
	h.position = 0
	h.current = ""

	return err
}

// SetCurrent 设置当前输入
func (h *History) SetCurrent(input string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.current = input
}

// GetCurrent 获取当前输入
func (h *History) GetCurrent() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.current
}

// Previous 获取上一条历史
func (h *History) Previous() (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.entries) == 0 {
		return "", false
	}

	// 首次按上，保存当前输入
	if h.position == 0 {
		// 保存当前输入
	}

	// 向前移动
	newPos := h.position - 1
	if -newPos > len(h.entries) {
		return "", false
	}

	h.position = newPos
	return h.entries[len(h.entries)+h.position].Input, true
}

// Next 获取下一条历史
func (h *History) Next() (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.position >= 0 {
		return h.current, false
	}

	h.position++
	if h.position == 0 {
		return h.current, true
	}

	return h.entries[len(h.entries)+h.position].Input, true
}

// Reset 重置浏览位置
func (h *History) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.position = 0
}

// Size 获取历史记录数量
func (h *History) Size() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.entries)
}

// GetEntry 获取指定位置的记录
// index: 0=最新, 负数=历史
func (h *History) GetEntry(index int) (Entry, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if index > 0 {
		return Entry{}, false
	}
	if index == 0 {
		return Entry{Input: h.current, Timestamp: float64(time.Now().Unix())}, true
	}

	pos := len(h.entries) + index
	if pos < 0 {
		return Entry{}, false
	}

	return h.entries[pos], true
}

// List 列出最近的历史记录
func (h *History) List(limit int) []Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if limit <= 0 || limit > len(h.entries) {
		limit = len(h.entries)
	}

	start := len(h.entries) - limit
	result := make([]Entry, limit)
	copy(result, h.entries[start:])

	// 反转顺序 (最新的在前)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// Search 搜索历史记录
func (h *History) Search(prefix string) []Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var matches []Entry
	for i := len(h.entries) - 1; i >= 0; i-- {
		entry := h.entries[i]
		if len(entry.Input) >= len(prefix) && entry.Input[:len(prefix)] == prefix {
			matches = append(matches, entry)
		}
	}

	return matches
}

// Clear 清除历史记录
func (h *History) Clear() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = make([]Entry, 0)
	h.position = 0
	h.current = ""

	// 清空文件
	return os.WriteFile(h.path, []byte{}, 0644)
}
