package httpdns

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPDNSClient(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"

	client := NewHTTPDNSClient(config)

	if client == nil {
		t.Fatal("NewHTTPDNSClient() returned nil")
	}

	if client.config != config {
		t.Error("NewHTTPDNSClient() config not set correctly")
	}

	if client.client == nil {
		t.Error("NewHTTPDNSClient() HTTP client not created")
	}
}

func TestNewHTTPClient(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name: "HTTP client",
			config: &Config{
				EnableHTTPS: false,
				Timeout:     5 * time.Second,
			},
		},
		{
			name: "HTTPS client",
			config: &Config{
				EnableHTTPS:  true,
				Timeout:      10 * time.Second,
				HTTPSSNIHost: DefaultHTTPSSNI,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newHTTPClient(tt.config)

			if client == nil {
				t.Fatal("newHTTPClient() returned nil")
			}

			if client.Timeout != tt.config.Timeout {
				t.Errorf("newHTTPClient() timeout = %v, want %v", client.Timeout, tt.config.Timeout)
			}

			transport, ok := client.Transport.(*http.Transport)
			if !ok {
				t.Fatal("newHTTPClient() transport is not *http.Transport")
			}

			if transport.MaxIdleConns != 100 {
				t.Errorf("newHTTPClient() MaxIdleConns = %v, want %v", transport.MaxIdleConns, 100)
			}

			if tt.config.EnableHTTPS {
				if transport.TLSClientConfig == nil {
					t.Error("newHTTPClient() TLS config not set for HTTPS")
				} else if transport.TLSClientConfig.ServerName != tt.config.HTTPSSNIHost {
					t.Errorf("newHTTPClient() TLS ServerName = %v, want %v", transport.TLSClientConfig.ServerName, tt.config.HTTPSSNIHost)
				}
			}
		})
	}
}

func TestRequestBuilder_BuildSingleResolveURL(t *testing.T) {
	config := &Config{
		AccountID:   "test123",
		EnableHTTPS: false,
	}

	tests := []struct {
		name        string
		authManager *AuthManager
		serviceIP   string
		domain      string
		clientIP    string
		queryType   QueryType
		wantContain []string
	}{
		{
			name:        "non-auth without client IP",
			authManager: nil,
			serviceIP:   "203.107.1.1",
			domain:      "example.com",
			clientIP:    "",
			queryType:   QueryIPv4,
			wantContain: []string{"http://203.107.1.1/test123/d", "host=example.com", "query=4"},
		},
		{
			name:        "non-auth with client IP",
			authManager: nil,
			serviceIP:   "203.107.1.1",
			domain:      "example.com",
			clientIP:    "1.2.3.4",
			queryType:   QueryBoth,
			wantContain: []string{"http://203.107.1.1/test123/d", "host=example.com", "query=4,6", "ip=1.2.3.4"},
		},
		{
			name:        "auth with client IP",
			authManager: NewAuthManager("secret123", 30*time.Second),
			serviceIP:   "203.107.1.1",
			domain:      "example.com",
			clientIP:    "1.2.3.4",
			queryType:   QueryIPv6,
			wantContain: []string{"http://203.107.1.1/test123/sign_d", "host=example.com", "query=6", "ip=1.2.3.4", "t=", "s="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewRequestBuilder(config, tt.authManager)
			url := builder.BuildSingleResolveURL(tt.serviceIP, tt.domain, tt.clientIP, tt.queryType)

			for _, contain := range tt.wantContain {
				if !strings.Contains(url, contain) {
					t.Errorf("BuildSingleResolveURL() = %v, should contain %v", url, contain)
				}
			}
		})
	}
}

func TestRequestBuilder_BuildBatchResolveURL(t *testing.T) {
	config := &Config{
		AccountID:   "test123",
		EnableHTTPS: false,
	}

	tests := []struct {
		name        string
		authManager *AuthManager
		serviceIP   string
		domains     []string
		clientIP    string
		wantContain []string
	}{
		{
			name:        "non-auth batch resolve",
			authManager: nil,
			serviceIP:   "203.107.1.1",
			domains:     []string{"example.com", "test.com"},
			clientIP:    "1.2.3.4",
			wantContain: []string{"http://203.107.1.1/test123/resolve", "host=example.com,test.com", "ip=1.2.3.4"},
		},
		{
			name:        "auth batch resolve",
			authManager: NewAuthManager("secret123", 30*time.Second),
			serviceIP:   "203.107.1.1",
			domains:     []string{"example.com", "test.com"},
			clientIP:    "",
			wantContain: []string{"http://203.107.1.1/test123/sign_resolve", "host=example.com,test.com", "t=", "s="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewRequestBuilder(config, tt.authManager)
			url := builder.BuildBatchResolveURL(tt.serviceIP, tt.domains, tt.clientIP)

			for _, contain := range tt.wantContain {
				if !strings.Contains(url, contain) {
					t.Errorf("BuildBatchResolveURL() = %v, should contain %v", url, contain)
				}
			}
		})
	}
}

func TestRequestBuilder_BuildServiceIPURL(t *testing.T) {
	tests := []struct {
		name        string
		enableHTTPS bool
		bootstrapIP string
		expected    string
	}{
		{
			name:        "HTTP service IP URL",
			enableHTTPS: false,
			bootstrapIP: "203.107.1.1",
			expected:    "http://203.107.1.1/test123/ss",
		},
		{
			name:        "HTTPS service IP URL",
			enableHTTPS: true,
			bootstrapIP: "203.107.1.1",
			expected:    "https://203.107.1.1/test123/ss",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				AccountID:   "test123",
				EnableHTTPS: tt.enableHTTPS,
			}
			builder := NewRequestBuilder(config, nil)
			url := builder.BuildServiceIPURL(tt.bootstrapIP)

			if url != tt.expected {
				t.Errorf("BuildServiceIPURL() = %v, want %v", url, tt.expected)
			}
		})
	}
}

func TestHTTPDNSClient_DoRequest(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"host":"example.com","ips":["1.2.3.4"],"ttl":300}`))
	}))
	defer server.Close()

	config := DefaultConfig()
	config.AccountID = "test123"
	client := NewHTTPDNSClient(config)

	ctx := context.Background()
	resp, err := client.DoRequest(ctx, server.URL)

	if err != nil {
		t.Fatalf("DoRequest() error = %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("DoRequest() status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	resp.Body.Close()
}

func TestHTTPDNSClient_DoRequest_Error(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"
	config.Timeout = 1 * time.Millisecond // 设置很短的超时时间
	client := NewHTTPDNSClient(config)

	ctx := context.Background()
	_, err := client.DoRequest(ctx, "http://192.0.2.1:12345") // 使用不存在的地址

	if err == nil {
		t.Error("DoRequest() should return error for invalid URL")
		return
	}

	httpDNSErr, ok := err.(*HTTPDNSError)
	if !ok {
		t.Errorf("DoRequest() error should be *HTTPDNSError, got %T", err)
	} else if httpDNSErr.Op != "http_request" {
		t.Errorf("DoRequest() error op = %v, want %v", httpDNSErr.Op, "http_request")
	}
}

func TestHTTPDNSClient_GetAvailableServiceIP(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"service_ip":["203.107.1.33","203.107.1.34"]}`))
	}))
	defer server.Close()

	config := DefaultConfig()
	config.AccountID = "test123"
	config.BootstrapIPs = []string{server.URL[7:]} // 去掉 "http://"
	client := NewHTTPDNSClient(config)

	ip, err := client.GetAvailableServiceIP()
	if err != nil {
		t.Errorf("GetAvailableServiceIP() error = %v", err)
	}

	expectedIPs := []string{"203.107.1.33", "203.107.1.34"}
	found := false
	for _, expectedIP := range expectedIPs {
		if ip == expectedIP {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("GetAvailableServiceIP() = %v, want one of %v", ip, expectedIPs)
	}
}

func TestHTTPDNSClient_MarkServiceIPFailed(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"service_ip":["203.107.1.33","203.107.1.34"]}`))
	}))
	defer server.Close()

	config := DefaultConfig()
	config.AccountID = "test123"
	config.BootstrapIPs = []string{server.URL[7:]}
	client := NewHTTPDNSClient(config)

	// 获取第一个IP
	ip1, err := client.GetAvailableServiceIP()
	if err != nil {
		t.Fatalf("GetAvailableServiceIP() error = %v", err)
	}

	// 标记失败
	client.MarkServiceIPFailed(ip1)

	// 再次获取应该得到不同的IP（如果有多个IP的话）
	ip2, err := client.GetAvailableServiceIP()
	if err != nil {
		t.Fatalf("GetAvailableServiceIP() error = %v", err)
	}

	// 如果有多个IP，应该得到不同的IP
	serviceIPs := client.serviceIPManager.GetServiceIPs()
	if len(serviceIPs) > 1 && ip1 == ip2 {
		t.Error("GetAvailableServiceIP() should return different IP after marking failed")
	}
}

func TestHTTPDNSClient_ShouldUpdateServiceIPs(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"
	client := NewHTTPDNSClient(config)

	// 空的服务IP列表应该需要更新
	if !client.ShouldUpdateServiceIPs() {
		t.Error("ShouldUpdateServiceIPs() should return true for empty service IPs")
	}

	// 添加服务IP
	client.serviceIPManager.UpdateServiceIPs([]string{"1.2.3.4"})

	// 刚更新的不应该需要更新
	if client.ShouldUpdateServiceIPs() {
		t.Error("ShouldUpdateServiceIPs() should return false for recently updated service IPs")
	}
}

func TestExtractServiceIPFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "HTTP URL",
			url:      "http://203.107.1.1/test123/d?host=example.com",
			expected: "203.107.1.1",
		},
		{
			name:     "HTTPS URL",
			url:      "https://203.107.1.1:443/test123/d?host=example.com",
			expected: "203.107.1.1:443",
		},
		{
			name:     "URL without path",
			url:      "http://203.107.1.1",
			expected: "203.107.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractServiceIPFromURL(tt.url)
			if got != tt.expected {
				t.Errorf("extractServiceIPFromURL() = %v, want %v", got, tt.expected)
			}
		})
	}
}
