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
	"golang.org/x/net/proxy"
)

func main() {
	// 从环境变量读取配置
	accountID := os.Getenv("HTTPDNS_ACCOUNT_ID")
	secretKey := os.Getenv("HTTPDNS_SECRET_KEY")

	if accountID == "" {
		accountID = "your-account-id"
	}
	if secretKey == "" {
		secretKey = "your-secret-key"
	}

	// 创建 HTTPDNS 客户端
	config := httpdns.DefaultConfig()
	config.AccountID = accountID
	config.SecretKey = secretKey
	config.Timeout = 10 * time.Second

	dnsClient, err := httpdns.NewClient(config)
	if err != nil {
		log.Fatalf("创建 HTTPDNS 客户端失败: %v", err)
	}
	defer dnsClient.Close()

	fmt.Println("=== HTTPDNS + Proxy 同时配置测试 ===\n")

	// 测试 1: HTTPDNS + HTTP 代理
	testHTTPDNSWithHTTPProxy(dnsClient)
	fmt.Println()

	// 测试 2: HTTPDNS + SOCKS5 代理
	testHTTPDNSWithSOCKS5Proxy(dnsClient)
	fmt.Println()

	// 测试 3: 仅 HTTPDNS（对比）
	testHTTPDNSOnly(dnsClient)
	fmt.Println()

	// 测试 4: 仅 HTTP 代理（对比）
	testHTTPProxyOnly()
	fmt.Println()

	// 测试 5: 仅 SOCKS5 代理（对比）
	testSOCKS5ProxyOnly()
}

// testHTTPDNSWithHTTPProxy 测试 HTTPDNS + HTTP 代理同时配置
func testHTTPDNSWithHTTPProxy(dnsClient httpdns.Client) {
	fmt.Println("=== 测试 1: HTTPDNS + HTTP 代理同时配置 ===")

	// 配置 HTTP 代理
	proxyURL, _ := url.Parse("http://127.0.0.1:7897")

	client := &http.Client{
		Transport: &http.Transport{
			// 配置 HTTPDNS
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, _ := net.SplitHostPort(addr)

				fmt.Printf("  [DialContext] 被调用，目标: %s\n", addr)

				// HTTPDNS 解析
				result, err := dnsClient.Resolve(ctx, host)
				if err == nil && len(result.IPv4) > 0 {
					resolvedIP := result.IPv4[0].String()
					addr = net.JoinHostPort(resolvedIP, port)
					fmt.Printf("  [HTTPDNS] 解析 %s -> %s\n", host, resolvedIP)
				}

				return (&net.Dialer{}).DialContext(ctx, network, addr)
			},
			// 同时配置 HTTP 代理
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 10 * time.Second,
	}

	testURL := "https://www.aliyun.com/solution"
	fmt.Printf("请求 URL: %s\n", testURL)
	fmt.Printf("配置: HTTPDNS + HTTP 代理 (%s)\n", proxyURL.String())

	resp, err := client.Get(testURL)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✅ 请求成功，状态码: %d, 响应大小: %d bytes\n", resp.StatusCode, len(body))
}

// testHTTPDNSWithSOCKS5Proxy 测试 HTTPDNS + SOCKS5 代理同时配置
func testHTTPDNSWithSOCKS5Proxy(dnsClient httpdns.Client) {
	fmt.Println("=== 测试 2: HTTPDNS + SOCKS5 代理同时配置 ===")

	// 创建 SOCKS5 代理
	proxyDialer, err := proxy.SOCKS5("tcp", "127.0.0.1:7897", nil, proxy.Direct)
	if err != nil {
		fmt.Printf("❌ 创建 SOCKS5 代理失败: %v\n", err)
		return
	}

	client := &http.Client{
		Transport: &http.Transport{
			// 配置 HTTPDNS
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, _ := net.SplitHostPort(addr)

				fmt.Printf("  [DialContext] 被调用，目标: %s\n", addr)

				// HTTPDNS 解析
				result, err := dnsClient.Resolve(ctx, host)
				if err == nil && len(result.IPv4) > 0 {
					resolvedIP := result.IPv4[0].String()
					addr = net.JoinHostPort(resolvedIP, port)
					fmt.Printf("  [HTTPDNS] 解析 %s -> %s\n", host, resolvedIP)
				}

				return (&net.Dialer{}).DialContext(ctx, network, addr)
			},
			// 同时配置 SOCKS5 代理（通过 Dial）
			Dial: proxyDialer.Dial,
		},
		Timeout: 10 * time.Second,
	}

	testURL := "https://www.aliyun.com/product/dns"
	fmt.Printf("请求 URL: %s\n", testURL)
	fmt.Printf("配置: HTTPDNS + SOCKS5 代理 (127.0.0.1:7897)\n")

	resp, err := client.Get(testURL)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✅ 请求成功，状态码: %d, 响应大小: %d bytes\n", resp.StatusCode, len(body))
}

// testHTTPDNSOnly 测试仅 HTTPDNS
func testHTTPDNSOnly(dnsClient httpdns.Client) {
	fmt.Println("=== 测试 3: 仅 HTTPDNS（对比）===")

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, _ := net.SplitHostPort(addr)

				result, err := dnsClient.Resolve(ctx, host)
				if err == nil && len(result.IPv4) > 0 {
					resolvedIP := result.IPv4[0].String()
					addr = net.JoinHostPort(resolvedIP, port)
					fmt.Printf("  [HTTPDNS] 解析 %s -> %s\n", host, resolvedIP)
				}

				return (&net.Dialer{}).DialContext(ctx, network, addr)
			},
		},
		Timeout: 10 * time.Second,
	}

	testURL := "https://www.aliyun.com/about"
	fmt.Printf("请求 URL: %s\n", testURL)

	resp, err := client.Get(testURL)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✅ 请求成功，状态码: %d, 响应大小: %d bytes\n", resp.StatusCode, len(body))
}

// testHTTPProxyOnly 测试仅 HTTP 代理
func testHTTPProxyOnly() {
	fmt.Println("=== 测试 4: 仅 HTTP 代理（对比）===")

	proxyURL, _ := url.Parse("http://127.0.0.1:7897")

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 10 * time.Second,
	}

	testURL := "https://www.aliyun.com/solution"
	fmt.Printf("请求 URL: %s\n", testURL)

	resp, err := client.Get(testURL)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✅ 请求成功，状态码: %d, 响应大小: %d bytes\n", resp.StatusCode, len(body))
}

// testSOCKS5ProxyOnly 测试仅 SOCKS5 代理
func testSOCKS5ProxyOnly() {
	fmt.Println("=== 测试 5: 仅 SOCKS5 代理（对比）===")

	proxyDialer, err := proxy.SOCKS5("tcp", "127.0.0.1:7897", nil, proxy.Direct)
	if err != nil {
		fmt.Printf("❌ 创建 SOCKS5 代理失败: %v\n", err)
		return
	}

	client := &http.Client{
		Transport: &http.Transport{
			Dial: proxyDialer.Dial,
		},
		Timeout: 10 * time.Second,
	}

	testURL := "https://www.aliyun.com/product/dns"
	fmt.Printf("请求 URL: %s\n", testURL)

	resp, err := client.Get(testURL)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✅ 请求成功，状态码: %d, 响应大小: %d bytes\n", resp.StatusCode, len(body))
}
