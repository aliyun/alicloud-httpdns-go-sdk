# 更新日志

本文档记录了 HTTPDNS Go SDK 的所有重要变更。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
并且本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [1.0.0] - 2025-12-29

### 新增
- 🎉 初始版本发布
- ✅ 支持阿里云 EMAS HTTPDNS 服务接入
- ✅ 支持单域名和批量域名解析
- ✅ 支持鉴权和非鉴权解析模式
- ✅ 支持 IPv4 和 IPv6 双栈解析
- ✅ 支持客户端 IP 传递，实现智能调度
- ✅ 实现启动 IP 冗余和服务 IP 轮转机制
- ✅ 支持故障转移和 LocalDNS 降级
- ✅ 支持异步解析功能
- ✅ 实现智能IP选择功能
- ✅ 支持监控指标收集和统计
- ✅ 实现指数退避重试机制
- ✅ 支持连接池和 HTTP/2
- ✅ 完整的错误处理和分类
- ✅ 支持配置验证和默认值
- ✅ 线程安全的客户端实现
- ✅ 优雅的客户端生命周期管理
- ✅ 完整的单元测试覆盖
- ✅ 详细的 API 文档和使用示例

### 技术特性
- **简洁设计**: 优先使用 Go 标准库，避免过度抽象
- **高可用性**: 多层故障转移机制，确保服务可用性
- **性能优化**: 连接复用、请求去重、智能IP选择
- **安全性**: 支持 HTTPS、证书校验、鉴权签名
- **可观测性**: 详细的指标统计和结构化日志
- **易用性**: 简单的 API 设计和丰富的配置选项

### API 接口
- `NewClient(config *Config) (Client, error)` - 创建客户端
- `Resolve(ctx, domain, clientIP, ...opts) (*ResolveResult, error)` - 单域名解析
- `ResolveBatch(ctx, domains, clientIP, ...opts) ([]*ResolveResult, error)` - 批量解析
- `ResolveAsync(ctx, domain, clientIP, callback, ...opts)` - 异步解析
- `Close() error` - 关闭客户端
- `GetMetrics() MetricsStats` - 获取指标统计
- `ResetMetrics()` - 重置指标
- `UpdateServiceIPs(ctx) error` - 更新服务 IP
- `GetServiceIPs() []string` - 获取服务 IP 列表
- `IsHealthy() bool` - 健康检查

### 配置选项
- `AccountID` - 账户 ID（必需）
- `SecretKey` - 密钥（可选，用于鉴权）
- `BootstrapIPs` - 启动 IP 列表
- `Timeout` - 请求超时时间
- `MaxRetries` - 最大重试次数

- `EnableHTTPS` - 启用 HTTPS
- `EnableLocalDNS` - 启用 LocalDNS 降级
- `EnableMetrics` - 启用指标收集
- `Logger` - 日志记录器

### 解析选项
- `WithIPv4Only()` - 仅解析 IPv4
- `WithIPv6Only()` - 仅解析 IPv6
- `WithBothIP()` - 解析 IPv4 和 IPv6
- `WithTimeout(duration)` - 设置超时时间

### 错误类型
- `HTTPDNSError` - HTTPDNS 特定错误
- `ErrInvalidConfig` - 配置错误
- `ErrAuthFailed` - 认证失败
- `ErrNetworkTimeout` - 网络超时
- `ErrInvalidDomain` - 无效域名
- `ErrServiceUnavailable` - 服务不可用
- `ErrLocalDNSDisabled` - LocalDNS 降级被禁用

### 指标统计
- 总解析次数和成功率
- 解析延迟统计（平均/最小/最大）
- API 请求次数和错误率
- 降级解析次数
- 错误分类统计（网络/认证/验证错误）
- 缓存命中率（预留）

### 示例代码
- `examples/basic/` - 基础使用示例
- `examples/advanced/` - 高级功能示例
- `examples/server/` - HTTP 服务器示例

### 文档
- `README.md` - 完整的使用文档
- `CHANGELOG.md` - 更新日志
- 内联 API 文档和注释
- 故障排查指南
- 性能优化建议

### 测试
- 单元测试覆盖率 > 90%
- 集成测试覆盖主要功能
- 性能基准测试
- 并发安全测试
- 故障转移测试

---

## 版本说明

### 版本号规则
本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/) 规范：

- **主版本号**：当你做了不兼容的 API 修改
- **次版本号**：当你做了向下兼容的功能性新增
- **修订号**：当你做了向下兼容的问题修正

### 发布周期
- **主版本**：根据需要发布，包含重大功能更新或 API 变更
- **次版本**：每月发布，包含新功能和改进
- **修订版本**：根据需要发布，主要用于 bug 修复

### 支持政策
- **当前版本**：提供完整支持，包括新功能、改进和 bug 修复
- **前一个主版本**：提供 bug 修复和安全更新
- **更早版本**：仅提供关键安全更新

### 迁移指南
当发布包含破坏性变更的新版本时，我们会提供详细的迁移指南，帮助用户平滑升级。