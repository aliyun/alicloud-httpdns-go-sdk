# HTTPDNS Go SDK 代码评审总结

## 🔴 P0 - 严重问题

### 1. 重试逻辑无法切换服务 IP
- **问题**：重试时使用固定 URL，多次重试都请求同一个已失败的 IP
- **影响**：重试机制形同虚设，无法实现故障转移
- **位置**：
  - `pkg/httpdns/network.go:234-265` - `DoRequestWithRetry()`
  - `pkg/httpdns/resolver.go:35-79` - `ResolveSingle()`
  - `pkg/httpdns/resolver.go:122-186` - `ResolveBatch()`

### 2. 批量解析完全无法工作
- **问题**：代码依赖 `type` 字段区分 IPv4/IPv6，但 API 不返回此字段，导致所有数据被丢弃
- **影响**：批量解析返回空结果（静默失败）
- **位置**：
  - `pkg/httpdns/resolver.go:217-233` - Type 字段判断逻辑
  - `pkg/httpdns/types.go:127` - `HTTPDNSResponse.Type` 字段

---

## 🟡 P1 - 中等问题

### 3. 服务 IP 管理器的并发安全问题
- **问题**：`GetAvailableIP()` 使用读锁但修改了 `currentIP` 字段
- **影响**：高并发场景下可能出现数据竞争
- **位置**：`internal/pool/service_ip.go:32-58`

---

## 🟢 P2 - 轻微问题

### 4. 存在未使用的代码
- **问题**：多处定义但未使用的代码
- **影响**：增加维护负担，不影响功能
- **位置**：
  - `pkg/httpdns/errors.go:11-12` - `ErrAuthFailed`, `ErrNetworkTimeout`
  - `pkg/httpdns/metrics.go` - `RecordAPIRequest()`, `APIRequests`, `APIErrors`, `APIResponseTime`, `CacheHits`
  - `pkg/httpdns/types.go:135-141` - `ServiceIPList` 结构体
  - `pkg/httpdns/types.go:48` - `ResolveResult.Error` 字段
  - `pkg/httpdns/types.go` - `HTTPDNSResponse.OriginTTL`, `ClientIP`, `Type` 字段
  - `pkg/httpdns/resolver.go:303-313` - `parseQueryType()` 函数
