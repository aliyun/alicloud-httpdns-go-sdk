//go:build integration
// +build integration

package httpdns

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestIntegration_EndToEnd 端到端集成测试
func TestIntegration_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := DefaultConfig()
	config.AccountID = "test123" // 在实际测试中需要使用真实的 Account ID
	config.EnableMetrics = true

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	t.Run("SingleResolve", func(t *testing.T) {
		result, err := client.Resolve(ctx, "example.com", WithClientIP("1.2.3.4"))
		if err != nil {
			t.Errorf("Resolve failed: %v", err)
			return
		}

		if result.Domain != "example.com" {
			t.Errorf("Domain = %v, want example.com", result.Domain)
		}

		if len(result.IPv4) == 0 && len(result.IPv6) == 0 {
			t.Error("No IP addresses returned")
		}

		if result.TTL <= 0 {
			t.Error("TTL should be positive")
		}
	})

	t.Run("BatchResolve", func(t *testing.T) {
		domains := []string{"example.com", "google.com", "github.com"}
		results, err := client.ResolveBatch(ctx, domains, WithClientIP("1.2.3.4"))
		if err != nil {
			t.Errorf("ResolveBatch failed: %v", err)
			return
		}

		if len(results) != len(domains) {
			t.Errorf("Got %d results, want %d", len(results), len(domains))
		}

		for i, result := range results {
			if result.Domain != domains[i] {
				t.Errorf("Result[%d] domain = %v, want %v", i, result.Domain, domains[i])
			}
		}
	})

	t.Run("AsyncResolve", func(t *testing.T) {
		done := make(chan bool, 1)
		var result *ResolveResult
		var asyncErr error

		client.ResolveAsync(ctx, "example.com", func(r *ResolveResult, err error) {
			result = r
			asyncErr = err
			done <- true
		}, WithClientIP("1.2.3.4"))

		select {
		case <-done:
			if asyncErr != nil {
				t.Errorf("Async resolve failed: %v", asyncErr)
			}
			if result == nil {
				t.Error("Async resolve returned nil result")
			}
		case <-time.After(30 * time.Second):
			t.Error("Async resolve timeout")
		}
	})

	t.Run("Metrics", func(t *testing.T) {
		stats := client.GetMetrics()
		if stats.TotalResolves == 0 {
			t.Error("No resolves recorded in metrics")
		}
	})
}

// TestIntegration_Concurrency 并发测试
func TestIntegration_Concurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := DefaultConfig()
	config.AccountID = "test123"
	config.EnableMetrics = true

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	concurrency := 50
	iterations := 10

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*iterations)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				domain := "example.com"

				_, err := client.Resolve(ctx, domain, WithClientIP("1.2.3.4"))
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for err := range errors {
		t.Logf("Concurrent resolve error: %v", err)
		errorCount++
	}

	stats := client.GetMetrics()
	t.Logf("Concurrent test stats: Total=%d, Success=%d, Failed=%d",
		stats.TotalResolves, stats.SuccessResolves, stats.FailedResolves)

	// 允许一定的错误率
	totalRequests := int64(concurrency * iterations)
	if errorCount > int(totalRequests/10) { // 允许10%的错误率
		t.Errorf("Too many errors in concurrent test: %d/%d", errorCount, totalRequests)
	}
}

// TestIntegration_FailoverAndRecovery 故障转移和恢复测试
func TestIntegration_FailoverAndRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := DefaultConfig()
	config.AccountID = "test123"
	config.EnableMetrics = true
	// 使用一些不可达的启动IP来测试故障转移
	config.BootstrapIPs = []string{"192.0.2.1", "192.0.2.2", "203.107.1.1"}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 尝试解析，由于没有可用的服务IP，应该返回错误
	_, err = client.Resolve(ctx, "localhost", WithClientIP("127.0.0.1"))
	if err == nil {
		t.Error("Expected error with unreachable bootstrap IPs, but got none")
		return
	}

	// 检查解析结果
	stats := client.GetMetrics()
	t.Logf("Total resolves: %d, Success resolves: %d, Failed resolves: %d",
		stats.TotalResolves, stats.SuccessResolves, stats.FailedResolves)
	t.Logf("Expected error with unreachable bootstrap IPs: %v", err)
}

// TestIntegration_NetworkConditions 不同网络条件测试
func TestIntegration_NetworkConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testCases := []struct {
		name    string
		timeout time.Duration
		retries int
	}{
		{"FastNetwork", 10 * time.Second, 3},
		{"SlowNetwork", 30 * time.Second, 5},
		{"UnstableNetwork", 5 * time.Second, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultConfig()
			config.AccountID = "test123"
			config.Timeout = tc.timeout
			config.MaxRetries = tc.retries
			config.EnableMetrics = true

			client, err := NewClient(config)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			ctx := context.Background()
			start := time.Now()

			result, err := client.Resolve(ctx, "example.com", WithClientIP("1.2.3.4"))
			duration := time.Since(start)

			if err != nil {
				t.Logf("Resolve failed after %v: %v", duration, err)
			} else {
				t.Logf("Resolve succeeded after %v, got %d IPv4 addresses",
					duration, len(result.IPv4))
			}

			stats := client.GetMetrics()
			t.Logf("Stats: Total=%d, Success=%d, Failed=%d, AvgLatency=%v",
				stats.TotalResolves, stats.SuccessResolves, stats.FailedResolves, stats.AvgLatency)
		})
	}
}

// TestIntegration_LongRunning 长时间运行测试
func TestIntegration_LongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := DefaultConfig()
	config.AccountID = "test123"
	config.EnableMetrics = true

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	duration := 2 * time.Minute // 运行2分钟
	interval := 5 * time.Second // 每5秒解析一次

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timeout := time.After(duration)
	resolveCount := 0

	for {
		select {
		case <-ticker.C:
			_, err := client.Resolve(ctx, "example.com", WithClientIP("1.2.3.4"))
			if err != nil {
				t.Logf("Long running resolve error: %v", err)
			}
			resolveCount++

		case <-timeout:
			stats := client.GetMetrics()
			t.Logf("Long running test completed:")
			t.Logf("  Duration: %v", duration)
			t.Logf("  Resolves attempted: %d", resolveCount)
			t.Logf("  Total resolves: %d", stats.TotalResolves)
			t.Logf("  Success rate: %.2f%%", stats.SuccessRate*100)
			t.Logf("  Average latency: %v", stats.AvgLatency)
			return
		}
	}
}
