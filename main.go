package main

import (
	"os"

	"github.com/sirupsen/logrus"

	"gitee.com/openeuler/uos-openldap-exporter/cmd"
	"gitee.com/openeuler/uos-openldap-exporter/internal/collector"
	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"gitee.com/openeuler/uos-openldap-exporter/internal/logger"
	"gitee.com/openeuler/uos-openldap-exporter/internal/server"
)

func main() {
	log := logger.New("info", "text")

	cfg, err := config.Load("")
	if err != nil {
		log.Errorf("Failed to load config: %v", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		log.Errorf("Invalid config: %v", err)
		os.Exit(1)
	}

	logLevel, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		log.Errorf("Invalid log level: %v", err)
		os.Exit(1)
	}
	log.SetLevel(logLevel)

	if cfg.Log.Format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{})
	}

	ldapCollector := collector.New(cfg, log)

	// 配置并注册所有需要的插件
	pm := ldapCollector.GetPluginManager()
	pm.ConfigurePlugins(cfg.Plugins.Enabled)


	// 注册默认插件
	ldapCollector.RegisterDefaultPlugins(pm)

	server.New(cfg.Web.ListenAddress, cfg.Web.MetricsPath, ldapCollector, log)

	if err := cmd.Execute(); err != nil {
		log.Errorf("Server stopped with error: %v", err)
		os.Exit(1)
	}
}
