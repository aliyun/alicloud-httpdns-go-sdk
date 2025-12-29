package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

func main() {
	// 创建高级配置
	config := &httpdns.Config{
		// 认证信息
		AccountID: "your-account-id",
		SecretKey: "your-secret-key", // 启用鉴权解析

		// 网络配置
		BootstrapIPs: []string{
			"203.107.1.1",
			"203.107.1.97",
			"203.107.1.100",
		},
		Timeout:    10 * time.Second,
		MaxRetries: 5,

		// 功能开关
		EnableHTTPS:   false, // 使用 HTTP
		EnableMetrics: true,  // 启用指标收集

		// 日志配置
		Logger: log.New(os.Stdout, "[HTTPDNS] ", log.LstdFlags),
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	// 创建客户端
	client, err := httpdns.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 检查客户端健康状态
	if !client.IsHealthy() {
		log.Fatal("Client is not healthy")
	}

	fmt.Println("=== 高级功能演示 ===")

	// 演示异步解析
	demonstrateAsyncResolve(client)

	// 演示指标监控
	demonstrateMetrics(client)

	// 演示服务 IP 管理
	demonstrateServiceIPManagement(client)

	// 演示错误处理
	demonstrateErrorHandling(client)
}

func demonstrateAsyncResolve(client httpdns.Client) {
	fmt.Println("\n=== 异步解析演示 ===")

	ctx := context.Background()
	done := make(chan bool, 3)

	domains := []string{"google.com", "github.com", "stackoverflow.com"}

	for _, domain := range domains {
		client.ResolveAsync(ctx, domain, "1.2.3.4", func(result *httpdns.ResolveResult, err error) {
			if err != nil {
				log.Printf("Async resolve %s failed: %v", domain, err)
			} else {
				fmt.Printf("Async result: %s -> %v (Source: %s)\n",
					result.Domain, result.IPv4, result.Source)
			}
			done <- true
		})
	}

	// 等待所有异步解析完成
	for i := 0; i < len(domains); i++ {
		<-done
	}
}

func demonstrateMetrics(client httpdns.Client) {
	fmt.Println("\n=== 指标监控演示 ===")

	ctx := context.Background()

	// 执行一些解析操作
	domains := []string{"example.com", "google.com", "invalid-domain-that-does-not-exist.com"}
	for _, domain := range domains {
		_, err := client.Resolve(ctx, domain, "1.2.3.4")
		if err != nil {
			log.Printf("Resolve %s failed: %v", domain, err)
		}
	}

	// 获取指标统计
	stats := client.GetMetrics()
	fmt.Printf("总解析次数: %d\n", stats.TotalResolves)
	fmt.Printf("成功解析次数: %d\n", stats.SuccessResolves)
	fmt.Printf("失败解析次数: %d\n", stats.FailedResolves)
	fmt.Printf("成功率: %.2f%%\n", stats.SuccessRate*100)
	fmt.Printf("平均延迟: %v\n", stats.AvgLatency)
	fmt.Printf("最小延迟: %v\n", stats.MinLatency)
	fmt.Printf("最大延迟: %v\n", stats.MaxLatency)

	fmt.Printf("API 请求次数: %d\n", stats.APIRequests)
	fmt.Printf("API 错误次数: %d\n", stats.APIErrors)
	fmt.Printf("网络错误次数: %d\n", stats.NetworkErrors)
}

func demonstrateServiceIPManagement(client httpdns.Client) {
	fmt.Println("\n=== 服务 IP 管理演示 ===")

	// 获取当前服务 IP 列表
	ips := client.GetServiceIPs()
	fmt.Printf("当前服务 IP 列表: %v\n", ips)

	// 手动更新服务 IP
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := client.UpdateServiceIPs(ctx)
	if err != nil {
		log.Printf("更新服务 IP 失败: %v", err)
	} else {
		fmt.Println("服务 IP 更新成功")

		// 再次获取服务 IP 列表
		newIPs := client.GetServiceIPs()
		fmt.Printf("更新后的服务 IP 列表: %v\n", newIPs)
	}
}

func demonstrateErrorHandling(client httpdns.Client) {
	fmt.Println("\n=== 错误处理演示 ===")

	ctx := context.Background()

	// 尝试解析一个不存在的域名
	_, err := client.Resolve(ctx, "this-domain-definitely-does-not-exist.com", "1.2.3.4")
	if err != nil {
		if httpDNSErr, ok := err.(*httpdns.HTTPDNSError); ok {
			fmt.Printf("HTTPDNS 错误:\n")
			fmt.Printf("  操作: %s\n", httpDNSErr.Op)
			fmt.Printf("  域名: %s\n", httpDNSErr.Domain)
			fmt.Printf("  错误: %v\n", httpDNSErr.Err)
		} else {
			fmt.Printf("其他错误: %v\n", err)
		}
	}

	// 演示超时处理
	shortCtx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err = client.Resolve(shortCtx, "example.com", "1.2.3.4")
	if err != nil {
		fmt.Printf("超时错误: %v\n", err)
	}
}
