// Package acp 实现了ACP请求和响应类型
package acp

// ======================== 初始化 ========================

// InitializeParams 初始化请求参数
type InitializeParams struct {
	ProtocolVersion    int                `json:"protocolVersion"`
	ClientCapabilities ClientCapabilities `json:"clientCapabilities"`
	ClientInfo         Implementation     `json:"clientInfo"`
}

// InitializeResponse 初始化响应
type InitializeResponse struct {
	ProtocolVersion   int                `json:"protocolVersion"`
	AgentCapabilities *AgentCapabilities `json:"agentCapabilities,omitempty"`
	AuthMethods       []AuthMethod       `json:"authMethods,omitempty"`
}

// ======================== 会话管理 ========================

// NewSessionParams 新建会话请求参数
type NewSessionParams struct {
	Cwd        string      `json:"cwd"`
	McpServers []McpServer `json:"mcpServers"`
}

// NewSessionResponse 新建会话响应
type NewSessionResponse struct {
	SessionID string             `json:"sessionId"`
	Models    *SessionModelState `json:"models,omitempty"`
	Modes     *SessionModeState  `json:"modes,omitempty"`
}

// SessionPromptParams 会话提示请求参数
type SessionPromptParams struct {
	Prompt    []ContentBlock `json:"prompt"`
	SessionID string         `json:"sessionId"`
}

// SessionPromptResponse 会话提示响应
type SessionPromptResponse struct {
	StopReason string `json:"stopReason"` // "end_turn", "max_tokens", "max_turn_requests", "refusal", "cancelled"
}

// SetSessionModeParams 设置会话模式请求参数
type SetSessionModeParams struct {
	SessionID string `json:"sessionId"`
	ModeID    string `json:"modeId"`
}

// SetSessionModeResponse 设置会话模式响应
type SetSessionModeResponse struct{}

// SessionCancelParams 取消会话请求参数
type SessionCancelParams struct {
	SessionID string                 `json:"sessionId"`
	Meta      map[string]interface{} `json:"_meta,omitempty"`
}

// ======================== 文件系统 ========================

// ReadTextFileParams 读取文本文件请求参数
type ReadTextFileParams struct {
	SessionID string `json:"sessionId"`
	Path      string `json:"path"`
	Line      *int   `json:"line,omitempty"`
	Limit     *int   `json:"limit,omitempty"`
}

// ReadTextFileResponse 读取文本文件响应
type ReadTextFileResponse struct {
	Content string `json:"content"`
}

// WriteTextFileParams 写入文本文件请求参数
type WriteTextFileParams struct {
	SessionID string `json:"sessionId"`
	Path      string `json:"path"`
	Content   string `json:"content"`
}

// ======================== 终端 ========================

// CreateTerminalParams 创建终端请求参数
type CreateTerminalParams struct {
	SessionID       string        `json:"sessionId,omitempty"`
	Command         string        `json:"command"`
	Args            []string      `json:"args,omitempty"`
	Cwd             string        `json:"cwd,omitempty"`
	Env             []EnvVariable `json:"env,omitempty"`
	OutputByteLimit *int          `json:"outputByteLimit,omitempty"`
}

// CreateTerminalResponse 创建终端响应
type CreateTerminalResponse struct {
	TerminalID string `json:"terminalId"`
}

// KillTerminalParams 杀死终端请求参数
type KillTerminalParams struct {
	SessionID  string `json:"sessionId"`
	TerminalID string `json:"terminalId"`
}

// KillTerminalResponse 杀死终端响应
type KillTerminalResponse struct{}

// TerminalOutputParams 终端输出请求参数
type TerminalOutputParams struct {
	SessionID  string `json:"sessionId"`
	TerminalID string `json:"terminalId"`
}

// TerminalOutputResponse 终端输出响应
type TerminalOutputResponse struct {
	Output     string              `json:"output"`
	Truncated  bool                `json:"truncated"`
	ExitStatus *TerminalExitStatus `json:"exitStatus,omitempty"`
}

// ReleaseTerminalParams 释放终端请求参数
type ReleaseTerminalParams struct {
	SessionID  string `json:"sessionId"`
	TerminalID string `json:"terminalId"`
}

// ReleaseTerminalResponse 释放终端响应
type ReleaseTerminalResponse struct{}

// WaitForTerminalExitParams 等待终端退出请求参数
type WaitForTerminalExitParams struct {
	SessionID  string `json:"sessionId"`
	TerminalID string `json:"terminalId"`
}

// WaitForTerminalExitResponse 等待终端退出响应
type WaitForTerminalExitResponse struct {
	ExitCode *int   `json:"exitCode,omitempty"`
	Signal   string `json:"signal,omitempty"`
}

// ======================== 权限请求 ========================

// RequestPermissionParams 请求权限参数
type RequestPermissionParams struct {
	SessionID string              `json:"sessionId"`
	Options   []PermissionOption  `json:"options"`
	ToolCall  *ToolCallUpdate     `json:"toolCall"`
}

// RequestPermissionResponse 请求权限响应
type RequestPermissionResponse struct {
	Outcome RequestPermissionOutcome `json:"outcome"`
}

// ======================== 会话更新通知 ========================

// SessionUpdateParams 会话更新通知参数
type SessionUpdateParams struct {
	SessionID string                 `json:"sessionId"`
	Update    map[string]interface{} `json:"update"`
}
