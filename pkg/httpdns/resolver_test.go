package httpdns

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewResolver(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"

	resolver := NewResolver(config)

	if resolver == nil {
		t.Fatal("NewResolver() returned nil")
	}

	if resolver.config != config {
		t.Error("NewResolver() config not set correctly")
	}

	if resolver.httpClient == nil {
		t.Error("NewResolver() HTTP client not created")
	}
}

func TestNewResolver_WithAuth(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"
	config.SecretKey = "secret123"

	resolver := NewResolver(config)

	if resolver.httpClient.authManager == nil {
		t.Error("NewResolver() should create auth manager when SecretKey is provided")
	}
}

func TestResolver_ResolveSingle(t *testing.T) {
	// 创建测试服务器
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/test123/ss" {
			// 服务IP响应 - 返回测试服务器自己的地址
			serverAddr := server.URL[7:] // 去掉 "http://"
			response := map[string]interface{}{
				"service_ip": []string{serverAddr},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/test123/d" {
			// 域名解析响应
			response := HTTPDNSResponse{
				Host: "example.com",
				IPs:  []string{"1.2.3.4", "5.6.7.8"},
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
	config.BootstrapIPs = []string{server.URL[7:]} // 去掉 "http://"

	resolver := NewResolver(config)

	ctx := context.Background()
	result, err := resolver.ResolveSingle(ctx, "example.com", WithClientIP("1.2.3.4"))

	if err != nil {
		t.Errorf("ResolveSingle() error = %v", err)
	}

	if result == nil {
		t.Fatal("ResolveSingle() returned nil result")
	}

	if result.Domain != "example.com" {
		t.Errorf("ResolveSingle() domain = %v, want %v", result.Domain, "example.com")
	}

	if result.ClientIP != "1.2.3.4" {
		t.Errorf("ResolveSingle() clientIP = %v, want %v", result.ClientIP, "1.2.3.4")
	}

	if len(result.IPv4) != 2 {
		t.Errorf("ResolveSingle() got %d IPv4 addresses, want 2", len(result.IPv4))
	}

	if result.TTL != 300*time.Second {
		t.Errorf("ResolveSingle() TTL = %v, want %v", result.TTL, 300*time.Second)
	}

	if result.Source != SourceHTTPDNS {
		t.Errorf("ResolveSingle() source = %v, want %v", result.Source, SourceHTTPDNS)
	}
}

func TestResolver_ResolveSingle_WithOptions(t *testing.T) {
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
			// 检查查询参数
			queryType := r.URL.Query().Get("query")
			if queryType != "4" {
				t.Errorf("Expected query=4, got query=%s", queryType)
			}

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

	resolver := NewResolver(config)

	ctx := context.Background()
	result, err := resolver.ResolveSingle(ctx, "example.com", WithIPv4Only())

	if err != nil {
		t.Errorf("ResolveSingle() error = %v", err)
	}

	if result == nil {
		t.Fatal("ResolveSingle() returned nil result")
	}
}

func TestResolver_ResolveBatch(t *testing.T) {
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
			// 批量解析响应
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

	resolver := NewResolver(config)

	ctx := context.Background()
	domains := []string{"example.com", "test.com"}
	results, err := resolver.ResolveBatch(ctx, domains, WithClientIP("1.2.3.4"))

	if err != nil {
		t.Errorf("ResolveBatch() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("ResolveBatch() got %d results, want 2", len(results))
	}

	if len(results) > 0 {
		// 检查第一个结果
		if results[0].Domain != "example.com" {
			t.Errorf("ResolveBatch() result[0].Domain = %v, want %v", results[0].Domain, "example.com")
		}

		if len(results[0].IPv4) != 1 {
			t.Errorf("ResolveBatch() result[0] got %d IPv4 addresses, want 1", len(results[0].IPv4))
		}
	}

	if len(results) > 1 {
		// 检查第二个结果
		if results[1].Domain != "test.com" {
			t.Errorf("ResolveBatch() result[1].Domain = %v, want %v", results[1].Domain, "test.com")
		}
	}
}

func TestResolver_ResolveBatch_EmptyDomains(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"
	resolver := NewResolver(config)

	ctx := context.Background()
	_, err := resolver.ResolveBatch(ctx, []string{})

	if err == nil {
		t.Error("ResolveBatch() should return error for empty domains")
	}

	httpDNSErr, ok := err.(*HTTPDNSError)
	if !ok {
		t.Errorf("ResolveBatch() error should be *HTTPDNSError, got %T", err)
	} else if httpDNSErr.Op != "resolve_batch" {
		t.Errorf("ResolveBatch() error op = %v, want %v", httpDNSErr.Op, "resolve_batch")
	}
}

func TestResolver_ResolveBatch_TooManyDomains(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"
	resolver := NewResolver(config)

	ctx := context.Background()
	// 创建超过5个域名的列表
	domains := []string{
		"domain1.com",
		"domain2.com",
		"domain3.com",
		"domain4.com",
		"domain5.com",
		"domain6.com", // 第6个域名，应该触发错误
	}

	_, err := resolver.ResolveBatch(ctx, domains)

	if err == nil {
		t.Error("ResolveBatch() should return error for too many domains")
	}

	httpDNSErr, ok := err.(*HTTPDNSError)
	if !ok {
		t.Errorf("ResolveBatch() error should be *HTTPDNSError, got %T", err)
		return
	}

	if httpDNSErr.Op != "resolve_batch" {
		t.Errorf("ResolveBatch() error op = %v, want %v", httpDNSErr.Op, "resolve_batch")
	}

	if httpDNSErr.Err != ErrTooManyDomains {
		t.Errorf("ResolveBatch() error should be ErrTooManyDomains, got %v", httpDNSErr.Err)
	}
}

func TestResolver_ResolveBatch_ExactlyFiveDomains(t *testing.T) {
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
			// 批量解析响应 - 返回5个域名的结果
			response := BatchResolveResponse{
				DNS: []HTTPDNSResponse{
					{Host: "domain1.com", IPs: []string{"1.1.1.1"}, TTL: 300},
					{Host: "domain2.com", IPs: []string{"2.2.2.2"}, TTL: 300},
					{Host: "domain3.com", IPs: []string{"3.3.3.3"}, TTL: 300},
					{Host: "domain4.com", IPs: []string{"4.4.4.4"}, TTL: 300},
					{Host: "domain5.com", IPs: []string{"5.5.5.5"}, TTL: 300},
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

	resolver := NewResolver(config)

	ctx := context.Background()
	// 创建正好5个域名的列表
	domains := []string{
		"domain1.com",
		"domain2.com",
		"domain3.com",
		"domain4.com",
		"domain5.com",
	}

	results, err := resolver.ResolveBatch(ctx, domains, WithClientIP("1.2.3.4"))

	if err != nil {
		t.Errorf("ResolveBatch() should not return error for exactly 5 domains, got: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("ResolveBatch() got %d results, want 5", len(results))
	}
}

func TestResolver_ResolveAsync(t *testing.T) {
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

	resolver := NewResolver(config)

	ctx := context.Background()
	resultChan := make(chan *ResolveResult, 1)
	errorChan := make(chan error, 1)

	resolver.ResolveAsync(ctx, "example.com", func(result *ResolveResult, err error) {
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

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{
			name:    "valid domain",
			domain:  "example.com",
			wantErr: false,
		},
		{
			name:    "empty domain",
			domain:  "",
			wantErr: true,
		},
		{
			name:    "too long domain",
			domain:  string(make([]byte, 254)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDomain(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDomain() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

