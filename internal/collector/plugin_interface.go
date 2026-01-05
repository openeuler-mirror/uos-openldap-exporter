package collector

import (
	"github.com/prometheus/client_golang/prometheus"
	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"github.com/sirupsen/logrus"
)


// PluginCollector 接口定义插件收集器的基本方法
type PluginCollector interface {
	// Name 返回插件的唯一名称
	Name() string
	
	// Describe 描述插件提供的指标
	Describe(ch chan<- *prometheus.Desc)
	
	// Collect 收集插件的指标
	Collect(ch chan<- prometheus.Metric, client LDAPClientInterface, server string) error
	
	// Enabled 检查插件是否启用
	Enabled() bool
	
	// SetEnabled 设置插件是否启用
	SetEnabled(enabled bool)
}

// BasePluginCollector 提供插件收集器的基础实现
type BasePluginCollector struct {
	enabled bool
}

// Enabled 返回插件是否启用
func (b *BasePluginCollector) Enabled() bool {
	return b.enabled
}

// SetEnabled 设置插件是否启用
func (b *BasePluginCollector) SetEnabled(enabled bool) {
	b.enabled = enabled
}

// PluginManager 管理插件的注册和收集
type PluginManager struct {
	plugins map[string]PluginCollector
	logger  *logrus.Logger
	config  *config.Config
}

// NewPluginManager 创建一个新的插件管理器
func NewPluginManager(logger *logrus.Logger, config *config.Config) *PluginManager {
	return &PluginManager{
		plugins: make(map[string]PluginCollector),
		logger:  logger,
		config:  config,
	}
}

// RegisterPlugin 注册一个插件
func (pm *PluginManager) RegisterPlugin(plugin PluginCollector) {
	pm.plugins[plugin.Name()] = plugin
	pm.logger.Infof("Registered plugin: %s (enabled: %t)", plugin.Name(), plugin.Enabled())
}

// GetPlugin 获取指定名称的插件
func (pm *PluginManager) GetPlugin(name string) (PluginCollector, bool) {
	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// ListPlugins 返回所有插件名称
func (pm *PluginManager) ListPlugins() []string {
	var names []string
	for name := range pm.plugins {
		names = append(names, name)
	}
	return names
}

// ConfigurePlugins 根据配置启用/禁用插件
func (pm *PluginManager) ConfigurePlugins(enabledPlugins []string) {
	// 如果没有指定启用的插件，则启用所有插件
	if len(enabledPlugins) == 0 {
		for _, plugin := range pm.plugins {
			plugin.SetEnabled(true)
		}
		return
	}

	// 先禁用所有插件
	for _, plugin := range pm.plugins {
		plugin.SetEnabled(false)
	}

	// 启用指定的插件
	for _, name := range enabledPlugins {
		if plugin, exists := pm.plugins[name]; exists {
			plugin.SetEnabled(true)
			pm.logger.Infof("Enabled plugin: %s", name)
		} else {
			pm.logger.Warnf("Plugin not found: %s", name)
		}
	}
}

// CollectAll 收集所有启用插件的指标
func (pm *PluginManager) CollectAll(ch chan<- prometheus.Metric, client LDAPClientInterface, server string) {
	for name, plugin := range pm.plugins {
		if plugin.Enabled() {
			if err := plugin.Collect(ch, client, server); err != nil {
				pm.logger.Errorf("Error collecting metrics from plugin %s: %v", name, err)
			}
		} else {
			pm.logger.Debugf("Skipping disabled plugin: %s", name)
		}
	}
}

// DescribeAll 描述所有启用插件的指标
func (pm *PluginManager) DescribeAll(ch chan<- *prometheus.Desc) {
	for name, plugin := range pm.plugins {
		if plugin.Enabled() {
			plugin.Describe(ch)
		} else {
			pm.logger.Debugf("Skipping Describe for disabled plugin: %s", name)
		}
	}
}