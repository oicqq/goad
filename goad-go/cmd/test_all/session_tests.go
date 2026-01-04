// 会话管理测试
package main

import (
	"fmt"
	"sync"

	"github.com/anthropics/goad/internal/session"
)

// GetSessionTests 返回会话管理测试用例
func GetSessionTests() []TestCase {
	return []TestCase{
		{"创建会话管理器", "会话管理", testSessionManager},
		{"创建新会话", "会话管理", testSessionCreate},
		{"会话消息操作", "会话管理", testSessionMessages},
		{"会话保存加载", "会话管理", testSessionSaveLoad},
		{"会话列表", "会话管理", testSessionList},
		{"会话删除", "会话管理", testSessionDelete},
		{"会话按代理过滤", "会话管理", testSessionByAgent},
		{"会话并发操作", "会话管理", testSessionConcurrent},
	}
}

func testSessionManager() error {
	mgr, err := session.NewManager()
	if err != nil {
		return fmt.Errorf("创建管理器失败: %w", err)
	}
	if mgr == nil {
		return fmt.Errorf("管理器为空")
	}
	return nil
}

func testSessionCreate() error {
	mgr, err := session.NewManager()
	if err != nil {
		return fmt.Errorf("创建管理器失败: %w", err)
	}

	sess := mgr.Create("test-agent", "/tmp/test-project")
	if sess == nil {
		return fmt.Errorf("创建会话失败")
	}
	if sess.ID == "" {
		return fmt.Errorf("会话ID为空")
	}
	if sess.AgentID != "test-agent" {
		return fmt.Errorf("代理ID不正确: %s", sess.AgentID)
	}
	if sess.CreatedAt.IsZero() {
		return fmt.Errorf("创建时间未设置")
	}

	// 清理
	mgr.Delete(sess.ID)

	return nil
}

func testSessionMessages() error {
	mgr, err := session.NewManager()
	if err != nil {
		return fmt.Errorf("创建管理器失败: %w", err)
	}

	sess := mgr.Create("test-agent", "/tmp/test-project")

	// 添加消息
	sess.AddMessage("user", "Hello")
	sess.AddMessage("assistant", "Hi there!")
	sess.AddMessage("user", "How are you?")

	if len(sess.Messages) != 3 {
		return fmt.Errorf("消息数量不正确: %d", len(sess.Messages))
	}

	// 验证消息内容
	if sess.Messages[0].Role != "user" || sess.Messages[0].Content != "Hello" {
		return fmt.Errorf("第一条消息内容不正确")
	}
	if sess.Messages[1].Role != "assistant" || sess.Messages[1].Content != "Hi there!" {
		return fmt.Errorf("第二条消息内容不正确")
	}

	// 清理
	mgr.Delete(sess.ID)

	return nil
}

func testSessionSaveLoad() error {
	mgr, err := session.NewManager()
	if err != nil {
		return fmt.Errorf("创建管理器失败: %w", err)
	}

	// 创建并保存会话
	sess := mgr.Create("save-load-test", "/tmp/test-project")
	sess.AddMessage("user", "Test message for save/load")
	sessionID := sess.ID

	if err := mgr.Save(sess); err != nil {
		return fmt.Errorf("保存会话失败: %w", err)
	}

	// 加载会话
	loaded, err := mgr.Load(sessionID)
	if err != nil {
		return fmt.Errorf("加载会话失败: %w", err)
	}

	if loaded.ID != sessionID {
		return fmt.Errorf("会话ID不匹配")
	}
	if loaded.AgentID != "save-load-test" {
		return fmt.Errorf("代理ID不匹配")
	}
	if len(loaded.Messages) != 1 {
		return fmt.Errorf("消息数量不正确: %d", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != "Test message for save/load" {
		return fmt.Errorf("消息内容不匹配")
	}

	// 清理
	mgr.Delete(sessionID)

	return nil
}

func testSessionList() error {
	mgr, err := session.NewManager()
	if err != nil {
		return fmt.Errorf("创建管理器失败: %w", err)
	}

	// 创建多个会话
	var sessionIDs []string
	for i := 0; i < 3; i++ {
		sess := mgr.Create(fmt.Sprintf("list-test-%d", i), "/tmp/test-project")
		sess.AddMessage("user", fmt.Sprintf("Message %d", i))
		mgr.Save(sess)
		sessionIDs = append(sessionIDs, sess.ID)
	}

	// 获取列表
	sessions, err := mgr.List()
	if err != nil {
		return fmt.Errorf("获取会话列表失败: %w", err)
	}
	if len(sessions) < 3 {
		return fmt.Errorf("会话列表数量不足: %d", len(sessions))
	}

	// 清理
	for _, id := range sessionIDs {
		mgr.Delete(id)
	}

	return nil
}

func testSessionDelete() error {
	mgr, err := session.NewManager()
	if err != nil {
		return fmt.Errorf("创建管理器失败: %w", err)
	}

	// 创建会话
	sess := mgr.Create("delete-test", "/tmp/test-project")
	sess.AddMessage("user", "To be deleted")
	sessionID := sess.ID
	mgr.Save(sess)

	// 删除
	if err := mgr.Delete(sessionID); err != nil {
		return fmt.Errorf("删除会话失败: %w", err)
	}

	// 确认删除
	_, err = mgr.Load(sessionID)
	if err == nil {
		return fmt.Errorf("会话未被删除")
	}

	return nil
}

func testSessionByAgent() error {
	mgr, err := session.NewManager()
	if err != nil {
		return fmt.Errorf("创建管理器失败: %w", err)
	}

	// 创建不同代理的会话
	var sessionIDs []string
	for i := 0; i < 2; i++ {
		sess := mgr.Create("agent-filter-a", "/tmp/test-project")
		mgr.Save(sess)
		sessionIDs = append(sessionIDs, sess.ID)
	}
	for i := 0; i < 3; i++ {
		sess := mgr.Create("agent-filter-b", "/tmp/test-project")
		mgr.Save(sess)
		sessionIDs = append(sessionIDs, sess.ID)
	}

	// 按代理过滤
	sessionsA, err := mgr.ListByAgent("agent-filter-a")
	if err != nil {
		return fmt.Errorf("按代理过滤失败: %w", err)
	}
	if len(sessionsA) < 2 {
		return fmt.Errorf("代理A会话数量不正确: %d", len(sessionsA))
	}

	sessionsB, err := mgr.ListByAgent("agent-filter-b")
	if err != nil {
		return fmt.Errorf("按代理过滤失败: %w", err)
	}
	if len(sessionsB) < 3 {
		return fmt.Errorf("代理B会话数量不正确: %d", len(sessionsB))
	}

	// 清理
	for _, id := range sessionIDs {
		mgr.Delete(id)
	}

	return nil
}

func testSessionConcurrent() error {
	mgr, err := session.NewManager()
	if err != nil {
		return fmt.Errorf("创建管理器失败: %w", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 10)
	var sessionIDs []string
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sess := mgr.Create(fmt.Sprintf("concurrent-%d", idx), "/tmp/test-project")
			sess.AddMessage("user", fmt.Sprintf("Concurrent message %d", idx))
			if err := mgr.Save(sess); err != nil {
				errCh <- fmt.Errorf("并发保存失败: %w", err)
				return
			}
			mu.Lock()
			sessionIDs = append(sessionIDs, sess.ID)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		return err
	}

	// 验证
	sessions, err := mgr.List()
	if err != nil {
		return fmt.Errorf("获取列表失败: %w", err)
	}
	if len(sessions) < 10 {
		return fmt.Errorf("并发创建会话数量不足")
	}

	// 清理
	for _, id := range sessionIDs {
		mgr.Delete(id)
	}

	return nil
}
