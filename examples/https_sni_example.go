package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

func main() {
	// 创建配置，启用HTTPS
	config := httpdns.DefaultConfig()
	config.AccountID = "your-account-id" // 替换为您的账号ID
	config.EnableHTTPS = true   // 启用HTTPS
	config.Timeout = 10 * time.Second

	// 验证配置中的SNI设置
	fmt.Printf("HTTPS SNI Host: %s\n", config.HTTPSSNIHost)
	fmt.Printf("Default HTTPS SNI: %s\n", httpdns.DefaultHTTPSSNI)

	// 创建客户端
	client, err := httpdns.NewClient(config)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 测试解析可以成功解析的域名
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	domain := "www.aliyun.com"
	fmt.Printf("\n正在使用HTTPS解析域名: %s\n", domain)

	result, err := client.Resolve(ctx, domain)
	if err != nil {
		log.Printf("解析失败: %v", err)
		return
	}

	fmt.Printf("解析成功!\n")
	fmt.Printf("域名: %s\n", result.Domain)
	fmt.Printf("IPv4地址: %v\n", result.IPv4)
	fmt.Printf("IPv6地址: %v\n", result.IPv6)
	fmt.Printf("TTL: %d秒\n", int(result.TTL.Seconds()))
	fmt.Printf("来源: %s\n", result.Source)
}
