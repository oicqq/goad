// Package main 综合测试程序 - 主入口
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// TestResult 测试结果
type TestResult struct {
	Name     string
	Category string
	Passed   bool
	Duration time.Duration
	Error    string
}

// TestSuite 测试套件
type TestSuite struct {
	results []TestResult
	mu      sync.Mutex
}

// TestFunc 测试函数类型
type TestFunc func() error

// TestCase 测试用例
type TestCase struct {
	Name     string
	Category string
	Func     TestFunc
}

func (ts *TestSuite) AddResult(name, category string, passed bool, duration time.Duration, err error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	ts.results = append(ts.results, TestResult{
		Name:     name,
		Category: category,
		Passed:   passed,
		Duration: duration,
		Error:    errStr,
	})
}

func (ts *TestSuite) PrintResults() {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("测试结果汇总")
	fmt.Println(strings.Repeat("=", 70))

	// 按类别分组
	categories := make(map[string][]TestResult)
	for _, r := range ts.results {
		categories[r.Category] = append(categories[r.Category], r)
	}

	totalPassed := 0
	totalFailed := 0

	for cat, results := range categories {
		fmt.Printf("\n[%s]\n", cat)
		passed := 0
		failed := 0

		for _, r := range results {
			status := "✓ PASS"
			if !r.Passed {
				status = "✗ FAIL"
				failed++
				totalFailed++
			} else {
				passed++
				totalPassed++
			}

			fmt.Printf("  %s [%v] %s\n", status, r.Duration.Round(time.Millisecond), r.Name)
			if r.Error != "" {
				fmt.Printf("       错误: %s\n", r.Error)
			}
		}
		fmt.Printf("  小计: %d 通过, %d 失败\n", passed, failed)
	}

	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("总计: %d 测试, %d 通过, %d 失败\n", len(ts.results), totalPassed, totalFailed)

	if totalFailed > 0 {
		fmt.Println("状态: ✗ 部分测试失败")
	} else {
		fmt.Println("状态: ✓ 全部测试通过")
	}
}

func (ts *TestSuite) RunTest(tc TestCase) {
	start := time.Now()
	err := tc.Func()
	duration := time.Since(start)

	passed := err == nil
	ts.AddResult(tc.Name, tc.Category, passed, duration, err)

	status := "✓"
	if !passed {
		status = "✗"
	}
	fmt.Printf("  %s %s (%v)\n", status, tc.Name, duration.Round(time.Millisecond))
}

func (ts *TestSuite) RunTestsParallel(tests []TestCase) {
	var wg sync.WaitGroup
	for _, tc := range tests {
		wg.Add(1)
		go func(t TestCase) {
			defer wg.Done()
			ts.RunTest(t)
		}(tc)
	}
	wg.Wait()
}

func main() {
	// 命令行参数
	category := flag.String("c", "all", "测试类别: all, config, session, metrics, plugin, hotreload, acp")
	parallel := flag.Bool("p", false, "并行运行测试")
	flag.Parse()

	fmt.Println("=== Goad 综合测试程序 ===")
	fmt.Println("开始时间:", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("测试类别: %s\n", *category)
	fmt.Println()

	suite := &TestSuite{}

	// 收集测试用例
	var tests []TestCase

	if *category == "all" || *category == "config" {
		tests = append(tests, GetConfigTests()...)
	}
	if *category == "all" || *category == "session" {
		tests = append(tests, GetSessionTests()...)
	}
	if *category == "all" || *category == "metrics" {
		tests = append(tests, GetMetricsTests()...)
	}
	if *category == "all" || *category == "plugin" {
		tests = append(tests, GetPluginTests()...)
	}
	if *category == "all" || *category == "hotreload" {
		tests = append(tests, GetHotreloadTests()...)
	}
	if *category == "all" || *category == "acp" {
		tests = append(tests, GetACPTests()...)
	}

	if len(tests) == 0 {
		fmt.Println("未找到匹配的测试用例")
		os.Exit(1)
	}

	fmt.Printf("共 %d 个测试用例\n\n", len(tests))

	// 运行测试
	if *parallel {
		fmt.Println("--- 并行模式 ---")
		suite.RunTestsParallel(tests)
	} else {
		fmt.Println("--- 串行模式 ---")
		currentCategory := ""
		for _, tc := range tests {
			if tc.Category != currentCategory {
				currentCategory = tc.Category
				fmt.Printf("\n[%s]\n", currentCategory)
			}
			suite.RunTest(tc)
		}
	}

	// 打印结果
	suite.PrintResults()

	// 返回状态码
	for _, r := range suite.results {
		if !r.Passed {
			os.Exit(1)
		}
	}
}
