package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	fmt.Println("=== 测试 Proxy + DialContext 的实际行为 ===\n")

	// 测试 1: 只配置 Proxy（系统代理）
	fmt.Println("--- 测试 1: 只配置 Proxy（系统代理）---")
	testOnlyProxy()
	fmt.Println()

	// 测试 2: 只配置 DialContext（模拟 HTTPDNS）
	fmt.Println("--- 测试 2: 只配置 DialContext（模拟 HTTPDNS）---")
	testOnlyDialContext()
	fmt.Println()

	// 测试 3: 同时配置 Proxy + DialContext
	fmt.Println("--- 测试 3: 同时配置 Proxy + DialContext ---")
	testProxyAndDialContext()
	fmt.Println()

	// 测试 4: 默认 http.Client（什么都不配置）
	fmt.Println("--- 测试 4: 默认 http.Client ---")
	testDefaultClient()
}

// 测试 1: 只配置 Proxy
func testOnlyProxy() {
	proxyURL, _ := url.Parse("http://127.0.0.1:7897")

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://www.aliyun.com")
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("✓ 请求成功，状态码: %d\n", resp.StatusCode)
	fmt.Println("  说明: 请求通过代理成功")
}

// 测试 2: 只配置 DialContext
func testOnlyDialContext() {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				fmt.Printf("  DialContext 收到地址: %s\n", addr)

				// 模拟 HTTPDNS：解析域名
				host, port, _ := net.SplitHostPort(addr)
				fmt.Printf("  解析的 host: %s, port: %s\n", host, port)

				// 直接连接（不做 DNS 解析，只是打印）
				dialer := &net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}
				return dialer.DialContext(ctx, network, addr)
			},
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://www.aliyun.com")
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("✓ 请求成功，状态码: %d\n", resp.StatusCode)
	fmt.Println("  说明: DialContext 收到的是目标域名")
}

// 测试 3: 同时配置 Proxy + DialContext
func testProxyAndDialContext() {
	proxyURL, _ := url.Parse("http://127.0.0.1:7897")

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				fmt.Printf("  DialContext 收到地址: %s\n", addr)

				host, port, _ := net.SplitHostPort(addr)
				fmt.Printf("  解析的 host: %s, port: %s\n", host, port)

				// 检查是否是 IP 地址
				if net.ParseIP(host) != nil {
					fmt.Println("  ⚠️  收到的是 IP 地址（可能是代理地址）")
				} else {
					fmt.Println("  ✓ 收到的是域名")
				}

				// 直接连接
				dialer := &net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}
				return dialer.DialContext(ctx, network, addr)
			},
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://www.aliyun.com")
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✓ 请求成功，状态码: %d, 响应大小: %d bytes\n", resp.StatusCode, len(body))
	fmt.Println("  说明: 观察 DialContext 收到的是代理地址还是目标域名")
}

// 测试 4: 默认 http.Client
func testDefaultClient() {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 检查是否有系统代理环境变量
	fmt.Printf("  HTTP_PROXY 环境变量: %s\n", getEnv("HTTP_PROXY"))
	fmt.Printf("  HTTPS_PROXY 环境变量: %s\n", getEnv("HTTPS_PROXY"))

	resp, err := client.Get("https://www.aliyun.com")
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("✓ 请求成功，状态码: %d\n", resp.StatusCode)
	if getEnv("HTTP_PROXY") != "" || getEnv("HTTPS_PROXY") != "" {
		fmt.Println("  说明: 默认 Client 会自动使用系统代理")
	} else {
		fmt.Println("  说明: 没有系统代理，直接连接")
	}
}

func getEnv(key string) string {
	// 尝试大小写
	if v := os.Getenv(key); v != "" {
		return v
	}
	// 尝试小写
	return os.Getenv(strings.ToLower(key))
}
