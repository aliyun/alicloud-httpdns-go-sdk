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
	fmt.Println("🔍 调试IPv6解析问题")

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Printf("收到请求: %s\n", r.URL.Path)

		if r.URL.Path == "/test/ss" {
			response := map[string]interface{}{
				"service_ip": []string{server.URL[7:]},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/test/resolve" {
			response := httpdns.BatchResolveResponse{
				DNS: []httpdns.HTTPDNSResponse{
					{
						Host:  "test.com",
						IPs:   []string{"1.2.3.4"},
						IPsV6: []string{"2001:db8::1"},
						TTL:   300,
					},
				},
			}

			// 打印响应内容
			responseBytes, _ := json.MarshalIndent(response, "", "  ")
			fmt.Printf("返回响应:\n%s\n", responseBytes)

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
	results, err := client.ResolveBatch(ctx, []string{"test.com"}, "")
	if err != nil {
		fmt.Printf("❌ 批量解析失败: %v\n", err)
		return
	}

	fmt.Printf("\n解析结果:\n")
	for i, result := range results {
		fmt.Printf("结果[%d]:\n", i)
		fmt.Printf("  域名: %s\n", result.Domain)
		fmt.Printf("  IPv4数量: %d\n", len(result.IPv4))
		for j, ip := range result.IPv4 {
			fmt.Printf("    IPv4[%d]: %s\n", j, ip.String())
		}
		fmt.Printf("  IPv6数量: %d\n", len(result.IPv6))
		for j, ip := range result.IPv6 {
			fmt.Printf("    IPv6[%d]: %s\n", j, ip.String())
		}
	}
}
