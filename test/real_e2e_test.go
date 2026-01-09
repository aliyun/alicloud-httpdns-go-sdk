package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

// TestRealEndToEndBatchResolve 真实环境端到端测试
func TestRealEndToEndBatchResolve(t *testing.T) {
	// 检查是否设置了环境变量
	accountID := os.Getenv("HTTPDNS_ACCOUNT_ID")
	secretKey := os.Getenv("HTTPDNS_SECRET_KEY")

	if accountID == "" || secretKey == "" {
		t.Skip("跳过真实环境测试：需要设置 HTTPDNS_ACCOUNT_ID 和 HTTPDNS_SECRET_KEY 环境变量")
	}

	// 创建客户端配置
	config := httpdns.DefaultConfig()
	config.AccountID = accountID
	config.SecretKey = secretKey
	config.EnableMetrics = true
	config.Timeout = 10 * time.Second

	// 创建客户端
	client, err := httpdns.NewClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 测试域名（根据规则文档，这些域名可以成功解析）
	domains := []string{"www.aliyun.com", "www.alibaba.com"}

	t.Logf("开始真实环境批量解析测试，域名: %v", domains)

	// 执行批量解析
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	results, err := client.ResolveBatch(ctx, domains, httpdns.WithClientIP("192.168.1.1"))
	if err != nil {
		t.Fatalf("批量解析失败: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("批量解析返回空结果")
	}

	// 验证结果
	domainResults := make(map[string]*httpdns.ResolveResult)
	for _, result := range results {
		domainResults[result.Domain] = result
		t.Logf("域名 %s 解析结果:", result.Domain)
		t.Logf("  IPv4地址数量: %d", len(result.IPv4))
		for i, ip := range result.IPv4 {
			t.Logf("    IPv4[%d]: %s", i, ip.String())
		}
		t.Logf("  IPv6地址数量: %d", len(result.IPv6))
		for i, ip := range result.IPv6 {
			t.Logf("    IPv6[%d]: %s", i, ip.String())
		}
		t.Logf("  TTL: %v", result.TTL)
		t.Logf("  来源: %s", result.Source.String())
	}

	// 验证每个域名都有结果
	for _, domain := range domains {
		result, exists := domainResults[domain]
		if !exists {
			t.Errorf("域名 %s 没有解析结果", domain)
			continue
		}

		// 验证至少有一个IP地址（IPv4或IPv6）
		if len(result.IPv4) == 0 && len(result.IPv6) == 0 {
			t.Errorf("域名 %s 没有返回任何IP地址", domain)
		}

		// 验证TTL
		if result.TTL <= 0 {
			t.Errorf("域名 %s 的TTL无效: %v", domain, result.TTL)
		}

		// 验证时间戳
		if result.Timestamp.IsZero() {
			t.Errorf("域名 %s 的时间戳无效", domain)
		}
	}

	// 验证指标
	metrics := client.GetMetrics()
	t.Logf("解析指标:")
	t.Logf("  总解析次数: %d", metrics.TotalResolves)
	t.Logf("  成功次数: %d", metrics.SuccessResolves)
	t.Logf("  成功率: %.2f%%", metrics.SuccessRate*100)

	if metrics.TotalResolves == 0 {
		t.Error("指标统计异常: 总解析次数为0")
	}

	if metrics.SuccessRate < 1.0 {
		t.Errorf("解析成功率过低: %.2f%%", metrics.SuccessRate*100)
	}

	t.Log("✅ 真实环境批量解析测试通过")
}

// TestRealEndToEndSingleResolve 真实环境单个解析测试
func TestRealEndToEndSingleResolve(t *testing.T) {
	// 检查是否设置了环境变量
	accountID := os.Getenv("HTTPDNS_ACCOUNT_ID")
	secretKey := os.Getenv("HTTPDNS_SECRET_KEY")

	if accountID == "" || secretKey == "" {
		t.Skip("跳过真实环境测试：需要设置 HTTPDNS_ACCOUNT_ID 和 HTTPDNS_SECRET_KEY 环境变量")
	}

	// 创建客户端配置
	config := httpdns.DefaultConfig()
	config.AccountID = accountID
	config.SecretKey = secretKey
	config.EnableMetrics = true
	config.Timeout = 10 * time.Second

	// 创建客户端
	client, err := httpdns.NewClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 测试域名
	domain := "www.aliyun.com"

	t.Logf("开始真实环境单个解析测试，域名: %s", domain)

	// 执行单个解析
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := client.Resolve(ctx, domain, httpdns.WithClientIP("192.168.1.1"))
	if err != nil {
		t.Fatalf("单个解析失败: %v", err)
	}

	t.Logf("域名 %s 解析结果:", result.Domain)
	t.Logf("  IPv4地址数量: %d", len(result.IPv4))
	for i, ip := range result.IPv4 {
		t.Logf("    IPv4[%d]: %s", i, ip.String())
	}
	t.Logf("  IPv6地址数量: %d", len(result.IPv6))
	for i, ip := range result.IPv6 {
		t.Logf("    IPv6[%d]: %s", i, ip.String())
	}
	t.Logf("  TTL: %v", result.TTL)
	t.Logf("  来源: %s", result.Source.String())

	// 验证结果
	if len(result.IPv4) == 0 && len(result.IPv6) == 0 {
		t.Error("没有返回任何IP地址")
	}

	if result.TTL <= 0 {
		t.Errorf("TTL无效: %v", result.TTL)
	}

	if result.Timestamp.IsZero() {
		t.Error("时间戳无效")
	}

	// 验证指标
	metrics := client.GetMetrics()
	t.Logf("解析指标:")
	t.Logf("  总解析次数: %d", metrics.TotalResolves)
	t.Logf("  成功次数: %d", metrics.SuccessResolves)
	t.Logf("  成功率: %.2f%%", metrics.SuccessRate*100)

	if metrics.TotalResolves == 0 {
		t.Error("指标统计异常: 总解析次数为0")
	}

	if metrics.SuccessRate < 1.0 {
		t.Errorf("解析成功率过低: %.2f%%", metrics.SuccessRate*100)
	}

	t.Log("✅ 真实环境单个解析测试通过")
}

// TestRealEndToEndBehaviorValidation 真实环境行为验证测试
func TestRealEndToEndBehaviorValidation(t *testing.T) {
	// 检查是否设置了环境变量
	accountID := os.Getenv("HTTPDNS_ACCOUNT_ID")
	secretKey := os.Getenv("HTTPDNS_SECRET_KEY")

	if accountID == "" || secretKey == "" {
		t.Skip("跳过真实环境测试：需要设置 HTTPDNS_ACCOUNT_ID 和 HTTPDNS_SECRET_KEY 环境变量")
	}

	// 创建客户端配置
	config := httpdns.DefaultConfig()
	config.AccountID = accountID
	config.SecretKey = secretKey
	config.EnableMetrics = true
	config.Timeout = 10 * time.Second

	// 创建客户端
	client, err := httpdns.NewClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 测试各种域名的解析行为
	testCases := []struct {
		name    string
		domains []string
		desc    string
	}{
		{
			name:    "推荐域名",
			domains: []string{"www.aliyun.com", "www.alibaba.com"},
			desc:    "文档推荐的可解析域名",
		},
		{
			name:    "其他域名",
			domains: []string{"www.baidu.com", "www.qq.com"},
			desc:    "文档标注的不可解析域名（实际可能可以解析）",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("测试 %s: %v", tc.desc, tc.domains)

			// 执行批量解析
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			results, err := client.ResolveBatch(ctx, tc.domains, httpdns.WithClientIP("192.168.1.1"))
			
			if err != nil {
				t.Logf("解析出错: %v", err)
				return
			}

			t.Logf("返回了 %d 个结果", len(results))
			for _, result := range results {
				t.Logf("域名 %s:", result.Domain)
				t.Logf("  IPv4地址数量: %d", len(result.IPv4))
				t.Logf("  IPv6地址数量: %d", len(result.IPv6))
				t.Logf("  TTL: %v", result.TTL)
				
				// 验证基本要求
				if len(result.IPv4) == 0 && len(result.IPv6) == 0 {
					t.Logf("  警告: 域名 %s 没有返回任何IP地址", result.Domain)
				}
				
				if result.TTL <= 0 {
					t.Errorf("  错误: 域名 %s 的TTL无效: %v", result.Domain, result.TTL)
				}
			}
		})
	}

	// 验证指标记录了这些请求
	metrics := client.GetMetrics()
	t.Logf("总体解析指标:")
	t.Logf("  总解析次数: %d", metrics.TotalResolves)
	t.Logf("  成功次数: %d", metrics.SuccessResolves)
	t.Logf("  成功率: %.2f%%", metrics.SuccessRate*100)

	t.Log("✅ 真实环境行为验证测试完成")
}

// TestRealEndToEndIPv6Support 真实环境IPv6支持测试
func TestRealEndToEndIPv6Support(t *testing.T) {
	// 检查是否设置了环境变量
	accountID := os.Getenv("HTTPDNS_ACCOUNT_ID")
	secretKey := os.Getenv("HTTPDNS_SECRET_KEY")

	if accountID == "" || secretKey == "" {
		t.Skip("跳过真实环境测试：需要设置 HTTPDNS_ACCOUNT_ID 和 HTTPDNS_SECRET_KEY 环境变量")
	}

	// 创建客户端配置
	config := httpdns.DefaultConfig()
	config.AccountID = accountID
	config.SecretKey = secretKey
	config.EnableMetrics = true
	config.Timeout = 10 * time.Second

	// 创建客户端
	client, err := httpdns.NewClient(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	domain := "www.aliyun.com"

	t.Logf("测试IPv6支持，域名: %s", domain)

	// 测试不同的查询选项
	testCases := []struct {
		name string
		opts []httpdns.ResolveOption
		desc string
	}{
		{
			name: "默认查询",
			opts: []httpdns.ResolveOption{httpdns.WithClientIP("192.168.1.1")},
			desc: "默认查询（IPv4和IPv6）",
		},
		{
			name: "仅IPv4",
			opts: []httpdns.ResolveOption{httpdns.WithClientIP("192.168.1.1"), httpdns.WithIPv4Only()},
			desc: "仅查询IPv4地址",
		},
		{
			name: "仅IPv6",
			opts: []httpdns.ResolveOption{httpdns.WithClientIP("192.168.1.1"), httpdns.WithIPv6Only()},
			desc: "仅查询IPv6地址",
		},
		{
			name: "IPv4和IPv6",
			opts: []httpdns.ResolveOption{httpdns.WithClientIP("192.168.1.1"), httpdns.WithBothIP()},
			desc: "明确查询IPv4和IPv6",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("执行 %s", tc.desc)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			result, err := client.Resolve(ctx, domain, tc.opts...)
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}

			t.Logf("结果:")
			t.Logf("  IPv4地址数量: %d", len(result.IPv4))
			for i, ip := range result.IPv4 {
				t.Logf("    IPv4[%d]: %s", i, ip.String())
			}
			t.Logf("  IPv6地址数量: %d", len(result.IPv6))
			for i, ip := range result.IPv6 {
				t.Logf("    IPv6[%d]: %s", i, ip.String())
			}

			// 根据查询选项验证结果
			switch tc.name {
			case "仅IPv4":
				if len(result.IPv6) > 0 {
					t.Errorf("仅IPv4查询不应返回IPv6地址，但返回了%d个", len(result.IPv6))
				}
				if len(result.IPv4) == 0 {
					t.Error("仅IPv4查询应该返回IPv4地址")
				}
			case "仅IPv6":
				if len(result.IPv4) > 0 {
					t.Errorf("仅IPv6查询不应返回IPv4地址，但返回了%d个", len(result.IPv4))
				}
				// IPv6可能不总是可用，所以不强制要求
			}
		})
	}

	t.Log("✅ 真实环境IPv6支持测试完成")
}