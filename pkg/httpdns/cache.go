package httpdns

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// CacheEntry 缓存条目（内存和持久化共用）
type CacheEntry struct {
	IPv4      []string  `json:"ipv4"`       // IPv4地址列表
	IPv6      []string  `json:"ipv6"`       // IPv6地址列表
	TTL       int       `json:"ttl"`        // TTL（秒）
	QueryTime time.Time `json:"query_time"` // 查询时间
}

// normalizeDomain 规范化域名（去空格 + 转小写 + 去尾点）
func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	domain = strings.ToLower(domain)
	domain = strings.TrimRight(domain, ".")
	return domain
}

// IsExpired 判断内存缓存是否过期
// 过期判断公式：当前时间 > 查询时间 + TTL
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.QueryTime.Add(time.Duration(e.TTL) * time.Second))
}

// IsPersistExpired 判断持久化缓存是否过期
// 过期判断公式：当前时间 > 查询时间 + TTL + threshold
func (e *CacheEntry) IsPersistExpired(threshold time.Duration) bool {
	return time.Now().After(e.QueryTime.Add(time.Duration(e.TTL)*time.Second + threshold))
}

// ToResolveResult 转换为 ResolveResult
func (e *CacheEntry) ToResolveResult(domain string) *ResolveResult {
	result := &ResolveResult{
		Domain:    domain,
		TTL:       time.Duration(e.TTL) * time.Second,
		Timestamp: e.QueryTime,
		Source:    SourceHTTPDNS,
	}

	for _, ipStr := range e.IPv4 {
		if ip := net.ParseIP(ipStr); ip != nil {
			result.IPv4 = append(result.IPv4, ip)
		}
	}

	for _, ipStr := range e.IPv6 {
		if ip := net.ParseIP(ipStr); ip != nil {
			result.IPv6 = append(result.IPv6, ip)
		}
	}

	return result
}


// CacheManager 统一缓存管理器（内存 + 持久化）
type CacheManager struct {
	// 内存缓存
	cache      map[string]*CacheEntry
	cacheMutex sync.RWMutex

	// 配置
	enabled      bool          // 是否启用内存缓存
	allowExpired bool          // 是否允许使用过期缓存
	persistent   bool          // 是否启用持久化
	threshold    time.Duration // 持久化缓存过期阈值

	// 持久化
	cacheDir  string     // 缓存目录
	fileMutex sync.Mutex // 文件写入锁

	// 异步保存控制（防止 goroutine 堆积）
	saveMu      sync.Mutex
	saving      bool // 是否正在保存
	savePending bool // 是否有待处理的保存请求

	logger Logger
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(config *Config) *CacheManager {
	cm := &CacheManager{
		cache:        make(map[string]*CacheEntry),
		enabled:      config.EnableMemoryCache,
		allowExpired: config.AllowExpiredCache,
		persistent:   config.EnablePersistentCache,
		threshold:    config.CacheExpireThreshold,
		logger:       config.Logger,
	}

	// 初始化持久化缓存目录
	if cm.persistent {
		cacheDir, err := getCacheDir(config.AccountID)
		if err != nil {
			if cm.logger != nil {
				cm.logger.Printf("Failed to get cache directory: %v, persistent cache disabled", err)
			}
			cm.persistent = false
		} else {
			cm.cacheDir = cacheDir
			if err := ensureCacheDir(cacheDir); err != nil {
				if cm.logger != nil {
					cm.logger.Printf("Failed to create cache directory: %v, persistent cache disabled", err)
				}
				cm.persistent = false
			}
		}
	}

	return cm
}

// Get 从内存缓存获取条目
// 返回值：entry（缓存条目）, hit（是否命中）, needAsyncUpdate（是否需要异步更新）
func (c *CacheManager) Get(domain string) (*CacheEntry, bool, bool) {
	if !c.enabled {
		return nil, false, false
	}

	domain = normalizeDomain(domain)

	c.cacheMutex.RLock()
	entry, exists := c.cache[domain]
	c.cacheMutex.RUnlock()

	if !exists {
		return nil, false, false
	}

	if entry.IsExpired() {
		if c.allowExpired {
			// 返回过期缓存，标记需要异步更新
			return entry, true, true
		}
		// 缓存过期且不允许使用过期缓存
		return nil, false, false
	}

	// 缓存命中且未过期
	return entry, true, false
}

// Set 设置内存缓存条目
func (c *CacheManager) Set(domain string, entry *CacheEntry) {
	if !c.enabled {
		return
	}

	// 校验 TTL，如果 <= 0 则设置为 60 秒
	if entry.TTL <= 0 {
		if c.logger != nil {
			c.logger.Printf("Invalid TTL %d for domain %s, using default 60s", entry.TTL, domain)
		}
		entry.TTL = 60
	}

	domain = normalizeDomain(domain)

	c.cacheMutex.Lock()
	c.cache[domain] = entry
	c.cacheMutex.Unlock()
}


// LoadFromDisk 从磁盘加载解析缓存到内存
func (c *CacheManager) LoadFromDisk() error {
	if !c.persistent || c.cacheDir == "" {
		return nil
	}

	filePath := filepath.Join(c.cacheDir, "resolve_cache.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在不是错误
		}
		return err
	}

	var cacheData ResolveCacheData
	if err := json.Unmarshal(data, &cacheData); err != nil {
		if c.logger != nil {
			c.logger.Printf("Failed to parse resolve cache file: %v", err)
		}
		return nil // 解析失败返回空缓存
	}

	// 过滤过期记录并加载到内存
	c.cacheMutex.Lock()
	validCount := 0
	expiredCount := 0
	for domain, entry := range cacheData.Records {
		if !entry.IsPersistExpired(c.threshold) {
			c.cache[domain] = entry
			validCount++
		} else {
			expiredCount++
		}
	}
	c.cacheMutex.Unlock()

	// 如果有过期记录，触发异步保存以删除磁盘上的过期记录
	if expiredCount > 0 {
		if c.logger != nil {
			c.logger.Printf("Loaded %d valid records, found %d expired records, scheduling rewrite", validCount, expiredCount)
		}
		c.SaveResolveCacheAsync()
	}

	return nil
}

// SaveResolveCacheAsync 异步保存解析缓存到磁盘（防止 goroutine 堆积）
func (c *CacheManager) SaveResolveCacheAsync() {
	if !c.persistent || c.cacheDir == "" {
		return
	}

	c.saveMu.Lock()
	if c.saving {
		// 已有保存在进行中，合并请求
		c.savePending = true
		c.saveMu.Unlock()
		return
	}
	c.saving = true
	c.saveMu.Unlock()

	go func() {
		for {
			c.doSaveResolveCache()

			c.saveMu.Lock()
			if c.savePending {
				// 有新的保存请求，再保存一次
				c.savePending = false
				c.saveMu.Unlock()
				continue
			}
			c.saving = false
			c.saveMu.Unlock()
			return
		}
	}()
}

// doSaveResolveCache 实际执行保存解析缓存的逻辑
func (c *CacheManager) doSaveResolveCache() {
	// 复制当前缓存
	c.cacheMutex.RLock()
	cacheCopy := make(map[string]*CacheEntry, len(c.cache))
	for k, v := range c.cache {
		cacheCopy[k] = v
	}
	c.cacheMutex.RUnlock()

	c.fileMutex.Lock()
	defer c.fileMutex.Unlock()

	cacheData := ResolveCacheData{Records: cacheCopy}
	if err := c.writeJSONFile("resolve_cache.json", cacheData); err != nil {
		if c.logger != nil {
			c.logger.Printf("Failed to save resolve cache: %v", err)
		}
	}
}


// LoadServiceIPs 从磁盘加载服务IP缓存
// 返回值：IPs列表, 更新时间, 错误
func (c *CacheManager) LoadServiceIPs() ([]string, time.Time, error) {
	if !c.persistent || c.cacheDir == "" {
		return nil, time.Time{}, nil
	}

	filePath := filepath.Join(c.cacheDir, "service_ips.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, err
	}

	var ipData ServiceIPCacheData
	if err := json.Unmarshal(data, &ipData); err != nil {
		if c.logger != nil {
			c.logger.Printf("Failed to parse service IP cache file: %v", err)
		}
		return nil, time.Time{}, nil
	}

	// 检查是否过期（24小时）
	if time.Since(ipData.UpdatedAt) > 24*time.Hour {
		return nil, time.Time{}, nil // 已过期
	}

	return ipData.IPs, ipData.UpdatedAt, nil
}

// SaveServiceIPsAsync 异步保存服务IP到磁盘
func (c *CacheManager) SaveServiceIPsAsync(ips []string) {
	if !c.persistent || c.cacheDir == "" {
		return
	}

	go func() {
		c.fileMutex.Lock()
		defer c.fileMutex.Unlock()

		ipData := ServiceIPCacheData{
			IPs:       ips,
			UpdatedAt: time.Now(),
		}
		if err := c.writeJSONFile("service_ips.json", ipData); err != nil {
			if c.logger != nil {
				c.logger.Printf("Failed to save service IPs: %v", err)
			}
		}
	}()
}


// writeJSONFile 原子性写入JSON文件
func (c *CacheManager) writeJSONFile(filename string, data interface{}) error {
	filePath := filepath.Join(c.cacheDir, filename)

	// 序列化为紧凑JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Windows：直接覆盖写入
	if runtime.GOOS == "windows" {
		return os.WriteFile(filePath, jsonData, 0600)
	}

	// 非 Windows：使用临时文件 + 原子重命名
	tempPath := filePath + ".tmp"

	// 写入临时文件
	if err := os.WriteFile(tempPath, jsonData, 0600); err != nil {
		return err
	}

	// 原子性重命名
	return os.Rename(tempPath, filePath)
}

// ServiceIPCacheData 服务IP缓存数据
type ServiceIPCacheData struct {
	IPs       []string  `json:"ips"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ResolveCacheData 解析结果缓存数据
type ResolveCacheData struct {
	Records map[string]*CacheEntry `json:"records"`
}

// getCacheDir 获取平台特定的缓存目录
func getCacheDir(accountID string) (string, error) {
	baseDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user cache dir: %w", err)
	}

	return filepath.Join(baseDir, "alicloud_httpdns", accountID), nil
}

// ensureCacheDir 确保缓存目录存在
func ensureCacheDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("cache directory path is empty")
	}
	return os.MkdirAll(dir, 0755)
}
