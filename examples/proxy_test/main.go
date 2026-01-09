package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

func main() {
	fmt.Println("=== Clash Verge 代理测试 ===\n")

	// 测试 1: HTTP 代理
	testHTTPProxy()
	fmt.Println()

	// 测试 2: SOCKS5 代理
	testSOCKS5Proxy()
	fmt.Println()

	// 测试 3: 不使用代理（对比）
	testDirect()
}

// testHTTPProxy 测试 HTTP 代理
func testHTTPProxy() {
	fmt.Println("=== 测试 1: HTTP 代理 ===")

	// 配置 HTTP 代理
	proxyURL, err := url.Parse("http://127.0.0.1:7897")
	if err != nil {
		fmt.Printf("❌ 解析代理 URL 失败: %v\n", err)
		return
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 10 * time.Second,
	}

	// 测试请求（改用国内可访问的网站）
	testURL := "https://www.aliyun.com/solution"
	fmt.Printf("请求 URL: %s\n", testURL)
	fmt.Printf("代理地址: %s\n", proxyURL.String())

	startTime := time.Now()
	resp, err := client.Get(testURL)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	elapsed := time.Since(startTime)

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("✅ 请求成功\n")
	fmt.Printf("  状态码: %d\n", resp.StatusCode)
	fmt.Printf("  响应大小: %d bytes\n", len(body))
	fmt.Printf("  耗时: %v\n", elapsed)
}

// testSOCKS5Proxy 测试 SOCKS5 代理
func testSOCKS5Proxy() {
	fmt.Println("=== 测试 2: SOCKS5 代理 ===")

	// 创建 SOCKS5 代理 Dialer
	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:7897", nil, proxy.Direct)
	if err != nil {
		fmt.Printf("❌ 创建 SOCKS5 代理失败: %v\n", err)
		return
	}

	client := &http.Client{
		Transport: &http.Transport{
			Dial: dialer.Dial,
		},
		Timeout: 10 * time.Second,
	}

	// 测试请求（改用国内可访问的网站）
	testURL := "https://www.aliyun.com/product/dns"
	fmt.Printf("请求 URL: %s\n", testURL)
	fmt.Printf("代理地址: socks5://127.0.0.1:7897\n")

	startTime := time.Now()
	resp, err := client.Get(testURL)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	elapsed := time.Since(startTime)

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("✅ 请求成功\n")
	fmt.Printf("  状态码: %d\n", resp.StatusCode)
	fmt.Printf("  响应大小: %d bytes\n", len(body))
	fmt.Printf("  耗时: %v\n", elapsed)
}

// testDirect 测试直连（不使用代理）
func testDirect() {
	fmt.Println("=== 测试 3: 直连（不使用代理）===")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 测试国内网站（不需要代理）
	testURL := "https://www.aliyun.com/"
	fmt.Printf("请求 URL: %s\n", testURL)

	startTime := time.Now()
	resp, err := client.Get(testURL)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	elapsed := time.Since(startTime)

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("✅ 请求成功\n")
	fmt.Printf("  状态码: %d\n", resp.StatusCode)
	fmt.Printf("  响应大小: %d bytes\n", len(body))
	fmt.Printf("  耗时: %v\n", elapsed)
}
