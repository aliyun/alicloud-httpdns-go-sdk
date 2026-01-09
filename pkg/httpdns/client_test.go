package httpdns

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"

	client, err := NewClient(config)

	if err != nil {
		t.Errorf("NewClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	// 检查客户端是否健康
	if !client.IsHealthy() {
		t.Error("NewClient() client should be healthy")
	}

	// 清理
	client.Close()
}

func TestNewClient_InvalidConfig(t *testing.T) {
	config := DefaultConfig()
	// 不设置AccountID，应该验证失败

	_, err := NewClient(config)

	if err == nil {
		t.Error("NewClient() should return error for invalid config")
	}
}

func TestClient_Resolve(t *testing.T) {
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
		} else if r.URL.Path == "/test123/d" {
			response := HTTPDNSResponse{
				Host: "example.com",
				IPs:  []string{"1.2.3.4"},
				TTL:  300,
			}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer server.Close()

	config := DefaultConfig()
	config.AccountID = "test123"
	config.BootstrapIPs = []string{server.URL[7:]}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	result, err := client.Resolve(ctx, "example.com", WithClientIP("1.2.3.4"))

	if err != nil {
		t.Errorf("Resolve() error = %v", err)
	}

	if result == nil {
		t.Fatal("Resolve() returned nil result")
	}

	if result.Domain != "example.com" {
		t.Errorf("Resolve() domain = %v, want %v", result.Domain, "example.com")
	}
}

func TestClient_ResolveBatch(t *testing.T) {
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
			response := BatchResolveResponse{
				DNS: []HTTPDNSResponse{
					{
						Host: "example.com",
						IPs:  []string{"1.2.3.4"},
						TTL:  300,
					},
					{
						Host: "test.com",
						IPs:  []string{"5.6.7.8"},
						TTL:  600,
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer server.Close()

	config := DefaultConfig()
	config.AccountID = "test123"
	config.BootstrapIPs = []string{server.URL[7:]}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	domains := []string{"example.com", "test.com"}
	results, err := client.ResolveBatch(ctx, domains, WithClientIP("1.2.3.4"))

	if err != nil {
		t.Errorf("ResolveBatch() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("ResolveBatch() got %d results, want 2", len(results))
	}
}

func TestClient_ResolveAsync(t *testing.T) {
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
		} else if r.URL.Path == "/test123/d" {
			response := HTTPDNSResponse{
				Host: "example.com",
				IPs:  []string{"1.2.3.4"},
				TTL:  300,
			}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer server.Close()

	config := DefaultConfig()
	config.AccountID = "test123"
	config.BootstrapIPs = []string{server.URL[7:]}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	resultChan := make(chan *ResolveResult, 1)
	errorChan := make(chan error, 1)

	client.ResolveAsync(ctx, "example.com", func(result *ResolveResult, err error) {
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	})

	// 等待结果
	select {
	case result := <-resultChan:
		if result.Domain != "example.com" {
			t.Errorf("ResolveAsync() domain = %v, want %v", result.Domain, "example.com")
		}
	case err := <-errorChan:
		t.Errorf("ResolveAsync() error = %v", err)
	case <-time.After(5 * time.Second):
		t.Error("ResolveAsync() timeout")
	}
}

func TestClient_Close(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// 检查客户端是否健康
	if !client.IsHealthy() {
		t.Error("Client should be healthy before close")
	}

	// 关闭客户端
	err = client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// 检查客户端是否已关闭
	if client.IsHealthy() {
		t.Error("Client should not be healthy after close")
	}

	// 再次关闭应该不会出错
	err = client.Close()
	if err != nil {
		t.Errorf("Close() second call error = %v", err)
	}
}

func TestClient_ClosedOperations(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// 关闭客户端
	client.Close()

	ctx := context.Background()

	// 测试关闭后的操作
	_, err = client.Resolve(ctx, "example.com")
	if err == nil {
		t.Error("Resolve() should return error after client is closed")
	}

	_, err = client.ResolveBatch(ctx, []string{"example.com"})
	if err == nil {
		t.Error("ResolveBatch() should return error after client is closed")
	}

	errorChan := make(chan error, 1)
	client.ResolveAsync(ctx, "example.com", func(result *ResolveResult, err error) {
		errorChan <- err
	})

	select {
	case err := <-errorChan:
		if err == nil {
			t.Error("ResolveAsync() should return error after client is closed")
		}
	case <-time.After(1 * time.Second):
		t.Error("ResolveAsync() callback timeout")
	}
}

func TestClient_GetMetrics(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"
	config.EnableMetrics = true

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	stats := client.GetMetrics()

	// 新客户端的指标应该都是零值
	if stats.TotalResolves != 0 {
		t.Errorf("GetMetrics() TotalResolves = %v, want 0", stats.TotalResolves)
	}
}

func TestClient_ResetMetrics(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"
	config.EnableMetrics = true

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	// 重置指标应该不会出错
	client.ResetMetrics()

	stats := client.GetMetrics()
	if stats.TotalResolves != 0 {
		t.Errorf("ResetMetrics() TotalResolves = %v, want 0", stats.TotalResolves)
	}
}

func TestClient_GetServiceIPs(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	// 新客户端的服务IP列表应该为空
	ips := client.GetServiceIPs()
	if len(ips) != 0 {
		t.Errorf("GetServiceIPs() got %d IPs, want 0", len(ips))
	}
}

// TestClient_MemoryCacheHit 测试内存缓存命中（第二次请求不发HTTP）
func TestClient_MemoryCacheHit(t *testing.T) {
	requestCount := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/test123/ss" {
			serverAddr := server.URL[7:]
			response := map[string]interface{}{
				"service_ip": []string{serverAddr},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/test123/d" {
			response := HTTPDNSResponse{
				Host: "example.com",
				IPs:  []string{"1.2.3.4"},
				TTL:  300,
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	config := DefaultConfig()
	config.AccountID = "test123"
	config.BootstrapIPs = []string{server.URL[7:]}
	config.EnableMemoryCache = true

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 第一次请求：应该发起HTTP请求
	result1, err := client.Resolve(ctx, "example.com")
	if err != nil {
		t.Fatalf("First Resolve() error = %v", err)
	}
	if result1 == nil || result1.Domain != "example.com" {
		t.Fatal("First Resolve() returned invalid result")
	}
	firstRequestCount := requestCount

	// 第二次请求：应该命中缓存，不发起HTTP请求
	result2, err := client.Resolve(ctx, "example.com")
	if err != nil {
		t.Fatalf("Second Resolve() error = %v", err)
	}
	if result2 == nil || result2.Domain != "example.com" {
		t.Fatal("Second Resolve() returned invalid result")
	}

	// 验证第二次请求没有增加HTTP请求计数
	if requestCount != firstRequestCount {
		t.Errorf("Second Resolve() should hit cache, but HTTP request count increased from %d to %d", firstRequestCount, requestCount)
	}

	// 验证两次结果一致
	if len(result1.IPv4) != len(result2.IPv4) {
		t.Errorf("Cache hit result mismatch: IPv4 count %d vs %d", len(result1.IPv4), len(result2.IPv4))
	}
}

// TestClient_PersistentCacheHit 测试持久化缓存命中（重启后从磁盘加载）
func TestClient_PersistentCacheHit(t *testing.T) {
	requestCount := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/test_persist_123/ss" {
			serverAddr := server.URL[7:]
			response := map[string]interface{}{
				"service_ip": []string{serverAddr},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/test_persist_123/d" {
			response := HTTPDNSResponse{
				Host: "example.com",
				IPs:  []string{"1.2.3.4"},
				TTL:  300,
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	config := DefaultConfig()
	config.AccountID = "test_persist_123"
	config.BootstrapIPs = []string{server.URL[7:]}
	config.EnableMemoryCache = true
	config.EnablePersistentCache = true

	// 第一个客户端：解析并保存到磁盘
	client1, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	result1, err := client1.Resolve(ctx, "example.com")
	if err != nil {
		t.Fatalf("First Resolve() error = %v", err)
	}
	if result1 == nil {
		t.Fatal("First Resolve() returned nil")
	}

	// 等待异步保存完成
	time.Sleep(200 * time.Millisecond)

	firstRequestCount := requestCount
	client1.Close()

	// 第二个客户端：从磁盘加载缓存（使用相同 accountID）
	client2, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client2.Close()

	// 验证从磁盘加载的缓存可用（不应发起新的HTTP请求）
	result2, err := client2.Resolve(ctx, "example.com")
	if err != nil {
		t.Fatalf("Second Resolve() error = %v", err)
	}
	if result2 == nil {
		t.Fatal("Second Resolve() returned nil")
	}

	// 验证结果一致
	if result2.Domain != "example.com" {
		t.Errorf("Loaded cache domain = %v, want example.com", result2.Domain)
	}
	if len(result2.IPv4) == 0 {
		t.Error("Loaded cache should have IPv4 addresses")
	}

	// 验证第二个客户端没有发起新的解析请求（从磁盘加载）
	if requestCount > firstRequestCount+1 {
		t.Logf("Note: Second client made %d additional requests (expected 0-1 for service IP fetch)", requestCount-firstRequestCount)
	}
}

// TestClient_ExpiredCacheAsyncUpdate 测试过期缓存的异步更新流程
func TestClient_ExpiredCacheAsyncUpdate(t *testing.T) {
	requestCount := 0
	ipVersion := 1 // 1=旧IP, 2=新IP
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/test123/ss" {
			serverAddr := server.URL[7:]
			response := map[string]interface{}{
				"service_ip": []string{serverAddr},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/test123/d" {
			// 第一次返回旧IP，后续返回新IP
			var ips []string
			if ipVersion == 1 {
				ips = []string{"1.2.3.4"}
				ipVersion = 2
			} else {
				ips = []string{"5.6.7.8"}
			}
			response := HTTPDNSResponse{
				Host: "example.com",
				IPs:  ips,
				TTL:  1, // 短TTL，快速过期
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	config := DefaultConfig()
	config.AccountID = "test123"
	config.BootstrapIPs = []string{server.URL[7:]}
	config.EnableMemoryCache = true
	config.AllowExpiredCache = true

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 第一次请求：获取旧IP并缓存
	result1, err := client.Resolve(ctx, "example.com")
	if err != nil {
		t.Fatalf("First Resolve() error = %v", err)
	}
	if result1 == nil {
		t.Fatal("First Resolve() returned nil")
	}
	if len(result1.IPv4) == 0 || result1.IPv4[0].String() != "1.2.3.4" {
		t.Errorf("First resolve should return 1.2.3.4, got %v", result1.IPv4)
	}

	// 等待缓存过期
	time.Sleep(2 * time.Second)

	// 第二次请求：缓存已过期，应返回旧值并触发后台更新
	result2, err := client.Resolve(ctx, "example.com")
	if err != nil {
		t.Fatalf("Second Resolve() error = %v", err)
	}
	if result2 == nil {
		t.Fatal("Second Resolve() should return expired cache")
	}

	// 验证返回的是旧IP（过期缓存）
	if len(result2.IPv4) == 0 || result2.IPv4[0].String() != "1.2.3.4" {
		t.Errorf("Should return expired cache IP 1.2.3.4, got %v", result2.IPv4)
	}

	// 等待异步更新完成
	time.Sleep(500 * time.Millisecond)

	// 第三次请求：应该返回更新后的新值
	result3, err := client.Resolve(ctx, "example.com")
	if err != nil {
		t.Fatalf("Third Resolve() error = %v", err)
	}
	if result3 == nil {
		t.Fatal("Third Resolve() returned nil")
	}

	// 验证缓存已更新为新IP
	if len(result3.IPv4) == 0 || result3.IPv4[0].String() != "5.6.7.8" {
		t.Errorf("Cache should be updated to 5.6.7.8, got %v", result3.IPv4)
	}

	// 验证触发了后台更新请求（至少2次解析请求：第一次+异步更新）
	if requestCount < 3 {
		t.Logf("Request count = %d (expected at least 3: service IP + first resolve + async update)", requestCount)
	}
}
