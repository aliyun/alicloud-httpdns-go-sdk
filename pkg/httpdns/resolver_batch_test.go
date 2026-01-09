package httpdns

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolver_ResolveBatch_TypeField(t *testing.T) {
	tests := []struct {
		name     string
		response BatchResolveResponse
		expected map[string]struct {
			ipv4Count int
			ipv6Count int
		}
	}{
		{
			name: "IPv4 only",
			response: BatchResolveResponse{
				DNS: []HTTPDNSResponse{
					{
						Host: "example.com",
						IPs:  []string{"1.2.3.4", "5.6.7.8"},
						TTL:  300,
					},
				},
			},
			expected: map[string]struct {
				ipv4Count int
				ipv6Count int
			}{
				"example.com": {ipv4Count: 2, ipv6Count: 0},
			},
		},
		{
			name: "IPv6 only",
			response: BatchResolveResponse{
				DNS: []HTTPDNSResponse{
					{
						Host:  "example.com",
						IPsV6: []string{"2001:db8::1", "2001:db8::2"},
						TTL:   300,
					},
				},
			},
			expected: map[string]struct {
				ipv4Count int
				ipv6Count int
			}{
				"example.com": {ipv4Count: 0, ipv6Count: 2},
			},
		},
		{
			name: "Mixed IPv4 and IPv6",
			response: BatchResolveResponse{
				DNS: []HTTPDNSResponse{
					{
						Host:  "example.com",
						IPs:   []string{"1.2.3.4"},
						IPsV6: []string{"2001:db8::1"},
						TTL:   300,
					},
				},
			},
			expected: map[string]struct {
				ipv4Count int
				ipv6Count int
			}{
				"example.com": {ipv4Count: 1, ipv6Count: 1},
			},
		},
		{
			name: "IPs and IPsV6 fields",
			response: BatchResolveResponse{
				DNS: []HTTPDNSResponse{
					{
						Host:  "example.com",
						IPs:   []string{"1.2.3.4"},
						IPsV6: []string{"2001:db8::1"},
						TTL:   300,
					},
				},
			},
			expected: map[string]struct {
				ipv4Count int
				ipv6Count int
			}{
				"example.com": {ipv4Count: 1, ipv6Count: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试服务器
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/test123/ss" {
					serverAddr := server.URL[7:]
					response := map[string]interface{}{
						"service_ip": []string{serverAddr},
					}
					json.NewEncoder(w).Encode(response)
				} else if r.URL.Path == "/test123/resolve" {
					json.NewEncoder(w).Encode(tt.response)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			config := DefaultConfig()
			config.AccountID = "test123"
			config.BootstrapIPs = []string{server.URL[7:]}

			resolver := NewResolver(config)

			ctx := context.Background()
			domains := []string{"example.com"}
			results, err := resolver.ResolveBatch(ctx, domains, WithClientIP("1.2.3.4"))

			if err != nil {
				t.Fatalf("ResolveBatch() error = %v", err)
			}

			// 统计每个域名的IPv4和IPv6地址数量
			actualCounts := make(map[string]struct {
				ipv4Count int
				ipv6Count int
			})

			for _, result := range results {
				counts := actualCounts[result.Domain]
				counts.ipv4Count += len(result.IPv4)
				counts.ipv6Count += len(result.IPv6)
				actualCounts[result.Domain] = counts
			}

			// 验证结果
			for domain, expected := range tt.expected {
				actual, exists := actualCounts[domain]
				if !exists {
					t.Errorf("Domain %s not found in results", domain)
					continue
				}

				if actual.ipv4Count != expected.ipv4Count {
					t.Errorf("Domain %s: IPv4 count = %d, want %d", domain, actual.ipv4Count, expected.ipv4Count)
				}

				if actual.ipv6Count != expected.ipv6Count {
					t.Errorf("Domain %s: IPv6 count = %d, want %d", domain, actual.ipv6Count, expected.ipv6Count)
				}
			}
		})
	}
}

func TestResolver_ResolveBatch_TypeFieldIntegration(t *testing.T) {
	// 创建测试服务器，模拟真实的批量解析响应
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/test123/ss" {
			serverAddr := server.URL[7:]
			response := map[string]interface{}{
				"service_ip": []string{serverAddr},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/test123/resolve" {
			// 模拟批量解析响应：IPs 为 IPv4，IPsV6 为 IPv6
			response := BatchResolveResponse{
				DNS: []HTTPDNSResponse{
					{
						Host:     "www.aliyun.com",
						ClientIP: "192.168.1.1",
						IPs:      []string{"47.246.11.132", "47.246.11.133"},
						IPsV6:    []string{"2400:3200:1300:0:0:0:47:f6"},
						TTL:      106,
					},
					{
						Host:     "www.taobao.com",
						ClientIP: "192.168.1.1",
						IPs:      []string{"140.205.94.189"},
						TTL:      46,
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := DefaultConfig()
	config.AccountID = "test123"
	config.BootstrapIPs = []string{server.URL[7:]}

	resolver := NewResolver(config)

	ctx := context.Background()
	domains := []string{"www.aliyun.com", "www.taobao.com"}
	results, err := resolver.ResolveBatch(ctx, domains, WithClientIP("192.168.1.1"))

	if err != nil {
		t.Fatalf("ResolveBatch() error = %v", err)
	}

	// 验证结果 - 每个域名应该返回一个合并的结果
	domainResults := make(map[string]*ResolveResult)
	for _, result := range results {
		domainResults[result.Domain] = result
	}

	// 验证 www.aliyun.com 的结果
	aliyunResult, exists := domainResults["www.aliyun.com"]
	if !exists {
		t.Error("Expected result for www.aliyun.com not found")
	} else {
		if len(aliyunResult.IPv4) != 2 {
			t.Errorf("Expected 2 IPv4 addresses for www.aliyun.com, got %d", len(aliyunResult.IPv4))
		}

		if len(aliyunResult.IPv6) != 1 {
			t.Errorf("Expected 1 IPv6 address for www.aliyun.com, got %d", len(aliyunResult.IPv6))
		}
	}

	// 验证 www.taobao.com 的结果
	taobaoResult, exists := domainResults["www.taobao.com"]
	if !exists {
		t.Error("Expected result for www.taobao.com not found")
	} else {
		if len(taobaoResult.IPv4) != 1 {
			t.Errorf("Expected 1 IPv4 address for www.taobao.com, got %d", len(taobaoResult.IPv4))
		}

		if len(taobaoResult.IPv6) != 0 {
			t.Errorf("Expected 0 IPv6 addresses for www.taobao.com, got %d", len(taobaoResult.IPv6))
		}
	}
}

func TestResolver_ResolveBatch_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name     string
		response BatchResolveResponse
		expected map[string]struct {
			ipv4Count int
			ipv6Count int
		}
		description string
	}{
		{
			name: "阿里云官方示例格式",
			response: BatchResolveResponse{
				DNS: []HTTPDNSResponse{
					{
						Host:     "www.aliyun.com",
						ClientIP: "192.168.1.100",
						IPs:      []string{"192.168.1.100"},
						TTL:      106,
					},
					{
						Host:     "www.taobao.com",
						ClientIP: "192.168.1.101",
						IPs:      []string{"192.168.1.101"},
						TTL:      46,
					},
				},
			},
			expected: map[string]struct {
				ipv4Count int
				ipv6Count int
			}{
				"www.aliyun.com": {ipv4Count: 1, ipv6Count: 0},
				"www.taobao.com": {ipv4Count: 1, ipv6Count: 0},
			},
			description: "测试官方文档示例格式的兼容性",
		},
		{
			name: "同一域名的IPv4和IPv6记录",
			response: BatchResolveResponse{
				DNS: []HTTPDNSResponse{
					{
						Host:     "example.com",
						ClientIP: "192.168.1.1",
						IPs:      []string{"1.2.3.4", "5.6.7.8"},
						IPsV6:    []string{"2001:db8::1", "2001:db8::2"},
						TTL:      300,
					},
				},
			},
			expected: map[string]struct {
				ipv4Count int
				ipv6Count int
			}{
				"example.com": {ipv4Count: 2, ipv6Count: 2},
			},
			description: "测试同一域名返回IPv4和IPv6的情况",
		},
		{
			name: "无效IP地址过滤",
			response: BatchResolveResponse{
				DNS: []HTTPDNSResponse{
					{
						Host:     "test.com",
						ClientIP: "192.168.1.1",
						IPs:      []string{"1.2.3.4", "invalid-ip", "5.6.7.8"},
						IPsV6:    []string{"2001:db8::1", "invalid-ipv6", "2001:db8::2"},
						TTL:      300,
					},
				},
			},
			expected: map[string]struct {
				ipv4Count int
				ipv6Count int
			}{
				"test.com": {ipv4Count: 2, ipv6Count: 2},
			},
			description: "测试无效IP地址的过滤功能",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试服务器
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/test123/ss" {
					serverAddr := server.URL[7:]
					response := map[string]interface{}{
						"service_ip": []string{serverAddr},
					}
					json.NewEncoder(w).Encode(response)
				} else if r.URL.Path == "/test123/resolve" {
					json.NewEncoder(w).Encode(tt.response)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			config := DefaultConfig()
			config.AccountID = "test123"
			config.BootstrapIPs = []string{server.URL[7:]}

			resolver := NewResolver(config)

			ctx := context.Background()
			domains := make([]string, 0)
			for domain := range tt.expected {
				domains = append(domains, domain)
			}

			results, err := resolver.ResolveBatch(ctx, domains, WithClientIP("192.168.1.1"))

			if err != nil {
				t.Fatalf("ResolveBatch() error = %v", err)
			}

			// 统计每个域名的IPv4和IPv6地址数量
			actualCounts := make(map[string]struct {
				ipv4Count int
				ipv6Count int
			})

			for _, result := range results {
				counts := actualCounts[result.Domain]
				counts.ipv4Count += len(result.IPv4)
				counts.ipv6Count += len(result.IPv6)
				actualCounts[result.Domain] = counts
			}

			// 验证结果
			for domain, expected := range tt.expected {
				actual, exists := actualCounts[domain]
				if !exists {
					t.Errorf("Domain %s not found in results", domain)
					continue
				}

				if actual.ipv4Count != expected.ipv4Count {
					t.Errorf("Domain %s: IPv4 count = %d, want %d (%s)",
						domain, actual.ipv4Count, expected.ipv4Count, tt.description)
				}

				if actual.ipv6Count != expected.ipv6Count {
					t.Errorf("Domain %s: IPv6 count = %d, want %d (%s)",
						domain, actual.ipv6Count, expected.ipv6Count, tt.description)
				}
			}
		})
	}
}
