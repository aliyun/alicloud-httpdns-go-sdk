package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

func main() {
	// 创建配置
	config := httpdns.DefaultConfig()
	config.AccountID = "your-account-id"
	config.SecretKey = "your-secret-key"

	// 创建客户端
	client, err := httpdns.NewClient(config)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试1: 正常批量解析（5个域名以内）
	fmt.Println("=== 测试1: 正常批量解析 ===")
	normalDomains := []string{
		"www.aliyun.com",
		"www.alibaba.com",
	}

	results, err := client.ResolveBatch(ctx, normalDomains, "1.2.3.4")
	if err != nil {
		log.Printf("正常批量解析失败: %v", err)
	} else {
		fmt.Printf("成功解析 %d 个域名:\n", len(results))
		for _, result := range results {
			fmt.Printf("  %s: %v\n", result.Domain, result.IPv4)
		}
	}
	fmt.Println()

	// 测试2: 正好5个域名的批量解析
	fmt.Println("=== 测试2: 5个域名的批量解析 ===")
	fiveDomains := []string{
		"www.aliyun.com",
		"www.alibaba.com",
		"ecs.aliyuncs.com",
		"oss.aliyuncs.com",
		"rds.aliyuncs.com",
	}

	results, err = client.ResolveBatch(ctx, fiveDomains, "1.2.3.4")
	if err != nil {
		log.Printf("5个域名批量解析失败: %v", err)
	} else {
		fmt.Printf("成功解析 %d 个域名:\n", len(results))
		for _, result := range results {
			fmt.Printf("  %s: %v\n", result.Domain, result.IPv4)
		}
	}
	fmt.Println()

	// 测试3: 超过5个域名的批量解析（应该失败）
	fmt.Println("=== 测试3: 超过5个域名的批量解析 ===")
	tooManyDomains := []string{
		"domain1.com",
		"domain2.com",
		"domain3.com",
		"domain4.com",
		"domain5.com",
		"domain6.com", // 第6个域名，应该触发错误
	}

	results, err = client.ResolveBatch(ctx, tooManyDomains, "1.2.3.4")
	if err != nil {
		// 检查是否是预期的错误类型
		if httpDNSErr, ok := err.(*httpdns.HTTPDNSError); ok {
			if httpDNSErr.Err == httpdns.ErrTooManyDomains {
				fmt.Printf("✅ 正确检测到域名数量超限: %v\n", err)
			} else {
				fmt.Printf("❌ 意外的错误类型: %v\n", err)
			}
		} else {
			fmt.Printf("❌ 错误类型不匹配: %v\n", err)
		}
	} else {
		fmt.Println("❌ 应该返回错误但没有返回")
	}
	fmt.Println()

	// 测试4: 空域名列表
	fmt.Println("=== 测试4: 空域名列表 ===")
	emptyDomains := []string{}

	results, err = client.ResolveBatch(ctx, emptyDomains, "1.2.3.4")
	if err != nil {
		fmt.Printf("✅ 正确检测到空域名列表: %v\n", err)
	} else {
		fmt.Println("❌ 应该返回错误但没有返回")
	}

	fmt.Println("\n=== 测试完成 ===")
}
