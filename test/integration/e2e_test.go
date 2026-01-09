//go:build e2e
// +build e2e

package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

// TestE2E_RealService 真实服务端到端测试
// 需要设置环境变量 HTTPDNS_ACCOUNT_ID 和 HTTPDNS_SECRET_KEY
func TestE2E_RealService(t *testing.T) {
	// 这个测试需要真实的阿里云HTTPDNS服务配置
	// 在CI/CD环境中可以通过环境变量配置
	accountID := getEnvOrSkip(t, "HTTPDNS_ACCOUNT_ID")
	secretKey := getEnvOrSkip(t, "HTTPDNS_SECRET_KEY")

	config := httpdns.DefaultConfig()
	config.AccountID = accountID
	config.SecretKey = secretKey
	config.EnableMetrics = true

	client, err := httpdns.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	t.Run("SingleDomainResolve", func(t *testing.T) {
		testDomains := []string{
			"www.aliyun.com",
			"www.taobao.com",
			"www.github.com",
		}

		for _, domain := range testDomains {
			t.Run(domain, func(t *testing.T) {
				result, err := client.Resolve(ctx, domain)
				if err != nil {
					t.Errorf("Failed to resolve %s: %v", domain, err)
					return
				}

				if result.Domain != domain {
					t.Errorf("Domain mismatch: got %s, want %s", result.Domain, domain)
				}

				if len(result.IPv4) == 0 && len(result.IPv6) == 0 {
					t.Errorf("No IP addresses returned for %s", domain)
				}

				if result.TTL <= 0 {
					t.Errorf("Invalid TTL for %s: %d", domain, result.TTL)
				}

				if result.Source != httpdns.SourceHTTPDNS {
					t.Logf("Domain %s resolved via %s instead of HTTPDNS", domain, result.Source)
				}

				t.Logf("Domain %s resolved: IPv4=%v, IPv6=%v, TTL=%d, Source=%s",
					domain, result.IPv4, result.IPv6, result.TTL, result.Source)
			})
		}
	})

	t.Run("BatchDomainResolve", func(t *testing.T) {
		domains := []string{
			"www.aliyun.com",
			"www.taobao.com",
			"ecs.aliyuncs.com",
		}

		results, err := client.ResolveBatch(ctx, domains)
		if err != nil {
			t.Fatalf("Failed to resolve batch: %v", err)
		}

		if len(results) != len(domains) {
			t.Fatalf("Result count mismatch: got %d, want %d", len(results), len(domains))
		}

		for i, result := range results {
			if result.Error != nil {
				t.Errorf("Failed to resolve %s: %v", domains[i], result.Error)
				continue
			}

			if result.Domain != domains[i] {
				t.Errorf("Domain mismatch at index %d: got %s, want %s", i, result.Domain, domains[i])
			}

			if len(result.IPv4) == 0 && len(result.IPv6) == 0 {
				t.Errorf("No IP addresses returned for %s", domains[i])
			}

			t.Logf("Batch result %d: %s -> IPv4=%v, IPv6=%v, TTL=%d",
				i, result.Domain, result.IPv4, result.IPv6, result.TTL)
		}
	})

	t.Run("AsyncResolve", func(t *testing.T) {
		domain := "www.aliyun.com"
		done := make(chan bool, 1)
		var result *httpdns.ResolveResult
		var asyncErr error

		client.ResolveAsync(ctx, domain, func(r *httpdns.ResolveResult, err error) {
			result = r
			asyncErr = err
			done <- true
		})

		select {
		case <-done:
			if asyncErr != nil {
				t.Errorf("Async resolve failed: %v", asyncErr)
			} else if result == nil {
				t.Error("Async resolve returned nil result")
			} else {
				t.Logf("Async resolve success: %s -> IPv4=%v, IPv6=%v",
					result.Domain, result.IPv4, result.IPv6)
			}
		case <-time.After(30 * time.Second):
			t.Error("Async resolve timeout")
		}
	})

	t.Run("MetricsValidation", func(t *testing.T) {
		// 重置指标
		client.ResetMetrics()

		// 执行一些解析操作
		testDomains := []string{"www.aliyun.com", "www.taobao.com"}
		for _, domain := range testDomains {
			client.Resolve(ctx, domain)
		}

		// 检查指标
		stats := client.GetMetrics()
		if stats.TotalResolves == 0 {
			t.Error("No resolves recorded in metrics")
		}

		t.Logf("Metrics: Total=%d, Success=%d, Failed=%d, SuccessRate=%.2f%%, AvgLatency=%v",
			stats.TotalResolves, stats.SuccessResolves, stats.FailedResolves,
			stats.SuccessRate*100, stats.AvgLatency)
	})
}

// TestE2E_PerformanceStress 性能压力测试
func TestE2E_PerformanceStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	accountID := getEnvOrSkip(t, "HTTPDNS_ACCOUNT_ID")
	secretKey := getEnvOrSkip(t, "HTTPDNS_SECRET_KEY")

	config := httpdns.DefaultConfig()
	config.AccountID = accountID
	config.SecretKey = secretKey
	config.EnableMetrics = true
	config.Timeout = 10 * time.Second

	client, err := httpdns.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	t.Run("HighConcurrency", func(t *testing.T) {
		concurrency := 100
		requestsPerWorker := 10
		domain := "www.aliyun.com"

		var wg sync.WaitGroup
		errors := make(chan error, concurrency*requestsPerWorker)
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < requestsPerWorker; j++ {
					_, err := client.Resolve(ctx, domain)
					if err != nil {
						errors <- fmt.Errorf("worker %d request %d: %v", workerID, j, err)
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)
		close(errors)

		errorCount := 0
		for err := range errors {
			t.Logf("Concurrent error: %v", err)
			errorCount++
		}

		totalRequests := concurrency * requestsPerWorker
		successRate := float64(totalRequests-errorCount) / float64(totalRequests) * 100
		qps := float64(totalRequests) / duration.Seconds()

		t.Logf("High concurrency test results:")
		t.Logf("  Concurrency: %d workers", concurrency)
		t.Logf("  Total requests: %d", totalRequests)
		t.Logf("  Duration: %v", duration)
		t.Logf("  Success rate: %.2f%% (%d/%d)", successRate, totalRequests-errorCount, totalRequests)
		t.Logf("  QPS: %.2f", qps)
		t.Logf("  Average latency: %v", duration/time.Duration(totalRequests))

		// 验证成功率不低于90%
		if successRate < 90.0 {
			t.Errorf("Success rate too low: %.2f%%, expected >= 90%%", successRate)
		}
	})

	t.Run("SustainedLoad", func(t *testing.T) {
		duration := 2 * time.Minute
		interval := 100 * time.Millisecond
		domain := "www.aliyun.com"

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		timeout := time.After(duration)
		requestCount := 0
		errorCount := 0
		start := time.Now()

		for {
			select {
			case <-ticker.C:
				_, err := client.Resolve(ctx, domain)
				requestCount++
				if err != nil {
					errorCount++
					t.Logf("Sustained load error: %v", err)
				}

			case <-timeout:
				actualDuration := time.Since(start)
				successRate := float64(requestCount-errorCount) / float64(requestCount) * 100
				qps := float64(requestCount) / actualDuration.Seconds()

				t.Logf("Sustained load test results:")
				t.Logf("  Duration: %v", actualDuration)
				t.Logf("  Total requests: %d", requestCount)
				t.Logf("  Success rate: %.2f%% (%d/%d)", successRate, requestCount-errorCount, requestCount)
				t.Logf("  QPS: %.2f", qps)

				stats := client.GetMetrics()
				t.Logf("  Final metrics: AvgLatency=%v, MinLatency=%v, MaxLatency=%v",
					stats.AvgLatency, stats.MinLatency, stats.MaxLatency)

				return
			}
		}
	})
}

// TestE2E_ErrorHandling 错误处理场景测试
func TestE2E_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error handling test in short mode")
	}

	accountID := getEnvOrSkip(t, "HTTPDNS_ACCOUNT_ID")
	secretKey := getEnvOrSkip(t, "HTTPDNS_SECRET_KEY")

	t.Run("InvalidBootstrapIPs", func(t *testing.T) {
		config := httpdns.DefaultConfig()
		config.AccountID = accountID
		config.SecretKey = secretKey
		config.EnableMetrics = true
		// 使用无效的启动IP来测试错误处理
		config.BootstrapIPs = []string{"192.0.2.1", "192.0.2.2"}

		client, err := httpdns.NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close()

		ctx := context.Background()
		_, err = client.Resolve(ctx, "www.aliyun.com")

		// 应该返回错误，因为无法连接到服务
		if err == nil {
			t.Error("Expected error with invalid bootstrap IPs, but got none")
		} else {
			t.Logf("Expected error with invalid bootstrap IPs: %v", err)
		}

		stats := client.GetMetrics()
		t.Logf("Error handling stats: Failed=%d, Total=%d", stats.FailedResolves, stats.TotalResolves)
	})

	t.Run("NetworkTimeout", func(t *testing.T) {
		config := httpdns.DefaultConfig()
		config.AccountID = accountID
		config.SecretKey = secretKey
		config.Timeout = 1 * time.Second // 很短的超时时间
		config.MaxRetries = 1

		client, err := httpdns.NewClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close()

		ctx := context.Background()
		start := time.Now()
		result, err := client.Resolve(ctx, "www.aliyun.com")
		duration := time.Since(start)

		t.Logf("Timeout test completed in %v", duration)
		if err != nil {
			t.Logf("Resolve with short timeout failed as expected: %v", err)
		} else {
			t.Logf("Resolve succeeded via %s", result.Source)
		}

		// 验证超时时间合理
		if duration > 10*time.Second {
			t.Errorf("Resolve took too long: %v", duration)
		}
	})
}

// TestE2E_ErrorCases 错误用例测试
func TestE2E_ErrorCases(t *testing.T) {
	accountID := getEnvOrSkip(t, "HTTPDNS_ACCOUNT_ID")
	secretKey := getEnvOrSkip(t, "HTTPDNS_SECRET_KEY")

	config := httpdns.DefaultConfig()
	config.AccountID = accountID
	config.SecretKey = secretKey
	// 测试 HTTPDNS 行为

	client, err := httpdns.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	t.Run("UnconfiguredDomains", func(t *testing.T) {
		// 根据配置规则，这些域名可能无法通过HTTPDNS解析或返回不一致的结果
		unconfiguredDomains := []string{
			"www.baidu.com",
			"www.qq.com",
		}

		for _, domain := range unconfiguredDomains {
			t.Run(domain, func(t *testing.T) {
				result, err := client.Resolve(ctx, domain)

				if err != nil {
					t.Logf("Domain %s failed to resolve: %v", domain, err)

					// 验证错误类型
					var httpDNSErr *httpdns.HTTPDNSError
					if errors.As(err, &httpDNSErr) {
						t.Logf("httpdns.HTTPDNSError details - Op: %s, Domain: %s, Err: %v",
							httpDNSErr.Op, httpDNSErr.Domain, httpDNSErr.Err)
					}
				} else if result != nil {
					hasIPs := len(result.IPv4) > 0 || len(result.IPv6) > 0
					if hasIPs {
						t.Logf("Domain %s resolved: IPv4=%v, IPv6=%v, Source=%s, TTL=%d",
							domain, result.IPv4, result.IPv6, result.Source, result.TTL)

						// 记录这是一个未配置域名但仍然解析成功的情况
						t.Logf("NOTE: Domain %s is marked as unconfigured but still resolved", domain)
					} else {
						t.Logf("Domain %s returned empty result: Source=%s, TTL=%d",
							domain, result.Source, result.TTL)
					}
				}
			})
		}
	})

	t.Run("BatchUnconfiguredDomains", func(t *testing.T) {
		// 测试批量解析未配置的域名
		unconfiguredDomains := []string{
			"www.baidu.com",
			"www.qq.com",
			"www.google.com", // 额外的未配置域名
		}

		results, err := client.ResolveBatch(ctx, unconfiguredDomains)
		if err != nil {
			t.Logf("Batch resolve of unconfigured domains failed: %v", err)
		} else {
			for i, result := range results {
				domain := unconfiguredDomains[i]
				if result.Error != nil {
					t.Logf("Batch result %d: %s failed: %v", i, domain, result.Error)
				} else {
					hasIPs := len(result.IPv4) > 0 || len(result.IPv6) > 0
					if hasIPs {
						t.Logf("Batch result %d: %s resolved: IPv4=%v, IPv6=%v",
							i, domain, result.IPv4, result.IPv6)
					} else {
						t.Logf("Batch result %d: %s returned empty result", i, domain)
					}
				}
			}
		}
	})

	t.Run("MixedConfiguredAndUnconfigured", func(t *testing.T) {
		// 测试混合配置和未配置的域名
		mixedDomains := []string{
			"www.aliyun.com", // 可以解析
			"www.baidu.com",  // 无法解析
			"www.taobao.com", // 可以解析
			"www.qq.com",     // 无法解析
		}

		results, err := client.ResolveBatch(ctx, mixedDomains)
		if err != nil {
			t.Logf("Mixed batch resolve failed: %v", err)
		} else {
			expectedResults := map[string]bool{
				"www.aliyun.com": true,  // 应该成功
				"www.baidu.com":  false, // 应该失败
				"www.taobao.com": true,  // 应该成功
				"www.qq.com":     false, // 应该失败
			}

			for i, result := range results {
				domain := mixedDomains[i]
				shouldSucceed := expectedResults[domain]

				hasIPs := len(result.IPv4) > 0 || len(result.IPv6) > 0

				if shouldSucceed {
					if result.Error != nil || !hasIPs {
						t.Errorf("Domain %s should have resolved but failed: Error=%v, IPv4=%v, IPv6=%v",
							domain, result.Error, result.IPv4, result.IPv6)
					} else {
						t.Logf("Domain %s resolved successfully as expected: IPv4=%v, IPv6=%v",
							domain, result.IPv4, result.IPv6)
					}
				} else {
					if result.Error == nil && hasIPs {
						t.Errorf("Domain %s should have failed but resolved: IPv4=%v, IPv6=%v",
							domain, result.IPv4, result.IPv6)
					} else {
						t.Logf("Domain %s failed to resolve as expected: Error=%v", domain, result.Error)
					}
				}
			}
		}
	})
}

// TestE2E_EdgeCases 边界情况测试
func TestE2E_EdgeCases(t *testing.T) {
	accountID := getEnvOrSkip(t, "HTTPDNS_ACCOUNT_ID")
	secretKey := getEnvOrSkip(t, "HTTPDNS_SECRET_KEY")

	config := httpdns.DefaultConfig()
	config.AccountID = accountID
	config.SecretKey = secretKey

	client, err := httpdns.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	t.Run("InvalidDomains", func(t *testing.T) {
		invalidDomains := []string{
			"",
			"invalid-domain-that-does-not-exist.invalid",
			"toolongdomainname" + string(make([]byte, 300)), // 超长域名
		}

		for _, domain := range invalidDomains {
			if domain == "" {
				continue // 空域名会在客户端验证阶段被拒绝
			}

			result, err := client.Resolve(ctx, domain)
			if err != nil {
				t.Logf("Invalid domain %s failed as expected: %v", domain, err)
			} else {
				t.Logf("Invalid domain %s unexpectedly succeeded: %v", domain, result.IPv4)
			}
		}
	})

	t.Run("SpecialCharacterDomains", func(t *testing.T) {
		specialDomains := []string{
			"xn--fsq.xn--0zwm56d",    // 中文域名的punycode
			"test-domain.com",        // 带连字符
			"sub.domain.example.com", // 多级子域名
		}

		for _, domain := range specialDomains {
			result, err := client.Resolve(ctx, domain)
			if err != nil {
				t.Logf("Special domain %s failed: %v", domain, err)
			} else {
				t.Logf("Special domain %s resolved: IPv4=%v", domain, result.IPv4)
			}
		}
	})

	t.Run("LargeBatchResolve", func(t *testing.T) {
		// 创建大批量域名列表
		domains := make([]string, 50)
		for i := 0; i < 50; i++ {
			domains[i] = fmt.Sprintf("test%d.aliyun.com", i)
		}

		start := time.Now()
		results, err := client.ResolveBatch(ctx, domains)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Large batch resolve failed: %v", err)
		} else {
			successCount := 0
			for _, result := range results {
				if result.Error == nil {
					successCount++
				}
			}
			t.Logf("Large batch resolve: %d/%d succeeded in %v",
				successCount, len(domains), duration)
		}
	})
}

// getEnvOrSkip 获取环境变量，如果不存在则跳过测试
func getEnvOrSkip(t *testing.T, key string) string {
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Environment variable %s not set, skipping test", key)
	}
	return value
}
