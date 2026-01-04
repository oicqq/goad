// Package plugin 实现了插件系统
package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

// PluginType 插件类型
type PluginType string

const (
	PluginTypeCommand    PluginType = "command"    // 命令插件
	PluginTypeHook       PluginType = "hook"       // 钩子插件
	PluginTypeFormatter  PluginType = "formatter"  // 格式化插件
	PluginTypeProvider   PluginType = "provider"   // 提供者插件
)

// HookPoint 钩子点
type HookPoint string

const (
	HookBeforePrompt    HookPoint = "before_prompt"    // 发送提示前
	HookAfterResponse   HookPoint = "after_response"   // 收到响应后
	HookBeforeToolCall  HookPoint = "before_tool_call" // 工具调用前
	HookAfterToolCall   HookPoint = "after_tool_call"  // 工具调用后
	HookOnError         HookPoint = "on_error"         // 发生错误时
	HookOnSessionStart  HookPoint = "on_session_start" // 会话开始
	HookOnSessionEnd    HookPoint = "on_session_end"   // 会话结束
)

// PluginConfig 插件配置
type PluginConfig struct {
	ID          string            `toml:"id" json:"id"`
	Name        string            `toml:"name" json:"name"`
	Version     string            `toml:"version" json:"version"`
	Type        PluginType        `toml:"type" json:"type"`
	Description string            `toml:"description" json:"description"`
	Author      string            `toml:"author" json:"author"`
	Command     string            `toml:"command" json:"command"`       // 执行命令
	Args        []string          `toml:"args" json:"args"`             // 命令参数
	Env         map[string]string `toml:"env" json:"env"`               // 环境变量
	Hooks       []HookPoint       `toml:"hooks" json:"hooks"`           // 监听的钩子点
	Enabled     bool              `toml:"enabled" json:"enabled"`       // 是否启用
	Priority    int               `toml:"priority" json:"priority"`     // 优先级 (越小越先执行)
	Timeout     int               `toml:"timeout" json:"timeout"`       // 超时时间(秒)
	ConfigFile  string            `toml:"-" json:"-"`                   // 配置文件路径
}

// Plugin 插件实例
type Plugin struct {
	Config  *PluginConfig
	running bool
	mu      sync.Mutex
}

// HookContext 钩子上下文
type HookContext struct {
	Point     HookPoint              `json:"point"`
	SessionID string                 `json:"session_id"`
	Data      map[string]interface{} `json:"data"`
}

// HookResult 钩子结果
type HookResult struct {
	Continue bool                   `json:"continue"` // 是否继续执行
	Modified map[string]interface{} `json:"modified"` // 修改的数据
	Error    string                 `json:"error"`    // 错误信息
}

// Manager 插件管理器
type Manager struct {
	plugins    map[string]*Plugin
	hooks      map[HookPoint][]*Plugin
	pluginDir  string
	mu         sync.RWMutex
	onReload   func()
}

// NewManager 创建插件管理器
func NewManager(pluginDir string) *Manager {
	return &Manager{
		plugins:   make(map[string]*Plugin),
		hooks:     make(map[HookPoint][]*Plugin),
		pluginDir: pluginDir,
	}
}

// SetOnReload 设置重新加载回调
func (m *Manager) SetOnReload(fn func()) {
	m.onReload = fn
}

// LoadPlugins 加载所有插件
func (m *Manager) LoadPlugins() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 确保目录存在
	if err := os.MkdirAll(m.pluginDir, 0755); err != nil {
		return fmt.Errorf("创建插件目录失败: %w", err)
	}

	// 读取目录
	entries, err := os.ReadDir(m.pluginDir)
	if err != nil {
		return fmt.Errorf("读取插件目录失败: %w", err)
	}

	// 清空现有插件
	m.plugins = make(map[string]*Plugin)
	m.hooks = make(map[HookPoint][]*Plugin)

	// 加载每个插件
	for _, entry := range entries {
		if entry.IsDir() {
			pluginPath := filepath.Join(m.pluginDir, entry.Name(), "plugin.toml")
			if _, err := os.Stat(pluginPath); err == nil {
				if err := m.loadPlugin(pluginPath); err != nil {
					// 记录错误但继续加载其他插件
					fmt.Fprintf(os.Stderr, "加载插件失败 %s: %v\n", entry.Name(), err)
				}
			}
		} else if strings.HasSuffix(entry.Name(), ".toml") {
			pluginPath := filepath.Join(m.pluginDir, entry.Name())
			if err := m.loadPlugin(pluginPath); err != nil {
				fmt.Fprintf(os.Stderr, "加载插件失败 %s: %v\n", entry.Name(), err)
			}
		}
	}

	// 按优先级排序钩子
	m.sortHooks()

	return nil
}

// loadPlugin 加载单个插件
func (m *Manager) loadPlugin(configPath string) error {
	var config PluginConfig
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return fmt.Errorf("解析插件配置失败: %w", err)
	}

	if config.ID == "" {
		return fmt.Errorf("插件ID不能为空")
	}

	config.ConfigFile = configPath

	plugin := &Plugin{
		Config: &config,
	}

	m.plugins[config.ID] = plugin

	// 注册钩子
	if config.Enabled {
		for _, hook := range config.Hooks {
			m.hooks[hook] = append(m.hooks[hook], plugin)
		}
	}

	return nil
}

// sortHooks 按优先级排序钩子
func (m *Manager) sortHooks() {
	for point := range m.hooks {
		plugins := m.hooks[point]
		// 简单冒泡排序
		for i := 0; i < len(plugins)-1; i++ {
			for j := 0; j < len(plugins)-i-1; j++ {
				if plugins[j].Config.Priority > plugins[j+1].Config.Priority {
					plugins[j], plugins[j+1] = plugins[j+1], plugins[j]
				}
			}
		}
	}
}

// Reload 重新加载插件
func (m *Manager) Reload() error {
	if err := m.LoadPlugins(); err != nil {
		return err
	}
	if m.onReload != nil {
		m.onReload()
	}
	return nil
}

// GetPlugin 获取插件
func (m *Manager) GetPlugin(id string) (*Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	plugin, ok := m.plugins[id]
	return plugin, ok
}

// ListPlugins 列出所有插件
func (m *Manager) ListPlugins() []*Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]*Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// EnablePlugin 启用插件
func (m *Manager) EnablePlugin(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, ok := m.plugins[id]
	if !ok {
		return fmt.Errorf("插件不存在: %s", id)
	}

	plugin.Config.Enabled = true

	// 重新注册钩子
	for _, hook := range plugin.Config.Hooks {
		// 检查是否已注册
		found := false
		for _, p := range m.hooks[hook] {
			if p.Config.ID == id {
				found = true
				break
			}
		}
		if !found {
			m.hooks[hook] = append(m.hooks[hook], plugin)
		}
	}

	m.sortHooks()
	return m.savePluginConfig(plugin)
}

// DisablePlugin 禁用插件
func (m *Manager) DisablePlugin(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, ok := m.plugins[id]
	if !ok {
		return fmt.Errorf("插件不存在: %s", id)
	}

	plugin.Config.Enabled = false

	// 从钩子中移除
	for hook := range m.hooks {
		filtered := make([]*Plugin, 0)
		for _, p := range m.hooks[hook] {
			if p.Config.ID != id {
				filtered = append(filtered, p)
			}
		}
		m.hooks[hook] = filtered
	}

	return m.savePluginConfig(plugin)
}

// savePluginConfig 保存插件配置
func (m *Manager) savePluginConfig(plugin *Plugin) error {
	if plugin.Config.ConfigFile == "" {
		return nil
	}

	f, err := os.Create(plugin.Config.ConfigFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return toml.NewEncoder(f).Encode(plugin.Config)
}

// ExecuteHook 执行钩子
func (m *Manager) ExecuteHook(ctx *HookContext) (*HookResult, error) {
	m.mu.RLock()
	plugins := m.hooks[ctx.Point]
	m.mu.RUnlock()

	result := &HookResult{
		Continue: true,
		Modified: ctx.Data,
	}

	for _, plugin := range plugins {
		if !plugin.Config.Enabled {
			continue
		}

		hookResult, err := plugin.Execute(ctx)
		if err != nil {
			result.Error = err.Error()
			// 继续执行其他插件
			continue
		}

		if hookResult != nil {
			if !hookResult.Continue {
				result.Continue = false
				break
			}
			// 合并修改
			for k, v := range hookResult.Modified {
				result.Modified[k] = v
			}
		}
	}

	return result, nil
}

// Execute 执行插件
func (p *Plugin) Execute(ctx *HookContext) (*HookResult, error) {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return nil, fmt.Errorf("插件正在运行中")
	}
	p.running = true
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
	}()

	if p.Config.Command == "" {
		return nil, fmt.Errorf("插件命令未配置")
	}

	// 准备输入
	input, err := json.Marshal(ctx)
	if err != nil {
		return nil, fmt.Errorf("序列化上下文失败: %w", err)
	}

	// 执行命令
	cmd := exec.Command(p.Config.Command, p.Config.Args...)

	// 设置环境变量
	cmd.Env = os.Environ()
	for k, v := range p.Config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// 设置工作目录为插件所在目录
	if p.Config.ConfigFile != "" {
		cmd.Dir = filepath.Dir(p.Config.ConfigFile)
	}

	// 设置标准输入
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("创建stdin管道失败: %w", err)
	}

	// 获取输出
	output, err := cmd.Output()
	go func() {
		defer stdin.Close()
		stdin.Write(input)
	}()

	if err != nil {
		return nil, fmt.Errorf("执行插件失败: %w", err)
	}

	// 解析结果
	var result HookResult
	if len(output) > 0 {
		if err := json.Unmarshal(output, &result); err != nil {
			// 如果不是JSON，视为成功
			result.Continue = true
		}
	} else {
		result.Continue = true
	}

	return &result, nil
}

// PluginDir 返回默认插件目录
func PluginDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "goad", "plugins"), nil
}

// CreateExamplePlugin 创建示例插件
func CreateExamplePlugin(dir string) error {
	exampleConfig := &PluginConfig{
		ID:          "example-logger",
		Name:        "Example Logger",
		Version:     "1.0.0",
		Type:        PluginTypeHook,
		Description: "记录所有提示和响应的示例插件",
		Author:      "Goad",
		Command:     "bash",
		Args:        []string{"-c", "cat >> /tmp/goad-log.txt"},
		Hooks:       []HookPoint{HookBeforePrompt, HookAfterResponse},
		Enabled:     false,
		Priority:    100,
		Timeout:     5,
	}

	pluginDir := filepath.Join(dir, "example-logger")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(pluginDir, "plugin.toml")
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return toml.NewEncoder(f).Encode(exampleConfig)
}
