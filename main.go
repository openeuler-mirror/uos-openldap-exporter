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
	// 修复：logger.New需要参数
	log := logger.New("info", "text")

	cfg, err := config.Load("")  // 修复：LoadConfig应该是Load
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
	
	// 修复：获取插件管理器的正确方式
	pm := collector.GetPluginManager()
	pm.ConfigurePlugins(cfg.Plugins.Enabled)
	
	// 修复：传递正确的参数到RegisterPlugin方法
	pm.RegisterPlugin(collector)

	// 修复：注册默认插件 - 修复调用方式为独立函数
	collector.RegisterDefaultPlugins(pm)

	// 修复：删除未使用的server变量声明
	srv := server.New(cfg.Web.ListenAddress, cfg.Web.MetricsPath, collector, log)

	// 修复：cmd.Execute不需要参数，因为服务器启动逻辑在cobra命令中定义
	if err := cmd.Execute(); err != nil {
		log.Errorf("Server stopped with error: %v", err)
		os.Exit(1)
	}
}