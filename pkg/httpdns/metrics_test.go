package httpdns

import (
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics()

	if metrics == nil {
		t.Fatal("NewMetrics() returned nil")
	}

	stats := metrics.GetStats()
	if stats.TotalResolves != 0 {
		t.Errorf("NewMetrics() TotalResolves = %v, want 0", stats.TotalResolves)
	}

	if stats.MinLatency != time.Duration(^uint64(0)>>1) {
		t.Errorf("NewMetrics() MinLatency should be max duration")
	}
}

func TestMetrics_RecordResolve(t *testing.T) {
	metrics := NewMetrics()

	// 记录成功解析
	metrics.RecordResolve(true, 100*time.Millisecond, SourceHTTPDNS)
	metrics.RecordResolve(true, 200*time.Millisecond, SourceHTTPDNS)

	// 记录失败解析
	metrics.RecordResolve(false, 50*time.Millisecond, SourceHTTPDNS)

	stats := metrics.GetStats()

	if stats.TotalResolves != 3 {
		t.Errorf("RecordResolve() TotalResolves = %v, want 3", stats.TotalResolves)
	}

	if stats.SuccessResolves != 2 {
		t.Errorf("RecordResolve() SuccessResolves = %v, want 2", stats.SuccessResolves)
	}

	if stats.FailedResolves != 1 {
		t.Errorf("RecordResolve() FailedResolves = %v, want 1", stats.FailedResolves)
	}

	expectedSuccessRate := float64(2) / float64(3)
	if stats.SuccessRate != expectedSuccessRate {
		t.Errorf("RecordResolve() SuccessRate = %v, want %v", stats.SuccessRate, expectedSuccessRate)
	}

	expectedAvgLatency := (100 + 200 + 50) * time.Millisecond / 3
	if stats.AvgLatency != expectedAvgLatency {
		t.Errorf("RecordResolve() AvgLatency = %v, want %v", stats.AvgLatency, expectedAvgLatency)
	}

	if stats.MinLatency != 50*time.Millisecond {
		t.Errorf("RecordResolve() MinLatency = %v, want %v", stats.MinLatency, 50*time.Millisecond)
	}

	if stats.MaxLatency != 200*time.Millisecond {
		t.Errorf("RecordResolve() MaxLatency = %v, want %v", stats.MaxLatency, 200*time.Millisecond)
	}
}

func TestMetrics_RecordAPIRequest(t *testing.T) {
	metrics := NewMetrics()

	// 记录成功API请求
	metrics.RecordAPIRequest(true, 100*time.Millisecond)
	metrics.RecordAPIRequest(true, 200*time.Millisecond)

	// 记录失败API请求
	metrics.RecordAPIRequest(false, 150*time.Millisecond)

	stats := metrics.GetStats()

	if stats.APIRequests != 3 {
		t.Errorf("RecordAPIRequest() APIRequests = %v, want 3", stats.APIRequests)
	}

	if stats.APIErrors != 1 {
		t.Errorf("RecordAPIRequest() APIErrors = %v, want 1", stats.APIErrors)
	}

	expectedAvgResponseTime := (100 + 200 + 150) * time.Millisecond / 3
	if stats.AvgAPIResponseTime != expectedAvgResponseTime {
		t.Errorf("RecordAPIRequest() AvgAPIResponseTime = %v, want %v", stats.AvgAPIResponseTime, expectedAvgResponseTime)
	}
}

func TestMetrics_RecordError(t *testing.T) {
	metrics := NewMetrics()

	// 记录不同类型的错误
	networkErr := NewHTTPDNSError("http_request", "example.com", ErrNetworkTimeout)
	authErr := NewHTTPDNSError("auth_failed", "example.com", ErrAuthFailed)
	validationErr := NewHTTPDNSError("validate_domain", "", ErrInvalidDomain)
	retryErr := NewHTTPDNSError("request_retry_failed", "example.com", ErrServiceUnavailable)

	metrics.RecordError(networkErr)
	metrics.RecordError(authErr)
	metrics.RecordError(validationErr)
	metrics.RecordError(retryErr)

	stats := metrics.GetStats()

	if stats.NetworkErrors != 2 { // http_request + request_retry_failed
		t.Errorf("RecordError() NetworkErrors = %v, want 2", stats.NetworkErrors)
	}

	if stats.AuthErrors != 1 {
		t.Errorf("RecordError() AuthErrors = %v, want 1", stats.AuthErrors)
	}

	if stats.ValidationErrors != 1 {
		t.Errorf("RecordError() ValidationErrors = %v, want 1", stats.ValidationErrors)
	}
}

func TestMetrics_Reset(t *testing.T) {
	metrics := NewMetrics()

	// 记录一些数据
	metrics.RecordResolve(true, 100*time.Millisecond, SourceHTTPDNS)
	metrics.RecordAPIRequest(true, 50*time.Millisecond)

	// 重置
	metrics.Reset()

	stats := metrics.GetStats()

	if stats.TotalResolves != 0 {
		t.Errorf("Reset() TotalResolves = %v, want 0", stats.TotalResolves)
	}

	if stats.APIRequests != 0 {
		t.Errorf("Reset() APIRequests = %v, want 0", stats.APIRequests)
	}

	if stats.MinLatency != time.Duration(^uint64(0)>>1) {
		t.Errorf("Reset() MinLatency should be max duration")
	}
}

func TestNewMetricsCollector(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		want    string
	}{
		{
			name:    "enabled collector",
			enabled: true,
			want:    "*httpdns.Metrics",
		},
		{
			name:    "disabled collector",
			enabled: false,
			want:    "*httpdns.NoOpMetrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewMetricsCollector(tt.enabled)

			if collector == nil {
				t.Fatal("NewMetricsCollector() returned nil")
			}

			// 检查类型
			gotType := ""
			switch collector.(type) {
			case *Metrics:
				gotType = "*httpdns.Metrics"
			case *NoOpMetrics:
				gotType = "*httpdns.NoOpMetrics"
			default:
				gotType = "unknown"
			}

			if gotType != tt.want {
				t.Errorf("NewMetricsCollector() type = %v, want %v", gotType, tt.want)
			}
		})
	}
}

func TestNoOpMetrics(t *testing.T) {
	metrics := &NoOpMetrics{}

	// 所有操作都应该是无操作的
	metrics.RecordResolve(true, 100*time.Millisecond, SourceHTTPDNS)
	metrics.RecordAPIRequest(true, 50*time.Millisecond)
	metrics.RecordError(ErrNetworkTimeout)
	metrics.Reset()

	stats := metrics.GetStats()

	// 所有统计都应该是零值
	if stats.TotalResolves != 0 {
		t.Errorf("NoOpMetrics.GetStats() TotalResolves = %v, want 0", stats.TotalResolves)
	}

	if stats.APIRequests != 0 {
		t.Errorf("NoOpMetrics.GetStats() APIRequests = %v, want 0", stats.APIRequests)
	}
}
