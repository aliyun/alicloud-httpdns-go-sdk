package httpdns

import (
	"sync"
	"time"
)

// Metrics 监控指标
type Metrics struct {
	// 解析统计
	TotalResolves   int64 // 总解析次数
	SuccessResolves int64 // 成功解析次数
	FailedResolves  int64 // 失败解析次数
	CacheHits       int64 // 缓存命中次数（当前实现中未使用缓存）

	// 延迟统计
	TotalLatency time.Duration // 总延迟时间
	MinLatency   time.Duration // 最小延迟
	MaxLatency   time.Duration // 最大延迟

	// API统计
	APIRequests     int64         // API请求次数
	APIErrors       int64         // API错误次数
	APIResponseTime time.Duration // API响应时间

	// 错误分类
	NetworkErrors    int64 // 网络错误
	AuthErrors       int64 // 认证错误
	ValidationErrors int64 // 验证错误

	mutex sync.RWMutex
}

// NewMetrics 创建新的指标收集器
func NewMetrics() *Metrics {
	return &Metrics{
		MinLatency: time.Duration(^uint64(0) >> 1), // 设置为最大值
	}
}

// RecordResolve 记录解析操作
func (m *Metrics) RecordResolve(success bool, latency time.Duration, source ResolveSource) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.TotalResolves++
	m.TotalLatency += latency

	if success {
		m.SuccessResolves++
	} else {
		m.FailedResolves++
	}

	// 更新延迟统计
	if latency < m.MinLatency {
		m.MinLatency = latency
	}
	if latency > m.MaxLatency {
		m.MaxLatency = latency
	}
}

// RecordAPIRequest 记录API请求
func (m *Metrics) RecordAPIRequest(success bool, responseTime time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.APIRequests++
	m.APIResponseTime += responseTime

	if !success {
		m.APIErrors++
	}
}

// RecordError 记录错误
func (m *Metrics) RecordError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if httpDNSErr, ok := err.(*HTTPDNSError); ok {
		switch httpDNSErr.Op {
		case "http_request", "request_retry_failed":
			m.NetworkErrors++
		case "auth_failed":
			m.AuthErrors++
		case "validate_domain":
			m.ValidationErrors++
		}
	}
}

// GetStats 获取统计信息
func (m *Metrics) GetStats() MetricsStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := MetricsStats{
		TotalResolves:    m.TotalResolves,
		SuccessResolves:  m.SuccessResolves,
		FailedResolves:   m.FailedResolves,
		CacheHits:        m.CacheHits,
		APIRequests:      m.APIRequests,
		APIErrors:        m.APIErrors,
		NetworkErrors:    m.NetworkErrors,
		AuthErrors:       m.AuthErrors,
		ValidationErrors: m.ValidationErrors,
	}

	// 计算成功率
	if m.TotalResolves > 0 {
		stats.SuccessRate = float64(m.SuccessResolves) / float64(m.TotalResolves)
	}

	// 计算平均延迟
	if m.TotalResolves > 0 {
		stats.AvgLatency = m.TotalLatency / time.Duration(m.TotalResolves)
	}

	stats.MinLatency = m.MinLatency
	stats.MaxLatency = m.MaxLatency

	// 计算API平均响应时间
	if m.APIRequests > 0 {
		stats.AvgAPIResponseTime = m.APIResponseTime / time.Duration(m.APIRequests)
	}

	return stats
}

// Reset 重置统计信息
func (m *Metrics) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 重置所有字段但保留mutex
	m.TotalResolves = 0
	m.SuccessResolves = 0
	m.FailedResolves = 0
	m.CacheHits = 0
	m.TotalLatency = 0
	m.MinLatency = time.Duration(^uint64(0) >> 1)
	m.MaxLatency = 0
	m.APIRequests = 0
	m.APIErrors = 0
	m.APIResponseTime = 0
	m.NetworkErrors = 0
	m.AuthErrors = 0
	m.ValidationErrors = 0
}

// MetricsStats 统计信息快照
type MetricsStats struct {
	// 解析统计
	TotalResolves   int64   `json:"total_resolves"`
	SuccessResolves int64   `json:"success_resolves"`
	FailedResolves  int64   `json:"failed_resolves"`
	CacheHits       int64   `json:"cache_hits"`
	SuccessRate     float64 `json:"success_rate"`

	// 延迟统计
	AvgLatency time.Duration `json:"avg_latency"`
	MinLatency time.Duration `json:"min_latency"`
	MaxLatency time.Duration `json:"max_latency"`

	// API统计
	APIRequests        int64         `json:"api_requests"`
	APIErrors          int64         `json:"api_errors"`
	AvgAPIResponseTime time.Duration `json:"avg_api_response_time"`

	// 错误分类
	NetworkErrors    int64 `json:"network_errors"`
	AuthErrors       int64 `json:"auth_errors"`
	ValidationErrors int64 `json:"validation_errors"`
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	RecordResolve(success bool, latency time.Duration, source ResolveSource)
	RecordAPIRequest(success bool, responseTime time.Duration)
	RecordError(err error)
	GetStats() MetricsStats
	Reset()
}

// NoOpMetrics 空操作指标收集器（用于禁用指标时）
type NoOpMetrics struct{}

func (n *NoOpMetrics) RecordResolve(success bool, latency time.Duration, source ResolveSource) {}
func (n *NoOpMetrics) RecordAPIRequest(success bool, responseTime time.Duration)               {}
func (n *NoOpMetrics) RecordError(err error)                                                   {}
func (n *NoOpMetrics) GetStats() MetricsStats                                                  { return MetricsStats{} }
func (n *NoOpMetrics) Reset()                                                                  {}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(enabled bool) MetricsCollector {
	if enabled {
		return NewMetrics()
	}
	return &NoOpMetrics{}
}
