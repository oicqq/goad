// Package highlight 提供代码语法高亮功能
package highlight

import (
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Highlighter 语法高亮器
type Highlighter struct {
	formatter chroma.Formatter
	style     *chroma.Style
}

// New 创建新的语法高亮器
func New() *Highlighter {
	// 使用适合终端的格式化器
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// 使用 monokai 主题，适合深色终端
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	return &Highlighter{
		formatter: formatter,
		style:     style,
	}
}

// SetStyle 设置高亮风格
func (h *Highlighter) SetStyle(name string) {
	if style := styles.Get(name); style != nil {
		h.style = style
	}
}

// Highlight 高亮代码
func (h *Highlighter) Highlight(code, language string) string {
	lexer := h.getLexer(language)
	if lexer == nil {
		return code
	}

	// 分词
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	// 格式化
	var b strings.Builder
	if err := h.formatter.Format(&b, h.style, iterator); err != nil {
		return code
	}

	return b.String()
}

// HighlightFile 根据文件路径高亮代码
func (h *Highlighter) HighlightFile(code, filePath string) string {
	language := DetectLanguage(filePath)
	return h.Highlight(code, language)
}

// getLexer 获取语言的词法分析器
func (h *Highlighter) getLexer(language string) chroma.Lexer {
	if language == "" {
		return nil
	}

	// 尝试按语言名获取
	lexer := lexers.Get(language)
	if lexer != nil {
		return chroma.Coalesce(lexer)
	}

	return nil
}

// DetectLanguage 根据文件扩展名检测语言
func DetectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	// 常见扩展名映射
	extMap := map[string]string{
		".go":    "go",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".jsx":   "jsx",
		".tsx":   "tsx",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".cc":    "cpp",
		".h":     "c",
		".hpp":   "cpp",
		".rs":    "rust",
		".rb":    "ruby",
		".php":   "php",
		".swift": "swift",
		".kt":    "kotlin",
		".scala": "scala",
		".cs":    "csharp",
		".sh":    "bash",
		".bash":  "bash",
		".zsh":   "bash",
		".ps1":   "powershell",
		".sql":   "sql",
		".json":  "json",
		".yaml":  "yaml",
		".yml":   "yaml",
		".toml":  "toml",
		".xml":   "xml",
		".html":  "html",
		".htm":   "html",
		".css":   "css",
		".scss":  "scss",
		".less":  "less",
		".md":    "markdown",
		".lua":   "lua",
		".vim":   "vim",
		".dockerfile": "docker",
		".makefile":   "make",
	}

	if lang, ok := extMap[ext]; ok {
		return lang
	}

	// 特殊文件名检测
	baseName := strings.ToLower(filepath.Base(filePath))
	nameMap := map[string]string{
		"dockerfile":  "docker",
		"makefile":    "make",
		"cmakelists.txt": "cmake",
		"jenkinsfile": "groovy",
		".gitignore":  "gitignore",
		".env":        "shell",
	}

	if lang, ok := nameMap[baseName]; ok {
		return lang
	}

	return ""
}

// HighlightDiff 高亮差异显示
func (h *Highlighter) HighlightDiff(diff string) string {
	lexer := lexers.Get("diff")
	if lexer == nil {
		return diff
	}

	iterator, err := lexer.Tokenise(nil, diff)
	if err != nil {
		return diff
	}

	var b strings.Builder
	if err := h.formatter.Format(&b, h.style, iterator); err != nil {
		return diff
	}

	return b.String()
}

// HighlightInline 高亮行内代码块
// 从 markdown 代码块提取语言和代码并高亮
func (h *Highlighter) HighlightInline(text string) string {
	// 简单实现：查找 ```language\ncode\n``` 模式
	result := text

	// 处理围栏代码块
	for {
		start := strings.Index(result, "```")
		if start == -1 {
			break
		}

		// 找到结束标记
		restAfterStart := result[start+3:]
		end := strings.Index(restAfterStart, "```")
		if end == -1 {
			break
		}

		// 提取代码块内容
		codeBlock := restAfterStart[:end]

		// 分离语言和代码
		lines := strings.SplitN(codeBlock, "\n", 2)
		language := strings.TrimSpace(lines[0])
		code := ""
		if len(lines) > 1 {
			code = lines[1]
		}

		// 高亮代码
		highlighted := h.Highlight(code, language)

		// 替换原始代码块
		fullMatch := result[start : start+3+end+3]
		replacement := "\n" + highlighted + "\n"
		result = strings.Replace(result, fullMatch, replacement, 1)
	}

	return result
}
