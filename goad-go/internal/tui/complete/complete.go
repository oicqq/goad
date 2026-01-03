// Package complete 实现路径自动补全
package complete

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PathCompleter 路径补全器
type PathCompleter struct {
	cache map[string][]os.DirEntry
}

// NewPathCompleter 创建路径补全器
func NewPathCompleter() *PathCompleter {
	return &PathCompleter{
		cache: make(map[string][]os.DirEntry),
	}
}

// CompletionResult 补全结果
type CompletionResult struct {
	Prefix      string   // 补全的前缀
	Suggestions []string // 候选列表
}

// ExcludeType 排除类型
type ExcludeType int

const (
	ExcludeNone ExcludeType = iota
	ExcludeFile
	ExcludeDir
)

// Complete 执行路径补全
func (c *PathCompleter) Complete(cwd, path string, exclude ExcludeType) CompletionResult {
	// 处理空路径
	if path == "" {
		return c.listDir(cwd, "", exclude)
	}

	// 展开~
	expandedPath := expandPath(cwd, path)

	// 判断是目录还是文件前缀
	var dirPath, prefix string
	if info, err := os.Stat(expandedPath); err == nil && info.IsDir() {
		dirPath = expandedPath
		prefix = ""
	} else {
		dirPath = filepath.Dir(expandedPath)
		prefix = filepath.Base(expandedPath)
	}

	return c.listDir(dirPath, prefix, exclude)
}

// listDir 列出目录内容并匹配
func (c *PathCompleter) listDir(dirPath, prefix string, exclude ExcludeType) CompletionResult {
	entries, err := c.readDir(dirPath)
	if err != nil {
		return CompletionResult{}
	}

	var matches []string
	for _, entry := range entries {
		name := entry.Name()

		// 排除类型过滤
		if exclude == ExcludeFile && !entry.IsDir() {
			continue
		}
		if exclude == ExcludeDir && entry.IsDir() {
			continue
		}

		// 前缀匹配
		if prefix == "" || strings.HasPrefix(name, prefix) {
			if entry.IsDir() {
				matches = append(matches, name+"/")
			} else {
				matches = append(matches, name)
			}
		}
	}

	sort.Strings(matches)

	if len(matches) == 0 {
		return CompletionResult{}
	}

	// 计算最长公共前缀
	lcp := longestCommonPrefix(matches)

	// 去除已有前缀
	completion := ""
	if len(lcp) > len(prefix) {
		completion = lcp[len(prefix):]
	}

	// 如果只有一个匹配且是目录，确保以/结尾
	if len(matches) == 1 && strings.HasSuffix(matches[0], "/") && !strings.HasSuffix(completion, "/") {
		completion += "/"
	}

	// 生成建议列表 (去除公共前缀)
	var suggestions []string
	for _, m := range matches {
		suffix := m[len(lcp):]
		if suffix != "" {
			suggestions = append(suggestions, suffix)
		}
	}

	return CompletionResult{
		Prefix:      completion,
		Suggestions: suggestions,
	}
}

// readDir 读取目录 (带缓存)
func (c *PathCompleter) readDir(dirPath string) ([]os.DirEntry, error) {
	if entries, ok := c.cache[dirPath]; ok {
		return entries, nil
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	c.cache[dirPath] = entries
	return entries, nil
}

// ClearCache 清除缓存
func (c *PathCompleter) ClearCache() {
	c.cache = make(map[string][]os.DirEntry)
}

// expandPath 展开路径
func expandPath(cwd, path string) string {
	// 展开~
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	// 相对路径转绝对路径
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}

	return filepath.Clean(path)
}

// longestCommonPrefix 计算最长公共前缀
func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	prefix := strs[0]
	for _, s := range strs[1:] {
		for len(prefix) > 0 && !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
		if prefix == "" {
			return ""
		}
	}
	return prefix
}

// CommandCompleter 命令补全器
type CommandCompleter struct {
	pathDirs []string
	commands map[string]bool
}

// NewCommandCompleter 创建命令补全器
func NewCommandCompleter() *CommandCompleter {
	c := &CommandCompleter{
		commands: make(map[string]bool),
	}
	c.loadPath()
	return c
}

// loadPath 加载PATH中的命令
func (c *CommandCompleter) loadPath() {
	pathEnv := os.Getenv("PATH")
	c.pathDirs = filepath.SplitList(pathEnv)

	for _, dir := range c.pathDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			// 检查是否可执行
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.Mode()&0111 != 0 {
				c.commands[entry.Name()] = true
			}
		}
	}
}

// Complete 补全命令
func (c *CommandCompleter) Complete(prefix string) []string {
	var matches []string
	for cmd := range c.commands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	sort.Strings(matches)
	return matches
}

// HasCommand 检查命令是否存在
func (c *CommandCompleter) HasCommand(name string) bool {
	return c.commands[name]
}
