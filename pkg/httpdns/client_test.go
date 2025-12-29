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
	result, err := client.Resolve(ctx, "example.com", "1.2.3.4")

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
	results, err := client.ResolveBatch(ctx, domains, "1.2.3.4")

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

	client.ResolveAsync(ctx, "example.com", "", func(result *ResolveResult, err error) {
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
	_, err = client.Resolve(ctx, "example.com", "")
	if err == nil {
		t.Error("Resolve() should return error after client is closed")
	}

	_, err = client.ResolveBatch(ctx, []string{"example.com"}, "")
	if err == nil {
		t.Error("ResolveBatch() should return error after client is closed")
	}

	errorChan := make(chan error, 1)
	client.ResolveAsync(ctx, "example.com", "", func(result *ResolveResult, err error) {
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
