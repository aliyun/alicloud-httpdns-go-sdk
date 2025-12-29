package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

func main() {
	// 创建默认配置
	config := httpdns.DefaultConfig()
	config.AccountID = "your-account-id" // 替换为你的 Account ID

	// 可选：启用鉴权解析
	// config.SecretKey = "your-secret-key"

	// 创建客户端
	client, err := httpdns.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 创建上下文
	ctx := context.Background()

	// 解析单个域名
	fmt.Println("=== 单域名解析 ===")
	result, err := client.Resolve(ctx, "example.com", "1.2.3.4")
	if err != nil {
		log.Printf("Resolve failed: %v", err)
	} else {
		fmt.Printf("Domain: %s\n", result.Domain)
		fmt.Printf("Client IP: %s\n", result.ClientIP)
		fmt.Printf("IPv4 addresses: %v\n", result.IPv4)
		fmt.Printf("IPv6 addresses: %v\n", result.IPv6)
		fmt.Printf("TTL: %v\n", result.TTL)
		fmt.Printf("Source: %s\n", result.Source)
		fmt.Printf("Timestamp: %v\n", result.Timestamp)
	}

	fmt.Println()

	// 批量解析域名
	fmt.Println("=== 批量域名解析 ===")
	domains := []string{"google.com", "github.com", "stackoverflow.com"}
	results, err := client.ResolveBatch(ctx, domains, "1.2.3.4")
	if err != nil {
		log.Printf("Batch resolve failed: %v", err)
	} else {
		for i, result := range results {
			fmt.Printf("Result %d:\n", i+1)
			fmt.Printf("  Domain: %s\n", result.Domain)
			fmt.Printf("  IPv4: %v\n", result.IPv4)
			fmt.Printf("  Source: %s\n", result.Source)
			if result.Error != nil {
				fmt.Printf("  Error: %v\n", result.Error)
			}
		}
	}

	fmt.Println()

	// 使用解析选项
	fmt.Println("=== 使用解析选项 ===")

	// 仅解析 IPv4
	result, err = client.Resolve(ctx, "example.com", "", httpdns.WithIPv4Only())
	if err != nil {
		log.Printf("IPv4 only resolve failed: %v", err)
	} else {
		fmt.Printf("IPv4 only result: %v\n", result.IPv4)
	}

	// 仅解析 IPv6
	result, err = client.Resolve(ctx, "example.com", "", httpdns.WithIPv6Only())
	if err != nil {
		log.Printf("IPv6 only resolve failed: %v", err)
	} else {
		fmt.Printf("IPv6 only result: %v\n", result.IPv6)
	}
}
