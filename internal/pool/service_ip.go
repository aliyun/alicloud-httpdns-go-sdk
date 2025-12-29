package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ServiceIPManager 服务IP管理器
type ServiceIPManager struct {
	serviceIPs []string
	currentIP  string
	failedIPs  map[string]time.Time // 记录失败的IP和失败时间
	updatedAt  time.Time
	mutex      sync.RWMutex
}

// NewServiceIPManager 创建服务IP管理器
func NewServiceIPManager() *ServiceIPManager {
	return &ServiceIPManager{
		failedIPs: make(map[string]time.Time),
	}
}

// UpdateServiceIPs 更新服务IP列表
func (m *ServiceIPManager) UpdateServiceIPs(ips []string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.serviceIPs = make([]string, len(ips))
	copy(m.serviceIPs, ips)
	m.updatedAt = time.Now()

	// 如果当前IP不在新列表中，清空当前IP
	if m.currentIP != "" {
		found := false
		for _, ip := range ips {
			if ip == m.currentIP {
				found = true
				break
			}
		}
		if !found {
			m.currentIP = ""
		}
	}
}

// GetAvailableIP 获取可用的服务IP
func (m *ServiceIPManager) GetAvailableIP() (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.serviceIPs) == 0 {
		return "", fmt.Errorf("no service IPs available")
	}

	// 正常情况下使用当前IP
	if m.currentIP != "" {
		// 检查当前IP是否在失败列表中
		if failTime, exists := m.failedIPs[m.currentIP]; !exists ||
			time.Since(failTime) > 5*time.Minute { // 5分钟后重试失败的IP
			return m.currentIP, nil
		}
	}

	// 异常情况下轮转到下一个可用IP
	for _, ip := range m.serviceIPs {
		if failTime, exists := m.failedIPs[ip]; !exists ||
			time.Since(failTime) > 5*time.Minute {
			m.currentIP = ip
			return ip, nil
		}
	}

	// 如果所有IP都失败，返回第一个IP（可能已经恢复）
	m.currentIP = m.serviceIPs[0]
	return m.currentIP, nil
}

// MarkIPFailed 标记IP失败
func (m *ServiceIPManager) MarkIPFailed(ip string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.failedIPs[ip] = time.Now()

	// 如果当前IP失败，清空当前IP，下次会自动选择其他IP
	if m.currentIP == ip {
		m.currentIP = ""
	}
}

// GetServiceIPs 获取所有服务IP
func (m *ServiceIPManager) GetServiceIPs() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make([]string, len(m.serviceIPs))
	copy(result, m.serviceIPs)
	return result
}

// GetUpdatedAt 获取更新时间
func (m *ServiceIPManager) GetUpdatedAt() time.Time {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.updatedAt
}

// IsEmpty 检查是否为空
func (m *ServiceIPManager) IsEmpty() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.serviceIPs) == 0
}

// ServiceIPResponse 服务IP列表响应
type ServiceIPResponse struct {
	ServiceIP   []string `json:"service_ip"`   // IPv4服务IP列表
	ServiceIPv6 []string `json:"service_ipv6"` // IPv6服务IP列表
}

// BootstrapManager 启动IP管理器
type BootstrapManager struct {
	bootstrapIPs []string
	domain       string
}

// NewBootstrapManager 创建启动IP管理器
func NewBootstrapManager(bootstrapIPs []string, domain string) *BootstrapManager {
	return &BootstrapManager{
		bootstrapIPs: bootstrapIPs,
		domain:       domain,
	}
}

// FetchServiceIPs 获取服务IP列表 - 启动IP使用for循环方式消费
func (b *BootstrapManager) FetchServiceIPs(ctx context.Context, client *http.Client, accountID string, enableHTTPS bool) ([]string, error) {
	protocol := "http"
	if enableHTTPS {
		protocol = "https"
	}

	// 遍历所有启动IP
	for _, bootstrapIP := range b.bootstrapIPs {
		url := fmt.Sprintf("%s://%s/%s/ss", protocol, bootstrapIP, accountID)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue // 尝试下一个启动IP
		}

		resp, err := client.Do(req)
		if err != nil {
			continue // 尝试下一个启动IP
		}

		if resp.StatusCode == http.StatusOK {
			var serviceResp ServiceIPResponse
			err := json.NewDecoder(resp.Body).Decode(&serviceResp)
			resp.Body.Close()

			if err == nil && len(serviceResp.ServiceIP) > 0 {
				return serviceResp.ServiceIP, nil
			}
		} else {
			resp.Body.Close()
		}
	}

	// 如果所有启动IP都失败，尝试使用启动域名
	if b.domain != "" {
		url := fmt.Sprintf("%s://%s/%s/ss", protocol, b.domain, accountID)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err == nil {
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var serviceResp ServiceIPResponse
					if err := json.NewDecoder(resp.Body).Decode(&serviceResp); err == nil && len(serviceResp.ServiceIP) > 0 {
						return serviceResp.ServiceIP, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("failed to fetch service IPs from all bootstrap IPs and domain")
}
