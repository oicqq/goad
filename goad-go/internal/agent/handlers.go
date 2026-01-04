// Package agent 实现了Agent的RPC处理方法
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/anthropics/goad/internal/acp"
)

// handleSessionUpdate 处理会话更新
func (a *Agent) handleSessionUpdate(updateData json.RawMessage) (interface{}, error) {
	msg, err := acp.ParseSessionUpdate(updateData)
	if err != nil {
		return nil, err
	}

	if msg != nil {
		// 处理工具调用追踪
		switch m := msg.(type) {
		case *acp.ToolCallMessage:
			a.toolCallsMu.Lock()
			a.toolCalls[m.ToolCall.ToolCallID] = m.ToolCall
			a.toolCallsMu.Unlock()

		case *acp.ToolCallUpdateMessage:
			a.toolCallsMu.Lock()
			if existing, ok := a.toolCalls[m.Update.ToolCallID]; ok {
				// 合并更新
				if m.Update.Title != "" {
					existing.Title = m.Update.Title
				}
				if m.Update.Status != "" {
					existing.Status = m.Update.Status
				}
				if m.Update.Kind != "" {
					existing.Kind = m.Update.Kind
				}
				if len(m.Update.Content) > 0 {
					existing.Content = m.Update.Content
				}
				m.ToolCall = existing
			}
			a.toolCallsMu.Unlock()
		}

		// 发送消息到UI
		select {
		case a.messages <- msg:
		default:
			// 通道满了，丢弃消息
		}
	}

	return nil, nil
}

// handleRequestPermission 处理权限请求
func (a *Agent) handleRequestPermission(p *acp.RequestPermissionParams) (*acp.RequestPermissionResponse, error) {
	// 获取或创建工具调用
	a.toolCallsMu.Lock()
	toolCall, ok := a.toolCalls[p.ToolCall.ToolCallID]
	if !ok {
		// 创建新的工具调用
		toolCall = &acp.ToolCall{
			SessionUpdate: "tool_call",
			ToolCallID:    p.ToolCall.ToolCallID,
			Title:         p.ToolCall.Title,
			Kind:          p.ToolCall.Kind,
			Status:        p.ToolCall.Status,
			Content:       p.ToolCall.Content,
		}
		a.toolCalls[p.ToolCall.ToolCallID] = toolCall
	}
	a.toolCallsMu.Unlock()

	// 创建响应通道
	responseCh := make(chan acp.PermissionResponse, 1)

	// 发送权限请求消息
	msg := &acp.PermissionRequestMessage{
		Options:    p.Options,
		ToolCall:   toolCall,
		ResponseCh: responseCh,
	}

	select {
	case a.messages <- msg:
	case <-a.ctx.Done():
		return nil, a.ctx.Err()
	}

	// 等待响应
	select {
	case resp := <-responseCh:
		return &acp.RequestPermissionResponse{
			Outcome: acp.RequestPermissionOutcome{
				OptionID: resp.OptionID,
				Outcome:  resp.Outcome,
			},
		}, nil
	case <-a.ctx.Done():
		return nil, a.ctx.Err()
	}
}

// handleReadTextFile 处理读取文件请求
func (a *Agent) handleReadTextFile(p *acp.ReadTextFileParams) (*acp.ReadTextFileResponse, error) {
	// 处理路径 - 支持绝对路径和相对路径
	var fullPath string
	if filepath.IsAbs(p.Path) {
		fullPath = filepath.Clean(p.Path)
	} else {
		fullPath = filepath.Join(a.projectRoot, p.Path)
	}

	// 安全检查：确保路径在项目根目录内（可选，暂时放宽限制）
	// if !strings.HasPrefix(fullPath, a.projectRoot) {
	// 	return nil, fmt.Errorf("路径不在项目目录内")
	// }

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return &acp.ReadTextFileResponse{Content: ""}, nil
	}

	text := string(content)

	// 处理行号和限制
	if p.Line != nil {
		lines := strings.Split(text, "\n")
		start := *p.Line - 1
		if start < 0 {
			start = 0
		}
		if start >= len(lines) {
			text = ""
		} else {
			end := len(lines)
			if p.Limit != nil {
				end = start + *p.Limit
				if end > len(lines) {
					end = len(lines)
				}
			}
			text = strings.Join(lines[start:end], "\n")
		}
	}

	return &acp.ReadTextFileResponse{Content: text}, nil
}

// handleWriteTextFile 处理写入文件请求
func (a *Agent) handleWriteTextFile(p *acp.WriteTextFileParams) (interface{}, error) {
	// 处理路径 - 支持绝对路径和相对路径
	var fullPath string
	if filepath.IsAbs(p.Path) {
		fullPath = filepath.Clean(p.Path)
	} else {
		fullPath = filepath.Join(a.projectRoot, p.Path)
	}

	// 确保目录存在
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	if err := os.WriteFile(fullPath, []byte(p.Content), 0644); err != nil {
		return nil, fmt.Errorf("写入文件失败: %w", err)
	}

	return nil, nil
}

// handleCreateTerminal 处理创建终端请求
func (a *Agent) handleCreateTerminal(p *acp.CreateTerminalParams) (*acp.CreateTerminalResponse, error) {
	a.terminalsMu.Lock()
	defer a.terminalsMu.Unlock()

	// 生成终端ID
	a.terminalID++
	terminalID := fmt.Sprintf("terminal-%d", a.terminalID)

	// 设置工作目录
	workDir := p.Cwd
	if workDir == "" {
		workDir = a.projectRoot
	}
	// 支持绝对路径
	if !filepath.IsAbs(workDir) {
		workDir = filepath.Join(a.projectRoot, workDir)
	}

	// 构建完整命令
	fullCommand := p.Command
	if len(p.Args) > 0 {
		for _, arg := range p.Args {
			// 对参数进行引用处理
			if strings.Contains(arg, " ") || strings.Contains(arg, "\"") {
				fullCommand += " \"" + strings.ReplaceAll(arg, "\"", "\\\"") + "\""
			} else {
				fullCommand += " " + arg
			}
		}
	}

	// 构建环境变量导出
	var envExports strings.Builder
	for _, ev := range p.Env {
		envExports.WriteString(fmt.Sprintf("export %s=%q; ", ev.Name, ev.Value))
	}

	// 使用 bash -c 直接执行命令（命令完成后自动退出）
	// 这样不会创建交互式 shell，命令执行完就会退出
	shellCmd := "/bin/bash"
	if fullCommand != "" {
		// 将命令包装为 bash -c 形式，执行完自动退出
		shellCmd = fmt.Sprintf("/bin/bash -c %q", envExports.String()+fullCommand)
	}

	t, err := a.terminalManager.CreateWithCommand(terminalID, shellCmd, 24, 80, workDir)
	if err != nil {
		return nil, fmt.Errorf("创建PTY终端失败: %w", err)
	}

	// 启动终端输出收集
	go a.collectTerminalOutput(terminalID, t)

	return &acp.CreateTerminalResponse{TerminalID: terminalID}, nil
}

// terminalOutputs 存储终端输出
var terminalOutputs = struct {
	sync.RWMutex
	data map[string]*strings.Builder
}{data: make(map[string]*strings.Builder)}

// collectTerminalOutput 收集终端输出
func (a *Agent) collectTerminalOutput(terminalID string, t interface{ Output() <-chan []byte; Done() <-chan struct{} }) {
	terminalOutputs.Lock()
	terminalOutputs.data[terminalID] = &strings.Builder{}
	terminalOutputs.Unlock()

	for {
		select {
		case data, ok := <-t.Output():
			if !ok {
				return
			}
			terminalOutputs.Lock()
			if buf, exists := terminalOutputs.data[terminalID]; exists {
				buf.Write(data)
			}
			terminalOutputs.Unlock()
		case <-t.Done():
			return
		case <-a.ctx.Done():
			return
		}
	}
}

// handleKillTerminal 处理杀死终端请求
func (a *Agent) handleKillTerminal(p *acp.KillTerminalParams) (*acp.KillTerminalResponse, error) {
	if err := a.terminalManager.Close(p.TerminalID); err != nil {
		return nil, err
	}
	return &acp.KillTerminalResponse{}, nil
}

// handleTerminalOutput 处理获取终端输出请求
func (a *Agent) handleTerminalOutput(p *acp.TerminalOutputParams) (*acp.TerminalOutputResponse, error) {
	terminalOutputs.RLock()
	buf, ok := terminalOutputs.data[p.TerminalID]
	terminalOutputs.RUnlock()

	if !ok {
		return &acp.TerminalOutputResponse{
			Output:    "",
			Truncated: false,
		}, nil
	}

	output := buf.String()

	resp := &acp.TerminalOutputResponse{
		Output:    output,
		Truncated: len(output) > 100000,
	}

	// 检查终端是否已退出
	t, exists := a.terminalManager.Get(p.TerminalID)
	if exists && !t.IsRunning() {
		exitCode := 0
		resp.ExitStatus = &acp.TerminalExitStatus{
			ExitCode: &exitCode,
		}
	}

	return resp, nil
}

// handleReleaseTerminal 处理释放终端请求
func (a *Agent) handleReleaseTerminal(p *acp.ReleaseTerminalParams) (*acp.ReleaseTerminalResponse, error) {
	a.terminalManager.Close(p.TerminalID)

	terminalOutputs.Lock()
	delete(terminalOutputs.data, p.TerminalID)
	terminalOutputs.Unlock()

	return &acp.ReleaseTerminalResponse{}, nil
}

// handleWaitForTerminalExit 处理等待终端退出请求
func (a *Agent) handleWaitForTerminalExit(p *acp.WaitForTerminalExitParams) (*acp.WaitForTerminalExitResponse, error) {
	t, ok := a.terminalManager.Get(p.TerminalID)
	if !ok {
		return &acp.WaitForTerminalExitResponse{}, nil
	}

	// 等待终端完成
	select {
	case <-t.Done():
	case <-a.ctx.Done():
		return nil, a.ctx.Err()
	}

	exitCode := 0
	return &acp.WaitForTerminalExitResponse{
		ExitCode: &exitCode,
	}, nil
}

// 原子计数器用于生成唯一ID
var terminalCounter int64

func nextTerminalID() string {
	id := atomic.AddInt64(&terminalCounter, 1)
	return fmt.Sprintf("terminal-%d", id)
}
