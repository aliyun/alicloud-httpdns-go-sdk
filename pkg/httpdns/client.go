package httpdns

import (
	"context"
	"sync"
	"time"
)

// client 主客户端实现
type client struct {
	resolver *Resolver
	config   *Config
	stopCh   chan struct{}
	wg       sync.WaitGroup
	started  bool
	mutex    sync.RWMutex
}

// NewClient 创建新的HTTPDNS客户端
func NewClient(config *Config) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	resolver := NewResolver(config)

	c := &client{
		resolver: resolver,
		config:   config,
		stopCh:   make(chan struct{}),
	}

	// 启动定时更新服务IP的goroutine
	c.start()

	return c, nil
}

// start 启动客户端
func (c *client) start() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.started {
		return
	}

	c.started = true
	c.wg.Add(1)

	go c.periodicUpdateServiceIPs()
}

// periodicUpdateServiceIPs 定时更新服务IP
func (c *client) periodicUpdateServiceIPs() {
	defer c.wg.Done()

	ticker := time.NewTicker(8 * time.Hour) // 每8小时更新一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
			if err := c.resolver.httpClient.FetchServiceIPs(ctx); err != nil {
				// 记录错误但不中断服务
				if c.config.Logger != nil {
					c.config.Logger.Printf("Failed to update service IPs: %v", err)
				}
			}
			cancel()
		case <-c.stopCh:
			return
		}
	}
}

// Resolve 解析单个域名
func (c *client) Resolve(ctx context.Context, domain string, clientIP string, opts ...ResolveOption) (*ResolveResult, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.started {
		return nil, NewHTTPDNSError("client_stopped", domain, ErrServiceUnavailable)
	}

	return c.resolver.ResolveSingle(ctx, domain, clientIP, opts...)
}

// ResolveBatch 批量解析域名
func (c *client) ResolveBatch(ctx context.Context, domains []string, clientIP string, opts ...ResolveOption) ([]*ResolveResult, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.started {
		return nil, NewHTTPDNSError("client_stopped", "", ErrServiceUnavailable)
	}

	return c.resolver.ResolveBatch(ctx, domains, clientIP, opts...)
}

// ResolveAsync 异步解析域名
func (c *client) ResolveAsync(ctx context.Context, domain string, clientIP string, callback func(*ResolveResult, error), opts ...ResolveOption) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.started {
		callback(nil, NewHTTPDNSError("client_stopped", domain, ErrServiceUnavailable))
		return
	}

	c.resolver.ResolveAsync(ctx, domain, clientIP, callback, opts...)
}

// Close 关闭客户端
func (c *client) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.started {
		return nil
	}

	c.started = false
	close(c.stopCh)
	c.wg.Wait()

	return nil
}

// GetMetrics 获取指标统计
func (c *client) GetMetrics() MetricsStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.started {
		return MetricsStats{}
	}

	return c.resolver.GetMetrics()
}

// ResetMetrics 重置指标统计
func (c *client) ResetMetrics() {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.started {
		c.resolver.ResetMetrics()
	}
}

// UpdateServiceIPs 手动更新服务IP
func (c *client) UpdateServiceIPs(ctx context.Context) error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.started {
		return NewHTTPDNSError("client_stopped", "", ErrServiceUnavailable)
	}

	return c.resolver.httpClient.FetchServiceIPs(ctx)
}

// GetServiceIPs 获取当前服务IP列表
func (c *client) GetServiceIPs() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.started {
		return nil
	}

	return c.resolver.httpClient.serviceIPManager.GetServiceIPs()
}

// IsHealthy 检查客户端健康状态
func (c *client) IsHealthy() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.started
}
