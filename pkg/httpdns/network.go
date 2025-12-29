package httpdns

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/alicloud-httpdns-go-sdk/internal/pool"
)

// HTTPDNSClient HTTP客户端封装
type HTTPDNSClient struct {
	client           *http.Client
	config           *Config
	authManager      *AuthManager
	serviceIPManager *pool.ServiceIPManager
	bootstrapManager *pool.BootstrapManager
}

// NewHTTPDNSClient 创建新的HTTP客户端
func NewHTTPDNSClient(config *Config) *HTTPDNSClient {
	return &HTTPDNSClient{
		client:           newHTTPClient(config),
		config:           config,
		serviceIPManager: pool.NewServiceIPManager(),
		bootstrapManager: pool.NewBootstrapManager(config.BootstrapIPs, DefaultBootstrapDomain),
	}
}

// SetAuthManager 设置鉴权管理器
func (c *HTTPDNSClient) SetAuthManager(authManager *AuthManager) {
	c.authManager = authManager
}

// newHTTPClient 创建HTTP客户端
func newHTTPClient(config *Config) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	// 如果启用HTTPS，配置TLS
	if config.EnableHTTPS {
		transport.TLSClientConfig = &tls.Config{
			ServerName: config.HTTPSSNIHost, // 使用配置的SNI主机名
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}
}

// RequestBuilder HTTP请求构建器
type RequestBuilder struct {
	config      *Config
	authManager *AuthManager
}

// NewRequestBuilder 创建请求构建器
func NewRequestBuilder(config *Config, authManager *AuthManager) *RequestBuilder {
	return &RequestBuilder{
		config:      config,
		authManager: authManager,
	}
}

// BuildSingleResolveURL 构建单域名解析URL
func (b *RequestBuilder) BuildSingleResolveURL(serviceIP, domain, clientIP string, queryType QueryType) string {
	protocol := "http"
	if b.config.EnableHTTPS {
		protocol = "https"
	}

	baseURL := fmt.Sprintf("%s://%s/%s", protocol, serviceIP, b.config.AccountID)

	if b.authManager != nil {
		// 鉴权解析
		timestamp, signature := b.authManager.GenerateSignature(domain)
		if clientIP != "" {
			return fmt.Sprintf("%s/sign_d?host=%s&query=%s&ip=%s&t=%s&s=%s",
				baseURL, domain, queryType, clientIP, timestamp, signature)
		} else {
			return fmt.Sprintf("%s/sign_d?host=%s&query=%s&t=%s&s=%s",
				baseURL, domain, queryType, timestamp, signature)
		}
	} else {
		// 非鉴权解析
		if clientIP != "" {
			return fmt.Sprintf("%s/d?host=%s&query=%s&ip=%s",
				baseURL, domain, queryType, clientIP)
		} else {
			return fmt.Sprintf("%s/d?host=%s&query=%s",
				baseURL, domain, queryType)
		}
	}
}

// BuildBatchResolveURL 构建批量域名解析URL
func (b *RequestBuilder) BuildBatchResolveURL(serviceIP string, domains []string, clientIP string) string {
	protocol := "http"
	if b.config.EnableHTTPS {
		protocol = "https"
	}

	baseURL := fmt.Sprintf("%s://%s/%s", protocol, serviceIP, b.config.AccountID)
	hostParam := strings.Join(domains, ",")

	if b.authManager != nil {
		// 鉴权解析
		timestamp, signature := b.authManager.GenerateBatchSignature(domains)
		if clientIP != "" {
			return fmt.Sprintf("%s/sign_resolve?host=%s&ip=%s&t=%s&s=%s",
				baseURL, hostParam, clientIP, timestamp, signature)
		} else {
			return fmt.Sprintf("%s/sign_resolve?host=%s&t=%s&s=%s",
				baseURL, hostParam, timestamp, signature)
		}
	} else {
		// 非鉴权解析
		if clientIP != "" {
			return fmt.Sprintf("%s/resolve?host=%s&ip=%s",
				baseURL, hostParam, clientIP)
		} else {
			return fmt.Sprintf("%s/resolve?host=%s",
				baseURL, hostParam)
		}
	}
}

// BuildServiceIPURL 构建服务IP获取URL
func (b *RequestBuilder) BuildServiceIPURL(bootstrapIP string) string {
	protocol := "http"
	if b.config.EnableHTTPS {
		protocol = "https"
	}

	return fmt.Sprintf("%s://%s/%s/ss", protocol, bootstrapIP, b.config.AccountID)
}

// DoRequest 执行HTTP请求
func (c *HTTPDNSClient) DoRequest(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, NewHTTPDNSError("create_request", "", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, NewHTTPDNSError("http_request", "", err)
	}

	return resp, nil
}

// AuthManager 鉴权管理器
type AuthManager struct {
	secretKey  string
	expireTime time.Duration
}

// NewAuthManager 创建鉴权管理器
func NewAuthManager(secretKey string, expireTime time.Duration) *AuthManager {
	return &AuthManager{
		secretKey:  secretKey,
		expireTime: expireTime,
	}
}

// GenerateSignature 生成单域名解析签名
func (a *AuthManager) GenerateSignature(host string) (timestamp, signature string) {
	// 使用当前时间加上过期时间作为时间戳，确保请求在有效期内
	expireAt := time.Now().Add(a.expireTime)
	timestamp = strconv.FormatInt(expireAt.Unix(), 10)
	signature = generateSignature(a.secretKey, host, timestamp)
	return
}

// GenerateBatchSignature 生成批量解析签名
func (a *AuthManager) GenerateBatchSignature(hosts []string) (timestamp, signature string) {
	// 使用当前时间加上过期时间作为时间戳，确保请求在有效期内
	expireAt := time.Now().Add(a.expireTime)
	timestamp = strconv.FormatInt(expireAt.Unix(), 10)
	signature = generateBatchSignature(a.secretKey, hosts, timestamp)
	return
}

// FetchServiceIPs 获取服务IP列表
func (c *HTTPDNSClient) FetchServiceIPs(ctx context.Context) error {
	ips, err := c.bootstrapManager.FetchServiceIPs(ctx, c.client, c.config.AccountID, c.config.EnableHTTPS)
	if err != nil {
		return NewHTTPDNSError("fetch_service_ips", "", err)
	}

	c.serviceIPManager.UpdateServiceIPs(ips)
	return nil
}

// GetAvailableServiceIP 获取可用的服务IP
func (c *HTTPDNSClient) GetAvailableServiceIP() (string, error) {
	// 如果没有服务IP，尝试获取
	if c.serviceIPManager.IsEmpty() {
		ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
		defer cancel()

		if err := c.FetchServiceIPs(ctx); err != nil {
			return "", err
		}
	}

	return c.serviceIPManager.GetAvailableIP()
}

// MarkServiceIPFailed 标记服务IP失败
func (c *HTTPDNSClient) MarkServiceIPFailed(ip string) {
	c.serviceIPManager.MarkIPFailed(ip)
}

// DoRequestWithRetry 执行HTTP请求并处理故障转移
func (c *HTTPDNSClient) DoRequestWithRetry(ctx context.Context, buildURL func() (string, error)) (*http.Response, error) {
	var lastErr error
	maxAttempts := c.config.MaxRetries + 1 // 至少执行一次请求

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// 每次重试都获取新的 URL
		url, err := buildURL()
		if err != nil {
			lastErr = err
			// 如果构建 URL 失败，等待后继续重试
			if attempt < maxAttempts-1 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Duration(attempt+1) * time.Second):
				}
			}
			continue
		}

		resp, err := c.DoRequest(ctx, url)
		if err == nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		if resp != nil {
			resp.Body.Close()
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = NewHTTPDNSError("http_status", "",
				fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status))
		}

		// 如果还有重试机会，进行重试准备
		if attempt < maxAttempts-1 {
			// 从URL中提取服务IP并标记为失败
			if serviceIP := extractServiceIPFromURL(url); serviceIP != "" {
				c.MarkServiceIPFailed(serviceIP)
			}

			// 等待一段时间后重试
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt+1) * time.Second): // 指数退避
			}
		}
	}

	return nil, NewHTTPDNSError("request_retry_failed", "", lastErr)
}

// extractServiceIPFromURL 从URL中提取服务IP
func extractServiceIPFromURL(url string) string {
	// 简单的URL解析，提取主机部分
	if strings.HasPrefix(url, "http://") {
		url = url[7:]
	} else if strings.HasPrefix(url, "https://") {
		url = url[8:]
	}

	if idx := strings.Index(url, "/"); idx != -1 {
		return url[:idx]
	}

	return url
}

// ShouldUpdateServiceIPs 检查是否需要更新服务IP
func (c *HTTPDNSClient) ShouldUpdateServiceIPs() bool {
	if c.serviceIPManager.IsEmpty() {
		return true
	}

	// 检查是否超过8小时未更新
	lastUpdate := c.serviceIPManager.GetUpdatedAt()
	return time.Since(lastUpdate) > 8*time.Hour
}

// UpdateServiceIPsIfNeeded 根据需要更新服务IP
func (c *HTTPDNSClient) UpdateServiceIPsIfNeeded(ctx context.Context) error {
	if c.ShouldUpdateServiceIPs() {
		return c.FetchServiceIPs(ctx)
	}
	return nil
}
