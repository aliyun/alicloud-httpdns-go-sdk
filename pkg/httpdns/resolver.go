package httpdns

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"time"
)

// Resolver 核心解析器
type Resolver struct {
	httpClient   *HTTPDNSClient
	config       *Config
	metrics      MetricsCollector
	cacheManager *CacheManager
	
	// 异步更新控制（防止同一域名重复刷新）
	updateMu sync.Mutex
	updating map[string]bool
}

// NewResolver 创建新的解析器
func NewResolver(config *Config) *Resolver {
	httpClient := NewHTTPDNSClient(config)

	// 如果配置了SecretKey，设置鉴权管理器
	if config.SecretKey != "" {
		authManager := NewAuthManager(config.SecretKey, config.SignatureExpireTime)
		httpClient.SetAuthManager(authManager)
	}

	// 创建缓存管理器
	cacheManager := NewCacheManager(config)

	// 如果启用持久化缓存，从磁盘加载缓存
	if config.EnablePersistentCache {
		if err := cacheManager.LoadFromDisk(); err != nil && config.Logger != nil {
			config.Logger.Printf("Failed to load cache from disk: %v", err)
		}
	}

	return &Resolver{
		httpClient:   httpClient,
		config:       config,
		metrics:      NewMetricsCollector(config.EnableMetrics),
		cacheManager: cacheManager,
		updating:     make(map[string]bool),
	}
}

// ResolveSingle 解析单个域名
func (r *Resolver) ResolveSingle(ctx context.Context, domain string, opts ...ResolveOption) (*ResolveResult, error) {
	startTime := time.Now()
	// 应用选项
	options := &ResolveOptions{
		QueryType: QueryBoth, // 默认查询IPv4和IPv6
		Timeout:   r.config.Timeout,
	}

	for _, opt := range opts {
		opt(options)
	}

	// 查询缓存
	if entry, hit, needAsyncUpdate := r.cacheManager.Get(domain); hit {
		if r.config.Logger != nil {
			r.config.Logger.Printf("Cache hit for domain: %s, expired: %v", domain, needAsyncUpdate)
		}
		
		// 如果需要异步更新，启动后台更新
		if needAsyncUpdate {
			r.tryAsyncUpdate(domain, func() {
				r.asyncUpdate(ctx, domain, options.ClientIP, options.QueryType)
			})
		}
		
		result := entry.ToResolveResult(domain)
		result.ClientIP = options.ClientIP
		
		// 记录指标
		latency := time.Since(startTime)
		r.metrics.RecordResolve(true, latency, result.Source)
		
		return result, nil
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
		return builder.BuildSingleResolveURL(serviceIP, domain, options.ClientIP, options.QueryType), nil
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
		ClientIP:  options.ClientIP,
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

	// 更新缓存
	r.updateCache(domain, result, dnsResp.TTL)

	// 记录指标
	latency := time.Since(startTime)
	r.metrics.RecordResolve(true, latency, result.Source)

	return result, nil
}

// ResolveBatch 批量解析域名
func (r *Resolver) ResolveBatch(ctx context.Context, domains []string, opts ...ResolveOption) ([]*ResolveResult, error) {
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

	// 查询缓存，分离命中和未命中的域名
	cachedResults := make([]*ResolveResult, 0)
	uncachedDomains := make([]string, 0)
	
	for _, domain := range domains {
		if entry, hit, needAsyncUpdate := r.cacheManager.Get(domain); hit {
			if r.config.Logger != nil {
				r.config.Logger.Printf("Cache hit for domain: %s, expired: %v", domain, needAsyncUpdate)
			}
			
			result := entry.ToResolveResult(domain)
			result.ClientIP = options.ClientIP
			cachedResults = append(cachedResults, result)
			
			// 如果需要异步更新，启动后台更新
			if needAsyncUpdate {
				r.tryAsyncUpdate(domain, func() {
					r.asyncUpdate(ctx, domain, options.ClientIP, options.QueryType)
				})
			}
		} else {
			uncachedDomains = append(uncachedDomains, domain)
		}
	}

	// 如果所有域名都命中缓存，直接返回
	if len(uncachedDomains) == 0 {
		latency := time.Since(startTime)
		r.metrics.RecordResolve(true, latency, SourceHTTPDNS)
		return cachedResults, nil
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
		return builder.BuildBatchResolveURL(serviceIP, uncachedDomains, options.ClientIP), nil
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
	domainTTLs := make(map[string]int) // 记录原始TTL用于缓存
	timestamp := time.Now()

	for _, dnsResp := range batchResp.DNS {
		domain := dnsResp.Host
		
		// 如果域名还没有结果，创建新的结果
		if domainResults[domain] == nil {
			domainResults[domain] = &ResolveResult{
				Domain:    domain,
				ClientIP:  options.ClientIP,
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
				domainTTLs[domain] = dnsResp.TTL
			}
		}
	}

	// 更新缓存并转换为结果列表
	networkResults := make([]*ResolveResult, 0, len(uncachedDomains))
	for _, domain := range uncachedDomains {
		if result, ok := domainResults[domain]; ok {
			r.updateCache(domain, result, domainTTLs[domain])
			networkResults = append(networkResults, result)
		}
	}

	// 合并缓存结果和网络结果
	allResults := append(cachedResults, networkResults...)

	// 记录成功指标
	latency := time.Since(startTime)
	r.metrics.RecordResolve(true, latency, SourceHTTPDNS)

	return allResults, nil
}

// ResolveAsync 异步解析域名
func (r *Resolver) ResolveAsync(ctx context.Context, domain string, callback func(*ResolveResult, error), opts ...ResolveOption) {
	go func() {
		result, err := r.ResolveSingle(ctx, domain, opts...)
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

// updateCache 更新缓存
func (r *Resolver) updateCache(domain string, result *ResolveResult, ttl int) {
	// 构建缓存条目
	entry := &CacheEntry{
		IPv4:      make([]string, 0, len(result.IPv4)),
		IPv6:      make([]string, 0, len(result.IPv6)),
		TTL:       ttl,
		QueryTime: result.Timestamp,
	}

	for _, ip := range result.IPv4 {
		entry.IPv4 = append(entry.IPv4, ip.String())
	}

	for _, ip := range result.IPv6 {
		entry.IPv6 = append(entry.IPv6, ip.String())
	}

	// 更新内存缓存
	r.cacheManager.Set(domain, entry)

	// 异步保存到磁盘
	r.cacheManager.SaveResolveCacheAsync()
}

// tryAsyncUpdate 尝试启动异步更新（防止同一域名重复刷新）
func (r *Resolver) tryAsyncUpdate(domain string, fn func()) {
	r.updateMu.Lock()
	if r.updating[domain] {
		r.updateMu.Unlock()
		return
	}
	r.updating[domain] = true
	r.updateMu.Unlock()
	
	go func() {
		defer func() {
			r.updateMu.Lock()
			delete(r.updating, domain)
			r.updateMu.Unlock()
		}()
		fn()
	}()
}

// asyncUpdate 异步更新缓存
func (r *Resolver) asyncUpdate(ctx context.Context, domain, clientIP string, queryType QueryType) {
	// 创建新的上下文，避免使用已取消的上下文
	asyncCtx, cancel := context.WithTimeout(context.Background(), r.config.Timeout)
	defer cancel()

	// 确保有可用的服务IP
	if err := r.httpClient.UpdateServiceIPsIfNeeded(asyncCtx); err != nil {
		if r.config.Logger != nil {
			r.config.Logger.Printf("Async update failed for %s: %v", domain, err)
		}
		return
	}

	// 执行HTTP请求
	builder := NewRequestBuilder(r.config, r.httpClient.authManager)
	resp, err := r.httpClient.DoRequestWithRetry(asyncCtx, func() (string, error) {
		serviceIP, err := r.httpClient.GetAvailableServiceIP()
		if err != nil {
			return "", err
		}
		return builder.BuildSingleResolveURL(serviceIP, domain, clientIP, queryType), nil
	})
	if err != nil {
		if r.config.Logger != nil {
			r.config.Logger.Printf("Async update failed for %s: %v", domain, err)
		}
		return
	}
	defer resp.Body.Close()

	// 解析响应
	var dnsResp HTTPDNSResponse
	if err := json.NewDecoder(resp.Body).Decode(&dnsResp); err != nil {
		if r.config.Logger != nil {
			r.config.Logger.Printf("Async update parse failed for %s: %v", domain, err)
		}
		return
	}

	// 构建结果
	result := &ResolveResult{
		Domain:    domain,
		ClientIP:  clientIP,
		Source:    SourceHTTPDNS,
		Timestamp: time.Now(),
	}

	for _, ipStr := range dnsResp.IPs {
		if ip := net.ParseIP(ipStr); ip != nil {
			result.IPv4 = append(result.IPv4, ip)
		}
	}

	for _, ipStr := range dnsResp.IPsV6 {
		if ip := net.ParseIP(ipStr); ip != nil {
			result.IPv6 = append(result.IPv6, ip)
		}
	}

	if dnsResp.TTL > 0 {
		result.TTL = time.Duration(dnsResp.TTL) * time.Second
	}

	// 更新缓存
	r.updateCache(domain, result, dnsResp.TTL)

	if r.config.Logger != nil {
		r.config.Logger.Printf("Async update completed for %s", domain)
	}
}
