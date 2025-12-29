package httpdns

import (
	"context"
	"encoding/json"
	"net"
	"time"
)

// Resolver 核心解析器
type Resolver struct {
	httpClient *HTTPDNSClient
	config     *Config
	metrics    MetricsCollector
}

// NewResolver 创建新的解析器
func NewResolver(config *Config) *Resolver {
	httpClient := NewHTTPDNSClient(config)

	// 如果配置了SecretKey，设置鉴权管理器
	if config.SecretKey != "" {
		authManager := NewAuthManager(config.SecretKey, config.SignatureExpireTime)
		httpClient.SetAuthManager(authManager)
	}

	return &Resolver{
		httpClient: httpClient,
		config:     config,
		metrics:    NewMetricsCollector(config.EnableMetrics),
	}
}

// ResolveSingle 解析单个域名
func (r *Resolver) ResolveSingle(ctx context.Context, domain string, clientIP string, opts ...ResolveOption) (*ResolveResult, error) {
	startTime := time.Now()
	// 应用选项
	options := &ResolveOptions{
		QueryType: QueryBoth, // 默认查询IPv4和IPv6
		Timeout:   r.config.Timeout,
	}

	for _, opt := range opts {
		opt(options)
	}

	// 创建带超时的上下文
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// 确保有可用的服务IP
	if err := r.httpClient.UpdateServiceIPsIfNeeded(ctx); err != nil {
		return nil, NewHTTPDNSError("resolve_single", domain, err)
	}

	// 执行HTTP请求（每次重试都会获取新的服务IP并构建URL）
	builder := NewRequestBuilder(r.config, r.httpClient.authManager)
	resp, err := r.httpClient.DoRequestWithRetry(ctx, func() (string, error) {
		serviceIP, err := r.httpClient.GetAvailableServiceIP()
		if err != nil {
			return "", err
		}
		return builder.BuildSingleResolveURL(serviceIP, domain, clientIP, options.QueryType), nil
	})
	if err != nil {
		// 记录错误指标
		r.metrics.RecordError(err)
		latency := time.Since(startTime)
		r.metrics.RecordResolve(false, latency, SourceHTTPDNS)
		return nil, NewHTTPDNSError("resolve_single", domain, err)
	}
	defer resp.Body.Close()

	// 解析响应
	var dnsResp HTTPDNSResponse
	if err := json.NewDecoder(resp.Body).Decode(&dnsResp); err != nil {
		return nil, NewHTTPDNSError("resolve_single", domain, err)
	}

	// 转换为ResolveResult
	result := &ResolveResult{
		Domain:    domain,
		ClientIP:  clientIP,
		Source:    SourceHTTPDNS,
		Timestamp: time.Now(),
	}

	// 解析IPv4地址
	for _, ipStr := range dnsResp.IPs {
		if ip := net.ParseIP(ipStr); ip != nil {
			result.IPv4 = append(result.IPv4, ip)
		}
	}

	// 解析IPv6地址
	for _, ipStr := range dnsResp.IPsV6 {
		if ip := net.ParseIP(ipStr); ip != nil {
			result.IPv6 = append(result.IPv6, ip)
		}
	}

	// 设置TTL
	if dnsResp.TTL > 0 {
		result.TTL = time.Duration(dnsResp.TTL) * time.Second
	}

	// 记录指标
	latency := time.Since(startTime)
	r.metrics.RecordResolve(true, latency, result.Source)

	return result, nil
}

// ResolveBatch 批量解析域名
func (r *Resolver) ResolveBatch(ctx context.Context, domains []string, clientIP string, opts ...ResolveOption) ([]*ResolveResult, error) {
	startTime := time.Now()
	
	if len(domains) == 0 {
		return nil, NewHTTPDNSError("resolve_batch", "", ErrInvalidDomain)
	}

	// 检查域名数量限制，最多支持5个域名
	if len(domains) > 5 {
		return nil, NewHTTPDNSError("resolve_batch", "", ErrTooManyDomains)
	}

	// 应用选项
	options := &ResolveOptions{
		QueryType: QueryBoth,
		Timeout:   r.config.Timeout,
	}

	for _, opt := range opts {
		opt(options)
	}

	// 创建带超时的上下文
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// 确保有可用的服务IP
	if err := r.httpClient.UpdateServiceIPsIfNeeded(ctx); err != nil {
		// 记录错误指标
		r.metrics.RecordError(err)
		latency := time.Since(startTime)
		r.metrics.RecordResolve(false, latency, SourceHTTPDNS)
		return nil, NewHTTPDNSError("resolve_batch", "", err)
	}

	// 执行HTTP请求（每次重试都会获取新的服务IP并构建URL）
	builder := NewRequestBuilder(r.config, r.httpClient.authManager)
	resp, err := r.httpClient.DoRequestWithRetry(ctx, func() (string, error) {
		serviceIP, err := r.httpClient.GetAvailableServiceIP()
		if err != nil {
			return "", err
		}
		return builder.BuildBatchResolveURL(serviceIP, domains, clientIP), nil
	})
	if err != nil {
		// 记录错误指标
		r.metrics.RecordError(err)
		latency := time.Since(startTime)
		r.metrics.RecordResolve(false, latency, SourceHTTPDNS)
		return nil, NewHTTPDNSError("resolve_batch", "", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var batchResp BatchResolveResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		// 记录错误指标
		r.metrics.RecordError(err)
		latency := time.Since(startTime)
		r.metrics.RecordResolve(false, latency, SourceHTTPDNS)
		return nil, NewHTTPDNSError("resolve_batch", "", err)
	}

	// 使用map来合并同一域名的多条记录
	domainResults := make(map[string]*ResolveResult)
	timestamp := time.Now()

	for _, dnsResp := range batchResp.DNS {
		domain := dnsResp.Host
		
		// 如果域名还没有结果，创建新的结果
		if domainResults[domain] == nil {
			domainResults[domain] = &ResolveResult{
				Domain:    domain,
				ClientIP:  clientIP,
				Source:    SourceHTTPDNS,
				Timestamp: timestamp,
				IPv4:      make([]net.IP, 0),
				IPv6:      make([]net.IP, 0),
			}
		}
		
		result := domainResults[domain]

		// 处理 IPv4 地址
		for _, ipStr := range dnsResp.IPs {
			if ip := net.ParseIP(ipStr); ip != nil {
				result.IPv4 = append(result.IPv4, ip)
			}
		}

		// 处理 IPv6 地址
		for _, ipStr := range dnsResp.IPsV6 {
			if ip := net.ParseIP(ipStr); ip != nil {
				result.IPv6 = append(result.IPv6, ip)
			}
		}

		// 设置TTL（使用最大的TTL值）
		if dnsResp.TTL > 0 {
			newTTL := time.Duration(dnsResp.TTL) * time.Second
			if newTTL > result.TTL {
				result.TTL = newTTL
			}
		}
	}

	// 转换为结果列表
	results := make([]*ResolveResult, 0, len(domainResults))
	for _, result := range domainResults {
		results = append(results, result)
	}

	// 记录成功指标
	latency := time.Since(startTime)
	r.metrics.RecordResolve(true, latency, SourceHTTPDNS)

	return results, nil
}

// ResolveAsync 异步解析域名
func (r *Resolver) ResolveAsync(ctx context.Context, domain string, clientIP string, callback func(*ResolveResult, error), opts ...ResolveOption) {
	go func() {
		result, err := r.ResolveSingle(ctx, domain, clientIP, opts...)
		callback(result, err)
	}()
}

// ValidateDomain 验证域名格式
func ValidateDomain(domain string) error {
	if domain == "" {
		return ErrInvalidDomain
	}

	// 简单的域名格式验证
	if len(domain) > 253 {
		return ErrInvalidDomain
	}

	return nil
}

// GetMetrics 获取指标统计
func (r *Resolver) GetMetrics() MetricsStats {
	return r.metrics.GetStats()
}

// ResetMetrics 重置指标统计
func (r *Resolver) ResetMetrics() {
	r.metrics.Reset()
}
