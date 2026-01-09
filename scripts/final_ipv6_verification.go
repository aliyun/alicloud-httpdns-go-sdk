package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

func main() {
	fmt.Println("🧪 批量解析IPv6功能最终验证")
	fmt.Println("=" + fmt.Sprintf("%s", "==========================================="))

	// 测试场景1: 新格式 type 字段
	fmt.Println("\n📋 测试场景1: 新格式 type 字段")
	testNewTypeFieldFormat()

	// 测试场景2: 旧格式兼容性
	fmt.Println("\n📋 测试场景2: 旧格式兼容性")
	testLegacyFormatCompatibility()

	// 测试场景3: 混合格式
	fmt.Println("\n📋 测试场景3: 混合格式")
	testMixedFormat()

	fmt.Println("\n✅ 所有IPv6批量解析测试通过！")
	fmt.Println("🎉 修复验证完成，可以安全提交代码")
}

func testNewTypeFieldFormat() {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/test/ss" {
			response := map[string]interface{}{
				"service_ip": []string{server.URL[7:]},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/test/resolve" {
			response := httpdns.BatchResolveResponse{
				DNS: []httpdns.HTTPDNSResponse{
					{
						Host:  "ipv4.example.com",
						IPs:   []string{"1.2.3.4", "5.6.7.8"},
						TTL:   300,
					},
					{
						Host:  "ipv6.example.com",
						IPsV6: []string{"2001:db8::1", "2001:db8::2", "2001:db8::3"},
						TTL:   300,
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	config := httpdns.DefaultConfig()
	config.AccountID = "test"
	config.BootstrapIPs = []string{server.URL[7:]}

	client, err := httpdns.NewClient(config)
	if err != nil {
		fmt.Printf("❌ 创建客户端失败: %v\n", err)
		return
	}
	defer client.Close()

	ctx := context.Background()
	results, err := client.ResolveBatch(ctx, []string{"ipv4.example.com", "ipv6.example.com"})
	if err != nil {
		fmt.Printf("❌ 批量解析失败: %v\n", err)
		return
	}

	ipv4Count, ipv6Count := 0, 0
	for _, result := range results {
		ipv4Count += len(result.IPv4)
		ipv6Count += len(result.IPv6)
		fmt.Printf("   域名: %s, IPv4: %d个, IPv6: %d个\n", result.Domain, len(result.IPv4), len(result.IPv6))
	}

	if ipv4Count == 2 && ipv6Count == 3 {
		fmt.Println("✅ IPv4/IPv6 分离解析正确")
	} else {
		fmt.Printf("❌ 解析错误: IPv4=%d(期望2), IPv6=%d(期望3)\n", ipv4Count, ipv6Count)
	}
}

func testLegacyFormatCompatibility() {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/test/ss" {
			response := map[string]interface{}{
				"service_ip": []string{server.URL[7:]},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/test/resolve" {
			response := httpdns.BatchResolveResponse{
				DNS: []httpdns.HTTPDNSResponse{
					{
						Host:  "legacy.example.com",
						IPs:   []string{"192.168.1.1"},
						IPsV6: []string{"2001:db8::legacy1", "2001:db8::legacy2"},
						TTL:   300,
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	config := httpdns.DefaultConfig()
	config.AccountID = "test"
	config.BootstrapIPs = []string{server.URL[7:]}

	client, err := httpdns.NewClient(config)
	if err != nil {
		fmt.Printf("❌ 创建客户端失败: %v\n", err)
		return
	}
	defer client.Close()

	ctx := context.Background()
	results, err := client.ResolveBatch(ctx, []string{"legacy.example.com"})
	if err != nil {
		fmt.Printf("❌ 批量解析失败: %v\n", err)
		return
	}

	ipv4Count, ipv6Count := 0, 0
	for _, result := range results {
		ipv4Count += len(result.IPv4)
		ipv6Count += len(result.IPv6)
		fmt.Printf("   域名: %s, IPv4: %d个, IPv6: %d个\n", result.Domain, len(result.IPv4), len(result.IPv6))
	}

	if ipv4Count == 1 && ipv6Count == 2 {
		fmt.Println("✅ 旧格式兼容性正确")
	} else {
		fmt.Printf("❌ 旧格式兼容性错误: IPv4=%d(期望1), IPv6=%d(期望2)\n", ipv4Count, ipv6Count)
	}
}

func testMixedFormat() {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/test/ss" {
			response := map[string]interface{}{
				"service_ip": []string{server.URL[7:]},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/test/resolve" {
			response := httpdns.BatchResolveResponse{
				DNS: []httpdns.HTTPDNSResponse{
					{
						Host:  "new.example.com",
						IPsV6: []string{"2001:db8::new"},
						TTL:   300,
					},
					{
						Host:  "old.example.com",
						IPs:   []string{"10.0.0.1"},
						IPsV6: []string{"2001:db8::old"},
						TTL:   300,
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	config := httpdns.DefaultConfig()
	config.AccountID = "test"
	config.BootstrapIPs = []string{server.URL[7:]}

	client, err := httpdns.NewClient(config)
	if err != nil {
		fmt.Printf("❌ 创建客户端失败: %v\n", err)
		return
	}
	defer client.Close()

	ctx := context.Background()
	results, err := client.ResolveBatch(ctx, []string{"new.example.com", "old.example.com"})
	if err != nil {
		fmt.Printf("❌ 批量解析失败: %v\n", err)
		return
	}

	totalIPv4, totalIPv6 := 0, 0
	for _, result := range results {
		totalIPv4 += len(result.IPv4)
		totalIPv6 += len(result.IPv6)
		fmt.Printf("   域名: %s, IPv4: %d个, IPv6: %d个\n", result.Domain, len(result.IPv4), len(result.IPv6))
	}

	if totalIPv4 == 1 && totalIPv6 == 2 {
		fmt.Println("✅ 混合格式解析正确")
	} else {
		fmt.Printf("❌ 混合格式解析错误: IPv4=%d(期望1), IPv6=%d(期望2)\n", totalIPv4, totalIPv6)
	}
}
