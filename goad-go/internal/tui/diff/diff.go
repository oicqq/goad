// Package diff 提供代码差异显示功能
package diff

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	dmp "github.com/sergi/go-diff/diffmatchpatch"
)

// Renderer 差异渲染器
type Renderer struct {
	dmp         *dmp.DiffMatchPatch
	addStyle    lipgloss.Style
	removeStyle lipgloss.Style
	equalStyle  lipgloss.Style
	lineNumStyle lipgloss.Style
}

// New 创建新的差异渲染器
func New() *Renderer {
	return &Renderer{
		dmp:         dmp.New(),
		addStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e22e")).Background(lipgloss.Color("#2d3c2d")),
		removeStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#f92672")).Background(lipgloss.Color("#3c2d2d")),
		equalStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#808080")),
		lineNumStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Width(4),
	}
}

// RenderDiff 渲染差异
func (r *Renderer) RenderDiff(oldText, newText string) string {
	// 计算差异
	diffs := r.dmp.DiffMain(oldText, newText, true)
	diffs = r.dmp.DiffCleanupSemantic(diffs)

	var b strings.Builder
	oldLineNum := 1
	newLineNum := 1

	for _, diff := range diffs {
		lines := strings.Split(diff.Text, "\n")
		for i, line := range lines {
			// 跳过最后一个空行（由split产生）
			if i == len(lines)-1 && line == "" {
				continue
			}

			switch diff.Type {
			case dmp.DiffDelete:
				lineNum := r.lineNumStyle.Render(strings.Repeat(" ", 4))
				oldLineNumStr := r.lineNumStyle.Render(pad(oldLineNum, 4))
				b.WriteString(oldLineNumStr + lineNum + " ")
				b.WriteString(r.removeStyle.Render("- " + line))
				oldLineNum++
			case dmp.DiffInsert:
				lineNum := r.lineNumStyle.Render(strings.Repeat(" ", 4))
				newLineNumStr := r.lineNumStyle.Render(pad(newLineNum, 4))
				b.WriteString(lineNum + newLineNumStr + " ")
				b.WriteString(r.addStyle.Render("+ " + line))
				newLineNum++
			case dmp.DiffEqual:
				oldLineNumStr := r.lineNumStyle.Render(pad(oldLineNum, 4))
				newLineNumStr := r.lineNumStyle.Render(pad(newLineNum, 4))
				b.WriteString(oldLineNumStr + newLineNumStr + " ")
				b.WriteString(r.equalStyle.Render("  " + line))
				oldLineNum++
				newLineNum++
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// RenderUnifiedDiff 渲染统一差异格式
func (r *Renderer) RenderUnifiedDiff(oldText, newText, path string) string {
	// 使用行模式差异
	a, b, lineArray := r.dmp.DiffLinesToChars(oldText, newText)
	diffs := r.dmp.DiffMain(a, b, false)
	diffs = r.dmp.DiffCharsToLines(diffs, lineArray)
	diffs = r.dmp.DiffCleanupSemantic(diffs)

	var result strings.Builder

	// 头部
	result.WriteString(r.removeStyle.Render("--- " + path))
	result.WriteString("\n")
	result.WriteString(r.addStyle.Render("+++ " + path))
	result.WriteString("\n")

	oldLineNum := 1
	newLineNum := 1
	contextLines := 3 // 上下文行数

	// 收集变更区块
	type chunk struct {
		oldStart, oldCount int
		newStart, newCount int
		lines              []struct {
			diffType dmp.Operation
			text     string
		}
	}

	var chunks []chunk
	var currentChunk *chunk

	for _, diff := range diffs {
		lines := strings.Split(strings.TrimSuffix(diff.Text, "\n"), "\n")
		for _, line := range lines {
			if diff.Type != dmp.DiffEqual {
				// 开始新的变更区块
				if currentChunk == nil {
					currentChunk = &chunk{
						oldStart: max(1, oldLineNum-contextLines),
						newStart: max(1, newLineNum-contextLines),
					}
				}
			}

			if currentChunk != nil {
				currentChunk.lines = append(currentChunk.lines, struct {
					diffType dmp.Operation
					text     string
				}{diff.Type, line})

				switch diff.Type {
				case dmp.DiffDelete:
					currentChunk.oldCount++
				case dmp.DiffInsert:
					currentChunk.newCount++
				case dmp.DiffEqual:
					currentChunk.oldCount++
					currentChunk.newCount++
				}
			}

			switch diff.Type {
			case dmp.DiffDelete:
				oldLineNum++
			case dmp.DiffInsert:
				newLineNum++
			case dmp.DiffEqual:
				oldLineNum++
				newLineNum++
			}
		}
	}

	if currentChunk != nil {
		chunks = append(chunks, *currentChunk)
	}

	// 渲染区块
	for _, c := range chunks {
		// 区块头
		header := lipgloss.NewStyle().Foreground(lipgloss.Color("#66d9ef")).Render(
			strings.Repeat(" ", 0) + "@@ " +
			formatRange(c.oldStart, c.oldCount) + " " +
			formatRange(c.newStart, c.newCount) + " @@",
		)
		result.WriteString(header)
		result.WriteString("\n")

		// 区块内容
		for _, l := range c.lines {
			switch l.diffType {
			case dmp.DiffDelete:
				result.WriteString(r.removeStyle.Render("-" + l.text))
			case dmp.DiffInsert:
				result.WriteString(r.addStyle.Render("+" + l.text))
			case dmp.DiffEqual:
				result.WriteString(r.equalStyle.Render(" " + l.text))
			}
			result.WriteString("\n")
		}
	}

	return result.String()
}

// RenderInlineDiff 渲染行内差异（字符级别）
func (r *Renderer) RenderInlineDiff(oldText, newText string) string {
	diffs := r.dmp.DiffMain(oldText, newText, true)
	diffs = r.dmp.DiffCleanupSemantic(diffs)

	var b strings.Builder
	for _, diff := range diffs {
		switch diff.Type {
		case dmp.DiffDelete:
			b.WriteString(r.removeStyle.Render(diff.Text))
		case dmp.DiffInsert:
			b.WriteString(r.addStyle.Render(diff.Text))
		case dmp.DiffEqual:
			b.WriteString(diff.Text)
		}
	}
	return b.String()
}

// pad 数字左填充
func pad(n, width int) string {
	s := strings.Repeat(" ", width)
	ns := string(rune('0') + rune(n%10))
	if n >= 10 {
		ns = string(rune('0')+rune((n/10)%10)) + ns
	}
	if n >= 100 {
		ns = string(rune('0')+rune((n/100)%10)) + ns
	}
	if n >= 1000 {
		ns = string(rune('0')+rune((n/1000)%10)) + ns
	}
	if len(ns) >= width {
		return ns
	}
	return s[:width-len(ns)] + ns
}

func formatRange(start, count int) string {
	if count == 1 {
		return strings.Repeat(" ", 0) + "-" + pad(start, 1)
	}
	return strings.Repeat(" ", 0) + "-" + pad(start, 1) + "," + pad(count, 1)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
