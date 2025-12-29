package httpdns

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// 验证默认值
	if config.MaxRetries != 0 {
		t.Errorf("Expected MaxRetries to be 0, got %d", config.MaxRetries)
	}

	if config.Timeout != 5*time.Second {
		t.Errorf("Expected Timeout to be 5s, got %v", config.Timeout)
	}

	if config.EnableHTTPS {
		t.Error("Expected EnableHTTPS to be false by default")
	}

	if config.EnableMetrics {
		t.Error("Expected EnableMetrics to be false by default")
	}

	if len(config.BootstrapIPs) == 0 {
		t.Error("Expected BootstrapIPs to be populated by default")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				AccountID:  "123456",
				Timeout:    5 * time.Second,
				MaxRetries: 0,
			},
			expectError: false,
		},
		{
			name: "missing account ID",
			config: &Config{
				Timeout:    5 * time.Second,
				MaxRetries: 0,
			},
			expectError: true,
		},
		{
			name: "zero timeout gets default",
			config: &Config{
				AccountID:  "123456",
				Timeout:    0,
				MaxRetries: 0,
			},
			expectError: false,
		},
		{
			name: "negative retries gets zero",
			config: &Config{
				AccountID:  "123456",
				Timeout:    5 * time.Second,
				MaxRetries: -1,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// 验证默认值设置
			if !tt.expectError {
				if tt.config.Timeout <= 0 {
					t.Error("Timeout should be set to default value")
				}
				if tt.config.MaxRetries < 0 {
					t.Error("MaxRetries should not be negative after validation")
				}
				if len(tt.config.BootstrapIPs) == 0 {
					t.Error("BootstrapIPs should be set to default values")
				}
			}
		})
	}
}
