package config

import (
	"crypto/tls"
	"fmt"
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
	Level string `mapstructure:"level"`
	Format string `mapstructure:"format"`  // 添加日志格式字段，支持json或text
}

// CustomSearch defines a custom LDAP search
type CustomSearch struct {
	Name   string `mapstructure:"name"`
	BaseDN string `mapstructure:"base_dn"`
	Filter string `mapstructure:"filter"`
}

// Load loads configuration from file or environment variables
func Load(configFile string) *Config {
	// Set default values
	viper.SetDefault("web.listen_address", ":9330")
	viper.SetDefault("web.metrics_path", "/metrics")
	viper.SetDefault("ldap.timeout", 10*time.Second)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "text")  // 设置日志格式默认值为text

	// Set environment variable prefix
	viper.SetEnvPrefix("OPENLDAP_EXPORTER")
	viper.AutomaticEnv()

	// Set config file if provided
	if configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			panic(fmt.Errorf("failed to read config file: %w", err))
		}
	}

	// Parse config into struct
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("failed to parse config: %w", err))
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

	return &cfg
}
