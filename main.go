package main

import (
	"os"

	"github.com/sirupsen/logrus"

	"gitee.com/openeuler/uos-openldap-exporter/cmd"
	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"gitee.com/openeuler/uos-openldap-exporter/internal/collector"
	"gitee.com/openeuler/uos-openldap-exporter/internal/logger"
	"gitee.com/openeuler/uos-openldap-exporter/internal/server"
)

func main() {
	log := logger.New()

	cfg, err := config.LoadConfig("")
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

	collector := collector.New(cfg, log)
	
	// 配置并注册所有需要的插件
	pm := collector.GetPluginManager()
	pm.ConfigurePlugins(cfg.Plugins.Enabled)
	pm.RegisterPlugin(collector)
	pm.RegisterPlugin(collector.NewBaseConnectionPlugin())
	pm.RegisterPlugin(collector.NewMonitorSpecificPlugin())
	pm.RegisterPlugin(collector.NewSecurityPlugin())

	server := server.New(cfg.Web.ListenAddress, cfg.Web.MetricsPath, collector, log)

	if err := cmd.Execute(server); err != nil {
		log.Errorf("Server stopped with error: %v", err)
		os.Exit(1)
	}
}

