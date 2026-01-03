// Package jsonrpc 实现了JSONRPC 2.0协议服务器
package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

// HandlerFunc 是RPC处理函数类型
type HandlerFunc func(ctx context.Context, params json.RawMessage) (interface{}, error)

// Server 是JSONRPC服务器，处理来自代理的RPC调用
type Server struct {
	handlers map[string]HandlerFunc
	mu       sync.RWMutex
}

// NewServer 创建新的JSONRPC服务器
func NewServer() *Server {
	return &Server{
		handlers: make(map[string]HandlerFunc),
	}
}

// Register 注册RPC处理函数
func (s *Server) Register(method string, handler HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = handler
}

// RegisterMethod 使用反射注册方法
// fn 的签名应该是 func(ctx context.Context, params T) (R, error)
func (s *Server) RegisterMethod(method string, fn interface{}) error {
	v := reflect.ValueOf(fn)
	t := v.Type()

	if t.Kind() != reflect.Func {
		return fmt.Errorf("fn 必须是函数类型")
	}

	if t.NumIn() != 2 || t.NumOut() != 2 {
		return fmt.Errorf("fn 签名必须是 func(ctx context.Context, params T) (R, error)")
	}

	handler := func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		// 获取参数类型
		paramsType := t.In(1)
		paramsValue := reflect.New(paramsType)

		if len(params) > 0 {
			if err := json.Unmarshal(params, paramsValue.Interface()); err != nil {
				return nil, fmt.Errorf("解析参数失败: %w", err)
			}
		}

		// 调用函数
		results := v.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			paramsValue.Elem(),
		})

		// 检查错误
		if !results[1].IsNil() {
			return nil, results[1].Interface().(error)
		}

		return results[0].Interface(), nil
	}

	s.Register(method, handler)
	return nil
}

// Handle 处理JSONRPC请求
func (s *Server) Handle(ctx context.Context, data []byte) (*Response, error) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return NewErrorResponse(nil, ParseError, "解析错误", nil)
	}

	if req.JSONRPC != "2.0" {
		return NewErrorResponse(req.ID, InvalidRequest, "无效的JSONRPC版本", nil)
	}

	s.mu.RLock()
	handler, ok := s.handlers[req.Method]
	s.mu.RUnlock()

	if !ok {
		return NewErrorResponse(req.ID, MethodNotFound, fmt.Sprintf("方法不存在: %s", req.Method), nil)
	}

	result, err := handler(ctx, req.Params)
	if err != nil {
		return NewErrorResponse(req.ID, InternalError, err.Error(), nil)
	}

	// 如果是通知（没有ID），不返回响应
	if req.ID == nil {
		return nil, nil
	}

	return NewResponse(req.ID, result)
}

// HandleBatch 处理批量请求
func (s *Server) HandleBatch(ctx context.Context, data []byte) ([]byte, error) {
	var requests []json.RawMessage
	if err := json.Unmarshal(data, &requests); err != nil {
		// 可能是单个请求
		resp, err := s.Handle(ctx, data)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, nil
		}
		return json.Marshal(resp)
	}

	var responses []*Response
	for _, reqData := range requests {
		resp, err := s.Handle(ctx, reqData)
		if err != nil {
			return nil, err
		}
		if resp != nil {
			responses = append(responses, resp)
		}
	}

	if len(responses) == 0 {
		return nil, nil
	}

	return json.Marshal(responses)
}
