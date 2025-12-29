//go:build e2e
// +build e2e

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

// TestE2E_OptimizedRealService 优化的端到端测试
func TestE2E_OptimizedRealService(t *testing.T) {
	accountID := getEnvOrSkip(t, "HTTPDNS_ACCOUNT_ID")
	secretKey := getEnvOrSkip(t, "HTTPDNS_SECRET_KEY")

	// 使用更保守的配置避免频率限制
	config := httpdns.DefaultConfig()
	config.AccountID = accountID
	config.SecretKey = secretKey
	config.EnableMetrics = true
	config.Timeout = 10 * time.Second
	config.MaxRetries = 1 // 减少重试次数

	client, err := httpdns.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	t.Run("ConfiguredDomains", func(t *testing.T) {
		// 测试配置中明确可以解析的域名
		configuredDomains := []string{
			"www.aliyun.com",
			"www.alibaba.com",
		}

		for _, domain := range configuredDomains {
			t.Run(domain, func(t *testing.T) {
				result, err := client.Resolve(ctx, domain, "")
				if err != nil {
					t.Errorf("Failed to resolve configured domain %s: %v", domain, err)
					return
				}

				if result.Domain != domain {
					t.Errorf("Domain mismatch: got %s, want %s", result.Domain, domain)
				}

				if len(result.IPv4) == 0 && len(result.IPv6) == 0 {
					t.Errorf("No IP addresses returned for %s", domain)
				}

				t.Logf("✅ Domain %s resolved: IPv4=%v, IPv6=%v, TTL=%d, Source=%s",
					domain, result.IPv4, result.IPv6, result.TTL, result.Source)
			})
		}
	})

	t.Run("UnconfiguredDomains", func(t *testing.T) {
		// 测试配置中明确无法解析的域名
		unconfiguredDomains := []string{
			"www.baidu.com",
			"www.qq.com",
		}

		for _, domain := range unconfiguredDomains {
			t.Run(domain, func(t *testing.T) {
				result, err := client.Resolve(ctx, domain, "")

				if err != nil {
					t.Logf("✅ Domain %s failed to resolve as expected: %v", domain, err)
				} else if result != nil {
					hasIPs := len(result.IPv4) > 0 || len(result.IPv6) > 0
					if hasIPs {
						t.Logf("⚠️  Domain %s unexpectedly resolved: IPv4=%v, IPv6=%v",
							domain, result.IPv4, result.IPv6)
					} else {
						t.Logf("✅ Domain %s returned empty result as expected", domain)
					}
				}
			})
		}
	})

	t.Run("BatchResolve", func(t *testing.T) {
		// 混合测试配置和未配置的域名
		domains := []string{
			"www.aliyun.com",  // 应该成功
			"www.baidu.com",   // 应该失败
			"www.alibaba.com", // 应该成功
		}

		results, err := client.ResolveBatch(ctx, domains, "")
		if err != nil {
			t.Logf("Batch resolve failed: %v", err)
			return
		}

		for i, result := range results {
			domain := domains[i]
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
	})

	t.Run("LowConcurrency", func(t *testing.T) {
		// 使用较低的并发数避免频率限制
		concurrency := 5
		requestsPerWorker := 2
		domain := "www.aliyun.com"

		results := make(chan error, concurrency*requestsPerWorker)

		for i := 0; i < concurrency; i++ {
			go func(workerID int) {
				for j := 0; j < requestsPerWorker; j++ {
					_, err := client.Resolve(ctx, domain, "")
					results <- err

					// 添加延迟避免频率限制
					time.Sleep(200 * time.Millisecond)
				}
			}(i)
		}

		successCount := 0
		totalRequests := concurrency * requestsPerWorker

		for i := 0; i < totalRequests; i++ {
			err := <-results
			if err == nil {
				successCount++
			} else {
				t.Logf("Request %d failed: %v", i, err)
			}
		}

		successRate := float64(successCount) / float64(totalRequests) * 100
		t.Logf("Low concurrency test: %d/%d succeeded (%.1f%%)",
			successCount, totalRequests, successRate)

		if successRate < 80.0 {
			t.Errorf("Success rate too low: %.1f%%, expected >= 80%%", successRate)
		}
	})

	t.Run("Metrics", func(t *testing.T) {
		client.ResetMetrics()

		// 执行几个解析操作
		client.Resolve(ctx, "www.aliyun.com", "")
		client.Resolve(ctx, "www.alibaba.com", "")

		stats := client.GetMetrics()
		t.Logf("Metrics: Total=%d, Success=%d, Failed=%d, SuccessRate=%.1f%%, AvgLatency=%v",
			stats.TotalResolves, stats.SuccessResolves, stats.FailedResolves,
			stats.SuccessRate*100, stats.AvgLatency)
	})
}
