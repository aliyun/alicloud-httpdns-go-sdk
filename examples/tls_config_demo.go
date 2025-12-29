package main

import (
	"fmt"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

func main() {
	fmt.Println("=== HTTPS SNI配置验证 ===")

	// 测试默认配置
	fmt.Println("\n1. 默认配置测试:")
	config := httpdns.DefaultConfig()
	fmt.Printf("   默认HTTPS SNI常量: %s\n", httpdns.DefaultHTTPSSNI)
	fmt.Printf("   配置中的HTTPS SNI: %s\n", config.HTTPSSNIHost)

	// 测试配置验证
	fmt.Println("\n2. 配置验证测试:")
	config.AccountID = "test123" // 设置必需的AccountID
	config.HTTPSSNIHost = ""     // 清空SNI配置
	err := config.Validate()
	if err != nil {
		fmt.Printf("   配置验证失败: %v\n", err)
	} else {
		fmt.Printf("   配置验证后的HTTPS SNI: %s\n", config.HTTPSSNIHost)
	}

	// 测试自定义SNI
	fmt.Println("\n3. 自定义SNI测试:")
	customConfig := httpdns.DefaultConfig()
	customConfig.AccountID = "test123" // 设置必需的AccountID
	customConfig.HTTPSSNIHost = "custom.example.com"
	fmt.Printf("   自定义HTTPS SNI: %s\n", customConfig.HTTPSSNIHost)

	// 验证配置不会被覆盖
	err = customConfig.Validate()
	if err != nil {
		fmt.Printf("   自定义配置验证失败: %v\n", err)
	} else {
		fmt.Printf("   验证后的自定义HTTPS SNI: %s\n", customConfig.HTTPSSNIHost)
	}

	fmt.Println("\n✅ HTTPS SNI配置验证完成")
	fmt.Println("\n修复说明:")
	fmt.Println("- 添加了DefaultHTTPSSNI常量，值为'resolver-cns.aliyuncs.com'")
	fmt.Println("- 在Config结构体中添加了HTTPSSNIHost字段")
	fmt.Println("- 修改了newHTTPClient函数，使用config.HTTPSSNIHost作为TLS ServerName")
	fmt.Println("- 更新了相关测试用例以验证新的SNI配置")
	fmt.Println("- 这样可以确保HTTPS请求使用正确的SNI，避免SSL握手失败")
}
