// Package acp 实现了Agent Client Protocol (ACP)
// https://agentclientprotocol.com/
package acp

// ProtocolVersion 当前支持的协议版本
const ProtocolVersion = 1

// ======================== 基础类型 ========================

// FileSystemCapability 文件系统能力
type FileSystemCapability struct {
	ReadTextFile  bool `json:"readTextFile,omitempty"`
	WriteTextFile bool `json:"writeTextFile,omitempty"`
}

// ClientCapabilities 客户端能力声明
type ClientCapabilities struct {
	FS       *FileSystemCapability `json:"fs,omitempty"`
	Terminal bool                  `json:"terminal,omitempty"`
}

// Implementation 实现信息
type Implementation struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

// PromptCapabilities 提示能力
type PromptCapabilities struct {
	Audio           bool `json:"audio,omitempty"`
	EmbeddedContent bool `json:"embeddedContent,omitempty"`
	Image           bool `json:"image,omitempty"`
}

// AgentCapabilities 代理能力
type AgentCapabilities struct {
	LoadSession        bool                `json:"loadSession,omitempty"`
	PromptCapabilities *PromptCapabilities `json:"promptCapabilities,omitempty"`
}

// AuthMethod 认证方法
type AuthMethod struct {
	Description string `json:"description,omitempty"`
	ID          string `json:"id"`
	Name        string `json:"name"`
}

// EnvVariable 环境变量
type EnvVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TerminalExitStatus 终端退出状态
type TerminalExitStatus struct {
	ExitCode *int   `json:"exitCode,omitempty"`
	Signal   string `json:"signal,omitempty"`
}

// McpServer MCP服务器配置
type McpServer struct {
	Args    []string      `json:"args,omitempty"`
	Command string        `json:"command,omitempty"`
	Env     []EnvVariable `json:"env,omitempty"`
	Name    string        `json:"name,omitempty"`
}

// ======================== 内容类型 ========================

// Annotations 注解
type Annotations struct {
	Audience     []string `json:"audience,omitempty"`
	Priority     float64  `json:"priority,omitempty"`
	LastModified string   `json:"lastModified,omitempty"`
}

// TextContent 文本内容
type TextContent struct {
	Type        string       `json:"type"` // "text"
	Text        string       `json:"text"`
	Annotations *Annotations `json:"annotations,omitempty"`
}

// ImageContent 图片内容
type ImageContent struct {
	Type        string       `json:"type"` // "image"
	Data        string       `json:"data"`
	MimeType    string       `json:"mimeType"`
	URL         string       `json:"url,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
}

// AudioContent 音频内容
type AudioContent struct {
	Type        string       `json:"type"` // "audio"
	Data        string       `json:"data"`
	MimeType    string       `json:"mimeType"`
	Annotations *Annotations `json:"annotations,omitempty"`
}

// EmbeddedResourceText 嵌入式文本资源
type EmbeddedResourceText struct {
	URI      string `json:"uri"`
	Text     string `json:"text"`
	MimeType string `json:"mimeType,omitempty"`
}

// EmbeddedResourceBlob 嵌入式二进制资源
type EmbeddedResourceBlob struct {
	URI      string `json:"uri"`
	Blob     string `json:"blob"`
	MimeType string `json:"mimeType,omitempty"`
}

// EmbeddedResourceContent 嵌入式资源内容
type EmbeddedResourceContent struct {
	Type     string      `json:"type"` // "resource"
	Resource interface{} `json:"resource"`
}

// ResourceLinkContent 资源链接内容
type ResourceLinkContent struct {
	Annotations *Annotations `json:"annotations,omitempty"`
	Description string       `json:"description,omitempty"`
	MimeType    string       `json:"mimeType,omitempty"`
	Name        string       `json:"name"`
	Size        *int         `json:"size,omitempty"`
	Title       string       `json:"title,omitempty"`
	Type        string       `json:"type"` // "resource_link"
	URI         string       `json:"uri"`
}

// ContentBlock 内容块（可以是多种类型）
type ContentBlock map[string]interface{}

// NewTextContent 创建文本内容块
func NewTextContent(text string) ContentBlock {
	return ContentBlock{
		"type": "text",
		"text": text,
	}
}

// NewResourceContent 创建资源内容块
func NewResourceContent(uri, text, mimeType string) ContentBlock {
	return ContentBlock{
		"type": "resource",
		"resource": map[string]interface{}{
			"uri":      uri,
			"text":     text,
			"mimeType": mimeType,
		},
	}
}

// NewBlobResourceContent 创建二进制资源内容块
func NewBlobResourceContent(uri, blob, mimeType string) ContentBlock {
	return ContentBlock{
		"type": "resource",
		"resource": map[string]interface{}{
			"uri":      uri,
			"blob":     blob,
			"mimeType": mimeType,
		},
	}
}

// ======================== 工具调用类型 ========================

// ToolKind 工具类型
type ToolKind string

const (
	ToolKindRead       ToolKind = "read"
	ToolKindEdit       ToolKind = "edit"
	ToolKindDelete     ToolKind = "delete"
	ToolKindMove       ToolKind = "move"
	ToolKindSearch     ToolKind = "search"
	ToolKindExecute    ToolKind = "execute"
	ToolKindThink      ToolKind = "think"
	ToolKindFetch      ToolKind = "fetch"
	ToolKindSwitchMode ToolKind = "switch_mode"
	ToolKindOther      ToolKind = "other"
)

// ToolCallStatus 工具调用状态
type ToolCallStatus string

const (
	ToolStatusPending    ToolCallStatus = "pending"
	ToolStatusInProgress ToolCallStatus = "in_progress"
	ToolStatusCompleted  ToolCallStatus = "completed"
	ToolStatusFailed     ToolCallStatus = "failed"
)

// ToolCallLocation 工具调用位置
type ToolCallLocation struct {
	Line *int   `json:"line,omitempty"`
	Path string `json:"path"`
}

// ToolCallContentContent 工具调用内容（内容类型）
type ToolCallContentContent struct {
	Type    string       `json:"type"` // "content"
	Content ContentBlock `json:"content"`
}

// ToolCallContentDiff 工具调用内容（差异类型）
type ToolCallContentDiff struct {
	Type    string `json:"type"` // "diff"
	Path    string `json:"path"`
	OldText string `json:"oldText,omitempty"`
	NewText string `json:"newText"`
}

// ToolCallContentTerminal 工具调用内容（终端类型）
type ToolCallContentTerminal struct {
	Type       string `json:"type"` // "terminal"
	TerminalID string `json:"terminalId"`
}

// ToolCallContent 工具调用内容（通用）
type ToolCallContent map[string]interface{}

// ToolCall 工具调用
type ToolCall struct {
	SessionUpdate string               `json:"sessionUpdate"` // "tool_call"
	ToolCallID    string               `json:"toolCallId"`
	Title         string               `json:"title"`
	Kind          ToolKind             `json:"kind,omitempty"`
	Status        ToolCallStatus       `json:"status,omitempty"`
	Content       []ToolCallContent    `json:"content,omitempty"`
	Locations     []ToolCallLocation   `json:"locations,omitempty"`
	RawInput      map[string]interface{} `json:"rawInput,omitempty"`
	RawOutput     map[string]interface{} `json:"rawOutput,omitempty"`
}

// ToolCallUpdate 工具调用更新
type ToolCallUpdate struct {
	SessionUpdate string               `json:"sessionUpdate"` // "tool_call_update"
	ToolCallID    string               `json:"toolCallId"`
	Title         string               `json:"title,omitempty"`
	Kind          ToolKind             `json:"kind,omitempty"`
	Status        ToolCallStatus       `json:"status,omitempty"`
	Content       []ToolCallContent    `json:"content,omitempty"`
	Locations     []ToolCallLocation   `json:"locations,omitempty"`
	RawInput      map[string]interface{} `json:"rawInput,omitempty"`
	RawOutput     map[string]interface{} `json:"rawOutput,omitempty"`
}

// ======================== 会话更新类型 ========================

// UserMessageChunk 用户消息块
type UserMessageChunk struct {
	SessionUpdate string       `json:"sessionUpdate"` // "user_message_chunk"
	Content       ContentBlock `json:"content"`
}

// AgentMessageChunk 代理消息块
type AgentMessageChunk struct {
	SessionUpdate string       `json:"sessionUpdate"` // "agent_message_chunk"
	Content       ContentBlock `json:"content"`
}

// AgentThoughtChunk 代理思考块
type AgentThoughtChunk struct {
	SessionUpdate string       `json:"sessionUpdate"` // "agent_thought_chunk"
	Content       ContentBlock `json:"content"`
}

// PlanEntry 计划条目
type PlanEntry struct {
	Content  string `json:"content"`
	Priority string `json:"priority,omitempty"` // "high", "medium", "low"
	Status   string `json:"status,omitempty"`   // "pending", "in_progress", "completed"
}

// Plan 计划
type Plan struct {
	SessionUpdate string      `json:"sessionUpdate"` // "plan"
	Entries       []PlanEntry `json:"entries"`
}

// AvailableCommandInput 可用命令输入
type AvailableCommandInput struct {
	Hint string `json:"hint"`
}

// AvailableCommand 可用命令
type AvailableCommand struct {
	Description string                 `json:"description"`
	Input       *AvailableCommandInput `json:"input,omitempty"`
	Name        string                 `json:"name"`
}

// AvailableCommandsUpdate 可用命令更新
type AvailableCommandsUpdate struct {
	SessionUpdate     string             `json:"sessionUpdate"` // "available_commands_update"
	AvailableCommands []AvailableCommand `json:"availableCommands"`
}

// CurrentModeUpdate 当前模式更新
type CurrentModeUpdate struct {
	SessionUpdate string `json:"sessionUpdate"` // "current_mode_update"
	CurrentModeID string `json:"currentModeId"`
}

// ======================== 会话模式 ========================

// SessionMode 会话模式
type SessionMode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// SessionModeState 会话模式状态
type SessionModeState struct {
	AvailableModes []SessionMode `json:"availableModes"`
	CurrentModeID  string        `json:"currentModeId"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	ModelID     string `json:"modelId"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// SessionModelState 会话模型状态
type SessionModelState struct {
	AvailableModels []ModelInfo `json:"availableModels"`
	CurrentModelID  string      `json:"currentModelId"`
}

// ======================== 权限请求 ========================

// PermissionOptionKind 权限选项类型
type PermissionOptionKind string

const (
	PermissionAllowOnce    PermissionOptionKind = "allow_once"
	PermissionAllowAlways  PermissionOptionKind = "allow_always"
	PermissionRejectOnce   PermissionOptionKind = "reject_once"
	PermissionRejectAlways PermissionOptionKind = "reject_always"
)

// PermissionOption 权限选项
type PermissionOption struct {
	Kind     PermissionOptionKind `json:"kind"`
	Name     string               `json:"name"`
	OptionID string               `json:"optionId"`
}

// OutcomeSelected 已选择的结果
type OutcomeSelected struct {
	OptionID string `json:"optionId"`
	Outcome  string `json:"outcome"` // "selected"
}

// OutcomeCancelled 已取消的结果
type OutcomeCancelled struct {
	Outcome string `json:"outcome"` // "cancelled"
}

// RequestPermissionOutcome 请求权限结果
type RequestPermissionOutcome struct {
	OptionID string `json:"optionId,omitempty"`
	Outcome  string `json:"outcome"` // "selected" or "cancelled"
}
