package main

import (
	"encoding/json"
	"fmt"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

func main() {
	fmt.Println("=== 批量解析接口修复验证 ===")
	fmt.Println()

	// 测试用例1：IPv4 和 IPv6 混合
	fmt.Println("1. 测试 IPv4 和 IPv6 混合:")
	testNewFormat()

	// 测试用例2：仅 IPv4
	fmt.Println("\n2. 测试仅 IPv4:")
	testLegacyFormat()

	// 测试用例3：多域名
	fmt.Println("\n3. 测试多域名:")
	testMixedFormat()

	fmt.Println("\n=== 验证完成 ===")
}

func testNewFormat() {
	// 模拟响应：使用 IPs 和 IPsV6 字段
	response := httpdns.BatchResolveResponse{
		DNS: []httpdns.HTTPDNSResponse{
			{
				Host:  "example.com",
				IPs:   []string{"1.2.3.4", "5.6.7.8"},
				IPsV6: []string{"2001:db8::1", "2001:db8::2"},
				TTL:   300,
			},
		},
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	fmt.Printf("响应格式:\n%s\n", data)

	// 验证解析逻辑
	ipv4Count, ipv6Count := countIPs(response)
	fmt.Printf("解析结果: IPv4=%d, IPv6=%d\n", ipv4Count, ipv6Count)

	if ipv4Count == 2 && ipv6Count == 2 {
		fmt.Println("✅ 新格式解析正确")
	} else {
		fmt.Println("❌ 新格式解析错误")
	}
}

func testLegacyFormat() {
	// 模拟响应：仅 IPv4
	response := httpdns.BatchResolveResponse{
		DNS: []httpdns.HTTPDNSResponse{
			{
				Host: "example.com",
				IPs:  []string{"1.2.3.4"},
				TTL:  300,
			},
		},
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	fmt.Printf("响应格式:\n%s\n", data)

	// 验证解析逻辑
	ipv4Count, ipv6Count := countIPs(response)
	fmt.Printf("解析结果: IPv4=%d, IPv6=%d\n", ipv4Count, ipv6Count)

	if ipv4Count == 1 && ipv6Count == 0 {
		fmt.Println("✅ 仅IPv4解析正确")
	} else {
		fmt.Println("❌ 仅IPv4解析错误")
	}
}

func testMixedFormat() {
	// 模拟多域名响应
	response := httpdns.BatchResolveResponse{
		DNS: []httpdns.HTTPDNSResponse{
			{
				Host: "domain1.example.com",
				IPs:  []string{"1.2.3.4"},
				TTL:  300,
			},
			{
				Host:  "domain2.example.com",
				IPs:   []string{"5.6.7.8"},
				IPsV6: []string{"2001:db8::1"},
				TTL:   300,
			},
		},
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	fmt.Printf("响应格式:\n%s\n", data)

	// 验证解析逻辑
	ipv4Count, ipv6Count := countIPs(response)
	fmt.Printf("解析结果: IPv4=%d, IPv6=%d\n", ipv4Count, ipv6Count)

	if ipv4Count == 2 && ipv6Count == 1 {
		fmt.Println("✅ 混合格式解析正确")
	} else {
		fmt.Println("❌ 混合格式解析错误")
	}
}

// countIPs 统计IP数量：IPs 为 IPv4，IPsV6 为 IPv6
func countIPs(response httpdns.BatchResolveResponse) (ipv4Count, ipv6Count int) {
	for _, dnsResp := range response.DNS {
		ipv4Count += len(dnsResp.IPs)
		ipv6Count += len(dnsResp.IPsV6)
	}
	return
}
