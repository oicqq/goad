// 指标收集测试
package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/anthropics/goad/internal/metrics"
)

// GetMetricsTests 返回指标收集测试用例
func GetMetricsTests() []TestCase {
	return []TestCase{
		{"创建收集器", "指标收集", testMetricsCollector},
		{"计数器指标", "指标收集", testMetricsCounter},
		{"计时器指标", "指标收集", testMetricsTiming},
		{"会话统计", "指标收集", testMetricsSession},
		{"并发写入", "指标收集", testMetricsConcurrent},
	}
}

func testMetricsCollector() error {
	collector := metrics.NewCollector()
	if collector == nil {
		return fmt.Errorf("收集器为空")
	}
	return nil
}

func testMetricsCounter() error {
	collector := metrics.NewCollector()
	collector.RecordCounter("requests", 1, "count", "请求计数")
	collector.RecordCounter("requests", 2, "count", "请求计数")

	metric, ok := collector.GetMetric("requests")
	if !ok {
		return fmt.Errorf("计数器指标未记录")
	}
	if metric.Value != 3 {
		return fmt.Errorf("计数器值不正确: %f", metric.Value)
	}
	return nil
}

func testMetricsTiming() error {
	collector := metrics.NewCollector()
	collector.StartTiming("operation")
	time.Sleep(20 * time.Millisecond)
	duration := collector.EndTiming("operation")

	if duration < 20*time.Millisecond {
		return fmt.Errorf("计时不正确: %v", duration)
	}
	return nil
}

func testMetricsSession() error {
	collector := metrics.NewCollector()
	collector.SetSessionID("test-session")
	collector.RecordPrompt()
	collector.RecordPrompt()
	collector.RecordResponse(100 * time.Millisecond)
	collector.RecordToolCall()
	collector.RecordTokens(100, 50)
	collector.RecordError()

	stats := collector.GetSessionStats()
	if stats.PromptCount != 2 {
		return fmt.Errorf("提示计数不正确: %d", stats.PromptCount)
	}
	if stats.TotalTokens != 150 {
		return fmt.Errorf("Token计数不正确: %d", stats.TotalTokens)
	}
	return nil
}

func testMetricsConcurrent() error {
	collector := metrics.NewCollector()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			collector.RecordCounter("concurrent", 1, "", "")
			collector.RecordPrompt()
		}()
	}
	wg.Wait()

	metric, _ := collector.GetMetric("concurrent")
	if metric.Value != 100 {
		return fmt.Errorf("并发计数不正确: %f", metric.Value)
	}
	return nil
}
