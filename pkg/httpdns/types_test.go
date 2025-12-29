package httpdns

import (
	"testing"
	"time"
)

func TestResolveSource_String(t *testing.T) {
	tests := []struct {
		source   ResolveSource
		expected string
	}{
		{SourceHTTPDNS, "HTTPDNS"},
		{ResolveSource(999), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.source.String(); got != tt.expected {
			t.Errorf("ResolveSource.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestResolveOptions(t *testing.T) {
	opts := &ResolveOptions{}

	// 测试WithIPv4Only
	WithIPv4Only()(opts)
	if opts.QueryType != QueryIPv4 {
		t.Errorf("WithIPv4Only() failed, got %v, want %v", opts.QueryType, QueryIPv4)
	}

	// 测试WithIPv6Only
	WithIPv6Only()(opts)
	if opts.QueryType != QueryIPv6 {
		t.Errorf("WithIPv6Only() failed, got %v, want %v", opts.QueryType, QueryIPv6)
	}

	// 测试WithBothIP
	WithBothIP()(opts)
	if opts.QueryType != QueryBoth {
		t.Errorf("WithBothIP() failed, got %v, want %v", opts.QueryType, QueryBoth)
	}

	// 测试WithTimeout
	timeout := 10 * time.Second
	WithTimeout(timeout)(opts)
	if opts.Timeout != timeout {
		t.Errorf("WithTimeout() failed, got %v, want %v", opts.Timeout, timeout)
	}
}
