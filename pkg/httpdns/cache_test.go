package httpdns

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCacheEntry_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		ttl       int
		queryTime time.Time
		expected  bool
	}{
		{
			name:      "not expired",
			ttl:       60,
			queryTime: time.Now(),
			expected:  false,
		},
		{
			name:      "expired",
			ttl:       1,
			queryTime: time.Now().Add(-2 * time.Second),
			expected:  true,
		},
		{
			name:      "just expired",
			ttl:       0,
			queryTime: time.Now().Add(-1 * time.Second),
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &CacheEntry{
				TTL:       tt.ttl,
				QueryTime: tt.queryTime,
			}
			if got := entry.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCacheEntry_IsPersistExpired(t *testing.T) {
	tests := []struct {
		name      string
		ttl       int
		queryTime time.Time
		threshold time.Duration
		expected  bool
	}{
		{
			name:      "not expired with threshold",
			ttl:       60,
			queryTime: time.Now(),
			threshold: 30 * time.Second,
			expected:  false,
		},
		{
			name:      "expired even with threshold",
			ttl:       1,
			queryTime: time.Now().Add(-5 * time.Second),
			threshold: 2 * time.Second,
			expected:  true,
		},
		{
			name:      "saved by threshold",
			ttl:       1,
			queryTime: time.Now().Add(-2 * time.Second),
			threshold: 5 * time.Second,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &CacheEntry{
				TTL:       tt.ttl,
				QueryTime: tt.queryTime,
			}
			if got := entry.IsPersistExpired(tt.threshold); got != tt.expected {
				t.Errorf("IsPersistExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCacheEntry_ToResolveResult(t *testing.T) {
	entry := &CacheEntry{
		IPv4:      []string{"1.2.3.4", "5.6.7.8"},
		IPv6:      []string{"2001:db8::1"},
		TTL:       300,
		QueryTime: time.Now(),
	}

	result := entry.ToResolveResult("example.com")

	if result.Domain != "example.com" {
		t.Errorf("Domain = %v, want example.com", result.Domain)
	}

	if len(result.IPv4) != 2 {
		t.Errorf("IPv4 count = %d, want 2", len(result.IPv4))
	}

	if len(result.IPv6) != 1 {
		t.Errorf("IPv6 count = %d, want 1", len(result.IPv6))
	}

	if result.TTL != 300*time.Second {
		t.Errorf("TTL = %v, want %v", result.TTL, 300*time.Second)
	}
}


func TestCacheManager_GetSet(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"
	config.EnableMemoryCache = true

	cm := NewCacheManager(config)

	entry := &CacheEntry{
		IPv4:      []string{"1.2.3.4"},
		TTL:       60,
		QueryTime: time.Now(),
	}

	// 测试 Set
	cm.Set("example.com", entry)

	// 测试 Get
	got, hit, needAsync := cm.Get("example.com")
	if !hit {
		t.Error("Get() should hit")
	}
	if needAsync {
		t.Error("Get() should not need async update")
	}
	if got == nil {
		t.Fatal("Get() returned nil")
	}
	if len(got.IPv4) != 1 {
		t.Errorf("Get() IPv4 count = %d, want 1", len(got.IPv4))
	}
}

func TestCacheManager_GetExpired(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"
	config.EnableMemoryCache = true
	config.AllowExpiredCache = false

	cm := NewCacheManager(config)

	// 设置已过期的缓存
	entry := &CacheEntry{
		IPv4:      []string{"1.2.3.4"},
		TTL:       1,
		QueryTime: time.Now().Add(-2 * time.Second),
	}
	cm.Set("example.com", entry)

	// 不允许使用过期缓存时应该返回未命中
	got, hit, _ := cm.Get("example.com")
	if hit {
		t.Error("Get() should not hit for expired cache when AllowExpiredCache=false")
	}
	if got != nil {
		t.Error("Get() should return nil for expired cache")
	}
}

func TestCacheManager_GetExpiredAllowed(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"
	config.EnableMemoryCache = true
	config.AllowExpiredCache = true

	cm := NewCacheManager(config)

	// 设置已过期的缓存
	entry := &CacheEntry{
		IPv4:      []string{"1.2.3.4"},
		TTL:       1,
		QueryTime: time.Now().Add(-2 * time.Second),
	}
	cm.Set("example.com", entry)

	// 允许使用过期缓存时应该返回缓存并标记需要异步更新
	got, hit, needAsync := cm.Get("example.com")
	if !hit {
		t.Error("Get() should hit for expired cache when AllowExpiredCache=true")
	}
	if !needAsync {
		t.Error("Get() should need async update for expired cache")
	}
	if got == nil {
		t.Fatal("Get() should return entry for expired cache")
	}
}

func TestCacheManager_Disabled(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"
	config.EnableMemoryCache = false

	cm := NewCacheManager(config)

	entry := &CacheEntry{
		IPv4:      []string{"1.2.3.4"},
		TTL:       60,
		QueryTime: time.Now(),
	}

	// Set 应该不生效
	cm.Set("example.com", entry)

	// Get 应该返回未命中
	got, hit, _ := cm.Get("example.com")
	if hit {
		t.Error("Get() should not hit when cache is disabled")
	}
	if got != nil {
		t.Error("Get() should return nil when cache is disabled")
	}
}

func TestGetCacheDir(t *testing.T) {
	testAccountID := "test_account_123"
	dir, err := getCacheDir(testAccountID)
	if err != nil {
		t.Fatalf("getCacheDir() error = %v", err)
	}
	if dir == "" {
		t.Error("getCacheDir() returned empty string")
	}
	// 验证路径包含 accountID
	if !strings.Contains(dir, testAccountID) {
		t.Errorf("getCacheDir() = %s, should contain accountID %s", dir, testAccountID)
	}
	t.Logf("Cache directory: %s", dir)
}

func TestCacheManager_Persistence(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "httpdns_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultConfig()
	config.AccountID = "test123"
	config.EnableMemoryCache = true
	config.EnablePersistentCache = true

	cm := &CacheManager{
		cache:      make(map[string]*CacheEntry),
		enabled:    true,
		persistent: true,
		cacheDir:   tempDir,
		threshold:  0,
	}

	// 设置缓存
	entry := &CacheEntry{
		IPv4:      []string{"1.2.3.4"},
		TTL:       60,
		QueryTime: time.Now(),
	}
	cm.Set("example.com", entry)

	// 同步保存（测试用）
	cm.fileMutex.Lock()
	cacheData := ResolveCacheData{Records: cm.cache}
	err = cm.writeJSONFile("resolve_cache.json", cacheData)
	cm.fileMutex.Unlock()
	if err != nil {
		t.Fatalf("writeJSONFile() error = %v", err)
	}

	// 验证文件存在
	filePath := filepath.Join(tempDir, "resolve_cache.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Cache file should exist")
	}

	// 创建新的 CacheManager 并加载
	cm2 := &CacheManager{
		cache:      make(map[string]*CacheEntry),
		enabled:    true,
		persistent: true,
		cacheDir:   tempDir,
		threshold:  0,
	}

	if err := cm2.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk() error = %v", err)
	}

	// 验证加载的数据
	got, hit, _ := cm2.Get("example.com")
	if !hit {
		t.Error("Get() should hit after loading from disk")
	}
	if got == nil {
		t.Fatal("Get() returned nil after loading from disk")
	}
	if len(got.IPv4) != 1 || got.IPv4[0] != "1.2.3.4" {
		t.Errorf("Loaded data mismatch: got %v", got.IPv4)
	}
}

// TestCacheManager_TTLValidation 测试 TTL <= 0 时设置为 60 秒
func TestCacheManager_TTLValidation(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"
	config.EnableMemoryCache = true

	cm := NewCacheManager(config)

	tests := []struct {
		name        string
		inputTTL    int
		expectedTTL int
	}{
		{
			name:        "zero TTL",
			inputTTL:    0,
			expectedTTL: 60,
		},
		{
			name:        "negative TTL",
			inputTTL:    -10,
			expectedTTL: 60,
		},
		{
			name:        "valid TTL",
			inputTTL:    300,
			expectedTTL: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &CacheEntry{
				IPv4:      []string{"1.2.3.4"},
				TTL:       tt.inputTTL,
				QueryTime: time.Now(),
			}

			cm.Set("example.com", entry)

			got, hit, _ := cm.Get("example.com")
			if !hit {
				t.Fatal("Get() should hit")
			}
			if got.TTL != tt.expectedTTL {
				t.Errorf("TTL = %d, want %d", got.TTL, tt.expectedTTL)
			}
		})
	}
}

// TestCacheManager_DomainNormalization 测试域名规范化（大小写、空格、尾点）
func TestCacheManager_DomainNormalization(t *testing.T) {
	config := DefaultConfig()
	config.AccountID = "test123"
	config.EnableMemoryCache = true

	cm := NewCacheManager(config)

	entry := &CacheEntry{
		IPv4:      []string{"1.2.3.4"},
		TTL:       60,
		QueryTime: time.Now(),
	}

	// 设置规范化的域名
	cm.Set("example.com", entry)

	tests := []struct {
		name   string
		domain string
		hit    bool
	}{
		{
			name:   "exact match",
			domain: "example.com",
			hit:    true,
		},
		{
			name:   "uppercase",
			domain: "EXAMPLE.COM",
			hit:    true,
		},
		{
			name:   "mixed case",
			domain: "Example.Com",
			hit:    true,
		},
		{
			name:   "with leading space",
			domain: " example.com",
			hit:    true,
		},
		{
			name:   "with trailing space",
			domain: "example.com ",
			hit:    true,
		},
		{
			name:   "with trailing dot",
			domain: "example.com.",
			hit:    true,
		},
		{
			name:   "with multiple trailing dots",
			domain: "example.com...",
			hit:    true,
		},
		{
			name:   "all combined",
			domain: "  EXAMPLE.COM.  ",
			hit:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, hit, _ := cm.Get(tt.domain)
			if hit != tt.hit {
				t.Errorf("Get(%q) hit = %v, want %v", tt.domain, hit, tt.hit)
			}
			if hit && got == nil {
				t.Error("Get() should return entry when hit")
			}
		})
	}
}

// TestCacheManager_LoadFromDisk_ExpiredRecords 测试启动时过期记录处理
func TestCacheManager_LoadFromDisk_ExpiredRecords(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "httpdns_expired_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultConfig()
	config.AccountID = "test123"
	config.EnableMemoryCache = true
	config.EnablePersistentCache = true

	// 创建包含过期和有效记录的缓存文件
	cacheData := ResolveCacheData{
		Records: map[string]*CacheEntry{
			"valid.com": {
				IPv4:      []string{"1.2.3.4"},
				TTL:       300,
				QueryTime: time.Now(),
			},
			"expired.com": {
				IPv4:      []string{"5.6.7.8"},
				TTL:       1,
				QueryTime: time.Now().Add(-10 * time.Second),
			},
		},
	}

	// 写入缓存文件
	cacheFile := filepath.Join(tempDir, "resolve_cache.json")
	data, err := json.Marshal(cacheData)
	if err != nil {
		t.Fatalf("Failed to marshal cache data: %v", err)
	}
	if err := os.WriteFile(cacheFile, data, 0600); err != nil {
		t.Fatalf("Failed to write cache file: %v", err)
	}

	// 创建 CacheManager 并加载
	cm := &CacheManager{
		cache:      make(map[string]*CacheEntry),
		enabled:    true,
		persistent: true,
		cacheDir:   tempDir,
		threshold:  0,
	}

	if err := cm.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk() error = %v", err)
	}

	// 验证只加载了有效记录
	validEntry, hit, _ := cm.Get("valid.com")
	if !hit {
		t.Error("valid.com should be loaded")
	}
	if validEntry == nil {
		t.Fatal("valid.com entry should not be nil")
	}

	// 验证过期记录未加载
	expiredEntry, hit, _ := cm.Get("expired.com")
	if hit {
		t.Error("expired.com should not be loaded")
	}
	if expiredEntry != nil {
		t.Error("expired.com entry should be nil")
	}

	// 等待异步保存完成
	time.Sleep(200 * time.Millisecond)

	// 验证磁盘文件已更新（过期记录被删除）
	data, err = os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}

	var reloadedData ResolveCacheData
	if err := json.Unmarshal(data, &reloadedData); err != nil {
		t.Fatalf("Failed to unmarshal cache data: %v", err)
	}

	if _, exists := reloadedData.Records["expired.com"]; exists {
		t.Error("expired.com should be removed from disk cache")
	}
	if _, exists := reloadedData.Records["valid.com"]; !exists {
		t.Error("valid.com should remain in disk cache")
	}
}
