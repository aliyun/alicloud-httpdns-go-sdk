package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

// TestResult 测试结果结构
type TestResult struct {
	TestName     string        `json:"test_name"`
	Status       string        `json:"status"`
	Duration     time.Duration `json:"duration"`
	ErrorMessage string        `json:"error_message,omitempty"`
	Metrics      TestMetrics   `json:"metrics,omitempty"`
}

// TestMetrics 测试指标
type TestMetrics struct {
	TotalRequests   int64         `json:"total_requests"`
	SuccessRequests int64         `json:"success_requests"`
	FailedRequests  int64         `json:"failed_requests"`
	SuccessRate     float64       `json:"success_rate"`
	AvgLatency      time.Duration `json:"avg_latency"`
	MinLatency      time.Duration `json:"min_latency"`
	MaxLatency      time.Duration `json:"max_latency"`
	P95Latency      time.Duration `json:"p95_latency"`
	P99Latency      time.Duration `json:"p99_latency"`
	QPS             float64       `json:"qps"`
	MemoryUsageMB   float64       `json:"memory_usage_mb"`
	CPUUsagePercent float64       `json:"cpu_usage_percent"`
}

// TestSuite 测试套件
type TestSuite struct {
	Name      string        `json:"name"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Results   []TestResult  `json:"results"`
	Summary   TestSummary   `json:"summary"`
}

// TestSummary 测试摘要
type TestSummary struct {
	TotalTests   int     `json:"total_tests"`
	PassedTests  int     `json:"passed_tests"`
	FailedTests  int     `json:"failed_tests"`
	SkippedTests int     `json:"skipped_tests"`
	PassRate     float64 `json:"pass_rate"`
}

// PerformanceThresholds 性能阈值
type PerformanceThresholds struct {
	MaxAvgLatency    time.Duration `json:"max_avg_latency"`
	MaxP95Latency    time.Duration `json:"max_p95_latency"`
	MaxP99Latency    time.Duration `json:"max_p99_latency"`
	MinSuccessRate   float64       `json:"min_success_rate"`
	MinQPS           float64       `json:"min_qps"`
	MaxMemoryUsageMB float64       `json:"max_memory_usage_mb"`
	MaxErrorRate     float64       `json:"max_error_rate"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	TestName   string   `json:"test_name"`
	Passed     bool     `json:"passed"`
	Violations []string `json:"violations,omitempty"`
}

// TestValidator 测试验证器
type TestValidator struct {
	thresholds PerformanceThresholds
}

// NewTestValidator 创建测试验证器
func NewTestValidator() *TestValidator {
	return &TestValidator{
		thresholds: PerformanceThresholds{
			MaxAvgLatency:    500 * time.Millisecond,
			MaxP95Latency:    1 * time.Second,
			MaxP99Latency:    2 * time.Second,
			MinSuccessRate:   0.95,
			MinQPS:           10.0,
			MaxMemoryUsageMB: 100.0,
			MaxErrorRate:     0.05,
		},
	}
}

// ValidateTestResult 验证单个测试结果
func (v *TestValidator) ValidateTestResult(result TestResult) ValidationResult {
	validation := ValidationResult{
		TestName:   result.TestName,
		Passed:     true,
		Violations: []string{},
	}

	// 检查测试状态
	if result.Status != "PASS" {
		validation.Passed = false
		validation.Violations = append(validation.Violations,
			fmt.Sprintf("Test failed with status: %s", result.Status))
		if result.ErrorMessage != "" {
			validation.Violations = append(validation.Violations,
				fmt.Sprintf("Error: %s", result.ErrorMessage))
		}
		return validation
	}

	metrics := result.Metrics

	// 验证延迟指标
	if metrics.AvgLatency > v.thresholds.MaxAvgLatency {
		validation.Passed = false
		validation.Violations = append(validation.Violations,
			fmt.Sprintf("Average latency %v exceeds threshold %v",
				metrics.AvgLatency, v.thresholds.MaxAvgLatency))
	}

	if metrics.P95Latency > v.thresholds.MaxP95Latency {
		validation.Passed = false
		validation.Violations = append(validation.Violations,
			fmt.Sprintf("P95 latency %v exceeds threshold %v",
				metrics.P95Latency, v.thresholds.MaxP95Latency))
	}

	if metrics.P99Latency > v.thresholds.MaxP99Latency {
		validation.Passed = false
		validation.Violations = append(validation.Violations,
			fmt.Sprintf("P99 latency %v exceeds threshold %v",
				metrics.P99Latency, v.thresholds.MaxP99Latency))
	}

	// 验证成功率
	if metrics.SuccessRate < v.thresholds.MinSuccessRate {
		validation.Passed = false
		validation.Violations = append(validation.Violations,
			fmt.Sprintf("Success rate %.2f%% below threshold %.2f%%",
				metrics.SuccessRate*100, v.thresholds.MinSuccessRate*100))
	}

	// 验证QPS
	if metrics.QPS < v.thresholds.MinQPS {
		validation.Passed = false
		validation.Violations = append(validation.Violations,
			fmt.Sprintf("QPS %.2f below threshold %.2f",
				metrics.QPS, v.thresholds.MinQPS))
	}

	// 验证内存使用
	if metrics.MemoryUsageMB > v.thresholds.MaxMemoryUsageMB {
		validation.Passed = false
		validation.Violations = append(validation.Violations,
			fmt.Sprintf("Memory usage %.2f MB exceeds threshold %.2f MB",
				metrics.MemoryUsageMB, v.thresholds.MaxMemoryUsageMB))
	}

	// 验证错误率
	errorRate := float64(metrics.FailedRequests) / float64(metrics.TotalRequests)
	if errorRate > v.thresholds.MaxErrorRate {
		validation.Passed = false
		validation.Violations = append(validation.Violations,
			fmt.Sprintf("Error rate %.2f%% exceeds threshold %.2f%%",
				errorRate*100, v.thresholds.MaxErrorRate*100))
	}

	return validation
}

// ValidateTestSuite 验证测试套件
func (v *TestValidator) ValidateTestSuite(suite TestSuite) []ValidationResult {
	var validations []ValidationResult

	for _, result := range suite.Results {
		validation := v.ValidateTestResult(result)
		validations = append(validations, validation)
	}

	return validations
}

// ParseGoTestOutput 解析Go测试输出
func ParseGoTestOutput(output string) (TestSuite, error) {
	suite := TestSuite{
		Name:      "HTTPDNS Go SDK Tests",
		StartTime: time.Now(),
		Results:   []TestResult{},
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析测试结果行
		if strings.HasPrefix(line, "--- PASS:") || strings.HasPrefix(line, "--- FAIL:") {
			result := parseTestResultLine(line)
			if result.TestName != "" {
				suite.Results = append(suite.Results, result)
			}
		}
	}

	suite.EndTime = time.Now()
	suite.Duration = suite.EndTime.Sub(suite.StartTime)
	suite.Summary = calculateSummary(suite.Results)

	return suite, nil
}

// parseTestResultLine 解析测试结果行
func parseTestResultLine(line string) TestResult {
	result := TestResult{}

	parts := strings.Fields(line)
	if len(parts) < 3 {
		return result
	}

	// 解析状态
	if strings.Contains(line, "PASS") {
		result.Status = "PASS"
	} else if strings.Contains(line, "FAIL") {
		result.Status = "FAIL"
	} else {
		result.Status = "UNKNOWN"
	}

	// 解析测试名称
	result.TestName = parts[2]

	// 解析持续时间
	if len(parts) >= 4 {
		durationStr := strings.Trim(parts[3], "()")
		if duration, err := time.ParseDuration(durationStr); err == nil {
			result.Duration = duration
		}
	}

	return result
}

// calculateSummary 计算测试摘要
func calculateSummary(results []TestResult) TestSummary {
	summary := TestSummary{
		TotalTests: len(results),
	}

	for _, result := range results {
		switch result.Status {
		case "PASS":
			summary.PassedTests++
		case "FAIL":
			summary.FailedTests++
		default:
			summary.SkippedTests++
		}
	}

	if summary.TotalTests > 0 {
		summary.PassRate = float64(summary.PassedTests) / float64(summary.TotalTests)
	}

	return summary
}

// GenerateReport 生成验证报告
func GenerateReport(validations []ValidationResult, outputPath string) error {
	report := map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339),
		"total_tests": len(validations),
		"validations": validations,
	}

	// 计算统计信息
	passedCount := 0
	for _, v := range validations {
		if v.Passed {
			passedCount++
		}
	}

	report["passed_tests"] = passedCount
	report["failed_tests"] = len(validations) - passedCount
	report["pass_rate"] = float64(passedCount) / float64(len(validations))

	// 生成JSON报告
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %v", err)
	}

	if err := ioutil.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write report: %v", err)
	}

	return nil
}

// main 主函数
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run validate_test_results.go <test_output_file> [output_report_file]")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := "validation_report.json"
	if len(os.Args) >= 3 {
		outputFile = os.Args[2]
	}

	// 读取测试输出
	content, err := ioutil.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("Failed to read input file: %v", err)
	}

	// 解析测试结果
	suite, err := ParseGoTestOutput(string(content))
	if err != nil {
		log.Fatalf("Failed to parse test output: %v", err)
	}

	// 验证测试结果
	validator := NewTestValidator()
	validations := validator.ValidateTestSuite(suite)

	// 生成报告
	if err := GenerateReport(validations, outputFile); err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	// 打印摘要
	fmt.Printf("Test Validation Summary:\n")
	fmt.Printf("========================\n")
	fmt.Printf("Total Tests: %d\n", len(validations))

	passedCount := 0
	failedCount := 0
	for _, v := range validations {
		if v.Passed {
			passedCount++
		} else {
			failedCount++
		}
	}

	fmt.Printf("Passed: %d\n", passedCount)
	fmt.Printf("Failed: %d\n", failedCount)
	fmt.Printf("Pass Rate: %.2f%%\n", float64(passedCount)/float64(len(validations))*100)
	fmt.Printf("\nReport saved to: %s\n", outputFile)

	// 显示失败的测试
	if failedCount > 0 {
		fmt.Printf("\nFailed Tests:\n")
		fmt.Printf("=============\n")
		for _, v := range validations {
			if !v.Passed {
				fmt.Printf("- %s\n", v.TestName)
				for _, violation := range v.Violations {
					fmt.Printf("  * %s\n", violation)
				}
			}
		}
		os.Exit(1)
	}
}
