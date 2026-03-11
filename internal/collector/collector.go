package collector

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Use PrometheusNamespace from constants.go

var (
	upDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "", "up"),
		"Whether the OpenLDAP server is reachable.",
		[]string{"server"}, nil)

	entriesTotalDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "", "entries_total"),
		"Total number of entries in the directory.",
		[]string{"server"}, nil)

	monitorCurrentConnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "monitor", "current_connections"),
		"Current number of connected clients.",
		[]string{"server"}, nil)

	monitorTotalConnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "monitor", "total_connections"),
		"Total number of connections since server startup.",
		[]string{"server"}, nil)

	monitorMaxConnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "monitor", "max_connections"),
		"Maximum number of connections allowed by server configuration.",
		[]string{"server"}, nil)

	monitorActiveOpsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "monitor", "active_operations"),
		"Number of currently active operations.",
		[]string{"server"}, nil)

	monitorPendingOpsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "monitor", "pending_operations"),
		"Number of pending operations.",
		[]string{"server"}, nil)

	monitorOpsInitDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "monitor", "operations_initiated_total"),
		"Number of initiated operations.",
		[]string{"server", "operation"}, nil)

	monitorOpsCompletedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "monitor", "operations_completed_total"),
		"Number of completed operations.",
		[]string{"server", "operation"}, nil)

	monitorOpsWaitingDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "monitor", "operations_waiting"),
		"Number of waiting operations.",
		[]string{"server", "operation"}, nil)

	monitorStatDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "monitor", "statistics"),
		"Various statistics.",
		[]string{"server", "statistic"}, nil)

	customSearchDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "", "custom_search_result_count"),
		"Result count of custom LDAP search.",
		[]string{"server", "name"}, nil)

	// New metrics for enhanced monitoring
	threadsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "monitor", "threads"),
		"Thread pool statistics.",
		[]string{"server", "state"}, nil)

	waitersDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "monitor", "waiters"),
		"Number of threads waiting on a resource.",
		[]string{"server"}, nil)

	timeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "monitor", "time_seconds"),
		"System time metrics from LDAP server.",
		[]string{"server", "type"}, nil)

	// SSL/TLS related metrics
	tlsConnectionsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "tls", "connections_total"),
		"Total number of TLS connections established.",
		[]string{"server"}, nil)

	tlsActiveConnectionsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "tls", "active_connections"),
		"Number of currently active TLS connections.",
		[]string{"server"}, nil)

	startTlsSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "tls", "starttls_success_total"),
		"Total number of successful StartTLS operations.",
		[]string{"server"}, nil)

	startTlsFailureDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "tls", "starttls_failure_total"),
		"Total number of failed StartTLS operations.",
		[]string{"server"}, nil)

	// Replication status metrics
	replicationProviderStatusDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "replication", "provider_status"),
		"Status of replication provider (1=up, 0=down).",
		[]string{"server", "provider"}, nil)

	replicationConsumerStatusDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "replication", "consumer_status"),
		"Status of replication consumer (1=up, 0=down).",
		[]string{"server", "consumer"}, nil)

	replicationProviderDelayDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "replication", "provider_delay_seconds"),
		"Replication delay in seconds.",
		[]string{"server", "provider"}, nil)

	replicationProviderLastUpdateDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "replication", "provider_last_update_time_seconds"),
		"Timestamp of last replication update.",
		[]string{"server", "provider"}, nil)

	// Performance metrics
	ldapOperationResponseTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "performance", "operation_response_time_seconds"),
		"Response time of LDAP operations in seconds.",
		[]string{"server", "operation"}, nil)

	// Security related metrics
	authenticationSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "security", "authentication_success_total"),
		"Total number of successful authentications.",
		[]string{"server"}, nil)

	authenticationFailureDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "security", "authentication_failure_total"),
		"Total number of failed authentications.",
		[]string{"server"}, nil)

	securitySaslBindCountDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "security", "sasl_bind_total"),
		"Total number of SASL bind operations.",
		[]string{"server"}, nil)

	securitySimpleBindCountDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "security", "simple_bind_total"),
		"Total number of simple bind operations.",
		[]string{"server"}, nil)

	securityStrongAuthCountDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "security", "strong_auth_total"),
		"Total number of strong authentication operations.",
		[]string{"server"}, nil)

	// Security and Performance stat descriptors
	securityStatDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "security", "statistics"),
		"Security related statistics.",
		[]string{"server", "statistic"}, nil)

	performanceStatDesc = prometheus.NewDesc(
		prometheus.BuildFQName(PrometheusNamespace, "performance", "statistics"),
		"Performance related statistics.",
		[]string{"server", "statistic"}, nil)
)

// OpenLDAPCollector implements the prometheus.Collector interface
type OpenLDAPCollector struct {
	config        *config.Config
	logger        *logrus.Logger
	pluginManager *PluginManager
	// ldapClientCreator 是一个函数，用于创建LDAP客户端，主要用于测试
	ldapClientCreator func(*config.LDAPConfig, *logrus.Logger) (LDAPClientInterface, error)
	// connectionPool 用于复用LDAP连接
	connectionPool *LDAPClientPool
	// usePool 启用连接池
	usePool bool
	// poolSize 连接池大小
	poolSize int
}

// New creates a new OpenLDAPCollector
func New(cfg *config.Config, logger *logrus.Logger) *OpenLDAPCollector {
	collector := &OpenLDAPCollector{
		config:        cfg,
		logger:        logger,
		pluginManager: NewPluginManager(logger, cfg),
		ldapClientCreator: func(cfg *config.LDAPConfig, logger *logrus.Logger) (LDAPClientInterface, error) {
			return NewLDAPClient(cfg, logger)
		},
	}

	// 配置插件
	collector.pluginManager.ConfigurePlugins(cfg.Plugins.Enabled)

	return collector
}

// GetLDAPConfig returns the LDAP configuration
func (c *OpenLDAPCollector) GetLDAPConfig() *config.LDAPConfig {
	return &c.config.LDAP
}

// GetPluginManager 返回插件管理器
func (c *OpenLDAPCollector) GetPluginManager() *PluginManager {
	return c.pluginManager
}

// CheckHealth performs a health check using existing LDAP client
func (c *OpenLDAPCollector) CheckHealth() (bool, string) {
	// Create LDAP client
	client, err := c.ldapClientCreator(&c.config.LDAP, c.logger)
	if err != nil {
		c.logger.Debugf("Health check failed to create LDAP client: %v", err)
		return false, fmt.Sprintf("Failed to create LDAP client: %v", err)
	}
	defer client.Close()

	// Use the client's CheckHealth method
	return client.CheckHealth()
}

// SetLDAPClientCreatorForTest allows setting the ldapClientCreator for testing purposes.
// This function is intended for use in tests only.
func (c *OpenLDAPCollector) SetLDAPClientCreatorForTest(creator func(*config.LDAPConfig, *logrus.Logger) (LDAPClientInterface, error)) {
	c.ldapClientCreator = creator
}

// RegisterDefaultPlugins 注册默认插件
func (c *OpenLDAPCollector) RegisterDefaultPlugins(pm *PluginManager) {
	// 注册内置插件
	RegisterDefaultPlugins(pm)
}

// Name 返回插件的唯一名称
func (c *OpenLDAPCollector) Name() string {
	return "openldap_base_collector"
}

// Describe implements the prometheus.Collector interface
func (c *OpenLDAPCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- upDesc
	ch <- entriesTotalDesc
	ch <- monitorCurrentConnDesc
	ch <- monitorTotalConnDesc
	ch <- monitorMaxConnDesc
	ch <- monitorActiveOpsDesc
	ch <- monitorPendingOpsDesc
	ch <- monitorOpsInitDesc
	ch <- monitorOpsCompletedDesc
	ch <- monitorOpsWaitingDesc
	ch <- monitorStatDesc
	ch <- customSearchDesc
	ch <- threadsDesc
	ch <- waitersDesc
	ch <- timeDesc
	// SSL/TLS related metrics
	ch <- tlsConnectionsDesc
	ch <- tlsActiveConnectionsDesc
	ch <- startTlsSuccessDesc
	ch <- startTlsFailureDesc
	// Replication status metrics
	ch <- replicationProviderStatusDesc
	ch <- replicationConsumerStatusDesc
	ch <- replicationProviderDelayDesc
	ch <- replicationProviderLastUpdateDesc
	// Performance metrics
	ch <- ldapOperationResponseTimeDesc
	// Security related metrics
	ch <- authenticationSuccessDesc
	ch <- authenticationFailureDesc
	ch <- securitySaslBindCountDesc
	ch <- securitySimpleBindCountDesc
	ch <- securityStrongAuthCountDesc

	// Also describe metrics from plugins
	c.pluginManager.DescribeAll(ch)
}

// Collect implements the prometheus.Collector interface
func (c *OpenLDAPCollector) Collect(ch chan<- prometheus.Metric) {
	if err := c.collectWithClient(ch, nil, ""); err != nil {
		c.logger.Errorf("Error during metrics collection: %v", err)
	}
}

// CollectWithClient implements the PluginCollector interface
func (c *OpenLDAPCollector) CollectWithClient(ch chan<- prometheus.Metric, client LDAPClientInterface, server string) error {
	return c.collectWithClient(ch, client, server)
}

// collectMetricWithErrorHandling safely collects a metric and handles errors
func (c *OpenLDAPCollector) collectMetricWithErrorHandling(
	ch chan<- prometheus.Metric,
	collectFunc func() (float64, error),
	desc *prometheus.Desc,
	labelValues ...string,
) {
	value, err := collectFunc()
	if err != nil {
		c.logger.Debugf("Failed to collect metric %s: %v", desc.String(), err)
		return
	}
	ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value, labelValues...)
}

// collectCounterMetricWithErrorHandling safely collects a counter metric and handles errors
func (c *OpenLDAPCollector) collectCounterMetricWithErrorHandling(
	ch chan<- prometheus.Metric,
	collectFunc func() (float64, error),
	desc *prometheus.Desc,
	labelValues ...string,
) {
	value, err := collectFunc()
	if err != nil {
		c.logger.Debugf("Failed to collect counter metric %s: %v", desc.String(), err)
		return
	}
	ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, value, labelValues...)
}

// collectMonitorMetric collects a metric from LDAP monitor with error handling
func (c *OpenLDAPCollector) collectMonitorMetric(
	ch chan<- prometheus.Metric,
	ldapClient LDAPClientInterface,
	dn, attr string,
	desc *prometheus.Desc,
	labelValues ...string,
) {
	value, err := ldapClient.SearchMonitor(dn, attr)
	if err != nil {
		c.logger.Debugf("Failed to get monitor metric %s from %s: %v", attr, dn, err)
		return
	}
	
	if n, err := strconv.ParseFloat(value, 64); err == nil {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, n, labelValues...)
	} else {
		c.logger.Debugf("Failed to parse monitor metric value %s: %v", value, err)
	}
}

// collectWithClient contains the actual implementation for collecting metrics
func (c *OpenLDAPCollector) collectWithClient(ch chan<- prometheus.Metric, client LDAPClientInterface, server string) error {
	labels := prometheus.Labels{"server": server}

	// Create LDAP client if not provided
	ldapClient := client
	if ldapClient == nil {
		var err error
		ldapClient, err = c.ldapClientCreator(&c.config.LDAP, c.logger)
		if err != nil {
			c.logger.Errorf("Failed to connect to LDAP: %v", err)
			ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 0.0, labels["server"])
			return err
		}
		defer ldapClient.Close()
	}

	// Connection successful
	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 1.0, labels["server"])

	// Check LDAP health status
	if ok, _ := ldapClient.CheckHealth(); ok {
		c.logger.Debug("LDAP health check passed")
	} else {
		c.logger.Debug("LDAP health check failed")
	}

	// Total entries
	c.collectMetricWithErrorHandling(ch, func() (float64, error) {
		count, err := ldapClient.SearchCount("", "(objectClass=*)")
		return float64(count), err
	}, entriesTotalDesc, labels["server"])

	// Monitor: connections details
	connDetails := []struct {
		dn   string
		attr string
		desc *prometheus.Desc
	}{
		{"cn=Current,cn=Connections,cn=Monitor", "monitorCounter", monitorCurrentConnDesc},
		{"cn=Total,cn=Connections,cn=Monitor", "monitorCounter", monitorTotalConnDesc},
		{"cn=Max File Descriptors,cn=Connections,cn=Monitor", "monitorCounter", monitorMaxConnDesc},
	}

	for _, detail := range connDetails {
		c.collectMonitorMetric(ch, ldapClient, detail.dn, detail.attr, detail.desc, labels["server"])
	}

	// Monitor: operations details
	c.collectMonitorMetric(ch, ldapClient, "cn=Operations,cn=Monitor", "monitorOpActive", monitorActiveOpsDesc, labels["server"])
	c.collectMonitorMetric(ch, ldapClient, "cn=Operations,cn=Monitor", "monitorOpPending", monitorPendingOpsDesc, labels["server"])

	// Monitor: operations initiated/completed
	opTypes := []string{"bind", "unbind", "search", "compare", "modify", "modrdn", "add", "delete", "abandon"}
	for _, opType := range opTypes {
		// Initiated operations
		c.collectCounterMetricWithErrorHandling(ch, func() (float64, error) {
			val, err := ldapClient.SearchMonitor("cn=Operations,cn=Monitor", "monitorOpInitiated-"+opType)
			if err != nil {
				return 0, err
			}
			return strconv.ParseFloat(val, 64)
		}, monitorOpsInitDesc, labels["server"], opType)

		// Completed operations
		c.collectCounterMetricWithErrorHandling(ch, func() (float64, error) {
			val, err := ldapClient.SearchMonitor("cn=Operations,cn=Monitor", "monitorOpCompleted-"+opType)
			if err != nil {
				return 0, err
			}
			return strconv.ParseFloat(val, 64)
		}, monitorOpsCompletedDesc, labels["server"], opType)

		// Waiting operations
		c.collectMonitorMetric(ch, ldapClient, "cn=Operations,cn=Monitor", "monitorOpWaiting-"+opType, monitorOpsWaitingDesc, labels["server"], opType)
	}

	// Monitor: statistics
	c.collectMonitorMetric(ch, ldapClient, "cn=Statistics,cn=Monitor", "monitorCounter", monitorStatDesc, labels["server"], "statistics")

	// Enhanced monitoring: thread pool stats
	threadStates := []string{"active", "idle", "max", "starting", "rdn", "wakeup"}
	for _, state := range threadStates {
		c.collectMonitorMetric(ch, ldapClient, "cn=ThreadPool,cn=Monitor", "nBackload"+state, threadsDesc, labels["server"], state)
	}

	// Enhanced monitoring: waiters
	c.collectMonitorMetric(ch, ldapClient, "cn=Waiters,cn=Monitor", "monitorCounter", waitersDesc, labels["server"])

	// Time metrics
	timeTypes := []string{"current", "uptime"}
	for _, ttype := range timeTypes {
		value, err := ldapClient.SearchMonitor("cn=Time,cn=Monitor", "monitorTimestamp-"+ttype)
		if err != nil {
			c.logger.Debugf("Failed to get time metric %s: %v", ttype, err)
			continue
		}
		
		// Convert LDAP timestamp to Unix timestamp
		if unixTime, err := parseLDAPTimestampToSeconds(value); err == nil {
			ch <- prometheus.MustNewConstMetric(timeDesc, prometheus.GaugeValue, unixTime, labels["server"], ttype)
		} else {
			c.logger.Debugf("Failed to parse time metric value %s: %v", value, err)
		}
	}

	// SSL/TLS related metrics
	if tlsStats, err := ldapClient.GetTLSStats(); err == nil {
		if tlsCountStr, exists := tlsStats["total_tls_connections"]; exists {
			if tlsCount, err := strconv.ParseFloat(tlsCountStr, 64); err == nil {
				ch <- prometheus.MustNewConstMetric(tlsConnectionsDesc, prometheus.CounterValue, tlsCount, labels["server"])
			}
		}
		if tlsActiveStr, exists := tlsStats["active_tls_connections"]; exists {
			if tlsActive, err := strconv.ParseFloat(tlsActiveStr, 64); err == nil {
				ch <- prometheus.MustNewConstMetric(tlsActiveConnectionsDesc, prometheus.GaugeValue, tlsActive, labels["server"])
			}
		}
	} else {
		c.logger.Debugf("Failed to get TLS stats: %v", err)
	}

	// STARTTLS metrics
	c.collectCounterMetricWithErrorHandling(ch, func() (float64, error) {
		val, err := ldapClient.SearchMonitor("cn=Statistics,cn=Monitor", "monitorCounter-starttls_success")
		if err != nil {
			return 0, err
		}
		return strconv.ParseFloat(val, 64)
	}, startTlsSuccessDesc, labels["server"])

	c.collectCounterMetricWithErrorHandling(ch, func() (float64, error) {
		val, err := ldapClient.SearchMonitor("cn=Statistics,cn=Monitor", "monitorCounter-starttls_failure")
		if err != nil {
			return 0, err
		}
		return strconv.ParseFloat(val, 64)
	}, startTlsFailureDesc, labels["server"])

	// Replication status metrics
	if replStats, err := ldapClient.GetReplicationStatus(); err == nil {
		if replStatus, exists := replStats["provider_status"]; exists {
			statusValue := 0.0
			if replStatus == "available" {
				statusValue = 1.0
			}
			ch <- prometheus.MustNewConstMetric(replicationProviderStatusDesc, prometheus.GaugeValue, statusValue, labels["server"])
		}
		if replStatus, exists := replStats["consumer_status"]; exists {
			statusValue := 0.0
			if replStatus == "available" {
				statusValue = 1.0
			}
			ch <- prometheus.MustNewConstMetric(replicationConsumerStatusDesc, prometheus.GaugeValue, statusValue, labels["server"])
		}
		if delayStr, exists := replStats["provider_delay"]; exists {
			if delay, err := strconv.ParseFloat(delayStr, 64); err == nil {
				ch <- prometheus.MustNewConstMetric(replicationProviderDelayDesc, prometheus.GaugeValue, delay, labels["server"])
			}
		}
		if lastUpdateStr, exists := replStats["provider_last_update"]; exists {
			if lastUpdate, err := strconv.ParseFloat(lastUpdateStr, 64); err == nil {
				ch <- prometheus.MustNewConstMetric(replicationProviderLastUpdateDesc, prometheus.GaugeValue, lastUpdate, labels["server"])
			}
		}
	} else {
		c.logger.Debugf("Failed to get replication status: %v", err)
	}

	// Security metrics
	if secStats, err := ldapClient.GetSecurityStats(); err == nil {
		for key, value := range secStats {
			if n, err := strconv.ParseFloat(value, 64); err == nil {
				ch <- prometheus.MustNewConstMetric(securityStatDesc, prometheus.GaugeValue, n, labels["server"], key)
			}
		}
	} else {
		c.logger.Debugf("Failed to get security stats: %v", err)
	}

	// Performance metrics
	if perfStats, err := ldapClient.GetPerformanceStats(); err == nil {
		for key, value := range perfStats {
			if n, err := strconv.ParseFloat(value, 64); err == nil {
				ch <- prometheus.MustNewConstMetric(performanceStatDesc, prometheus.GaugeValue, n, labels["server"], key)
			}
		}
	} else {
		c.logger.Debugf("Failed to get performance stats: %v", err)
	}

	// Custom searches - use the top-level config instead of LDAP config
	for _, search := range c.config.CustomSearches {
		c.collectMetricWithErrorHandling(ch, func() (float64, error) {
			count, err := ldapClient.SearchCount(search.BaseDN, search.Filter)
			return float64(count), err
		}, customSearchDesc, labels["server"], search.Name)
	}

	// Collect metrics from plugins
	c.pluginManager.CollectAll(ch, ldapClient, server)

	return nil
}

// PluginAdapter 是一个适配器，用于将OpenLDAPCollector作为PluginCollector使用
type PluginAdapter struct {
	collector *OpenLDAPCollector
}

// NewPluginAdapter 创建一个新的插件适配器
func NewPluginAdapter(collector *OpenLDAPCollector) *PluginAdapter {
	return &PluginAdapter{
		collector: collector,
	}
}

// Name 返回插件名称
func (p *PluginAdapter) Name() string {
	return "openldap_base_collector"
}

// Describe 描述插件提供的指标
func (p *PluginAdapter) Describe(ch chan<- *prometheus.Desc) {
	p.collector.Describe(ch)
}

// Collect 收集插件的指标
func (p *PluginAdapter) Collect(ch chan<- prometheus.Metric, client LDAPClientInterface, server string) error {
	return p.collector.CollectWithClient(ch, client, server)
}

// Enabled 检查插件是否启用
func (p *PluginAdapter) Enabled() bool {
	return true
}

// SetEnabled 设置插件是否启用
func (p *PluginAdapter) SetEnabled(enabled bool) {
	// 基础收集器不能被禁用
}

// parseLDAPTimestampToSeconds converts LDAP timestamp format to seconds since epoch
func parseLDAPTimestampToSeconds(timestamp string) (float64, error) {
	// LDAP Generalized Time format: YYYYMMDDHHMMSS[.sss]Z or YYYYMMDDHHMMSS[.sss]+HHMM
	// For simplicity, we'll parse the basic format without milliseconds

	// Check if timestamp is long enough and ends with Z
	if len(timestamp) < 15 || timestamp[len(timestamp)-1] != 'Z' {
		return 0, nil // fallback for invalid format
	}

	// Parse format: YYYYMMDDHHMMSS using time.Parse directly
	// This is more efficient than string concatenation
	t, err := time.Parse("20060102150405Z", timestamp)
	if err != nil {
		return 0, err
	}
	return float64(t.Unix()), nil
}
