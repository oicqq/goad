// Package agent 实现了ACP代理的核心逻辑
package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/anthropics/goad/internal/acp"
	"github.com/anthropics/goad/internal/config"
	"github.com/anthropics/goad/internal/jsonrpc"
	"github.com/anthropics/goad/internal/terminal"
)

// 应用信息
const (
	AppName    = "goad"
	AppTitle   = "Goad - AI Agent Terminal"
	AppVersion = "0.1.0"
)

// Agent 代理实例
type Agent struct {
	config      *config.AgentConfig
	apiConfig   *config.APIConfig
	projectRoot string
	sessionID   string

	// 进程管理
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	// JSONRPC
	client *jsonrpc.Client
	server *jsonrpc.Server

	// 代理能力
	capabilities *acp.AgentCapabilities
	authMethods  []acp.AuthMethod
	modes        map[string]*acp.SessionMode
	currentMode  string

	// 工具调用追踪
	toolCalls   map[string]*acp.ToolCall
	toolCallsMu sync.RWMutex

	// PTY终端管理
	terminalManager *terminal.Manager
	terminalID      int
	terminalsMu     sync.RWMutex

	// 消息通道
	messages chan acp.Message

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc

	// 状态
	running bool
	mu      sync.RWMutex
}

// NewAgent 创建新的代理实例
func NewAgent(agentConfig *config.AgentConfig, apiConfig *config.APIConfig, projectRoot string) *Agent {
	ctx, cancel := context.WithCancel(context.Background())
	return &Agent{
		config:          agentConfig,
		apiConfig:       apiConfig,
		projectRoot:     projectRoot,
		toolCalls:       make(map[string]*acp.ToolCall),
		terminalManager: terminal.NewManager(),
		modes:           make(map[string]*acp.SessionMode),
		messages:        make(chan acp.Message, 100),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start 启动代理进程
func (a *Agent) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return fmt.Errorf("代理已在运行")
	}

	command := a.config.GetRunCommand()
	if command == "" {
		return fmt.Errorf("未找到适合当前操作系统的运行命令")
	}

	// 准备环境变量
	env := os.Environ()
	env = append(env, fmt.Sprintf("GOAD_CWD=%s", a.projectRoot))

	// 添加API配置的环境变量
	if a.apiConfig != nil {
		for k, v := range a.apiConfig.GetEnvVars(a.config.Identity) {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// 创建进程
	a.cmd = exec.CommandContext(a.ctx, "sh", "-c", command)
	a.cmd.Dir = a.projectRoot
	a.cmd.Env = env

	var err error
	a.stdin, err = a.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("创建stdin管道失败: %w", err)
	}

	a.stdout, err = a.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建stdout管道失败: %w", err)
	}

	a.stderr, err = a.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建stderr管道失败: %w", err)
	}

	if err := a.cmd.Start(); err != nil {
		return fmt.Errorf("启动代理进程失败: %w", err)
	}

	// 初始化JSONRPC客户端
	a.client = jsonrpc.NewClient(a.stdin)

	// 初始化JSONRPC服务器（处理代理的RPC调用）
	a.server = jsonrpc.NewServer()
	a.registerServerMethods()

	a.running = true

	// 启动读取goroutine
	go a.readLoop()
	go a.readStderr()

	// 初始化ACP协议
	if err := a.initialize(); err != nil {
		a.Stop()
		return fmt.Errorf("初始化ACP协议失败: %w", err)
	}

	// 创建会话
	if err := a.newSession(); err != nil {
		a.Stop()
		return fmt.Errorf("创建会话失败: %w", err)
	}

	a.messages <- &acp.AgentReadyMessage{}
	return nil
}

// Stop 停止代理进程
func (a *Agent) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	a.cancel()
	a.running = false

	// 关闭所有PTY终端
	if a.terminalManager != nil {
		a.terminalManager.CloseAll()
	}

	if a.stdin != nil {
		a.stdin.Close()
	}

	if a.cmd != nil && a.cmd.Process != nil {
		a.cmd.Process.Kill()
		a.cmd.Wait()
	}

	close(a.messages)
	return nil
}

// Messages 返回消息通道
func (a *Agent) Messages() <-chan acp.Message {
	return a.messages
}

// registerServerMethods 注册RPC方法
func (a *Agent) registerServerMethods() {
	// session/update - 会话更新
	a.server.Register("session/update", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p struct {
			SessionID string          `json:"sessionId"`
			Update    json.RawMessage `json:"update"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return a.handleSessionUpdate(p.Update)
	})

	// session/request_permission - 权限请求
	a.server.Register("session/request_permission", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p acp.RequestPermissionParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return a.handleRequestPermission(&p)
	})

	// fs/read_text_file - 读取文件
	a.server.Register("fs/read_text_file", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p acp.ReadTextFileParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return a.handleReadTextFile(&p)
	})

	// fs/write_text_file - 写入文件
	a.server.Register("fs/write_text_file", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p acp.WriteTextFileParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return a.handleWriteTextFile(&p)
	})

	// terminal/create - 创建终端
	a.server.Register("terminal/create", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p acp.CreateTerminalParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return a.handleCreateTerminal(&p)
	})

	// terminal/kill - 杀死终端
	a.server.Register("terminal/kill", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p acp.KillTerminalParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return a.handleKillTerminal(&p)
	})

	// terminal/output - 获取终端输出
	a.server.Register("terminal/output", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p acp.TerminalOutputParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return a.handleTerminalOutput(&p)
	})

	// terminal/release - 释放终端
	a.server.Register("terminal/release", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p acp.ReleaseTerminalParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return a.handleReleaseTerminal(&p)
	})

	// terminal/wait_for_exit - 等待终端退出
	a.server.Register("terminal/wait_for_exit", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p acp.WaitForTerminalExitParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return a.handleWaitForTerminalExit(&p)
	})
}

// readLoop 读取代理输出
func (a *Agent) readLoop() {
	scanner := bufio.NewScanner(a.stdout)
	buf := make([]byte, 10*1024*1024)
	scanner.Buffer(buf, len(buf))

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// 尝试解析JSON
		var data map[string]interface{}
		if err := json.Unmarshal(line, &data); err != nil {
			continue
		}

		// 检查是否是响应
		if _, hasResult := data["result"]; hasResult {
			a.client.ProcessResponse(&jsonrpc.Response{})
			var resp jsonrpc.Response
			json.Unmarshal(line, &resp)
			a.client.ProcessResponse(&resp)
			continue
		}
		if _, hasError := data["error"]; hasError {
			var resp jsonrpc.Response
			json.Unmarshal(line, &resp)
			a.client.ProcessResponse(&resp)
			continue
		}

		// 是RPC请求，处理它
		resp, err := a.server.Handle(a.ctx, line)
		if err != nil {
			continue
		}
		if resp != nil {
			respData, _ := json.Marshal(resp)
			a.stdin.Write(append(respData, '\n'))
		}
	}
}

// readStderr 读取错误输出
func (a *Agent) readStderr() {
	scanner := bufio.NewScanner(a.stderr)
	for scanner.Scan() {
		// 可以记录日志或忽略
		_ = scanner.Text()
	}
}

// initialize 初始化ACP协议
func (a *Agent) initialize() error {
	params := &acp.InitializeParams{
		ProtocolVersion: acp.ProtocolVersion,
		ClientCapabilities: acp.ClientCapabilities{
			FS: &acp.FileSystemCapability{
				ReadTextFile:  true,
				WriteTextFile: true,
			},
			Terminal: true,
		},
		ClientInfo: acp.Implementation{
			Name:    AppName,
			Title:   AppTitle,
			Version: AppVersion,
		},
	}

	resp, err := a.client.Call("initialize", params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return resp.Error
	}

	var initResp acp.InitializeResponse
	if err := json.Unmarshal(resp.Result, &initResp); err != nil {
		return err
	}

	if initResp.AgentCapabilities != nil {
		a.capabilities = initResp.AgentCapabilities
	}
	a.authMethods = initResp.AuthMethods

	return nil
}

// newSession 创建新会话
func (a *Agent) newSession() error {
	params := &acp.NewSessionParams{
		Cwd:        a.projectRoot,
		McpServers: nil,
	}

	resp, err := a.client.Call("session/new", params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return resp.Error
	}

	var sessionResp acp.NewSessionResponse
	if err := json.Unmarshal(resp.Result, &sessionResp); err != nil {
		return err
	}

	a.sessionID = sessionResp.SessionID

	if sessionResp.Modes != nil {
		a.currentMode = sessionResp.Modes.CurrentModeID
		for _, mode := range sessionResp.Modes.AvailableModes {
			m := mode
			a.modes[mode.ID] = &m
		}
	}

	return nil
}

// SendPrompt 发送提示
func (a *Agent) SendPrompt(prompt string) (string, error) {
	a.mu.RLock()
	if !a.running {
		a.mu.RUnlock()
		return "", fmt.Errorf("代理未运行")
	}
	a.mu.RUnlock()

	// 构建提示内容
	contentBlocks := []acp.ContentBlock{
		acp.NewTextContent(prompt),
	}

	params := &acp.SessionPromptParams{
		Prompt:    contentBlocks,
		SessionID: a.sessionID,
	}

	resp, err := a.client.Call("session/prompt", params)
	if err != nil {
		return "", err
	}

	if resp.Error != nil {
		return "", resp.Error
	}

	var promptResp acp.SessionPromptResponse
	if err := json.Unmarshal(resp.Result, &promptResp); err != nil {
		return "", err
	}

	return promptResp.StopReason, nil
}

// SetMode 设置模式
func (a *Agent) SetMode(modeID string) error {
	params := &acp.SetSessionModeParams{
		SessionID: a.sessionID,
		ModeID:    modeID,
	}

	resp, err := a.client.Call("session/set_mode", params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return resp.Error
	}

	a.currentMode = modeID
	return nil
}

// Cancel 取消当前操作
func (a *Agent) Cancel() error {
	params := &acp.SessionCancelParams{
		SessionID: a.sessionID,
	}

	return a.client.Notify("session/cancel", params)
}

// GetModes 获取可用模式
func (a *Agent) GetModes() map[string]*acp.SessionMode {
	return a.modes
}

// GetCurrentMode 获取当前模式
func (a *Agent) GetCurrentMode() string {
	return a.currentMode
}

// ProjectRoot 返回项目根目录
func (a *Agent) ProjectRoot() string {
	return a.projectRoot
}

// Config 返回代理配置
func (a *Agent) Config() *config.AgentConfig {
	return a.config
}
