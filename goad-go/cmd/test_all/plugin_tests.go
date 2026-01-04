// 插件系统测试
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/goad/internal/plugin"
)

// GetPluginTests 返回插件系统测试用例
func GetPluginTests() []TestCase {
	return []TestCase{
		{"创建插件管理器", "插件系统", testPluginManager},
		{"加载空目录", "插件系统", testPluginLoadEmpty},
		{"创建示例插件", "插件系统", testPluginCreateExample},
		{"插件启用禁用", "插件系统", testPluginEnableDisable},
		{"钩子上下文", "插件系统", testPluginHookContext},
	}
}

func testPluginManager() error {
	tmpDir, err := os.MkdirTemp("", "goad-plugin-test")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	mgr := plugin.NewManager(tmpDir)
	if mgr == nil {
		return fmt.Errorf("管理器为空")
	}
	return nil
}

func testPluginLoadEmpty() error {
	tmpDir, err := os.MkdirTemp("", "goad-plugin-test")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	mgr := plugin.NewManager(tmpDir)
	if err := mgr.LoadPlugins(); err != nil {
		return fmt.Errorf("加载空目录失败: %w", err)
	}

	plugins := mgr.ListPlugins()
	if len(plugins) != 0 {
		return fmt.Errorf("空目录应该没有插件")
	}
	return nil
}

func testPluginCreateExample() error {
	tmpDir, err := os.MkdirTemp("", "goad-plugin-test")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// 创建示例插件
	if err := plugin.CreateExamplePlugin(tmpDir); err != nil {
		return fmt.Errorf("创建示例插件失败: %w", err)
	}

	// 验证文件存在
	pluginPath := filepath.Join(tmpDir, "example-logger", "plugin.toml")
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("插件配置文件未创建")
	}

	// 加载并验证
	mgr := plugin.NewManager(tmpDir)
	if err := mgr.LoadPlugins(); err != nil {
		return fmt.Errorf("加载插件失败: %w", err)
	}

	p, ok := mgr.GetPlugin("example-logger")
	if !ok {
		return fmt.Errorf("未找到示例插件")
	}
	if p.Config.Name != "Example Logger" {
		return fmt.Errorf("插件名称不正确: %s", p.Config.Name)
	}
	return nil
}

func testPluginEnableDisable() error {
	tmpDir, err := os.MkdirTemp("", "goad-plugin-test")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	plugin.CreateExamplePlugin(tmpDir)
	mgr := plugin.NewManager(tmpDir)
	mgr.LoadPlugins()

	// 启用插件
	if err := mgr.EnablePlugin("example-logger"); err != nil {
		return fmt.Errorf("启用插件失败: %w", err)
	}

	p, _ := mgr.GetPlugin("example-logger")
	if !p.Config.Enabled {
		return fmt.Errorf("插件未启用")
	}

	// 禁用插件
	if err := mgr.DisablePlugin("example-logger"); err != nil {
		return fmt.Errorf("禁用插件失败: %w", err)
	}

	p, _ = mgr.GetPlugin("example-logger")
	if p.Config.Enabled {
		return fmt.Errorf("插件未禁用")
	}
	return nil
}

func testPluginHookContext() error {
	ctx := &plugin.HookContext{
		Point:     plugin.HookBeforePrompt,
		SessionID: "test-session",
		Data: map[string]interface{}{
			"prompt": "Hello",
		},
	}

	if ctx.Point != plugin.HookBeforePrompt {
		return fmt.Errorf("钩子点不正确")
	}
	if ctx.SessionID != "test-session" {
		return fmt.Errorf("会话ID不正确")
	}
	if ctx.Data["prompt"] != "Hello" {
		return fmt.Errorf("数据不正确")
	}
	return nil
}
