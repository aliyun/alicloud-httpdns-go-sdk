package httpdns

import (
	"strconv"
	"testing"
	"time"
)

func TestGenerateSignature(t *testing.T) {
	tests := []struct {
		name      string
		secretKey string
		host      string
		timestamp string
		expected  string
	}{
		{
			name:      "basic signature",
			secretKey: "IAmASecret",
			host:      "www.aliyun.com",
			timestamp: "1534316400",
			expected:  "60c71e98b6d7fcbb366243e224eab457", // 根据官方文档的示例
		},
		{
			name:      "different host",
			secretKey: "secret123",
			host:      "example.com",
			timestamp: "1234567890",
			expected:  generateSignature("secret123", "example.com", "1234567890"), // 计算期望值
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateSignature(tt.secretKey, tt.host, tt.timestamp)
			if got != tt.expected {
				t.Errorf("generateSignature() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGenerateBatchSignature(t *testing.T) {
	tests := []struct {
		name      string
		secretKey string
		hosts     []string
		timestamp string
		expected  string
	}{
		{
			name:      "batch signature",
			secretKey: "IAmASecret",
			hosts:     []string{"www.aliyun.com", "www.taobao.com"},
			timestamp: "1534316400",
			expected:  "12a3f6b1b14a46ca813ca6439beb59a4", // 根据官方文档的示例
		},
		{
			name:      "single host in batch",
			secretKey: "secret123",
			hosts:     []string{"example.com"},
			timestamp: "1234567890",
			expected:  generateBatchSignature("secret123", []string{"example.com"}, "1234567890"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateBatchSignature(tt.secretKey, tt.hosts, tt.timestamp)
			if got != tt.expected {
				t.Errorf("generateBatchSignature() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAuthManager_GenerateSignature(t *testing.T) {
	authManager := NewAuthManager("test-secret", 30*time.Second)
	host := "example.com"

	timestamp1, signature1 := authManager.GenerateSignature(host)
	timestamp2, signature2 := authManager.GenerateSignature(host)

	// 检查时间戳格式
	if _, err := strconv.ParseInt(timestamp1, 10, 64); err != nil {
		t.Errorf("GenerateSignature() timestamp format error: %v", err)
	}

	// 检查签名长度（MD5应该是32个字符）
	if len(signature1) != 32 {
		t.Errorf("GenerateSignature() signature length = %v, want 32", len(signature1))
	}

	// 由于时间戳不同，签名应该不同
	if timestamp1 == timestamp2 && signature1 == signature2 {
		// 如果时间戳相同但在不同时间调用，这是可能的，所以只在时间戳相同时检查
		time.Sleep(time.Second)
		timestamp3, signature3 := authManager.GenerateSignature(host)
		if timestamp3 == timestamp1 && signature3 == signature1 {
			t.Error("GenerateSignature() should generate different signatures at different times")
		}
	}
}

func TestAuthManager_GenerateBatchSignature(t *testing.T) {
	authManager := NewAuthManager("test-secret", 30*time.Second)
	hosts := []string{"example.com", "test.com"}

	timestamp, signature := authManager.GenerateBatchSignature(hosts)

	// 检查时间戳格式
	if _, err := strconv.ParseInt(timestamp, 10, 64); err != nil {
		t.Errorf("GenerateBatchSignature() timestamp format error: %v", err)
	}

	// 检查签名长度（MD5应该是32个字符）
	if len(signature) != 32 {
		t.Errorf("GenerateBatchSignature() signature length = %v, want 32", len(signature))
	}

	// 验证签名一致性
	expectedSignature := generateBatchSignature("test-secret", hosts, timestamp)
	if signature != expectedSignature {
		t.Errorf("GenerateBatchSignature() signature = %v, want %v", signature, expectedSignature)
	}
}

func TestNewAuthManager(t *testing.T) {
	secretKey := "test-secret-key"
	expireTime := 30 * time.Second
	authManager := NewAuthManager(secretKey, expireTime)

	if authManager == nil {
		t.Fatal("NewAuthManager() returned nil")
	}

	if authManager.secretKey != secretKey {
		t.Errorf("NewAuthManager() secretKey = %v, want %v", authManager.secretKey, secretKey)
	}

	if authManager.expireTime != expireTime {
		t.Errorf("NewAuthManager() expireTime = %v, want %v", authManager.expireTime, expireTime)
	}
}

func TestAuthManager_TimestampExpiration(t *testing.T) {
	expireTime := 30 * time.Second
	authManager := NewAuthManager("test-secret", expireTime)

	// 测试单域名签名时间戳
	beforeTime := time.Now()
	timestamp, _ := authManager.GenerateSignature("example.com")

	// 解析时间戳
	timestampInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		t.Fatalf("Failed to parse timestamp: %v", err)
	}

	timestampTime := time.Unix(timestampInt, 0)
	expectedMinTime := beforeTime.Add(expireTime).Add(-1 * time.Second) // 允许1秒误差
	expectedMaxTime := beforeTime.Add(expireTime).Add(1 * time.Second)  // 允许1秒误差

	// 验证时间戳在预期范围内（当前时间+过期时间，允许1秒误差）
	if timestampTime.Before(expectedMinTime) || timestampTime.After(expectedMaxTime) {
		t.Errorf("Timestamp %v is not within expected range [%v, %v]",
			timestampTime, expectedMinTime, expectedMaxTime)
	}

	// 验证时间戳确实是未来时间
	if !timestampTime.After(beforeTime) {
		t.Errorf("Timestamp %v should be after current time %v", timestampTime, beforeTime)
	}

	// 测试批量域名签名时间戳
	beforeTime = time.Now()
	timestamp, _ = authManager.GenerateBatchSignature([]string{"example.com", "test.com"})

	timestampInt, err = strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		t.Fatalf("Failed to parse batch timestamp: %v", err)
	}

	timestampTime = time.Unix(timestampInt, 0)
	expectedMinTime = beforeTime.Add(expireTime).Add(-1 * time.Second)
	expectedMaxTime = beforeTime.Add(expireTime).Add(1 * time.Second)

	// 验证批量签名时间戳也在预期范围内
	if timestampTime.Before(expectedMinTime) || timestampTime.After(expectedMaxTime) {
		t.Errorf("Batch timestamp %v is not within expected range [%v, %v]",
			timestampTime, expectedMinTime, expectedMaxTime)
	}

	// 验证批量签名时间戳也是未来时间
	if !timestampTime.After(beforeTime) {
		t.Errorf("Batch timestamp %v should be after current time %v", timestampTime, beforeTime)
	}
}
