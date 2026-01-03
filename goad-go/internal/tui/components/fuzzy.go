// Package components 实现TUI的UI组件
package components

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/goad/internal/tui/styles"
)

// FuzzyFinder 模糊搜索组件
type FuzzyFinder struct {
	items         []FuzzyItem
	filtered      []FuzzyItem
	query         string
	selectedIndex int
	width         int
	height        int
	maxResults    int
	active        bool
}

// FuzzyItem 可搜索项
type FuzzyItem struct {
	Text     string  // 显示文本
	Value    string  // 实际值
	Score    int     // 匹配分数
	Matches  []int   // 匹配位置
	Category string  // 分类
}

// NewFuzzyFinder 创建模糊搜索组件
func NewFuzzyFinder() *FuzzyFinder {
	return &FuzzyFinder{
		items:      make([]FuzzyItem, 0),
		filtered:   make([]FuzzyItem, 0),
		maxResults: 10,
	}
}

// SetItems 设置搜索项
func (f *FuzzyFinder) SetItems(items []FuzzyItem) {
	f.items = items
	f.filter()
}

// AddItem 添加搜索项
func (f *FuzzyFinder) AddItem(item FuzzyItem) {
	f.items = append(f.items, item)
}

// SetQuery 设置搜索查询
func (f *FuzzyFinder) SetQuery(query string) {
	f.query = query
	f.selectedIndex = 0
	f.filter()
}

// AppendQuery 追加查询字符
func (f *FuzzyFinder) AppendQuery(ch rune) {
	f.query += string(ch)
	f.selectedIndex = 0
	f.filter()
}

// DeleteChar 删除查询字符
func (f *FuzzyFinder) DeleteChar() {
	if len(f.query) > 0 {
		f.query = f.query[:len(f.query)-1]
		f.selectedIndex = 0
		f.filter()
	}
}

// ClearQuery 清除查询
func (f *FuzzyFinder) ClearQuery() {
	f.query = ""
	f.selectedIndex = 0
	f.filter()
}

// GetQuery 获取当前查询
func (f *FuzzyFinder) GetQuery() string {
	return f.query
}

// SetSize 设置大小
func (f *FuzzyFinder) SetSize(width, height int) {
	f.width = width
	f.height = height
	f.maxResults = height - 4
	if f.maxResults < 3 {
		f.maxResults = 3
	}
}

// Activate 激活组件
func (f *FuzzyFinder) Activate() {
	f.active = true
	f.selectedIndex = 0
	f.filter()
}

// Deactivate 停用组件
func (f *FuzzyFinder) Deactivate() {
	f.active = false
	f.query = ""
}

// IsActive 是否激活
func (f *FuzzyFinder) IsActive() bool {
	return f.active
}

// MoveUp 向上移动
func (f *FuzzyFinder) MoveUp() {
	if f.selectedIndex > 0 {
		f.selectedIndex--
	}
}

// MoveDown 向下移动
func (f *FuzzyFinder) MoveDown() {
	if f.selectedIndex < len(f.filtered)-1 {
		f.selectedIndex++
	}
}

// GetSelected 获取选中项
func (f *FuzzyFinder) GetSelected() *FuzzyItem {
	if f.selectedIndex >= 0 && f.selectedIndex < len(f.filtered) {
		return &f.filtered[f.selectedIndex]
	}
	return nil
}

// GetFilteredCount 获取筛选结果数量
func (f *FuzzyFinder) GetFilteredCount() int {
	return len(f.filtered)
}

// filter 执行模糊匹配筛选
func (f *FuzzyFinder) filter() {
	if f.query == "" {
		// 无查询时显示全部
		f.filtered = make([]FuzzyItem, len(f.items))
		copy(f.filtered, f.items)
		return
	}

	queryLower := strings.ToLower(f.query)
	f.filtered = make([]FuzzyItem, 0)

	for _, item := range f.items {
		score, matches := fuzzyMatch(queryLower, strings.ToLower(item.Text))
		if score > 0 {
			item.Score = score
			item.Matches = matches
			f.filtered = append(f.filtered, item)
		}
	}

	// 按分数排序
	sort.Slice(f.filtered, func(i, j int) bool {
		return f.filtered[i].Score > f.filtered[j].Score
	})
}

// fuzzyMatch 模糊匹配算法
// 返回分数和匹配位置
func fuzzyMatch(pattern, text string) (int, []int) {
	if pattern == "" {
		return 1, nil
	}
	if text == "" {
		return 0, nil
	}

	patternRunes := []rune(pattern)
	textRunes := []rune(text)

	patternIdx := 0
	matches := make([]int, 0, len(patternRunes))
	score := 0

	// 连续匹配加分
	consecutiveBonus := 0
	// 单词边界加分
	wordBoundaryBonus := 0

	for i, ch := range textRunes {
		if patternIdx < len(patternRunes) && ch == patternRunes[patternIdx] {
			matches = append(matches, i)
			patternIdx++

			// 基础分
			score += 10

			// 连续匹配加分
			if len(matches) > 1 && matches[len(matches)-1]-matches[len(matches)-2] == 1 {
				consecutiveBonus += 5
				score += consecutiveBonus
			} else {
				consecutiveBonus = 0
			}

			// 单词边界加分 (首字母、下划线后、大写字母)
			if i == 0 || (i > 0 && (textRunes[i-1] == '_' || textRunes[i-1] == '/' ||
				textRunes[i-1] == '.' || textRunes[i-1] == '-' ||
				(unicode.IsLower(textRunes[i-1]) && unicode.IsUpper(rune(text[i]))))) {
				wordBoundaryBonus++
				score += 15
			}
		}
	}

	// 未完全匹配
	if patternIdx < len(patternRunes) {
		return 0, nil
	}

	// 前缀匹配加分
	if len(matches) > 0 && matches[0] == 0 {
		score += 20
	}

	// 短文本优先
	score -= len(textRunes) / 10

	return score, matches
}

// View 渲染组件
func (f *FuzzyFinder) View() string {
	if !f.active {
		return ""
	}

	var b strings.Builder
	width := f.width
	if width < 40 {
		width = 40
	}

	// 搜索框
	promptStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	queryStyle := lipgloss.NewStyle().Foreground(styles.TextPrimary)

	b.WriteString(promptStyle.Render("> "))
	b.WriteString(queryStyle.Render(f.query))
	b.WriteString("█") // 光标
	b.WriteString("\n")

	// 结果数量
	countStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	b.WriteString(countStyle.Render(fmt.Sprintf("  %d/%d", len(f.filtered), len(f.items))))
	b.WriteString("\n")

	// 分隔线
	b.WriteString(strings.Repeat("─", width-4))
	b.WriteString("\n")

	// 结果列表
	displayCount := f.maxResults
	if displayCount > len(f.filtered) {
		displayCount = len(f.filtered)
	}

	for i := 0; i < displayCount; i++ {
		item := f.filtered[i]

		cursor := "  "
		if i == f.selectedIndex {
			cursor = "▶ "
		}

		// 渲染带高亮的文本
		text := f.renderHighlighted(item.Text, item.Matches, i == f.selectedIndex)

		// 分类标签
		category := ""
		if item.Category != "" {
			catStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
			category = catStyle.Render(" [" + item.Category + "]")
		}

		b.WriteString(cursor)
		b.WriteString(text)
		b.WriteString(category)
		b.WriteString("\n")
	}

	if len(f.filtered) > f.maxResults {
		b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("  ... 还有 %d 项", len(f.filtered)-f.maxResults)))
		b.WriteString("\n")
	}

	if len(f.filtered) == 0 && f.query != "" {
		b.WriteString(styles.HelpStyle.Render("  无匹配结果"))
		b.WriteString("\n")
	}

	// 帮助
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("↑↓: 选择  Enter: 确认  Esc: 取消"))

	// 边框
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Width(width)

	return boxStyle.Render(b.String())
}

// renderHighlighted 渲染高亮文本
func (f *FuzzyFinder) renderHighlighted(text string, matches []int, selected bool) string {
	if len(matches) == 0 {
		if selected {
			return lipgloss.NewStyle().Foreground(styles.TextPrimary).Bold(true).Render(text)
		}
		return text
	}

	textRunes := []rune(text)
	var result strings.Builder

	matchSet := make(map[int]bool)
	for _, m := range matches {
		matchSet[m] = true
	}

	normalStyle := lipgloss.NewStyle().Foreground(styles.TextSecondary)
	matchStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	if selected {
		normalStyle = lipgloss.NewStyle().Foreground(styles.TextPrimary)
		matchStyle = lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	}

	for i, ch := range textRunes {
		if matchSet[i] {
			result.WriteString(matchStyle.Render(string(ch)))
		} else {
			result.WriteString(normalStyle.Render(string(ch)))
		}
	}

	return result.String()
}

// LoadFilesFromDir 从目录加载文件列表
func (f *FuzzyFinder) LoadFilesFromDir(root string, maxDepth int) error {
	f.items = make([]FuzzyItem, 0)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // 忽略错误继续
		}

		// 跳过隐藏目录和常见的忽略目录
		name := d.Name()
		if strings.HasPrefix(name, ".") ||
			name == "node_modules" ||
			name == "vendor" ||
			name == "__pycache__" ||
			name == "target" ||
			name == "build" ||
			name == "dist" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 计算深度
		relPath, _ := filepath.Rel(root, path)
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 只添加文件
		if !d.IsDir() {
			category := ""
			ext := filepath.Ext(name)
			switch ext {
			case ".go":
				category = "Go"
			case ".py":
				category = "Python"
			case ".js", ".ts", ".jsx", ".tsx":
				category = "JS/TS"
			case ".md":
				category = "Markdown"
			case ".json", ".yaml", ".yml", ".toml":
				category = "Config"
			}

			f.items = append(f.items, FuzzyItem{
				Text:     relPath,
				Value:    path,
				Category: category,
			})
		}

		return nil
	})

	f.filter()
	return err
}
