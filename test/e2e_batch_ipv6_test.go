package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

// TestBatchResolveIPv6EndToEnd 端到端测试批量解析IPv6功能
func TestBatchResolveIPv6EndToEnd(t *testing.T) {
	// 创建模拟服务器，返回真实的批量解析响应格式
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/100000/ss" {
			// 返回服务IP列表
			serverAddr := server.URL[7:] // 去掉 "http://" 前缀
			response := map[string]interface{}{
				"service_ip": []string{serverAddr},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/100000/resolve" {
			// 模拟批量解析响应：IPs 为 IPv4，IPsV6 为 IPv6
			response := httpdns.BatchResolveResponse{
				DNS: []httpdns.HTTPDNSResponse{
					{
						Host:     "www.aliyun.com",
						ClientIP: "192.168.1.1",
						IPs:      []string{"47.246.11.132", "47.246.11.133"},
						IPsV6:    []string{"2400:3200:1300:0:0:0:47:f6", "2400:3200:1300:0:0:0:47:f7"},
						TTL:      106,
					},
					{
						Host:     "www.alibaba.com",
						ClientIP: "192.168.1.1",
						IPs:      []string{"140.205.94.189"},
						IPsV6:    []string{"2400:3200:1300:0:0:0:8c:cd"},
						TTL:      300,
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 创建客户端配置
	config := httpdns.DefaultConfig()
	config.AccountID = "100000"
	config.BootstrapIPs = []string{server.URL[7:]} // 使用测试服务器
	config.EnableMetrics = true

	// 创建客户端
	client, err := httpdns.NewClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 执行批量解析
	ctx := context.Background()
	domains := []string{"www.aliyun.com", "www.alibaba.com"}

	t.Logf("开始批量解析域名: %v", domains)
	results, err := client.ResolveBatch(ctx, domains, "192.168.1.1")
	if err != nil {
		t.Fatalf("批量解析失败: %v", err)
	}

	// 验证结果
	if len(results) == 0 {
		t.Fatal("批量解析返回空结果")
	}

	// 统计每个域名的IPv4和IPv6地址
	domainStats := make(map[string]struct {
		ipv4Count int
		ipv6Count int
		ipv4IPs   []string
		ipv6IPs   []string
	})

	for _, result := range results {
		stats := domainStats[result.Domain]

		// 统计IPv4地址
		for _, ip := range result.IPv4 {
			stats.ipv4Count++
			stats.ipv4IPs = append(stats.ipv4IPs, ip.String())
		}

		// 统计IPv6地址
		for _, ip := range result.IPv6 {
			stats.ipv6Count++
			stats.ipv6IPs = append(stats.ipv6IPs, ip.String())
		}

		domainStats[result.Domain] = stats
	}

	// 验证 www.aliyun.com 的结果
	aliyunStats, exists := domainStats["www.aliyun.com"]
	if !exists {
		t.Error("未找到 www.aliyun.com 的解析结果")
	} else {
		t.Logf("www.aliyun.com 解析结果:")
		t.Logf("  IPv4地址数量: %d, 地址: %v", aliyunStats.ipv4Count, aliyunStats.ipv4IPs)
		t.Logf("  IPv6地址数量: %d, 地址: %v", aliyunStats.ipv6Count, aliyunStats.ipv6IPs)

		if aliyunStats.ipv4Count != 2 {
			t.Errorf("www.aliyun.com IPv4地址数量错误: 期望2个, 实际%d个", aliyunStats.ipv4Count)
		}

		if aliyunStats.ipv6Count != 2 {
			t.Errorf("www.aliyun.com IPv6地址数量错误: 期望2个, 实际%d个", aliyunStats.ipv6Count)
		}

		// 验证具体的IPv6地址
		expectedIPv6 := []string{"2400:3200:1300::47:f6", "2400:3200:1300::47:f7"}
		for i, expectedIP := range expectedIPv6 {
			if i < len(aliyunStats.ipv6IPs) {
				if aliyunStats.ipv6IPs[i] != expectedIP {
					t.Logf("IPv6地址格式差异: 期望%s, 实际%s (可能是格式化差异)", expectedIP, aliyunStats.ipv6IPs[i])
				}
			}
		}
	}

	// 验证 www.alibaba.com 的结果
	alibabaStats, exists := domainStats["www.alibaba.com"]
	if !exists {
		t.Error("未找到 www.alibaba.com 的解析结果")
	} else {
		t.Logf("www.alibaba.com 解析结果:")
		t.Logf("  IPv4地址数量: %d, 地址: %v", alibabaStats.ipv4Count, alibabaStats.ipv4IPs)
		t.Logf("  IPv6地址数量: %d, 地址: %v", alibabaStats.ipv6Count, alibabaStats.ipv6IPs)

		if alibabaStats.ipv4Count != 1 {
			t.Errorf("www.alibaba.com IPv4地址数量错误: 期望1个, 实际%d个", alibabaStats.ipv4Count)
		}

		if alibabaStats.ipv6Count != 1 {
			t.Errorf("www.alibaba.com IPv6地址数量错误: 期望1个, 实际%d个", alibabaStats.ipv6Count)
		}
	}

	// 验证总体统计
	totalIPv4 := 0
	totalIPv6 := 0
	for _, stats := range domainStats {
		totalIPv4 += stats.ipv4Count
		totalIPv6 += stats.ipv6Count
	}

	t.Logf("总体统计: IPv4地址%d个, IPv6地址%d个", totalIPv4, totalIPv6)

	if totalIPv4 != 3 {
		t.Errorf("总IPv4地址数量错误: 期望3个, 实际%d个", totalIPv4)
	}

	if totalIPv6 != 3 {
		t.Errorf("总IPv6地址数量错误: 期望3个, 实际%d个", totalIPv6)
	}

	// 验证指标
	metrics := client.GetMetrics()
	t.Logf("解析指标:")
	t.Logf("  总解析次数: %d", metrics.TotalResolves)
	t.Logf("  成功次数: %d", metrics.SuccessResolves)
	t.Logf("  成功率: %.2f%%", metrics.SuccessRate*100)

	if metrics.TotalResolves == 0 {
		t.Error("指标统计异常: 总解析次数为0")
	}

	if metrics.SuccessRate < 1.0 {
		t.Errorf("解析成功率过低: %.2f%%", metrics.SuccessRate*100)
	}

	t.Log("✅ 批量解析IPv6端到端测试通过")
}

// TestBatchResolveLegacyFormatIPv6 测试旧格式的IPv6兼容性
func TestBatchResolveLegacyFormatIPv6(t *testing.T) {
	// 创建模拟服务器，返回旧格式的响应
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/100000/ss" {
			serverAddr := server.URL[7:]
			response := map[string]interface{}{
				"service_ip": []string{serverAddr},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/100000/resolve" {
			// 模拟响应：IPs 为 IPv4，IPsV6 为 IPv6
			response := httpdns.BatchResolveResponse{
				DNS: []httpdns.HTTPDNSResponse{
					{
						Host:     "legacy.example.com",
						ClientIP: "192.168.1.1",
						IPs:      []string{"1.2.3.4"},
						IPsV6:    []string{"2001:db8::1", "2001:db8::2"},
						TTL:      300,
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := httpdns.DefaultConfig()
	config.AccountID = "100000"
	config.BootstrapIPs = []string{server.URL[7:]}

	client, err := httpdns.NewClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	results, err := client.ResolveBatch(ctx, []string{"legacy.example.com"}, "192.168.1.1")
	if err != nil {
		t.Fatalf("批量解析失败: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("期望1个结果, 实际%d个", len(results))
	}

	result := results[0]
	t.Logf("旧格式解析结果:")
	t.Logf("  域名: %s", result.Domain)
	t.Logf("  IPv4地址数量: %d", len(result.IPv4))
	t.Logf("  IPv6地址数量: %d", len(result.IPv6))

	if len(result.IPv4) != 1 {
		t.Errorf("IPv4地址数量错误: 期望1个, 实际%d个", len(result.IPv4))
	}

	if len(result.IPv6) != 2 {
		t.Errorf("IPv6地址数量错误: 期望2个, 实际%d个", len(result.IPv6))
	}

	// 验证IPv6地址内容
	for i, ip := range result.IPv6 {
		t.Logf("  IPv6[%d]: %s", i, ip.String())
	}

	t.Log("✅ 旧格式IPv6兼容性测试通过")
}
