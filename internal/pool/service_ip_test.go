package pool

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewServiceIPManager(t *testing.T) {
	manager := NewServiceIPManager()

	if manager == nil {
		t.Fatal("NewServiceIPManager() returned nil")
	}

	if manager.failedIPs == nil {
		t.Error("NewServiceIPManager() failedIPs not initialized")
	}

	if !manager.IsEmpty() {
		t.Error("NewServiceIPManager() should be empty initially")
	}
}

func TestServiceIPManager_UpdateServiceIPs(t *testing.T) {
	manager := NewServiceIPManager()
	ips := []string{"1.2.3.4", "5.6.7.8"}

	manager.UpdateServiceIPs(ips)

	if manager.IsEmpty() {
		t.Error("UpdateServiceIPs() should not be empty after update")
	}

	gotIPs := manager.GetServiceIPs()
	if len(gotIPs) != len(ips) {
		t.Errorf("UpdateServiceIPs() got %d IPs, want %d", len(gotIPs), len(ips))
	}

	for i, ip := range ips {
		if gotIPs[i] != ip {
			t.Errorf("UpdateServiceIPs() got IP[%d] = %v, want %v", i, gotIPs[i], ip)
		}
	}

	if manager.GetUpdatedAt().IsZero() {
		t.Error("UpdateServiceIPs() should set updated time")
	}
}

func TestServiceIPManager_GetAvailableIP(t *testing.T) {
	manager := NewServiceIPManager()

	// 测试空列表
	_, err := manager.GetAvailableIP()
	if err == nil {
		t.Error("GetAvailableIP() should return error for empty list")
	}

	// 添加IP
	ips := []string{"1.2.3.4", "5.6.7.8"}
	manager.UpdateServiceIPs(ips)

	// 获取可用IP
	ip, err := manager.GetAvailableIP()
	if err != nil {
		t.Errorf("GetAvailableIP() error = %v", err)
	}

	found := false
	for _, validIP := range ips {
		if ip == validIP {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("GetAvailableIP() returned invalid IP: %v", ip)
	}
}

func TestServiceIPManager_MarkIPFailed(t *testing.T) {
	manager := NewServiceIPManager()
	ips := []string{"1.2.3.4", "5.6.7.8"}
	manager.UpdateServiceIPs(ips)

	// 获取第一个IP
	ip1, err := manager.GetAvailableIP()
	if err != nil {
		t.Fatalf("GetAvailableIP() error = %v", err)
	}

	// 标记失败
	manager.MarkIPFailed(ip1)

	// 再次获取应该得到不同的IP
	ip2, err := manager.GetAvailableIP()
	if err != nil {
		t.Fatalf("GetAvailableIP() error = %v", err)
	}

	if ip1 == ip2 && len(ips) > 1 {
		t.Error("GetAvailableIP() should return different IP after marking failed")
	}
}

func TestServiceIPManager_FailedIPRecovery(t *testing.T) {
	manager := NewServiceIPManager()
	ips := []string{"1.2.3.4"}
	manager.UpdateServiceIPs(ips)

	// 标记IP失败
	manager.MarkIPFailed(ips[0])

	// 立即获取应该还是返回这个IP（因为没有其他选择）
	ip, err := manager.GetAvailableIP()
	if err != nil {
		t.Errorf("GetAvailableIP() error = %v", err)
	}
	if ip != ips[0] {
		t.Errorf("GetAvailableIP() = %v, want %v", ip, ips[0])
	}
}

func TestNewBootstrapManager(t *testing.T) {
	bootstrapIPs := []string{"1.2.3.4", "5.6.7.8"}
	domain := "example.com"

	manager := NewBootstrapManager(bootstrapIPs, domain)

	if manager == nil {
		t.Fatal("NewBootstrapManager() returned nil")
	}

	if len(manager.bootstrapIPs) != len(bootstrapIPs) {
		t.Errorf("NewBootstrapManager() got %d bootstrap IPs, want %d", len(manager.bootstrapIPs), len(bootstrapIPs))
	}

	if manager.domain != domain {
		t.Errorf("NewBootstrapManager() domain = %v, want %v", manager.domain, domain)
	}
}

func TestBootstrapManager_FetchServiceIPs(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test123/ss" {
			response := ServiceIPResponse{
				ServiceIP: []string{"203.107.1.33", "203.107.1.34"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 从测试服务器URL中提取IP和端口
	serverURL := server.URL[7:] // 去掉 "http://"

	manager := NewBootstrapManager([]string{serverURL}, "")
	client := &http.Client{Timeout: 5 * time.Second}

	ctx := context.Background()
	ips, err := manager.FetchServiceIPs(ctx, client, "test123", false)

	if err != nil {
		t.Errorf("FetchServiceIPs() error = %v", err)
	}

	if len(ips) != 2 {
		t.Errorf("FetchServiceIPs() got %d IPs, want 2", len(ips))
	}

	expectedIPs := []string{"203.107.1.33", "203.107.1.34"}
	for i, expectedIP := range expectedIPs {
		if ips[i] != expectedIP {
			t.Errorf("FetchServiceIPs() IP[%d] = %v, want %v", i, ips[i], expectedIP)
		}
	}
}

func TestBootstrapManager_FetchServiceIPs_AllFailed(t *testing.T) {
	// 使用不存在的IP
	manager := NewBootstrapManager([]string{"192.0.2.1"}, "")
	client := &http.Client{Timeout: 100 * time.Millisecond}

	ctx := context.Background()
	_, err := manager.FetchServiceIPs(ctx, client, "test123", false)

	if err == nil {
		t.Error("FetchServiceIPs() should return error when all bootstrap IPs fail")
	}
}

func TestBootstrapManager_FetchServiceIPs_DomainFallback(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ServiceIPResponse{
			ServiceIP: []string{"203.107.1.35"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// 从测试服务器URL中提取IP和端口
	serverURL := server.URL[7:] // 去掉 "http://"

	// 使用不存在的启动IP，但提供域名兜底
	manager := NewBootstrapManager([]string{"192.0.2.1"}, serverURL)
	client := &http.Client{Timeout: 100 * time.Millisecond}

	ctx := context.Background()
	ips, err := manager.FetchServiceIPs(ctx, client, "test123", false)

	if err != nil {
		t.Errorf("FetchServiceIPs() error = %v", err)
	}

	if len(ips) != 1 || ips[0] != "203.107.1.35" {
		t.Errorf("FetchServiceIPs() = %v, want [203.107.1.35]", ips)
	}
}
