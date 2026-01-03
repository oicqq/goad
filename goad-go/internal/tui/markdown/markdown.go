// Package markdown 提供Markdown渲染功能
package markdown

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

// Renderer Markdown渲染器
type Renderer struct {
	renderer *glamour.TermRenderer
	width    int
}

// New 创建新的Markdown渲染器
func New(width int) *Renderer {
	if width <= 0 {
		width = 80
	}

	// 创建暗色主题渲染器
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		// 回退到简单渲染器
		renderer, _ = glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(width),
		)
	}

	return &Renderer{
		renderer: renderer,
		width:    width,
	}
}

// SetWidth 设置渲染宽度
func (r *Renderer) SetWidth(width int) {
	if width <= 0 {
		return
	}
	r.width = width

	// 重新创建渲染器
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err == nil {
		r.renderer = renderer
	}
}

// Render 渲染Markdown内容
func (r *Renderer) Render(content string) string {
	if r.renderer == nil {
		return content
	}

	rendered, err := r.renderer.Render(content)
	if err != nil {
		return content
	}

	// 移除尾部多余的空行
	return strings.TrimRight(rendered, "\n")
}

// RenderWithStyle 使用指定风格渲染Markdown
func (r *Renderer) RenderWithStyle(content, style string) string {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath(style),
		glamour.WithWordWrap(r.width),
	)
	if err != nil {
		return r.Render(content)
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}

	return strings.TrimRight(rendered, "\n")
}

// HasMarkdown 检测内容是否包含Markdown语法
func HasMarkdown(content string) bool {
	// 检测常见的Markdown标记
	markers := []string{
		"```",     // 代码块
		"# ",      // 标题
		"## ",     // 二级标题
		"### ",    // 三级标题
		"- ",      // 列表
		"* ",      // 列表
		"1. ",     // 有序列表
		"**",      // 粗体
		"__",      // 粗体
		"*",       // 斜体
		"_",       // 斜体
		"[",       // 链接
		"> ",      // 引用
		"|",       // 表格
	}

	for _, marker := range markers {
		if strings.Contains(content, marker) {
			return true
		}
	}

	return false
}

// StripMarkdown 移除Markdown语法，返回纯文本
func StripMarkdown(content string) string {
	// 简单的Markdown剥离
	result := content

	// 移除代码块
	for {
		start := strings.Index(result, "```")
		if start == -1 {
			break
		}
		end := strings.Index(result[start+3:], "```")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+3:start+3+end] + result[start+3+end+3:]
	}

	// 移除行内代码
	result = strings.ReplaceAll(result, "`", "")

	// 移除粗体
	result = strings.ReplaceAll(result, "**", "")
	result = strings.ReplaceAll(result, "__", "")

	// 移除标题标记
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		line = strings.TrimLeft(line, "#")
		line = strings.TrimLeft(line, " ")
		lines[i] = line
	}
	result = strings.Join(lines, "\n")

	return result
}
