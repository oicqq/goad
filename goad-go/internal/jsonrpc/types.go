// Package jsonrpc 实现了JSONRPC 2.0协议
package jsonrpc

import (
	"encoding/json"
	"sync/atomic"
)

// 错误码定义 (JSONRPC 2.0规范)
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// Request 表示JSONRPC请求
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      *int64          `json:"id,omitempty"`
}

// Response 表示JSONRPC响应
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	ID      *int64          `json:"id,omitempty"`
}

// Error 表示JSONRPC错误
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return e.Message
}

// Notification 表示JSONRPC通知（无ID的请求）
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// IDGenerator 用于生成唯一的请求ID
type IDGenerator struct {
	counter int64
}

// NewIDGenerator 创建新的ID生成器
func NewIDGenerator() *IDGenerator {
	return &IDGenerator{}
}

// Next 返回下一个ID
func (g *IDGenerator) Next() int64 {
	return atomic.AddInt64(&g.counter, 1)
}

// NewRequest 创建新的JSONRPC请求
func NewRequest(method string, params interface{}, id int64) (*Request, error) {
	var paramsData json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		paramsData = data
	}
	return &Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsData,
		ID:      &id,
	}, nil
}

// NewNotification 创建新的JSONRPC通知（不需要响应）
func NewNotification(method string, params interface{}) (*Notification, error) {
	var paramsData json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		paramsData = data
	}
	return &Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsData,
	}, nil
}

// NewResponse 创建成功响应
func NewResponse(id *int64, result interface{}) (*Response, error) {
	var resultData json.RawMessage
	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		resultData = data
	}
	return &Response{
		JSONRPC: "2.0",
		Result:  resultData,
		ID:      id,
	}, nil
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(id *int64, code int, message string, data interface{}) (*Response, error) {
	var dataRaw json.RawMessage
	if data != nil {
		d, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		dataRaw = d
	}
	return &Response{
		JSONRPC: "2.0",
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    dataRaw,
		},
		ID: id,
	}, nil
}
