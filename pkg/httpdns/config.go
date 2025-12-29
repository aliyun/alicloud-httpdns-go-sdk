package httpdns

import "time"

// 默认EMAS HTTPDNS启动IP（中国内地）
var DefaultBootstrapIPs = []string{
	"203.107.1.1",
	"203.107.1.97",
	"203.107.1.100",
	"203.119.238.240",
	"106.11.25.239",
	"59.82.99.47",
}

// 默认启动域名（兜底）
var DefaultBootstrapDomain = "resolvers-cn.httpdns.aliyuncs.com"

// 默认HTTPS SNI域名
var DefaultHTTPSSNI = "resolver-cns.aliyuncs.com"

// Config 客户端配置
type Config struct {
	// 认证信息
	AccountID string
	SecretKey string // 可选，用于鉴权解析

	// 网络配置
	BootstrapIPs []string // 默认使用DefaultBootstrapIPs，支持用户自定义
	Timeout      time.Duration
	MaxRetries   int // 重试次数，默认0不重试，避免频率限制

	// 功能开关
	EnableHTTPS   bool // 是否使用HTTPS，默认false使用HTTP
	EnableMetrics bool

	// HTTPS配置
	HTTPSSNIHost string // HTTPS SNI主机名，默认使用DefaultHTTPSSNI

	// 签名配置
	SignatureExpireTime time.Duration // 签名过期时间，默认30秒

	// 日志配置
	Logger Logger
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		BootstrapIPs:        DefaultBootstrapIPs,
		Timeout:             5 * time.Second,
		MaxRetries:          0,     // 默认不重试，避免频率限制问题
		EnableHTTPS:         false, // 默认使用HTTP
		EnableMetrics:       false,
		HTTPSSNIHost:        DefaultHTTPSSNI,  // 默认HTTPS SNI主机名
		SignatureExpireTime: 30 * time.Second, // 默认30秒签名过期时间
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.AccountID == "" {
		return ErrInvalidConfig
	}
	if c.Timeout <= 0 {
		c.Timeout = 5 * time.Second
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = 0 // 允许0次重试
	}
	if len(c.BootstrapIPs) == 0 {
		c.BootstrapIPs = DefaultBootstrapIPs
	}
	if c.SignatureExpireTime <= 0 {
		c.SignatureExpireTime = 30 * time.Second
	}
	if c.HTTPSSNIHost == "" {
		c.HTTPSSNIHost = DefaultHTTPSSNI
	}
	return nil
}
