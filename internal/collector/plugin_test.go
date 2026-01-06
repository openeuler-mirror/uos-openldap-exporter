package collector

import (
	"testing"

	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestPluginManager(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.Config{}
	pm := NewPluginManager(logger, cfg)

	// 测试插件注册
	plugin := NewBaseConnectionPlugin()
	pm.RegisterPlugin(plugin)

	// 验证插件是否已注册
	_, exists := pm.GetPlugin("base_connection")
	assert.True(t, exists, "Plugin should be registered")

	// 验证插件列表
	plugins := pm.ListPlugins()
	assert.Contains(t, plugins, "base_connection")

	// 测试插件启用/禁用
	pm.ConfigurePlugins([]string{"base_connection"})
	registeredPlugin, exists := pm.GetPlugin("base_connection")
	assert.True(t, exists, "Plugin should exist")
	assert.True(t, registeredPlugin.Enabled(), "Plugin should be enabled")

	// 测试禁用所有插件
	pm.ConfigurePlugins([]string{})
	registeredPlugin, exists = pm.GetPlugin("base_connection")
	assert.True(t, exists, "Plugin should exist")
	assert.False(t, registeredPlugin.Enabled(), "Plugin should be disabled")
}

func TestPluginManagerEnableSpecificPlugins(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.Config{}
	pm := NewPluginManager(logger, cfg)

	// 注册多个插件
	basePlugin := NewBaseConnectionPlugin()
	monitorPlugin := NewMonitorSpecificPlugin()
	securityPlugin := NewSecurityPlugin()

	pm.RegisterPlugin(basePlugin)
	pm.RegisterPlugin(monitorPlugin)
	pm.RegisterPlugin(securityPlugin)

	// 启用特定插件
	pm.ConfigurePlugins([]string{"base_connection", "security"})

	// 验证插件状态
	registeredBasePlugin, _ := pm.GetPlugin("base_connection")
	registeredMonitorPlugin, _ := pm.GetPlugin("monitor_specific")
	registeredSecurityPlugin, _ := pm.GetPlugin("security")

	assert.True(t, registeredBasePlugin.Enabled(), "base_connection plugin should be enabled")
	assert.False(t, registeredMonitorPlugin.Enabled(), "monitor_specific plugin should be disabled")
	assert.True(t, registeredSecurityPlugin.Enabled(), "security plugin should be enabled")
}

func TestPluginDescribeAndCollect(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := &config.Config{}
	pm := NewPluginManager(logger, cfg)

	// 注册插件并启用
	plugin := NewBaseConnectionPlugin()
	pm.RegisterPlugin(plugin)
	pm.ConfigurePlugins([]string{"base_connection"})

	// 测试 Describe
	descChan := make(chan *prometheus.Desc, 10)
	go func() {
		defer close(descChan)
		pm.DescribeAll(descChan)
	}()

	descCount := 0
	for range descChan {
		descCount++
	}
	assert.Greater(t, descCount, 0, "At least one descriptor should be sent")
}
