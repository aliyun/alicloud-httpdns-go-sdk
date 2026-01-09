package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

func main() {
	// 从环境变量获取配置
	accountID := os.Getenv("HTTPDNS_ACCOUNT_ID")
	secretKey := os.Getenv("HTTPDNS_SECRET_KEY")

	if accountID == "" {
		log.Fatal("HTTPDNS_ACCOUNT_ID environment variable is required")
	}

	// 使用默认配置（不重试）
	config := httpdns.DefaultConfig()
	config.AccountID = accountID
	config.SecretKey = secretKey
	config.EnableMetrics = true

	fmt.Printf("Configuration:\n")
	fmt.Printf("  MaxRetries: %d (no retry by default)\n", config.MaxRetries)
	fmt.Printf("  Timeout: %v\n", config.Timeout)
	fmt.Printf("\n")

	// 创建客户端
	client, err := httpdns.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 测试可解析的域名
	fmt.Println("Testing resolvable domains:")
	resolvableDomains := []string{"www.aliyun.com", "www.alibaba.com"}

	for _, domain := range resolvableDomains {
		start := time.Now()
		result, err := client.Resolve(ctx, domain)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("  ❌ %s: %v (took %v)\n", domain, err, duration)
		} else {
			fmt.Printf("  ✅ %s: IPv4=%v, IPv6=%v (took %v)\n",
				domain, result.IPv4, result.IPv6, duration)
		}
	}

	fmt.Println("\nTesting unresolvable domains:")
	unresolvableDomains := []string{"www.baidu.com", "www.qq.com"}

	for _, domain := range unresolvableDomains {
		start := time.Now()
		result, err := client.Resolve(ctx, domain)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("  ❌ %s: %v (took %v)\n", domain, err, duration)
		} else {
			hasIPs := len(result.IPv4) > 0 || len(result.IPv6) > 0
			if hasIPs {
				fmt.Printf("  ⚠️  %s: unexpectedly resolved IPv4=%v, IPv6=%v (took %v)\n",
					domain, result.IPv4, result.IPv6, duration)
			} else {
				fmt.Printf("  ✅ %s: empty result as expected (took %v)\n", domain, duration)
			}
		}
	}

	// 显示指标
	fmt.Println("\nMetrics:")
	stats := client.GetMetrics()
	fmt.Printf("  Total resolves: %d\n", stats.TotalResolves)
	fmt.Printf("  Success rate: %.1f%%\n", stats.SuccessRate*100)
	fmt.Printf("  Average latency: %v\n", stats.AvgLatency)

	// 演示自定义重试配置
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Demonstrating custom retry configuration:")

	customConfig := httpdns.DefaultConfig()
	customConfig.AccountID = accountID
	customConfig.SecretKey = secretKey
	customConfig.MaxRetries = 2 // 自定义重试2次
	customConfig.Timeout = 3 * time.Second

	fmt.Printf("Custom config MaxRetries: %d\n", customConfig.MaxRetries)

	customClient, err := httpdns.NewClient(customConfig)
	if err != nil {
		log.Fatalf("Failed to create custom client: %v", err)
	}
	defer customClient.Close()

	// 测试一个域名看重试行为
	start := time.Now()
	result, err := customClient.Resolve(ctx, "www.aliyun.com")
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("Custom client resolve failed: %v (took %v)\n", err, duration)
	} else {
		fmt.Printf("Custom client resolve succeeded: IPv4=%v (took %v)\n",
			result.IPv4, duration)
	}
}
