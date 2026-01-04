// Package components 实现TUI的UI组件
package components

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/goad/internal/tui/styles"
)

// CompletionType 补全类型
type CompletionType int

const (
	CompletionNone CompletionType = iota
	CompletionCommand             // 斜杠命令
	CompletionPath                // 文件路径
	CompletionHistory             // 历史记录
)

// Completion 补全项
type Completion struct {
	Text        string         // 补全文本
	Display     string         // 显示文本
	Description string         // 描述
	Type        CompletionType // 类型
}

// AutoComplete 自动补全组件
type AutoComplete struct {
	completions   []Completion
	selectedIndex int
	active        bool
	width         int
	maxDisplay    int

	// 补全源
	commands     []Completion // 斜杠命令
	history      []string     // 历史记录
	projectRoot  string       // 项目根目录
	pathCache    map[string][]string
	cacheMaxSize int
}

// NewAutoComplete 创建自动补全组件
func NewAutoComplete() *AutoComplete {
	return &AutoComplete{
		completions:  make([]Completion, 0),
		commands:     make([]Completion, 0),
		history:      make([]string, 0),
		pathCache:    make(map[string][]string),
		cacheMaxSize: 100,
		maxDisplay:   8,
		width:        60,
	}
}

// SetWidth 设置宽度
func (a *AutoComplete) SetWidth(w int) {
	a.width = w
}

// SetProjectRoot 设置项目根目录
func (a *AutoComplete) SetProjectRoot(root string) {
	a.projectRoot = root
}

// SetCommands 设置可用命令
func (a *AutoComplete) SetCommands(commands []Completion) {
	a.commands = commands
}

// AddCommand 添加命令
func (a *AutoComplete) AddCommand(cmd, description string) {
	a.commands = append(a.commands, Completion{
		Text:        cmd,
		Display:     cmd,
		Description: description,
		Type:        CompletionCommand,
	})
}

// SetHistory 设置历史记录
func (a *AutoComplete) SetHistory(history []string) {
	a.history = history
}

// AddHistory 添加历史记录
func (a *AutoComplete) AddHistory(item string) {
	// 去重
	for i, h := range a.history {
		if h == item {
			// 移到最前
			a.history = append(a.history[:i], a.history[i+1:]...)
			break
		}
	}
	a.history = append([]string{item}, a.history...)

	// 限制大小
	if len(a.history) > 100 {
		a.history = a.history[:100]
	}
}

// Update 根据输入更新补全列表
func (a *AutoComplete) Update(input string) {
	a.completions = make([]Completion, 0)
	a.selectedIndex = 0

	if input == "" {
		a.active = false
		return
	}

	// 检测补全类型并生成补全列表
	if strings.HasPrefix(input, "/") {
		// 斜杠命令补全
		a.completeCommands(input)
	} else if a.containsPath(input) {
		// 路径补全
		a.completePaths(input)
	} else {
		// 历史记录补全
		a.completeHistory(input)
	}

	a.active = len(a.completions) > 0
}

// containsPath 检测输入是否包含路径
func (a *AutoComplete) containsPath(input string) bool {
	// 检测常见路径模式
	if strings.Contains(input, "/") || strings.Contains(input, "./") || strings.Contains(input, "~/") {
		return true
	}
	// 检测是否以常见路径命令开头
	pathCommands := []string{"cd ", "cat ", "ls ", "vim ", "nano ", "code ", "open "}
	for _, cmd := range pathCommands {
		if strings.HasPrefix(input, cmd) {
			return true
		}
	}
	return false
}

// completeCommands 补全斜杠命令
func (a *AutoComplete) completeCommands(input string) {
	inputLower := strings.ToLower(input)

	for _, cmd := range a.commands {
		if strings.HasPrefix(strings.ToLower(cmd.Text), inputLower) {
			a.completions = append(a.completions, cmd)
		}
	}

	// 按名称排序
	sort.Slice(a.completions, func(i, j int) bool {
		return a.completions[i].Text < a.completions[j].Text
	})
}

// completePaths 补全文件路径
func (a *AutoComplete) completePaths(input string) {
	// 提取路径部分
	pathStart := a.findPathStart(input)
	pathPart := input[pathStart:]

	// 展开 ~
	expandedPath := pathPart
	homeDir, _ := os.UserHomeDir()
	if strings.HasPrefix(pathPart, "~/") {
		expandedPath = filepath.Join(homeDir, pathPart[2:])
	} else if pathPart == "~" {
		expandedPath = homeDir
	}

	// 获取目录和前缀
	dir := filepath.Dir(expandedPath)
	prefix := filepath.Base(expandedPath)

	// 如果路径以 / 结尾，则在该目录下搜索
	if strings.HasSuffix(pathPart, "/") {
		dir = expandedPath
		prefix = ""
	}

	// 相对路径处理
	if !filepath.IsAbs(dir) && a.projectRoot != "" {
		dir = filepath.Join(a.projectRoot, dir)
	}

	// 列出目录内容
	entries := a.listDir(dir)
	prefixLower := strings.ToLower(prefix)

	for _, entry := range entries {
		if prefix == "" || strings.HasPrefix(strings.ToLower(entry), prefixLower) {
			fullPath := filepath.Join(dir, entry)
			info, err := os.Stat(fullPath)
			if err != nil {
				continue
			}

			displayPath := entry
			if info.IsDir() {
				displayPath += "/"
			}

			// 构建补全文本
			completionText := input[:pathStart] + filepath.Join(filepath.Dir(pathPart), entry)
			if info.IsDir() {
				completionText += "/"
			}

			desc := "file"
			if info.IsDir() {
				desc = "directory"
			}

			a.completions = append(a.completions, Completion{
				Text:        completionText,
				Display:     displayPath,
				Description: desc,
				Type:        CompletionPath,
			})
		}
	}

	// 目录优先，然后按名称排序
	sort.Slice(a.completions, func(i, j int) bool {
		iIsDir := strings.HasSuffix(a.completions[i].Display, "/")
		jIsDir := strings.HasSuffix(a.completions[j].Display, "/")
		if iIsDir != jIsDir {
			return iIsDir
		}
		return a.completions[i].Display < a.completions[j].Display
	})
}

// findPathStart 找到路径开始位置
func (a *AutoComplete) findPathStart(input string) int {
	// 从后向前找空格或特殊字符
	for i := len(input) - 1; i >= 0; i-- {
		ch := input[i]
		if ch == ' ' || ch == '"' || ch == '\'' || ch == '=' || ch == ':' {
			return i + 1
		}
	}
	return 0
}

// listDir 列出目录内容（带缓存）
func (a *AutoComplete) listDir(dir string) []string {
	// 检查缓存
	if entries, ok := a.pathCache[dir]; ok {
		return entries
	}

	entries := make([]string, 0)
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return entries
	}

	for _, entry := range dirEntries {
		name := entry.Name()
		// 跳过隐藏文件（除非明确输入.）
		if !strings.HasPrefix(name, ".") {
			entries = append(entries, name)
		}
	}

	// 更新缓存
	if len(a.pathCache) >= a.cacheMaxSize {
		// 简单清理：删除一半
		count := 0
		for k := range a.pathCache {
			if count >= a.cacheMaxSize/2 {
				break
			}
			delete(a.pathCache, k)
			count++
		}
	}
	a.pathCache[dir] = entries

	return entries
}

// completeHistory 补全历史记录
func (a *AutoComplete) completeHistory(input string) {
	inputLower := strings.ToLower(input)

	for _, h := range a.history {
		if strings.Contains(strings.ToLower(h), inputLower) && h != input {
			a.completions = append(a.completions, Completion{
				Text:        h,
				Display:     truncateString(h, a.width-10),
				Description: "history",
				Type:        CompletionHistory,
			})
		}

		// 限制数量
		if len(a.completions) >= a.maxDisplay {
			break
		}
	}
}

// truncateString 截断字符串
func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// IsActive 是否激活
func (a *AutoComplete) IsActive() bool {
	return a.active
}

// Deactivate 停用
func (a *AutoComplete) Deactivate() {
	a.active = false
	a.completions = nil
}

// MoveUp 向上移动
func (a *AutoComplete) MoveUp() {
	if a.selectedIndex > 0 {
		a.selectedIndex--
	}
}

// MoveDown 向下移动
func (a *AutoComplete) MoveDown() {
	if a.selectedIndex < len(a.completions)-1 {
		a.selectedIndex++
	}
}

// GetSelected 获取选中项
func (a *AutoComplete) GetSelected() *Completion {
	if a.selectedIndex >= 0 && a.selectedIndex < len(a.completions) {
		return &a.completions[a.selectedIndex]
	}
	return nil
}

// GetCompletionCount 获取补全数量
func (a *AutoComplete) GetCompletionCount() int {
	return len(a.completions)
}

// Accept 接受当前选择
func (a *AutoComplete) Accept() string {
	if comp := a.GetSelected(); comp != nil {
		a.active = false
		return comp.Text
	}
	return ""
}

// View 渲染组件
func (a *AutoComplete) View() string {
	if !a.active || len(a.completions) == 0 {
		return ""
	}

	var b strings.Builder
	width := a.width
	if width < 40 {
		width = 40
	}

	displayCount := a.maxDisplay
	if displayCount > len(a.completions) {
		displayCount = len(a.completions)
	}

	// 计算滚动偏移
	offset := 0
	if a.selectedIndex >= displayCount {
		offset = a.selectedIndex - displayCount + 1
	}

	for i := 0; i < displayCount; i++ {
		idx := i + offset
		if idx >= len(a.completions) {
			break
		}

		comp := a.completions[idx]
		isSelected := idx == a.selectedIndex

		// 样式
		itemStyle := lipgloss.NewStyle().Foreground(styles.TextSecondary)
		descStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)

		if isSelected {
			itemStyle = lipgloss.NewStyle().
				Foreground(styles.TextPrimary).
				Background(styles.Primary).
				Bold(true)
			descStyle = lipgloss.NewStyle().
				Foreground(styles.TextSecondary).
				Background(styles.Primary)
		}

		// 类型图标
		icon := ""
		switch comp.Type {
		case CompletionCommand:
			icon = "/"
		case CompletionPath:
			if strings.HasSuffix(comp.Display, "/") {
				icon = "D"
			} else {
				icon = "F"
			}
		case CompletionHistory:
			icon = "H"
		}

		// 构建行
		display := comp.Display
		maxDisplayLen := width - 15
		if len(display) > maxDisplayLen {
			display = display[:maxDisplayLen-3] + "..."
		}

		line := itemStyle.Render(icon+" "+display) +
			" " +
			descStyle.Render(comp.Description)

		b.WriteString(line)
		b.WriteString("\n")
	}

	// 显示更多提示
	if len(a.completions) > a.maxDisplay {
		moreStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
		b.WriteString(moreStyle.Render("  ... 还有更多"))
	}

	// 边框样式
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.TextMuted).
		Padding(0, 1).
		Width(width)

	return boxStyle.Render(b.String())
}

// ClearCache 清除路径缓存
func (a *AutoComplete) ClearCache() {
	a.pathCache = make(map[string][]string)
}
