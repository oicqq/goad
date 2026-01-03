// Package acp 实现了ACP消息类型，用于Agent和UI之间的通信
package acp

import "encoding/json"

// MessageType 消息类型
type MessageType int

const (
	MsgTypeUpdate MessageType = iota
	MsgTypeThinking
	MsgTypeToolCall
	MsgTypeToolCallUpdate
	MsgTypePlan
	MsgTypePermissionRequest
	MsgTypeModeUpdate
	MsgTypeCommandsUpdate
	MsgTypeAgentReady
	MsgTypeAgentFail
	MsgTypeStatusLine
)

// Message 是所有消息的基础接口
type Message interface {
	Type() MessageType
}

// UpdateMessage 代理消息更新
type UpdateMessage struct {
	ContentType string
	Text        string
}

func (m *UpdateMessage) Type() MessageType { return MsgTypeUpdate }

// ThinkingMessage 代理思考消息
type ThinkingMessage struct {
	ContentType string
	Text        string
}

func (m *ThinkingMessage) Type() MessageType { return MsgTypeThinking }

// ToolCallMessage 工具调用消息
type ToolCallMessage struct {
	ToolCall *ToolCall
}

func (m *ToolCallMessage) Type() MessageType { return MsgTypeToolCall }

// ToolCallUpdateMessage 工具调用更新消息
type ToolCallUpdateMessage struct {
	ToolCall *ToolCall
	Update   *ToolCallUpdate
}

func (m *ToolCallUpdateMessage) Type() MessageType { return MsgTypeToolCallUpdate }

// PlanMessage 计划消息
type PlanMessage struct {
	Entries []PlanEntry
}

func (m *PlanMessage) Type() MessageType { return MsgTypePlan }

// PermissionRequestMessage 权限请求消息
type PermissionRequestMessage struct {
	Options    []PermissionOption
	ToolCall   *ToolCall
	ResponseCh chan<- PermissionResponse
}

func (m *PermissionRequestMessage) Type() MessageType { return MsgTypePermissionRequest }

// PermissionResponse 权限响应
type PermissionResponse struct {
	OptionID string
	Outcome  string
}

// ModeUpdateMessage 模式更新消息
type ModeUpdateMessage struct {
	CurrentModeID string
}

func (m *ModeUpdateMessage) Type() MessageType { return MsgTypeModeUpdate }

// CommandsUpdateMessage 可用命令更新消息
type CommandsUpdateMessage struct {
	Commands []AvailableCommand
}

func (m *CommandsUpdateMessage) Type() MessageType { return MsgTypeCommandsUpdate }

// AgentReadyMessage 代理就绪消息
type AgentReadyMessage struct{}

func (m *AgentReadyMessage) Type() MessageType { return MsgTypeAgentReady }

// AgentFailMessage 代理失败消息
type AgentFailMessage struct {
	Reason  string
	Details string
}

func (m *AgentFailMessage) Type() MessageType { return MsgTypeAgentFail }

// StatusLineMessage 状态行消息
type StatusLineMessage struct {
	Text string
}

func (m *StatusLineMessage) Type() MessageType { return MsgTypeStatusLine }

// ParseSessionUpdate 解析会话更新
func ParseSessionUpdate(data json.RawMessage) (Message, error) {
	var update map[string]interface{}
	if err := json.Unmarshal(data, &update); err != nil {
		return nil, err
	}

	updateType, ok := update["sessionUpdate"].(string)
	if !ok {
		return nil, nil
	}

	switch updateType {
	case "agent_message_chunk":
		content, _ := update["content"].(map[string]interface{})
		contentType, _ := content["type"].(string)
		text, _ := content["text"].(string)
		return &UpdateMessage{
			ContentType: contentType,
			Text:        text,
		}, nil

	case "agent_thought_chunk":
		content, _ := update["content"].(map[string]interface{})
		contentType, _ := content["type"].(string)
		text, _ := content["text"].(string)
		return &ThinkingMessage{
			ContentType: contentType,
			Text:        text,
		}, nil

	case "tool_call":
		var tc ToolCall
		if err := json.Unmarshal(data, &tc); err != nil {
			return nil, err
		}
		return &ToolCallMessage{ToolCall: &tc}, nil

	case "tool_call_update":
		var tcu ToolCallUpdate
		if err := json.Unmarshal(data, &tcu); err != nil {
			return nil, err
		}
		return &ToolCallUpdateMessage{Update: &tcu}, nil

	case "plan":
		var plan Plan
		if err := json.Unmarshal(data, &plan); err != nil {
			return nil, err
		}
		return &PlanMessage{Entries: plan.Entries}, nil

	case "available_commands_update":
		var acu AvailableCommandsUpdate
		if err := json.Unmarshal(data, &acu); err != nil {
			return nil, err
		}
		return &CommandsUpdateMessage{Commands: acu.AvailableCommands}, nil

	case "current_mode_update":
		var cmu CurrentModeUpdate
		if err := json.Unmarshal(data, &cmu); err != nil {
			return nil, err
		}
		return &ModeUpdateMessage{CurrentModeID: cmu.CurrentModeID}, nil
	}

	return nil, nil
}
