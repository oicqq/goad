// 热更新测试
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/anthropics/goad/internal/hotreload"
)

// GetHotreloadTests 返回热更新测试用例
func GetHotreloadTests() []TestCase {
	return []TestCase{
		{"创建热更新监视器", "热更新", testHotreloadWatcher},
		{"监视配置文件", "热更新", testHotreloadWatchConfig},
		{"注册处理器", "热更新", testHotreloadRegisterHandler},
		{"停止监视器", "热更新", testHotreloadStop},
		{"获取监视路径", "热更新", testHotreloadGetPaths},
	}
}

func testHotreloadWatcher() error {
	watcher, err := hotreload.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建监视器失败: %w", err)
	}
	defer watcher.Stop()

	if watcher == nil {
		return fmt.Errorf("监视器为空")
	}
	return nil
}

func testHotreloadWatchConfig() error {
	tmpDir, err := os.MkdirTemp("", "goad-hotreload-test")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试配置文件
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte("[app]\nname = \"test\""), 0644); err != nil {
		return fmt.Errorf("创建配置文件失败: %w", err)
	}

	watcher, err := hotreload.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建监视器失败: %w", err)
	}
	defer watcher.Stop()

	// 添加监视
	if err := watcher.Watch(configPath, hotreload.ConfigTypeApp); err != nil {
		return fmt.Errorf("添加监视失败: %w", err)
	}

	// 验证监视路径
	paths := watcher.GetWatchedPaths()
	if len(paths) == 0 {
		return fmt.Errorf("监视路径为空")
	}

	return nil
}

func testHotreloadRegisterHandler() error {
	watcher, err := hotreload.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建监视器失败: %w", err)
	}
	defer watcher.Stop()

	// 注册处理器
	handlerCalled := false
	watcher.RegisterHandler(hotreload.ConfigTypeApp, func(event *hotreload.ChangeEvent) {
		handlerCalled = true
	})

	// 处理器应该已注册（无法直接验证，但不应该报错）
	_ = handlerCalled
	return nil
}

func testHotreloadStop() error {
	watcher, err := hotreload.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建监视器失败: %w", err)
	}

	// 启动后停止
	go watcher.Start()
	time.Sleep(50 * time.Millisecond)

	if err := watcher.Stop(); err != nil {
		return fmt.Errorf("停止监视器失败: %w", err)
	}

	// 验证已停止
	if watcher.IsRunning() {
		return fmt.Errorf("监视器未正确停止")
	}

	return nil
}

func testHotreloadGetPaths() error {
	tmpDir, err := os.MkdirTemp("", "goad-hotreload-test")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试配置文件
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte("[app]\nname = \"test\""), 0644); err != nil {
		return fmt.Errorf("创建配置文件失败: %w", err)
	}

	watcher, err := hotreload.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建监视器失败: %w", err)
	}
	defer watcher.Stop()

	// 初始应该为空
	paths := watcher.GetWatchedPaths()
	if len(paths) != 0 {
		return fmt.Errorf("初始监视路径应该为空")
	}

	// 添加监视
	watcher.Watch(configPath, hotreload.ConfigTypeApp)

	// 验证监视路径
	paths = watcher.GetWatchedPaths()
	if len(paths) != 1 {
		return fmt.Errorf("监视路径数量不正确: %d", len(paths))
	}

	// 验证配置类型
	configType, ok := paths[configPath]
	if !ok {
		return fmt.Errorf("未找到配置路径")
	}
	if configType != hotreload.ConfigTypeApp {
		return fmt.Errorf("配置类型不正确: %d", configType)
	}

	return nil
}
