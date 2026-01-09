package httpdns

import (
	"context"
	"net"
	"time"
)

// Client 是HTTPDNS客户端的主接口
type Client interface {
	// Resolve 解析单个域名
	Resolve(ctx context.Context, domain string, opts ...ResolveOption) (*ResolveResult, error)

	// ResolveBatch 批量解析域名
	ResolveBatch(ctx context.Context, domains []string, opts ...ResolveOption) ([]*ResolveResult, error)

	// ResolveAsync 异步解析域名
	ResolveAsync(ctx context.Context, domain string, callback func(*ResolveResult, error), opts ...ResolveOption)

	// Close 关闭客户端
	Close() error

	// GetMetrics 获取指标统计
	GetMetrics() MetricsStats

	// ResetMetrics 重置指标统计
	ResetMetrics()

	// UpdateServiceIPs 手动更新服务IP
	UpdateServiceIPs(ctx context.Context) error

	// GetServiceIPs 获取当前服务IP列表
	GetServiceIPs() []string

	// IsHealthy 检查客户端健康状态
	IsHealthy() bool
}

// ResolveResult 解析结果
type ResolveResult struct {
	Domain    string        // 域名
	ClientIP  string        // 客户端IP
	IPv4      []net.IP      // IPv4地址列表
	IPv6      []net.IP      // IPv6地址列表
	TTL       time.Duration // TTL时间
	Source    ResolveSource // 解析来源
	Timestamp time.Time     // 解析时间戳
	Error     error         // 错误信息
}

// ResolveSource 解析来源
type ResolveSource int

const (
	SourceHTTPDNS ResolveSource = iota
)

// String 返回解析来源的字符串表示
func (s ResolveSource) String() string {
	switch s {
	case SourceHTTPDNS:
		return "HTTPDNS"
	default:
		return "Unknown"
	}
}

// ResolveOption 解析选项
type ResolveOption func(*ResolveOptions)

// ResolveOptions 解析选项配置
type ResolveOptions struct {
	QueryType QueryType     // 查询类型
	Timeout   time.Duration // 超时时间
	ClientIP  string        // 客户端IP
}

// QueryType 查询类型，对应API中的query参数
type QueryType string

const (
	QueryIPv4 QueryType = "4"   // 仅IPv4
	QueryIPv6 QueryType = "6"   // 仅IPv6
	QueryBoth QueryType = "4,6" // IPv4和IPv6
)

// WithIPv4Only 仅解析IPv4
func WithIPv4Only() ResolveOption {
	return func(opts *ResolveOptions) {
		opts.QueryType = QueryIPv4
	}
}

// WithIPv6Only 仅解析IPv6
func WithIPv6Only() ResolveOption {
	return func(opts *ResolveOptions) {
		opts.QueryType = QueryIPv6
	}
}

// WithBothIP 解析IPv4和IPv6
func WithBothIP() ResolveOption {
	return func(opts *ResolveOptions) {
		opts.QueryType = QueryBoth
	}
}

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) ResolveOption {
	return func(opts *ResolveOptions) {
		opts.Timeout = timeout
	}
}

// WithClientIP 设置客户端IP
func WithClientIP(ip string) ResolveOption {
	return func(opts *ResolveOptions) {
		opts.ClientIP = ip
	}
}

// HTTPDNSResponse EMAS HTTPDNS API响应结构
type HTTPDNSResponse struct {
	Host      string   `json:"host"`
	IPs       []string `json:"ips"`   // IPv4地址列表
	IPsV6     []string `json:"ipsv6"` // IPv6地址列表
	TTL       int      `json:"ttl"`
	OriginTTL int      `json:"origin_ttl"` // 原始TTL
	ClientIP  string   `json:"client_ip"`  // 客户端IP（批量解析时返回）
}

// BatchResolveResponse 批量解析响应
type BatchResolveResponse struct {
	DNS []HTTPDNSResponse `json:"dns"`
}

// ServiceIPResponse 服务IP列表响应
type ServiceIPResponse struct {
	ServiceIP   []string `json:"service_ip"`   // IPv4服务IP列表
	ServiceIPv6 []string `json:"service_ipv6"` // IPv6服务IP列表
}

// Logger 日志接口
type Logger interface {
	Printf(format string, v ...interface{})
}
