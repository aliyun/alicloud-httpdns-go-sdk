package httpdns

import (
	"errors"
	"fmt"
)

// 定义具体的错误类型
var (
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrAuthFailed         = errors.New("authentication failed")
	ErrNetworkTimeout     = errors.New("network timeout")
	ErrInvalidDomain      = errors.New("invalid domain name")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrTooManyDomains     = errors.New("too many domains, maximum 5 domains allowed per batch request")
)

// HTTPDNSError 包装错误信息
type HTTPDNSError struct {
	Op     string // 操作名称
	Domain string // 相关域名
	Err    error  // 原始错误
}

func (e *HTTPDNSError) Error() string {
	if e.Domain != "" {
		return fmt.Sprintf("httpdns %s %s: %v", e.Op, e.Domain, e.Err)
	}
	return fmt.Sprintf("httpdns %s: %v", e.Op, e.Err)
}

func (e *HTTPDNSError) Unwrap() error {
	return e.Err
}

// NewHTTPDNSError 创建新的HTTPDNS错误
func NewHTTPDNSError(op, domain string, err error) *HTTPDNSError {
	return &HTTPDNSError{
		Op:     op,
		Domain: domain,
		Err:    err,
	}
}
