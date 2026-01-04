// Package metrics 实现性能监控
package metrics

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/anthropics/goad/internal/tui/styles"
)

// MetricType 指标类型
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"   // 计数器
	MetricTypeGauge     MetricType = "gauge"     // 仪表
	MetricTypeHistogram MetricType = "histogram" // 直方图
	MetricTypeTiming    MetricType = "timing"    // 计时
)

// Metric 指标
type Metric struct {
	Name        string
	Type        MetricType
	Value       float64
	Unit        string
	Description string
	Timestamp   time.Time
}

// TimingMetric 计时指标
type TimingMetric struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
}

// SessionStats 会话统计
type SessionStats struct {
	SessionID       string
	StartTime       time.Time
	PromptCount     int
	ResponseCount   int
	ToolCallCount   int
	TotalTokens     int64
	InputTokens     int64
	OutputTokens    int64
	TotalLatency    time.Duration
	AverageLatency  time.Duration
	ErrorCount      int
	LastPromptTime  time.Time
	LastResponseTime time.Time
}

// Collector 指标收集器
type Collector struct {
	metrics      map[string]*Metric
	timings      map[string]*TimingMetric
	sessionStats *SessionStats
	history      []*Metric
	maxHistory   int
	mu           sync.RWMutex
}

// NewCollector 创建指标收集器
func NewCollector() *Collector {
	return &Collector{
		metrics:    make(map[string]*Metric),
		timings:    make(map[string]*TimingMetric),
		history:    make([]*Metric, 0),
		maxHistory: 1000,
		sessionStats: &SessionStats{
			StartTime: time.Now(),
		},
	}
}

// SetSessionID 设置会话ID
func (c *Collector) SetSessionID(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionStats.SessionID = id
}

// RecordCounter 记录计数器
func (c *Collector) RecordCounter(name string, value float64, unit, description string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	metric := &Metric{
		Name:        name,
		Type:        MetricTypeCounter,
		Value:       value,
		Unit:        unit,
		Description: description,
		Timestamp:   time.Now(),
	}

	if existing, ok := c.metrics[name]; ok {
		existing.Value += value
		existing.Timestamp = time.Now()
	} else {
		c.metrics[name] = metric
	}

	c.addHistory(metric)
}

// RecordGauge 记录仪表值
func (c *Collector) RecordGauge(name string, value float64, unit, description string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	metric := &Metric{
		Name:        name,
		Type:        MetricTypeGauge,
		Value:       value,
		Unit:        unit,
		Description: description,
		Timestamp:   time.Now(),
	}

	c.metrics[name] = metric
	c.addHistory(metric)
}

// StartTiming 开始计时
func (c *Collector) StartTiming(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.timings[name] = &TimingMetric{
		Name:      name,
		StartTime: time.Now(),
	}
}

// EndTiming 结束计时
func (c *Collector) EndTiming(name string) time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()

	timing, ok := c.timings[name]
	if !ok {
		return 0
	}

	timing.EndTime = time.Now()
	timing.Duration = timing.EndTime.Sub(timing.StartTime)

	// 记录到指标
	metric := &Metric{
		Name:        name,
		Type:        MetricTypeTiming,
		Value:       float64(timing.Duration.Milliseconds()),
		Unit:        "ms",
		Description: fmt.Sprintf("Timing for %s", name),
		Timestamp:   time.Now(),
	}
	c.metrics[name] = metric
	c.addHistory(metric)

	return timing.Duration
}

// addHistory 添加历史记录
func (c *Collector) addHistory(metric *Metric) {
	c.history = append(c.history, metric)
	if len(c.history) > c.maxHistory {
		c.history = c.history[1:]
	}
}

// RecordPrompt 记录提示
func (c *Collector) RecordPrompt() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionStats.PromptCount++
	c.sessionStats.LastPromptTime = time.Now()
}

// RecordResponse 记录响应
func (c *Collector) RecordResponse(latency time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionStats.ResponseCount++
	c.sessionStats.TotalLatency += latency
	c.sessionStats.LastResponseTime = time.Now()
	if c.sessionStats.ResponseCount > 0 {
		c.sessionStats.AverageLatency = c.sessionStats.TotalLatency / time.Duration(c.sessionStats.ResponseCount)
	}
}

// RecordToolCall 记录工具调用
func (c *Collector) RecordToolCall() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionStats.ToolCallCount++
}

// RecordTokens 记录Token使用
func (c *Collector) RecordTokens(input, output int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionStats.InputTokens += input
	c.sessionStats.OutputTokens += output
	c.sessionStats.TotalTokens = c.sessionStats.InputTokens + c.sessionStats.OutputTokens
}

// RecordError 记录错误
func (c *Collector) RecordError() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionStats.ErrorCount++
}

// GetSessionStats 获取会话统计
func (c *Collector) GetSessionStats() *SessionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// 返回副本
	stats := *c.sessionStats
	return &stats
}

// GetMetric 获取指标
func (c *Collector) GetMetric(name string) (*Metric, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	metric, ok := c.metrics[name]
	return metric, ok
}

// GetAllMetrics 获取所有指标
func (c *Collector) GetAllMetrics() map[string]*Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*Metric)
	for k, v := range c.metrics {
		result[k] = v
	}
	return result
}

// Reset 重置收集器
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics = make(map[string]*Metric)
	c.timings = make(map[string]*TimingMetric)
	c.history = make([]*Metric, 0)
	c.sessionStats = &SessionStats{
		StartTime: time.Now(),
	}
}

// Dashboard 性能仪表盘组件
type Dashboard struct {
	collector     *Collector
	width         int
	height        int
	active        bool
	selectedTab   int
	tabs          []string
	refreshRate   time.Duration
	lastRefresh   time.Time
}

// NewDashboard 创建性能仪表盘
func NewDashboard(collector *Collector) *Dashboard {
	return &Dashboard{
		collector:   collector,
		width:       80,
		height:      24,
		tabs:        []string{"概览", "会话", "指标", "系统"},
		refreshRate: time.Second,
	}
}

// SetSize 设置大小
func (d *Dashboard) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Activate 激活仪表盘
func (d *Dashboard) Activate() {
	d.active = true
	d.lastRefresh = time.Now()
}

// Deactivate 停用仪表盘
func (d *Dashboard) Deactivate() {
	d.active = false
}

// IsActive 是否激活
func (d *Dashboard) IsActive() bool {
	return d.active
}

// NextTab 下一个标签
func (d *Dashboard) NextTab() {
	d.selectedTab = (d.selectedTab + 1) % len(d.tabs)
}

// PrevTab 上一个标签
func (d *Dashboard) PrevTab() {
	d.selectedTab--
	if d.selectedTab < 0 {
		d.selectedTab = len(d.tabs) - 1
	}
}

// View 渲染仪表盘
func (d *Dashboard) View() string {
	if !d.active {
		return ""
	}

	var b strings.Builder
	width := d.width
	if width < 60 {
		width = 60
	}

	// 标题
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true)
	b.WriteString(titleStyle.Render("性能监控仪表盘"))
	b.WriteString("\n\n")

	// 标签栏
	b.WriteString(d.renderTabs())
	b.WriteString("\n\n")

	// 内容
	switch d.selectedTab {
	case 0:
		b.WriteString(d.renderOverview())
	case 1:
		b.WriteString(d.renderSession())
	case 2:
		b.WriteString(d.renderMetrics())
	case 3:
		b.WriteString(d.renderSystem())
	}

	b.WriteString("\n\n")

	// 帮助
	helpStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	b.WriteString(helpStyle.Render("←→: 切换标签  R: 刷新  Esc: 关闭"))

	// 边框
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Width(width)

	return boxStyle.Render(b.String())
}

// renderTabs 渲染标签栏
func (d *Dashboard) renderTabs() string {
	var b strings.Builder

	for i, tab := range d.tabs {
		if i == d.selectedTab {
			style := lipgloss.NewStyle().
				Foreground(styles.TextPrimary).
				Background(styles.Primary).
				Padding(0, 2).
				Bold(true)
			b.WriteString(style.Render(tab))
		} else {
			style := lipgloss.NewStyle().
				Foreground(styles.TextSecondary).
				Padding(0, 2)
			b.WriteString(style.Render(tab))
		}
		b.WriteString(" ")
	}

	return b.String()
}

// renderOverview 渲染概览
func (d *Dashboard) renderOverview() string {
	stats := d.collector.GetSessionStats()
	var b strings.Builder

	labelStyle := lipgloss.NewStyle().Foreground(styles.TextMuted).Width(16)
	valueStyle := lipgloss.NewStyle().Foreground(styles.TextPrimary).Bold(true)
	unitStyle := lipgloss.NewStyle().Foreground(styles.TextSecondary)

	// 会话时长
	duration := time.Since(stats.StartTime)
	b.WriteString(labelStyle.Render("会话时长:"))
	b.WriteString(valueStyle.Render(formatDuration(duration)))
	b.WriteString("\n")

	// 提示/响应数
	b.WriteString(labelStyle.Render("提示/响应:"))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d / %d", stats.PromptCount, stats.ResponseCount)))
	b.WriteString("\n")

	// 工具调用
	b.WriteString(labelStyle.Render("工具调用:"))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", stats.ToolCallCount)))
	b.WriteString("\n")

	// 平均延迟
	b.WriteString(labelStyle.Render("平均延迟:"))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%.0f", float64(stats.AverageLatency.Milliseconds()))))
	b.WriteString(unitStyle.Render(" ms"))
	b.WriteString("\n")

	// Token使用
	b.WriteString(labelStyle.Render("Token使用:"))
	b.WriteString(valueStyle.Render(formatNumber(stats.TotalTokens)))
	b.WriteString(unitStyle.Render(fmt.Sprintf(" (输入:%s 输出:%s)",
		formatNumber(stats.InputTokens), formatNumber(stats.OutputTokens))))
	b.WriteString("\n")

	// 错误数
	b.WriteString(labelStyle.Render("错误数:"))
	if stats.ErrorCount > 0 {
		errorStyle := lipgloss.NewStyle().Foreground(styles.Error).Bold(true)
		b.WriteString(errorStyle.Render(fmt.Sprintf("%d", stats.ErrorCount)))
	} else {
		successStyle := lipgloss.NewStyle().Foreground(styles.Success)
		b.WriteString(successStyle.Render("0"))
	}

	return b.String()
}

// renderSession 渲染会话详情
func (d *Dashboard) renderSession() string {
	stats := d.collector.GetSessionStats()
	var b strings.Builder

	labelStyle := lipgloss.NewStyle().Foreground(styles.TextMuted).Width(16)
	valueStyle := lipgloss.NewStyle().Foreground(styles.TextPrimary)

	b.WriteString(labelStyle.Render("会话ID:"))
	if stats.SessionID != "" {
		b.WriteString(valueStyle.Render(truncateString(stats.SessionID, 36)))
	} else {
		b.WriteString(valueStyle.Render("未设置"))
	}
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("开始时间:"))
	b.WriteString(valueStyle.Render(stats.StartTime.Format("2006-01-02 15:04:05")))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("最后提示:"))
	if !stats.LastPromptTime.IsZero() {
		b.WriteString(valueStyle.Render(stats.LastPromptTime.Format("15:04:05")))
	} else {
		b.WriteString(valueStyle.Render("-"))
	}
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("最后响应:"))
	if !stats.LastResponseTime.IsZero() {
		b.WriteString(valueStyle.Render(stats.LastResponseTime.Format("15:04:05")))
	} else {
		b.WriteString(valueStyle.Render("-"))
	}
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("总延迟:"))
	b.WriteString(valueStyle.Render(formatDuration(stats.TotalLatency)))

	return b.String()
}

// renderMetrics 渲染指标列表
func (d *Dashboard) renderMetrics() string {
	metrics := d.collector.GetAllMetrics()
	var b strings.Builder

	if len(metrics) == 0 {
		mutedStyle := lipgloss.NewStyle().Foreground(styles.TextMuted).Italic(true)
		return mutedStyle.Render("暂无指标数据")
	}

	labelStyle := lipgloss.NewStyle().Foreground(styles.TextMuted).Width(20)
	valueStyle := lipgloss.NewStyle().Foreground(styles.TextPrimary)
	unitStyle := lipgloss.NewStyle().Foreground(styles.TextSecondary)

	count := 0
	for name, metric := range metrics {
		if count >= 10 {
			b.WriteString("\n...")
			break
		}

		b.WriteString(labelStyle.Render(truncateString(name, 18) + ":"))
		b.WriteString(valueStyle.Render(fmt.Sprintf("%.2f", metric.Value)))
		if metric.Unit != "" {
			b.WriteString(unitStyle.Render(" " + metric.Unit))
		}
		b.WriteString("\n")
		count++
	}

	return b.String()
}

// renderSystem 渲染系统信息
func (d *Dashboard) renderSystem() string {
	var b strings.Builder

	labelStyle := lipgloss.NewStyle().Foreground(styles.TextMuted).Width(16)
	valueStyle := lipgloss.NewStyle().Foreground(styles.TextPrimary)
	unitStyle := lipgloss.NewStyle().Foreground(styles.TextSecondary)

	// 内存统计
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	b.WriteString(labelStyle.Render("分配内存:"))
	b.WriteString(valueStyle.Render(formatBytes(memStats.Alloc)))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("系统内存:"))
	b.WriteString(valueStyle.Render(formatBytes(memStats.Sys)))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("GC次数:"))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", memStats.NumGC)))
	b.WriteString("\n")

	// Goroutine数量
	b.WriteString(labelStyle.Render("Goroutines:"))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", runtime.NumGoroutine())))
	b.WriteString("\n")

	// CPU核心数
	b.WriteString(labelStyle.Render("CPU核心:"))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", runtime.NumCPU())))
	b.WriteString("\n")

	// Go版本
	b.WriteString(labelStyle.Render("Go版本:"))
	b.WriteString(valueStyle.Render(runtime.Version()))
	b.WriteString(unitStyle.Render(" " + runtime.GOOS + "/" + runtime.GOARCH))

	return b.String()
}

// 辅助函数

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
