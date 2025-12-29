package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

func main() {
	// 从环境变量获取配置
	accountID := os.Getenv("HTTPDNS_ACCOUNT_ID")
	secretKey := os.Getenv("HTTPDNS_SECRET_KEY")

	if accountID == "" {
		log.Fatal("请设置环境变量 HTTPDNS_ACCOUNT_ID")
	}

	// 创建配置
	config := httpdns.DefaultConfig()
	config.AccountID = accountID
	config.SecretKey = secretKey
	config.EnableMetrics = true

	// 创建客户端
	client, err := httpdns.NewClient(config)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 批量解析域名
	domains := []string{"www.aliyun.com", "www.alibaba.com"}
	ctx := context.Background()

	fmt.Printf("正在批量解析域名: %v\n", domains)
	results, err := client.ResolveBatch(ctx, domains, "")
	if err != nil {
		log.Fatalf("批量解析失败: %v", err)
	}

	// 显示结果
	fmt.Printf("\n解析结果:\n")
	for _, result := range results {
		fmt.Printf("域名: %s\n", result.Domain)
		fmt.Printf("  来源: %s\n", result.Source)
		fmt.Printf("  TTL: %v\n", result.TTL)

		if len(result.IPv4) > 0 {
			fmt.Printf("  IPv4地址:\n")
			for _, ip := range result.IPv4 {
				fmt.Printf("    %s\n", ip.String())
			}
		}

		if len(result.IPv6) > 0 {
			fmt.Printf("  IPv6地址:\n")
			for _, ip := range result.IPv6 {
				fmt.Printf("    %s\n", ip.String())
			}
		}

		if result.Error != nil {
			fmt.Printf("  错误: %v\n", result.Error)
		}

		fmt.Println()
	}

	// 显示指标
	metrics := client.GetMetrics()
	fmt.Printf("指标统计:\n")
	fmt.Printf("  总解析次数: %d\n", metrics.TotalResolves)
	fmt.Printf("  成功次数: %d\n", metrics.SuccessResolves)
	fmt.Printf("  失败次数: %d\n", metrics.FailedResolves)
	fmt.Printf("  成功率: %.2f%%\n", metrics.SuccessRate*100)
	fmt.Printf("  平均延迟: %v\n", metrics.AvgLatency)
}
