package benchmark

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

// BenchmarkResolve 单域名解析性能基准测试
func BenchmarkResolve(b *testing.B) {
	config := httpdns.DefaultConfig()
	config.AccountID = "test123"

	client, err := httpdns.NewClient(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Resolve(ctx, "example.com", "1.2.3.4")
			if err != nil {
				b.Logf("Resolve error: %v", err)
			}
		}
	})
}

// BenchmarkResolveBatch 批量解析性能基准测试
func BenchmarkResolveBatch(b *testing.B) {
	config := httpdns.DefaultConfig()
	config.AccountID = "test123"

	client, err := httpdns.NewClient(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	domains := []string{"example.com", "google.com", "github.com", "stackoverflow.com", "reddit.com"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.ResolveBatch(ctx, domains, "1.2.3.4")
			if err != nil {
				b.Logf("ResolveBatch error: %v", err)
			}
		}
	})
}

// BenchmarkResolveAsync 异步解析性能基准测试
func BenchmarkResolveAsync(b *testing.B) {
	config := httpdns.DefaultConfig()
	config.AccountID = "test123"

	client, err := httpdns.NewClient(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			done := make(chan bool, 1)
			client.ResolveAsync(ctx, "example.com", "1.2.3.4", func(result *httpdns.ResolveResult, err error) {
				if err != nil {
					b.Logf("Async resolve error: %v", err)
				}
				done <- true
			})
			<-done
		}
	})
}

// BenchmarkConcurrentResolve 并发解析性能基准测试
func BenchmarkConcurrentResolve(b *testing.B) {
	config := httpdns.DefaultConfig()
	config.AccountID = "test123"
	config.EnableMetrics = true

	client, err := httpdns.NewClient(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	concurrency := 100

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(concurrency)

		for j := 0; j < concurrency; j++ {
			go func() {
				defer wg.Done()
				_, err := client.Resolve(ctx, "example.com", "1.2.3.4")
				if err != nil {
					b.Logf("Concurrent resolve error: %v", err)
				}
			}()
		}

		wg.Wait()
	}
}

// BenchmarkMetricsCollection 指标收集性能基准测试
func BenchmarkMetricsCollection(b *testing.B) {
	config := httpdns.DefaultConfig()
	config.AccountID = "test123"
	config.EnableMetrics = true

	client, err := httpdns.NewClient(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 预热，生成一些指标数据
	for i := 0; i < 100; i++ {
		client.Resolve(ctx, "example.com", "1.2.3.4")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = client.GetMetrics()
		}
	})
}

// BenchmarkClientCreation 客户端创建性能基准测试
func BenchmarkClientCreation(b *testing.B) {
	config := httpdns.DefaultConfig()
	config.AccountID = "test123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, err := httpdns.NewClient(config)
		if err != nil {
			b.Fatalf("Failed to create client: %v", err)
		}
		client.Close()
	}
}

// BenchmarkConfigValidation 配置验证性能基准测试
func BenchmarkConfigValidation(b *testing.B) {
	config := httpdns.DefaultConfig()
	config.AccountID = "test123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := config.Validate()
		if err != nil {
			b.Fatalf("Config validation failed: %v", err)
		}
	}
}

// BenchmarkMemoryUsage 内存使用基准测试
func BenchmarkMemoryUsage(b *testing.B) {
	config := httpdns.DefaultConfig()
	config.AccountID = "test123"
	config.EnableMetrics = true

	client, err := httpdns.NewClient(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 执行大量解析操作来测试内存使用
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		domains := []string{
			"example.com", "google.com", "github.com", "stackoverflow.com", "reddit.com",
			"amazon.com", "microsoft.com", "apple.com", "facebook.com", "twitter.com",
		}

		for _, domain := range domains {
			_, err := client.Resolve(ctx, domain, "1.2.3.4")
			if err != nil {
				b.Logf("Memory test resolve error: %v", err)
			}
		}
	}
}

// BenchmarkErrorHandling 错误处理性能基准测试
func BenchmarkErrorHandling(b *testing.B) {
	config := httpdns.DefaultConfig()
	config.AccountID = "test123"

	client, err := httpdns.NewClient(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 尝试解析一个不存在的域名来触发错误处理
		_, err := client.Resolve(ctx, "this-domain-does-not-exist-for-sure.invalid", "1.2.3.4")
		if err == nil {
			b.Log("Expected error but got none")
		}
	}
}

// 性能测试报告函数
func TestPerformanceReport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance report in short mode")
	}

	config := httpdns.DefaultConfig()
	config.AccountID = "test123"
	config.EnableMetrics = true

	client, err := httpdns.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 执行性能测试
	t.Log("=== Performance Report ===")

	// 单次解析延迟测试
	start := time.Now()
	_, err = client.Resolve(ctx, "example.com", "1.2.3.4")
	singleLatency := time.Since(start)
	if err != nil {
		t.Logf("Single resolve error: %v", err)
	} else {
		t.Logf("Single resolve latency: %v", singleLatency)
	}

	// 批量解析延迟测试
	domains := []string{"google.com", "github.com", "stackoverflow.com", "reddit.com", "amazon.com"}
	start = time.Now()
	results, err := client.ResolveBatch(ctx, domains, "1.2.3.4")
	batchLatency := time.Since(start)
	if err != nil {
		t.Logf("Batch resolve error: %v", err)
	} else {
		t.Logf("Batch resolve latency (%d domains): %v", len(domains), batchLatency)
		t.Logf("Average per domain: %v", batchLatency/time.Duration(len(domains)))
		successCount := 0
		for _, result := range results {
			if result.Error == nil {
				successCount++
			}
		}
		t.Logf("Batch success rate: %d/%d (%.1f%%)", successCount, len(results), float64(successCount)/float64(len(results))*100)
	}

	// 并发性能测试
	concurrency := 50
	iterations := 10
	start = time.Now()

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				client.Resolve(ctx, "example.com", "1.2.3.4")
			}
		}()
	}
	wg.Wait()

	concurrentLatency := time.Since(start)
	totalRequests := concurrency * iterations
	t.Logf("Concurrent test (%d goroutines × %d requests): %v", concurrency, iterations, concurrentLatency)
	t.Logf("Average per request: %v", concurrentLatency/time.Duration(totalRequests))
	t.Logf("Requests per second: %.1f", float64(totalRequests)/concurrentLatency.Seconds())

	// 指标统计
	stats := client.GetMetrics()
	t.Logf("Final metrics:")
	t.Logf("  Total resolves: %d", stats.TotalResolves)
	t.Logf("  Success rate: %.2f%%", stats.SuccessRate*100)
	t.Logf("  Average latency: %v", stats.AvgLatency)
	t.Logf("  Min latency: %v", stats.MinLatency)
	t.Logf("  Max latency: %v", stats.MaxLatency)
	t.Logf("  API requests: %d", stats.APIRequests)
	t.Logf("  API errors: %d", stats.APIErrors)
}
