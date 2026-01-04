package config

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds the configuration for the OpenLDAP Exporter
type Config struct {
	Web            WebConfig      `mapstructure:"web"`
	LDAP           LDAPConfig     `mapstructure:"ldap"`
	Log            LogConfig      `mapstructure:"log"`
	CustomSearches []CustomSearch `mapstructure:"custom_searches"`
}

func NewConfig() *Config {
	return &Config{
		CustomSearches: []CustomSearch{},
	}
}

// WebConfig holds the web server configuration
type WebConfig struct {
	ListenAddress string `mapstructure:"listen_address"`
	MetricsPath   string `mapstructure:"metrics_path"`
}

// LDAPConfig holds the LDAP connection configuration
type LDAPConfig struct {
	Server             string        `mapstructure:"server"`
	BindDN             string        `mapstructure:"bind_dn"`
	BindPassword       string        `mapstructure:"bind_password"`
	Timeout            time.Duration `mapstructure:"timeout"`
	StartTLS           bool          `mapstructure:"start_tls"`
	InsecureSkipVerify bool          `mapstructure:"insecure_skip_verify"`
	TLSConfig          *tls.Config   // Private field, built at runtime
}

// LogConfig holds the logging configuration
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`      // 日志输出位置，默认 stdout
	MaxSize    int    `mapstructure:"max_size"`    // 每个日志文件最大大小(MB)，默认100
	MaxAge     int    `mapstructure:"max_age"`     // 保留旧日志文件的最大天数，默认30
	MaxBackups int    `mapstructure:"max_backups"` // 保留旧日志文件的最大个数，默认3
	LocalTime  bool   `mapstructure:"local_time"`  // 是否使用本地时间，默认false(UTC)
	Compress   bool   `mapstructure:"compress"`    // 是否压缩轮转的日志文件，默认false
}

// CustomSearch defines a custom LDAP search
type CustomSearch struct {
	Name   string `mapstructure:"name"`
	BaseDN string `mapstructure:"base_dn"`
	Filter string `mapstructure:"filter"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate LDAP configuration
	if c.LDAP.Server == "" {
		return fmt.Errorf("ldap.server is required")
	}

	// Validate timeout
	if c.LDAP.Timeout <= 0 {
		return fmt.Errorf("ldap.timeout must be positive, got %v", c.LDAP.Timeout)
	}

	// Validate log level
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
		// Valid log levels
	default:
		return fmt.Errorf("invalid log.level: %s, must be one of debug, info, warn, error", c.Log.Level)
	}

	// Validate log format
	switch c.Log.Format {
	case "text", "json":
		// Valid log formats
	default:
		return fmt.Errorf("invalid log.format: %s, must be one of text, json", c.Log.Format)
	}

	// Validate web configuration
	if c.Web.ListenAddress == "" {
		return fmt.Errorf("web.listen_address is required")
	}

	if c.Web.MetricsPath == "" {
		return fmt.Errorf("web.metrics_path is required")
	}

	// Validate custom searches
	for i, cs := range c.CustomSearches {
		if cs.Name == "" {
			return fmt.Errorf("custom_searches[%d].name is required", i)
		}
		if cs.BaseDN == "" {
			return fmt.Errorf("custom_searches[%d].base_dn is required", i)
		}
		if cs.Filter == "" {
			return fmt.Errorf("custom_searches[%d].filter is required", i)
		}
	}

	return nil
}

// Load loads configuration from file or environment variables
func Load(configFile string) (*Config, error) {
	// Print configuration sources precedence
	// Precedence (highest to lowest):
	// 1. Command line flags (bound via viper.BindPFlag in main)
	// 2. Environment variables
	// 3. Config file
	// 4. Default values

	// Set default values
	viper.SetDefault("web.listen_address", ":9330")
	viper.SetDefault("web.metrics_path", "/metrics")
	viper.SetDefault("ldap.server", "")
	viper.SetDefault("ldap.bind_dn", "")
	viper.SetDefault("ldap.bind_password", "")
	viper.SetDefault("ldap.timeout", 10*time.Second)
	viper.SetDefault("ldap.start_tls", false)
	viper.SetDefault("ldap.insecure_skip_verify", false)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "text")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("log.max_size", 100)
	viper.SetDefault("log.max_age", 30)
	viper.SetDefault("log.max_backups", 3)
	viper.SetDefault("log.local_time", false)
	viper.SetDefault("log.compress", false)

	// Set environment variable prefix
	viper.SetEnvPrefix("OPENLDAP_EXPORTER")

	// 设置环境变量键名替换规则，将点(.)替换为下划线(_)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read in environment variables that match
	viper.AutomaticEnv()

	// Set config file if provided
	if configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file '%s': %w", configFile, err)
		}
	}

	// Parse config into struct
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// 初始化CustomSearches为空切片而不是nil
	if cfg.CustomSearches == nil {
		cfg.CustomSearches = []CustomSearch{}
	}

	// Build TLS configuration
	if cfg.LDAP.InsecureSkipVerify {
		// #nosec G402 InsecureSkipVerify is intentionally configured by user to skip certificate verification
		cfg.LDAP.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		}
	} else {
		cfg.LDAP.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}
	
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}
