// Package jsonrpc 实现了JSONRPC 2.0协议客户端
package jsonrpc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// PendingCall 表示等待响应的调用
type PendingCall struct {
	ID       int64
	Response chan *Response
	Done     chan struct{}
}

// Client 是JSONRPC客户端
type Client struct {
	idGen    *IDGenerator
	pending  map[int64]*PendingCall
	mu       sync.RWMutex
	writer   io.Writer
	writerMu sync.Mutex
}

// NewClient 创建新的JSONRPC客户端
func NewClient(writer io.Writer) *Client {
	return &Client{
		idGen:   NewIDGenerator(),
		pending: make(map[int64]*PendingCall),
		writer:  writer,
	}
}

// Call 发送请求并等待响应
func (c *Client) Call(method string, params interface{}) (*Response, error) {
	id := c.idGen.Next()
	req, err := NewRequest(method, params, id)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 创建待处理调用
	pending := &PendingCall{
		ID:       id,
		Response: make(chan *Response, 1),
		Done:     make(chan struct{}),
	}

	c.mu.Lock()
	c.pending[id] = pending
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	// 发送请求
	if err := c.send(req); err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	// 等待响应
	resp := <-pending.Response
	return resp, nil
}

// CallAsync 异步发送请求，返回PendingCall用于后续获取响应
func (c *Client) CallAsync(method string, params interface{}) (*PendingCall, error) {
	id := c.idGen.Next()
	req, err := NewRequest(method, params, id)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	pending := &PendingCall{
		ID:       id,
		Response: make(chan *Response, 1),
		Done:     make(chan struct{}),
	}

	c.mu.Lock()
	c.pending[id] = pending
	c.mu.Unlock()

	if err := c.send(req); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	return pending, nil
}

// Notify 发送通知（不等待响应）
func (c *Client) Notify(method string, params interface{}) error {
	notification, err := NewNotification(method, params)
	if err != nil {
		return fmt.Errorf("创建通知失败: %w", err)
	}
	return c.send(notification)
}

// send 发送JSON数据
func (c *Client) send(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	c.writerMu.Lock()
	defer c.writerMu.Unlock()

	if _, err := c.writer.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

// ProcessResponse 处理从代理收到的响应
func (c *Client) ProcessResponse(resp *Response) {
	if resp.ID == nil {
		return
	}

	c.mu.RLock()
	pending, ok := c.pending[*resp.ID]
	c.mu.RUnlock()

	if ok {
		pending.Response <- resp
	}
}

// RemovePending 移除待处理的调用
func (c *Client) RemovePending(id int64) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

// ReadResponses 从reader读取响应并处理
func (c *Client) ReadResponses(reader io.Reader, handler func(data []byte) error) error {
	scanner := bufio.NewScanner(reader)
	// 增大缓冲区以处理大消息
	buf := make([]byte, 10*1024*1024)
	scanner.Buffer(buf, len(buf))

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// 尝试解析为响应
		var resp Response
		if err := json.Unmarshal(line, &resp); err == nil {
			if resp.Result != nil || resp.Error != nil {
				c.ProcessResponse(&resp)
				continue
			}
		}

		// 不是响应，交给handler处理
		if handler != nil {
			if err := handler(line); err != nil {
				return err
			}
		}
	}

	return scanner.Err()
}
