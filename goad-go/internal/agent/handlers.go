// Package agent 实现了Agent的RPC处理方法
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	// 确保路径在项目根目录内
	fullPath := filepath.Join(a.projectRoot, p.Path)
	if !strings.HasPrefix(fullPath, a.projectRoot) {
		return nil, fmt.Errorf("路径不在项目目录内")
	}

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
	// 确保路径在项目根目录内
	fullPath := filepath.Join(a.projectRoot, p.Path)
	if !strings.HasPrefix(fullPath, a.projectRoot) {
		return nil, fmt.Errorf("路径不在项目目录内")
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

	// 构建命令
	args := append([]string{"-c", p.Command}, p.Args...)
	cmd := exec.CommandContext(a.ctx, "sh", args...)

	// 设置工作目录
	if p.Cwd != "" {
		cmd.Dir = p.Cwd
	} else {
		cmd.Dir = a.projectRoot
	}

	// 设置环境变量
	cmd.Env = os.Environ()
	for _, ev := range p.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", ev.Name, ev.Value))
	}

	// 创建终端记录
	terminal := &Terminal{
		ID:   terminalID,
		Cmd:  cmd,
		Done: make(chan struct{}),
	}

	// 获取输出
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// 读取输出
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				terminal.Output.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				terminal.Output.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	// 等待完成
	go func() {
		cmd.Wait()
		exitCode := cmd.ProcessState.ExitCode()
		terminal.ExitCode = &exitCode
		close(terminal.Done)
	}()

	a.terminals[terminalID] = terminal

	return &acp.CreateTerminalResponse{TerminalID: terminalID}, nil
}

// handleKillTerminal 处理杀死终端请求
func (a *Agent) handleKillTerminal(p *acp.KillTerminalParams) (*acp.KillTerminalResponse, error) {
	a.terminalsMu.RLock()
	terminal, ok := a.terminals[p.TerminalID]
	a.terminalsMu.RUnlock()

	if !ok {
		return &acp.KillTerminalResponse{}, nil
	}

	if terminal.Cmd != nil && terminal.Cmd.Process != nil {
		terminal.Cmd.Process.Kill()
	}

	return &acp.KillTerminalResponse{}, nil
}

// handleTerminalOutput 处理获取终端输出请求
func (a *Agent) handleTerminalOutput(p *acp.TerminalOutputParams) (*acp.TerminalOutputResponse, error) {
	a.terminalsMu.RLock()
	terminal, ok := a.terminals[p.TerminalID]
	a.terminalsMu.RUnlock()

	if !ok {
		return &acp.TerminalOutputResponse{
			Output:    "",
			Truncated: false,
		}, nil
	}

	resp := &acp.TerminalOutputResponse{
		Output:    terminal.Output.String(),
		Truncated: false,
	}

	if terminal.ExitCode != nil {
		resp.ExitStatus = &acp.TerminalExitStatus{
			ExitCode: terminal.ExitCode,
		}
	}

	return resp, nil
}

// handleReleaseTerminal 处理释放终端请求
func (a *Agent) handleReleaseTerminal(p *acp.ReleaseTerminalParams) (*acp.ReleaseTerminalResponse, error) {
	a.terminalsMu.Lock()
	delete(a.terminals, p.TerminalID)
	a.terminalsMu.Unlock()

	return &acp.ReleaseTerminalResponse{}, nil
}

// handleWaitForTerminalExit 处理等待终端退出请求
func (a *Agent) handleWaitForTerminalExit(p *acp.WaitForTerminalExitParams) (*acp.WaitForTerminalExitResponse, error) {
	a.terminalsMu.RLock()
	terminal, ok := a.terminals[p.TerminalID]
	a.terminalsMu.RUnlock()

	if !ok {
		return &acp.WaitForTerminalExitResponse{}, nil
	}

	// 等待终端完成
	select {
	case <-terminal.Done:
	case <-a.ctx.Done():
		return nil, a.ctx.Err()
	}

	resp := &acp.WaitForTerminalExitResponse{}
	if terminal.ExitCode != nil {
		resp.ExitCode = terminal.ExitCode
	}

	return resp, nil
}

// 原子计数器用于生成唯一ID
var terminalCounter int64

func nextTerminalID() string {
	id := atomic.AddInt64(&terminalCounter, 1)
	return fmt.Sprintf("terminal-%d", id)
}
