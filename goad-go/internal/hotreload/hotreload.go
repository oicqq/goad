// Package hotreload 实现配置热更新
package hotreload

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ConfigType 配置类型
type ConfigType string

const (
	ConfigTypeApp    ConfigType = "app"    // 应用配置
	ConfigTypeAgent  ConfigType = "agent"  // 代理配置
	ConfigTypePlugin ConfigType = "plugin" // 插件配置
	ConfigTypeTheme  ConfigType = "theme"  // 主题配置
)

// ChangeEvent 配置变更事件
type ChangeEvent struct {
	Type      ConfigType
	Path      string
	Operation string // "create", "write", "remove", "rename"
	Timestamp time.Time
}

// Handler 配置变更处理器
type Handler func(event *ChangeEvent)

// Watcher 配置监视器
type Watcher struct {
	watcher    *fsnotify.Watcher
	handlers   map[ConfigType][]Handler
	paths      map[string]ConfigType
	debounce   time.Duration
	pending    map[string]*ChangeEvent
	mu         sync.RWMutex
	stopCh     chan struct{}
	running    bool
}

// NewWatcher 创建配置监视器
func NewWatcher() (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		watcher:  watcher,
		handlers: make(map[ConfigType][]Handler),
		paths:    make(map[string]ConfigType),
		debounce: 500 * time.Millisecond,
		pending:  make(map[string]*ChangeEvent),
		stopCh:   make(chan struct{}),
	}, nil
}

// SetDebounce 设置防抖时间
func (w *Watcher) SetDebounce(d time.Duration) {
	w.debounce = d
}

// RegisterHandler 注册配置变更处理器
func (w *Watcher) RegisterHandler(configType ConfigType, handler Handler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers[configType] = append(w.handlers[configType], handler)
}

// Watch 添加监视路径
func (w *Watcher) Watch(path string, configType ConfigType) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查路径是否存在
	info, err := os.Stat(path)
	if err != nil {
		// 如果路径不存在，尝试监视父目录
		parentDir := filepath.Dir(path)
		if _, err := os.Stat(parentDir); err != nil {
			return err
		}
		path = parentDir
	} else if info.IsDir() {
		// 监视目录
	}

	if err := w.watcher.Add(path); err != nil {
		return err
	}

	w.paths[path] = configType
	return nil
}

// Unwatch 移除监视路径
func (w *Watcher) Unwatch(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.watcher.Remove(path); err != nil {
		return err
	}

	delete(w.paths, path)
	return nil
}

// Start 启动监视
func (w *Watcher) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	go w.run()
	go w.processDebounced()
}

// Stop 停止监视
func (w *Watcher) Stop() error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopCh)
	return w.watcher.Close()
}

// run 运行监视循环
func (w *Watcher) run() {
	for {
		select {
		case <-w.stopCh:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			// 记录错误但继续运行
			_ = err
		}
	}
}

// handleEvent 处理文件系统事件
func (w *Watcher) handleEvent(event fsnotify.Event) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 确定配置类型
	configType := w.determineConfigType(event.Name)

	// 确定操作类型
	var operation string
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		operation = "create"
	case event.Op&fsnotify.Write == fsnotify.Write:
		operation = "write"
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		operation = "remove"
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		operation = "rename"
	default:
		return
	}

	changeEvent := &ChangeEvent{
		Type:      configType,
		Path:      event.Name,
		Operation: operation,
		Timestamp: time.Now(),
	}

	// 添加到待处理队列（防抖）
	w.pending[event.Name] = changeEvent
}

// determineConfigType 确定配置类型
func (w *Watcher) determineConfigType(path string) ConfigType {
	// 检查精确匹配
	for watchPath, configType := range w.paths {
		if path == watchPath || filepath.Dir(path) == watchPath {
			return configType
		}
	}

	// 根据路径推断
	base := filepath.Base(path)
	dir := filepath.Base(filepath.Dir(path))

	switch {
	case base == "config.toml":
		return ConfigTypeApp
	case dir == "agents" || filepath.Ext(path) == ".toml" && dir != "plugins":
		return ConfigTypeAgent
	case dir == "plugins":
		return ConfigTypePlugin
	case dir == "themes":
		return ConfigTypeTheme
	default:
		return ConfigTypeApp
	}
}

// processDebounced 处理防抖后的事件
func (w *Watcher) processDebounced() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processPending()
		}
	}
}

// processPending 处理待处理事件
func (w *Watcher) processPending() {
	w.mu.Lock()
	now := time.Now()
	toProcess := make([]*ChangeEvent, 0)

	for path, event := range w.pending {
		if now.Sub(event.Timestamp) >= w.debounce {
			toProcess = append(toProcess, event)
			delete(w.pending, path)
		}
	}
	w.mu.Unlock()

	// 触发处理器
	for _, event := range toProcess {
		w.triggerHandlers(event)
	}
}

// triggerHandlers 触发处理器
func (w *Watcher) triggerHandlers(event *ChangeEvent) {
	w.mu.RLock()
	handlers := w.handlers[event.Type]
	w.mu.RUnlock()

	for _, handler := range handlers {
		handler(event)
	}
}

// WatchConfigDir 监视配置目录
func (w *Watcher) WatchConfigDir(configDir string) error {
	// 监视主配置文件
	configFile := filepath.Join(configDir, "config.toml")
	if err := w.Watch(configFile, ConfigTypeApp); err != nil {
		// 如果文件不存在，监视目录
		if err := w.Watch(configDir, ConfigTypeApp); err != nil {
			return err
		}
	}

	// 监视代理目录
	agentsDir := filepath.Join(configDir, "agents")
	if _, err := os.Stat(agentsDir); err == nil {
		if err := w.Watch(agentsDir, ConfigTypeAgent); err != nil {
			return err
		}
	}

	// 监视插件目录
	pluginsDir := filepath.Join(configDir, "plugins")
	if _, err := os.Stat(pluginsDir); err == nil {
		if err := w.Watch(pluginsDir, ConfigTypePlugin); err != nil {
			return err
		}
	}

	// 监视主题目录
	themesDir := filepath.Join(configDir, "themes")
	if _, err := os.Stat(themesDir); err == nil {
		if err := w.Watch(themesDir, ConfigTypeTheme); err != nil {
			return err
		}
	}

	return nil
}

// IsRunning 是否正在运行
func (w *Watcher) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// GetWatchedPaths 获取监视的路径
func (w *Watcher) GetWatchedPaths() map[string]ConfigType {
	w.mu.RLock()
	defer w.mu.RUnlock()

	result := make(map[string]ConfigType)
	for k, v := range w.paths {
		result[k] = v
	}
	return result
}
