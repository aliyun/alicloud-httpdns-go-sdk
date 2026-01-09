# HTTPDNS Go SDK

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/aliyun/alicloud-httpdns-go-sdk)](https://goreportcard.com/report/github.com/aliyun/alicloud-httpdns-go-sdk)

HTTPDNS Go SDK 是一个轻量级的 DNS 解析库，通过 HTTP/HTTPS 协议提供域名解析服务。支持阿里云 EMAS HTTPDNS 服务，为传统 DNS 解析提供更好的性能、安全性和可靠性。

## 特性

- ✅ **简洁设计**: 优先使用 Go 标准库，避免过度抽象
- ✅ **高可用性**: 启动 IP 冗余、服务 IP 轮转、故障转移
- ✅ **智能调度**: 支持客户端 IP 传递，实现就近接入
- ✅ **安全认证**: 支持鉴权解析，MD5 签名算法
- ✅ **智能缓存**: 内存缓存、持久化缓存、过期缓存使用
- ✅ **性能优化**: 异步解析、连接复用、请求去重
- ✅ **监控指标**: 解析延迟、成功率、错误分类统计
- ✅ **IPv6 支持**: 完整的 IPv4/IPv6 双栈支持

## 安装

```bash
go get github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns
```

## 快速开始

### 基础使用

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

func main() {
    // 创建配置
    config := httpdns.DefaultConfig()
    config.AccountID = "your-account-id"
    
    // 创建客户端
    client, err := httpdns.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // 解析域名
    ctx := context.Background()
    result, err := client.Resolve(ctx, "example.com")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Domain: %s\n", result.Domain)
    fmt.Printf("IPv4: %v\n", result.IPv4)
    fmt.Printf("IPv6: %v\n", result.IPv6)
    fmt.Printf("TTL: %v\n", result.TTL)
    fmt.Printf("Source: %s\n", result.Source)
}
```

### 鉴权解析

```go
config := httpdns.DefaultConfig()
config.AccountID = "your-account-id"
config.SecretKey = "your-secret-key"  // 启用鉴权解析

client, err := httpdns.NewClient(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### 批量解析

```go
// 批量解析最多支持5个域名
domains := []string{"example.com", "google.com", "github.com"}
results, err := client.ResolveBatch(ctx, domains)
if err != nil {
    log.Fatal(err)
}

for _, result := range results {
    fmt.Printf("Domain: %s, IPs: %v\n", result.Domain, result.IPv4)
}
```

**注意**: 单次批量解析最多支持5个域名，超过限制将返回 `ErrTooManyDomains` 错误。

### 异步解析

```go
client.ResolveAsync(ctx, "example.com", func(result *httpdns.ResolveResult, err error) {
    if err != nil {
        log.Printf("Async resolve error: %v", err)
        return
    }
    fmt.Printf("Async result: %s -> %v\n", result.Domain, result.IPv4)
})
```

## 高级配置

### 完整配置示例

```go
config := &httpdns.Config{
    // 认证信息
    AccountID: "your-account-id",
    SecretKey: "your-secret-key",
    
    // 网络配置
    BootstrapIPs: []string{"203.107.1.1", "203.107.1.97"},
    Timeout:      5 * time.Second,
    MaxRetries:   3,
    
    // 功能开关
    EnableHTTPS:   false, // 使用 HTTP
    EnableMetrics: true,  // 启用指标
    
    // 缓存配置
    EnableMemoryCache:     true,  // 启用内存缓存（默认 true）
    EnablePersistentCache: false, // 启用持久化缓存（默认 false）
    AllowExpiredCache:     false, // 允许使用过期缓存（默认 false）
    CacheExpireThreshold:  0,     // 持久化缓存过期阈值（默认 0）
    
    // 日志配置
    Logger: log.New(os.Stdout, "[HTTPDNS] ", log.LstdFlags),
}
```

### 解析选项

```go
// 仅解析 IPv4
result, err := client.Resolve(ctx, "example.com", 
    httpdns.WithIPv4Only(),
    httpdns.WithClientIP("1.2.3.4"))

// 仅解析 IPv6
result, err := client.Resolve(ctx, "example.com", 
    httpdns.WithIPv6Only(),
    httpdns.WithClientIP("1.2.3.4"))

// 自定义超时
result, err := client.Resolve(ctx, "example.com", 
    httpdns.WithTimeout(10*time.Second),
    httpdns.WithClientIP("1.2.3.4"))
```

## 缓存配置

### 基础缓存使用

```go
config := httpdns.DefaultConfig()
config.AccountID = "your-account-id"
// 默认已启用内存缓存，无需额外配置

client, _ := httpdns.NewClient(config)
defer client.Close()

// 第一次解析：发起 HTTP 请求
result1, _ := client.Resolve(ctx, "example.com")

// 第二次解析：命中内存缓存，不发 HTTP 请求
result2, _ := client.Resolve(ctx, "example.com")
```

### 启用持久化缓存

```go
config := httpdns.DefaultConfig()
config.AccountID = "your-account-id"
config.EnablePersistentCache = true  // 启用持久化缓存

client, _ := httpdns.NewClient(config)
defer client.Close()

// 解析结果会自动保存到磁盘
// 下次启动时自动从磁盘加载
result, _ := client.Resolve(ctx, "example.com")
```

**缓存目录**：
- macOS: `~/Library/Caches/alicloud_httpdns/{accountID}/`
- Linux: `~/.cache/alicloud_httpdns/{accountID}/`
- Windows: `%LocalAppData%\alicloud_httpdns\{accountID}\`

### 允许使用过期缓存

```go
config := httpdns.DefaultConfig()
config.AccountID = "your-account-id"
config.AllowExpiredCache = true  // 允许使用过期缓存

client, _ := httpdns.NewClient(config)
defer client.Close()

// 即使缓存过期，也会立即返回旧值
// 同时后台异步发起请求更新缓存
result, _ := client.Resolve(ctx, "example.com")
```

**适用场景**：对延迟敏感的应用，可以接受短暂的过期数据。

### 持久化缓存宽限期

```go
config := httpdns.DefaultConfig()
config.AccountID = "your-account-id"
config.EnablePersistentCache = true
config.CacheExpireThreshold = 5 * time.Minute  // 宽限 5 分钟

client, _ := httpdns.NewClient(config)

// 启动时加载磁盘缓存：
// - 如果记录过期 < 5 分钟：仍然加载
// - 如果记录过期 > 5 分钟：丢弃
```

## 监控和指标

```go
// 获取指标统计
stats := client.GetMetrics()
fmt.Printf("Total Resolves: %d\n", stats.TotalResolves)
fmt.Printf("Success Rate: %.2f%%\n", stats.SuccessRate*100)
fmt.Printf("Average Latency: %v\n", stats.AvgLatency)

// 重置指标
client.ResetMetrics()
```

## 服务管理

```go
// 手动更新服务 IP
err := client.UpdateServiceIPs(ctx)
if err != nil {
    log.Printf("Update service IPs failed: %v", err)
}

// 获取当前服务 IP 列表
ips := client.GetServiceIPs()
fmt.Printf("Service IPs: %v\n", ips)

// 检查客户端健康状态
if client.IsHealthy() {
    fmt.Println("Client is healthy")
}
```

## 错误处理

SDK 提供了结构化的错误处理：

```go
result, err := client.Resolve(ctx, "example.com")
if err != nil {
    if httpDNSErr, ok := err.(*httpdns.HTTPDNSError); ok {
        fmt.Printf("Operation: %s\n", httpDNSErr.Op)
        fmt.Printf("Domain: %s\n", httpDNSErr.Domain)
        fmt.Printf("Error: %v\n", httpDNSErr.Err)
    }
    return
}
```

## 最佳实践

### 1. 客户端生命周期管理

```go
// 应用启动时创建客户端
client, err := httpdns.NewClient(config)
if err != nil {
    log.Fatal(err)
}

// 应用关闭时优雅关闭客户端
defer func() {
    if err := client.Close(); err != nil {
        log.Printf("Close client error: %v", err)
    }
}()
```

### 2. 上下文管理

```go
// 使用带超时的上下文
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

result, err := client.Resolve(ctx, "example.com")
```

### 3. 错误处理

```go
result, err := client.Resolve(ctx, "example.com")
if err != nil {
    log.Printf("Resolve failed: %v", err)
    return
}

// 检查解析来源
if result.Source == httpdns.SourceHTTPDNS {
    log.Println("Used HTTPDNS successfully")
}
```

### 4. 性能优化

```go
// 启用指标监控
config.EnableMetrics = true

// 定期检查指标
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        stats := client.GetMetrics()
        if stats.SuccessRate < 0.95 {
            log.Printf("Low success rate: %.2f%%", stats.SuccessRate*100)
        }
    }
}()
```

## HTTP 客户端集成

通过自定义 `DialContext`，可以将 HTTPDNS 解析结果无缝集成到标准 `net/http` 客户端中，同时支持 HTTP 和 HTTPS。

```go
// 创建集成了 HTTPDNS 的 HTTP 客户端
func createHTTPDNSClient(dnsClient httpdns.Client) *http.Client {
    return &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
                // 解析域名和端口
                host, port, err := net.SplitHostPort(addr)
                if err != nil {
                    return nil, err
                }
                
                // 使用 HTTPDNS 解析域名
                result, err := dnsClient.Resolve(ctx, host)
                if err != nil {
                    return nil, err
                }
                
                if len(result.IPv4) == 0 {
                    return nil, fmt.Errorf("no IP address found for %s", host)
                }
                
                // 使用解析的 IP 建立连接
                resolvedIP := result.IPv4[0].String()
                return net.Dial(network, net.JoinHostPort(resolvedIP, port))
            },
        },
    }
}

// 使用示例
httpClient := createHTTPDNSClient(client)

// 可以请求任意域名，自动使用 HTTPDNS 解析
resp1, _ := httpClient.Get("https://www.example.com/api")
resp2, _ := httpClient.Get("https://www.another.com/data")
```

**关键点**：
- 在 `DialContext` 中动态解析每个域名，支持请求多个不同域名
- URL 使用原始域名（不是 IP），确保 HTTPS 的 SNI 和证书校验正常工作
- 同时支持 HTTP 和 HTTPS，无需区分处理
- 自动利用 HTTPDNS 的缓存机制，避免重复解析

## 故障排查

### 常见问题

1. **认证失败**
   ```
   Error: httpdns auth_failed: authentication failed
   ```
   检查 AccountID 和 SecretKey 是否正确。

2. **网络超时**
   ```
   Error: httpdns http_request: network timeout
   ```
   检查网络连接，考虑增加超时时间。

3. **服务不可用**
   ```
   Error: httpdns fetch_service_ips: service unavailable
   ```
   检查启动 IP 配置，确保至少有一个可用的启动 IP。

### 调试日志

```go
config.Logger = log.New(os.Stdout, "[HTTPDNS] ", log.LstdFlags|log.Lshortfile)
```

### 健康检查

```go
if !client.IsHealthy() {
    log.Println("Client is not healthy, recreating...")
    client.Close()
    client, err = httpdns.NewClient(config)
}
```

## 性能基准

在标准测试环境下的性能表现：

- **单域名解析**: ~50ms (包含网络延迟)
- **批量解析**: ~100ms (5个域名)
- **缓存命中**: ~1ms
- **内存使用**: ~10MB (1000个缓存条目)
- **并发支持**: 1000+ 并发请求

## 许可证

本项目采用 Apache 2.0 许可证。详见 [LICENSE](LICENSE) 文件。

## 贡献

欢迎提交 Issue 和 Pull Request！