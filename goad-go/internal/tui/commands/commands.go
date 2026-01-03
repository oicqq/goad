// Package commands 实现斜杠命令支持
package commands

import (
	"sort"
	"strings"
)

// SlashCommand 斜杠命令
type SlashCommand struct {
	Name        string // 命令名称 (如 /help)
	Description string // 命令描述
	Hint        string // 输入提示
	Handler     func(args string) error
}

// Registry 命令注册表
type Registry struct {
	commands map[string]*SlashCommand
}

// NewRegistry 创建命令注册表
func NewRegistry() *Registry {
	r := &Registry{
		commands: make(map[string]*SlashCommand),
	}
	r.registerBuiltins()
	return r
}

// registerBuiltins 注册内置命令
func (r *Registry) registerBuiltins() {
	// 内置命令
	builtins := []*SlashCommand{
		{Name: "/help", Description: "显示帮助信息"},
		{Name: "/clear", Description: "清除对话历史"},
		{Name: "/exit", Description: "退出程序"},
		{Name: "/settings", Description: "打开设置"},
		{Name: "/model", Description: "切换模型", Hint: "<model_name>"},
		{Name: "/mode", Description: "切换模式", Hint: "<mode_name>"},
		{Name: "/export", Description: "导出对话", Hint: "[filename]"},
		{Name: "/history", Description: "查看历史会话"},
		{Name: "/cancel", Description: "取消当前操作"},
		{Name: "/compact", Description: "压缩对话上下文"},
	}

	for _, cmd := range builtins {
		r.commands[cmd.Name] = cmd
	}
}

// Register 注册命令
func (r *Registry) Register(cmd *SlashCommand) {
	r.commands[cmd.Name] = cmd
}

// RegisterMany 批量注册命令
func (r *Registry) RegisterMany(cmds []*SlashCommand) {
	for _, cmd := range cmds {
		r.commands[cmd.Name] = cmd
	}
}

// Get 获取命令
func (r *Registry) Get(name string) (*SlashCommand, bool) {
	cmd, ok := r.commands[name]
	return cmd, ok
}

// List 列出所有命令
func (r *Registry) List() []*SlashCommand {
	result := make([]*SlashCommand, 0, len(r.commands))
	for _, cmd := range r.commands {
		result = append(result, cmd)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Match 匹配命令 (支持前缀匹配)
func (r *Registry) Match(prefix string) []*SlashCommand {
	if !strings.HasPrefix(prefix, "/") {
		return nil
	}

	var matches []*SlashCommand
	for _, cmd := range r.commands {
		if strings.HasPrefix(cmd.Name, prefix) {
			matches = append(matches, cmd)
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name < matches[j].Name
	})

	return matches
}

// Parse 解析命令行
func Parse(input string) (command string, args string, isCommand bool) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return "", input, false
	}

	parts := strings.SplitN(input, " ", 2)
	command = parts[0]
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}
	return command, args, true
}

// FormatHelp 格式化帮助文本
func (r *Registry) FormatHelp() string {
	var b strings.Builder
	b.WriteString("可用命令:\n\n")

	for _, cmd := range r.List() {
		b.WriteString("  ")
		b.WriteString(cmd.Name)
		if cmd.Hint != "" {
			b.WriteString(" ")
			b.WriteString(cmd.Hint)
		}
		b.WriteString("\n    ")
		b.WriteString(cmd.Description)
		b.WriteString("\n\n")
	}

	return b.String()
}
