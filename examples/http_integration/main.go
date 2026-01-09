package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

func main() {
	// 从环境变量读取配置
	accountID := os.Getenv("HTTPDNS_ACCOUNT_ID")
	secretKey := os.Getenv("HTTPDNS_SECRET_KEY")
	
	if accountID == "" {
		accountID = "your-account-id" // 默认占位符
	}
	if secretKey == "" {
		secretKey = "your-secret-key" // 默认占位符
	}

	// 创建 HTTPDNS 客户端
	config := httpdns.DefaultConfig()
	config.AccountID = accountID
	config.SecretKey = secretKey
	config.EnableHTTPS = true  // 使用 HTTPS 测试
	config.Timeout = 10 * time.Second

	dnsClient, err := httpdns.NewClient(config)
	if err != nil {
		log.Fatalf("创建 HTTPDNS 客户端失败: %v", err)
	}
	defer dnsClient.Close()

	fmt.Println("=== HTTPDNS HTTP/HTTPS 集成示例 ===\n")

	// 测试多个不同的 HTTPS 域名，验证 SNI 和证书校验
	testDomains := []string{
		"www.aliyun.com",
		"www.taobao.com",
		"www.tmall.com",
		"www.1688.com",
	}

	for _, domain := range testDomains {
		testHTTPSRequest(dnsClient, domain)
		fmt.Println()
	}

	// 测试 HTTP 请求
	testHTTPRequest(dnsClient, "www.aliyun.com")
	fmt.Println()

	// 测试 IPv6
	testIPv6Request(dnsClient, "www.aliyun.com")
	fmt.Println()

	// 测试使用 net/url 构建复杂 URL
	testComplexURL(dnsClient, "www.aliyun.com")
}

// testHTTPSRequest 演示如何使用 HTTPDNS 进行 HTTPS 请求
func testHTTPSRequest(dnsClient httpdns.Client, domain string) {
	fmt.Printf("=== HTTPS 请求测试: %s ===\n", domain)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. 使用 HTTPDNS 解析域名
	result, err := dnsClient.Resolve(ctx, domain)
	if err != nil {
		log.Printf("域名解析失败: %v", err)
		return
	}

	if len(result.IPv4) == 0 {
		log.Printf("未获取到 IPv4 地址")
		return
	}

	resolvedIP := result.IPv4[0].String()
	fmt.Printf("✓ 域名解析成功: %s -> %s\n", domain, resolvedIP)

	// 2. 创建自定义 HTTP 客户端，使用解析的 IP
	client := createHTTPSClient(domain, resolvedIP)

	// 3. 发起 HTTPS 请求
	url := fmt.Sprintf("https://%s/solution", domain)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("创建请求失败: %v", err)
		return
	}

	// 4. 发送请求
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	elapsed := time.Since(startTime)

	// 5. 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应失败: %v", err)
		return
	}

	fmt.Printf("✓ HTTPS 请求成功\n")
	fmt.Printf("  状态码: %d\n", resp.StatusCode)
	fmt.Printf("  响应大小: %d bytes\n", len(body))
	fmt.Printf("  耗时: %v\n", elapsed)
	fmt.Printf("  使用 IP: %s\n", resolvedIP)
}

// testHTTPRequest 演示如何使用 HTTPDNS 进行 HTTP 请求
func testHTTPRequest(dnsClient httpdns.Client, domain string) {
	fmt.Printf("=== HTTP 请求测试: %s ===\n", domain)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. 使用 HTTPDNS 解析域名
	result, err := dnsClient.Resolve(ctx, domain)
	if err != nil {
		log.Printf("域名解析失败: %v", err)
		return
	}

	if len(result.IPv4) == 0 {
		log.Printf("未获取到 IPv4 地址")
		return
	}

	resolvedIP := result.IPv4[0].String()
	fmt.Printf("✓ 域名解析成功: %s -> %s\n", domain, resolvedIP)

	// 2. 创建自定义 HTTP 客户端
	client := createHTTPClient(resolvedIP)

	// 3. 发起 HTTP 请求（注意：使用 IP 构建 URL，但 Host 头会是域名）
	url := fmt.Sprintf("http://%s/about", resolvedIP)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("创建请求失败: %v", err)
		return
	}

	// 关键：设置 Host 头为原始域名
	req.Host = domain

	// 4. 发送请求
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	elapsed := time.Since(startTime)

	// 5. 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应失败: %v", err)
		return
	}

	fmt.Printf("✓ HTTP 请求成功\n")
	fmt.Printf("  状态码: %d\n", resp.StatusCode)
	fmt.Printf("  响应大小: %d bytes\n", len(body))
	fmt.Printf("  耗时: %v\n", elapsed)
	fmt.Printf("  使用 IP: %s\n", resolvedIP)
}

// testIPv6Request 演示如何使用 IPv6 地址进行请求
func testIPv6Request(dnsClient httpdns.Client, domain string) {
	fmt.Printf("=== IPv6 请求测试: %s ===\n", domain)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. 使用 HTTPDNS 解析域名（同时获取 IPv6）
	result, err := dnsClient.Resolve(ctx, domain)
	if err != nil {
		log.Printf("域名解析失败: %v", err)
		return
	}

	if len(result.IPv6) == 0 {
		log.Printf("未获取到 IPv6 地址")
		return
	}

	resolvedIPv6 := result.IPv6[0].String()
	fmt.Printf("✓ 域名解析成功: %s -> %s\n", domain, resolvedIPv6)

	// 2. 使用同一个 createHTTPSClient 方法（自动处理 IPv6）
	client := createHTTPSClient(domain, resolvedIPv6)

	// 3. 发起 HTTPS 请求
	url := fmt.Sprintf("https://%s/product/dns", domain)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("创建请求失败: %v", err)
		return
	}

	// 4. 发送请求
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	elapsed := time.Since(startTime)

	// 5. 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应失败: %v", err)
		return
	}

	fmt.Printf("✓ IPv6 HTTPS 请求成功\n")
	fmt.Printf("  状态码: %d\n", resp.StatusCode)
	fmt.Printf("  响应大小: %d bytes\n", len(body))
	fmt.Printf("  耗时: %v\n", elapsed)
	fmt.Printf("  使用 IPv6: %s\n", resolvedIPv6)
}

// createHTTPSClient 创建配置了 IP 直连的 HTTPS 客户端（支持 IPv4 和 IPv6）
func createHTTPSClient(domain, ip string) *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// 解析原始地址中的端口
				_, port, err := net.SplitHostPort(addr)
				if err != nil {
					port = "443" // HTTPS 默认端口
				}

				// 使用解析的 IP 替换域名（自动处理 IPv4 和 IPv6）
				targetAddr := net.JoinHostPort(ip, port)
				
				dialer := &net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}
				
				return dialer.DialContext(ctx, network, targetAddr)
			},
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// createHTTPClient 创建配置了 IP 直连的 HTTP 客户端
func createHTTPClient(ip string) *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// 对于 HTTP，直接使用 IP
				dialer := &net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}
				
				return dialer.DialContext(ctx, network, addr)
			},
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// testComplexURL 演示如何使用 net/url 构建复杂 URL
func testComplexURL(dnsClient httpdns.Client, domain string) {
	fmt.Printf("=== 使用 net/url 构建复杂 URL 测试: %s ===\n", domain)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. 使用 HTTPDNS 解析域名
	result, err := dnsClient.Resolve(ctx, domain)
	if err != nil {
		log.Printf("域名解析失败: %v", err)
		return
	}

	if len(result.IPv4) == 0 {
		log.Printf("未获取到 IPv4 地址")
		return
	}

	resolvedIP := result.IPv4[0].String()
	fmt.Printf("✓ 域名解析成功: %s -> %s\n", domain, resolvedIP)

	// 2. 使用 net/url 构建 URL
	u := &url.URL{
		Scheme: "https",
		Host:   domain,
		Path:   "/product/emas",
		RawQuery: url.Values{
			"category": []string{"httpdns"},
			"version":  []string{"2.0"},
		}.Encode(),
	}

	fmt.Printf("✓ 构建的 URL: %s\n", u.String())

	// 3. 创建自定义 HTTP 客户端
	client := createHTTPSClient(domain, resolvedIP)

	// 4. 发起 HTTPS 请求
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		log.Printf("创建请求失败: %v", err)
		return
	}

	// 5. 发送请求
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	elapsed := time.Since(startTime)

	// 6. 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应失败: %v", err)
		return
	}

	fmt.Printf("✓ 复杂 URL 请求成功\n")
	fmt.Printf("  状态码: %d\n", resp.StatusCode)
	fmt.Printf("  响应大小: %d bytes\n", len(body))
	fmt.Printf("  耗时: %v\n", elapsed)
	fmt.Printf("  使用 IP: %s\n", resolvedIP)
}
