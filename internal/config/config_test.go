package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_DefaultValues(t *testing.T) {
	// 清理环境变量确保测试不互相影响
	defer os.Unsetenv("OPENLDAP_EXPORTER_WEB_LISTEN_ADDRESS")
	defer os.Unsetenv("OPENLDAP_EXPORTER_LDAP_SERVER")
	defer os.Unsetenv("OPENLDAP_EXPORTER_LDAP_TIMEOUT")

	// 设置必要的LDAP服务器配置来通过验证
	os.Setenv("OPENLDAP_EXPORTER_LDAP_SERVER", "ldap://localhost:389")
	defer os.Unsetenv("OPENLDAP_EXPORTER_LDAP_SERVER")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	// 测试默认值
	if cfg.Web.ListenAddress != ":9330" {
		t.Errorf("Expected default listen address :9330, got %s", cfg.Web.ListenAddress)
	}

	if cfg.Web.MetricsPath != "/metrics" {
		t.Errorf("Expected default metrics path /metrics, got %s", cfg.Web.MetricsPath)
	}

	if cfg.LDAP.Timeout != 10*time.Second {
		t.Errorf("Expected default timeout 10s, got %v", cfg.LDAP.Timeout)
	}

	if cfg.Log.Level != "info" {
		t.Errorf("Expected default log level info, got %s", cfg.Log.Level)
	}

	if cfg.Log.Format != "text" {
		t.Errorf("Expected default log format text, got %s", cfg.Log.Format)
	}
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	// 清理环境变量确保测试不互相影响
	defer os.Unsetenv("OPENLDAP_EXPORTER_WEB_LISTEN_ADDRESS")
	defer os.Unsetenv("OPENLDAP_EXPORTER_LDAP_SERVER")
	defer os.Unsetenv("OPENLDAP_EXPORTER_LDAP_TIMEOUT")
	defer os.Unsetenv("OPENLDAP_EXPORTER_LOG_LEVEL")

	// 设置环境变量
	os.Setenv("OPENLDAP_EXPORTER_WEB_LISTEN_ADDRESS", ":8080")
	os.Setenv("OPENLDAP_EXPORTER_LDAP_SERVER", "ldap.example.com:389")
	os.Setenv("OPENLDAP_EXPORTER_LDAP_TIMEOUT", "30s")
	os.Setenv("OPENLDAP_EXPORTER_LOG_LEVEL", "debug")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load config from environment: %v", err)
	}

	if cfg.Web.ListenAddress != ":8080" {
		t.Errorf("Expected listen address :8080 from env, got %s", cfg.Web.ListenAddress)
	}

	if cfg.LDAP.Server != "ldap.example.com:389" {
		t.Errorf("Expected LDAP server ldap.example.com:389 from env, got %s", cfg.LDAP.Server)
	}

	if cfg.LDAP.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s from env, got %v", cfg.LDAP.Timeout)
	}

	if cfg.Log.Level != "debug" {
		t.Errorf("Expected log level debug from env, got %s", cfg.Log.Level)
	}
}

func TestLoad_TLSConfig(t *testing.T) {
	// 设置必要的LDAP服务器配置来通过验证
	os.Setenv("OPENLDAP_EXPORTER_LDAP_SERVER", "ldap://localhost:389")
	defer os.Unsetenv("OPENLDAP_EXPORTER_LDAP_SERVER")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	// 默认情况下应该有TLS配置且InsecureSkipVerify应为false
	if cfg.LDAP.TLSConfig == nil {
		t.Error("Expected TLSConfig to be initialized")
	}

	if cfg.LDAP.TLSConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be false by default")
	}

	// 模拟设置InsecureSkipVerify为true的情况
	oldValue := os.Getenv("OPENLDAP_EXPORTER_LDAP_INSECURE_SKIP_VERIFY")
	os.Setenv("OPENLDAP_EXPORTER_LDAP_INSECURE_SKIP_VERIFY", "true")
	defer os.Setenv("OPENLDAP_EXPORTER_LDAP_INSECURE_SKIP_VERIFY", oldValue)

	cfgWithInsecure, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load config with insecure setting: %v", err)
	}
	
	if !cfgWithInsecure.LDAP.TLSConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true when set via environment")
	}
}

func TestConfig_Structure(t *testing.T) {
	cfg := NewConfig()

	// 确保所有嵌套结构都已正确初始化
	if cfg.CustomSearches == nil {
		t.Error("Expected CustomSearches to be initialized as empty slice, not nil")
	}
}

func TestConfig_Validation(t *testing.T) {
	// 测试缺少必需字段的情况
	cfg := &Config{}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation to fail when ldap.server is empty")
	}

	// 测试无效的日志级别
	cfg = &Config{
		LDAP: LDAPConfig{
			Server:  "ldap://localhost:389",
			Timeout: 10 * time.Second,
		},
		Log: LogConfig{
			Level: "invalid",
		},
		Web: WebConfig{
			ListenAddress: ":9330",
			MetricsPath:   "/metrics",
		},
	}
	err = cfg.Validate()
	if err == nil {
		t.Error("Expected validation to fail with invalid log level")
	}

	// 测试无效的超时值
	cfg = &Config{
		LDAP: LDAPConfig{
			Server:  "ldap://localhost:389",
			Timeout: -1 * time.Second,
		},
		Log: LogConfig{
			Level: "info",
		},
		Web: WebConfig{
			ListenAddress: ":9330",
			MetricsPath:   "/metrics",
		},
	}
	err = cfg.Validate()
	if err == nil {
		t.Error("Expected validation to fail with negative timeout")
	}

	// 测试有效的配置
	cfg = &Config{
		LDAP: LDAPConfig{
			Server:  "ldap://localhost:389",
			Timeout: 10 * time.Second,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		Web: WebConfig{
			ListenAddress: ":9330",
			MetricsPath:   "/metrics",
		},
		CustomSearches: []CustomSearch{
			{
				Name:   "test",
				BaseDN: "ou=people,dc=example,dc=com",
				Filter: "(objectClass=person)",
			},
		},
	}
	err = cfg.Validate()
	if err != nil {
		t.Errorf("Expected validation to pass for valid config, got: %v", err)
	}
}

func TestConfig_Priority(t *testing.T) {
	// 测试配置优先级: Flag > Env > File > Default
	// 由于Load函数不直接处理flag，我们测试Env > Default
	
	// 设置环境变量
	os.Setenv("OPENLDAP_EXPORTER_WEB_LISTEN_ADDRESS", ":8080")
	os.Setenv("OPENLDAP_EXPORTER_LDAP_SERVER", "ldap://localhost:389")
	defer os.Unsetenv("OPENLDAP_EXPORTER_WEB_LISTEN_ADDRESS")
	defer os.Unsetenv("OPENLDAP_EXPORTER_LDAP_SERVER")
	
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// 应该使用环境变量而不是默认值
	if cfg.Web.ListenAddress != ":8080" {
		t.Errorf("Expected :8080 from environment, got %s", cfg.Web.ListenAddress)
	}
}