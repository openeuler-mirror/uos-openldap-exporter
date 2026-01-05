package collector

import (
	"fmt"
	"strconv"
	"time"

	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	namespace = "openldap"
)

var (
	upDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Whether the OpenLDAP server is reachable.",
		[]string{"server"}, nil)

	entriesTotalDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "entries_total"),
		"Total number of entries in the directory.",
		[]string{"server"}, nil)

	monitorCurrentConnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "current_connections"),
		"Current number of connected clients.",
		[]string{"server"}, nil)

	monitorTotalConnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "total_connections"),
		"Total number of connections since server startup.",
		[]string{"server"}, nil)

	monitorMaxConnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "max_connections"),
		"Maximum number of connections allowed by server configuration.",
		[]string{"server"}, nil)

	monitorActiveOpsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "active_operations"),
		"Number of currently active operations.",
		[]string{"server"}, nil)

	monitorPendingOpsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "pending_operations"),
		"Number of pending operations.",
		[]string{"server"}, nil)

	monitorOpsInitDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "operations_initiated_total"),
		"Number of initiated operations.",
		[]string{"server", "operation"}, nil)

	monitorOpsCompletedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "operations_completed_total"),
		"Number of completed operations.",
		[]string{"server", "operation"}, nil)

	monitorOpsWaitingDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "operations_waiting"),
		"Number of waiting operations.",
		[]string{"server", "operation"}, nil)

	monitorStatDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "statistics"),
		"Various statistics.",
		[]string{"server", "statistic"}, nil)

	customSearchDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "custom_search_result_count"),
		"Result count of custom LDAP search.",
		[]string{"server", "name"}, nil)

	// New metrics for enhanced monitoring
	threadsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "threads"),
		"Thread pool statistics.",
		[]string{"server", "state"}, nil)

	waitersDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "waiters"),
		"Number of threads waiting on a resource.",
		[]string{"server"}, nil)

	timeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "time_seconds"),
		"System time metrics from LDAP server.",
		[]string{"server", "type"}, nil)

	// SSL/TLS related metrics
	tlsConnectionsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "tls", "connections_total"),
		"Total number of TLS connections established.",
		[]string{"server"}, nil)

	tlsActiveConnectionsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "tls", "active_connections"),
		"Number of currently active TLS connections.",
		[]string{"server"}, nil)

	startTlsSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "tls", "starttls_success_total"),
		"Total number of successful StartTLS operations.",
		[]string{"server"}, nil)

	startTlsFailureDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "tls", "starttls_failure_total"),
		"Total number of failed StartTLS operations.",
		[]string{"server"}, nil)

	// Replication status metrics
	replicationProviderStatusDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "replication", "provider_status"),
		"Status of replication provider (1=up, 0=down).",
		[]string{"server", "provider"}, nil)

	replicationConsumerStatusDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "replication", "consumer_status"),
		"Status of replication consumer (1=up, 0=down).",
		[]string{"server", "consumer"}, nil)

	replicationProviderDelayDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "replication", "provider_delay_seconds"),
		"Replication delay in seconds.",
		[]string{"server", "provider"}, nil)

	replicationProviderLastUpdateDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "replication", "provider_last_update_time_seconds"),
		"Timestamp of last replication update.",
		[]string{"server", "provider"}, nil)

	// Performance metrics
	ldapOperationResponseTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "performance", "operation_response_time_seconds"),
		"Response time of LDAP operations in seconds.",
		[]string{"server", "operation"}, nil)

	// Security related metrics
	authenticationSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "security", "authentication_success_total"),
		"Total number of successful authentications.",
		[]string{"server"}, nil)

	authenticationFailureDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "security", "authentication_failure_total"),
		"Total number of failed authentications.",
		[]string{"server"}, nil)

	securitySaslBindCountDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "security", "sasl_bind_total"),
		"Total number of SASL bind operations.",
		[]string{"server"}, nil)

	securitySimpleBindCountDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "security", "simple_bind_total"),
		"Total number of simple bind operations.",
		[]string{"server"}, nil)

	securityStrongAuthCountDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "security", "strong_auth_total"),
		"Total number of strong authentication operations.",
		[]string{"server"}, nil)
)

// OpenLDAPCollector implements the prometheus.Collector interface
type OpenLDAPCollector struct {
	config *config.Config
	logger *logrus.Logger
	// ldapClientCreator 是一个函数，用于创建LDAP客户端，主要用于测试
	ldapClientCreator func(*config.LDAPConfig, *logrus.Logger) (LDAPClientInterface, error)
}

// New creates a new OpenLDAPCollector
func New(cfg *config.Config, logger *logrus.Logger) *OpenLDAPCollector {
	return &OpenLDAPCollector{
		config: cfg,
		logger: logger,
		ldapClientCreator: func(cfg *config.LDAPConfig, logger *logrus.Logger) (LDAPClientInterface, error) {
			return NewLDAPClient(cfg, logger)
		},
	}
}

// GetLDAPConfig returns the LDAP configuration
func (c *OpenLDAPCollector) GetLDAPConfig() *config.LDAPConfig {
	return &c.config.LDAP
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
}

// Collect implements the prometheus.Collector interface
func (c *OpenLDAPCollector) Collect(ch chan<- prometheus.Metric) {
	labels := prometheus.Labels{"server": c.config.LDAP.Server}

	// Create LDAP client
	client, err := c.ldapClientCreator(&c.config.LDAP, c.logger)
	if err != nil {
		c.logger.Errorf("Failed to connect to LDAP: %v", err)
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 0.0, labels["server"])
		return
	}
	defer client.Close()

	// Connection successful
	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 1.0, labels["server"])

	// Check LDAP health status
	if ok, _ := client.CheckHealth(); ok {
		c.logger.Debug("LDAP health check passed")
	} else {
		c.logger.Debug("LDAP health check failed")
	}

	// Total entries
	if count, err := client.SearchCount("", "(objectClass=*)"); err == nil {
		ch <- prometheus.MustNewConstMetric(entriesTotalDesc, prometheus.GaugeValue, float64(count), labels["server"])
	} else {
		c.logger.Warnf("Failed to get total entries: %v", err)
	}

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
		if val, err := client.SearchMonitor(detail.dn, detail.attr); err == nil {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				ch <- prometheus.MustNewConstMetric(detail.desc, prometheus.GaugeValue, n, labels["server"])
			}
		}
	}

	// Monitor: operations details
	if val, err := client.SearchMonitor("cn=Operations,cn=Monitor", "monitorOpActive"); err == nil {
		if n, err := strconv.ParseFloat(val, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(monitorActiveOpsDesc, prometheus.GaugeValue, n, labels["server"])
		}
	}

	if val, err := client.SearchMonitor("cn=Operations,cn=Monitor", "monitorOpPending"); err == nil {
		if n, err := strconv.ParseFloat(val, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(monitorPendingOpsDesc, prometheus.GaugeValue, n, labels["server"])
		}
	}

	// Monitor: operations (initiated, completed, waiting)
	ops := []string{"Bind", "Unbind", "Search", "Modify", "Add", "Delete"}
	for _, op := range ops {
		dn := "cn=" + op + ",cn=Operations,cn=Monitor"

		// Initiated operations
		if val, err := client.SearchMonitor(dn, "monitorOpInitiated"); err == nil {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				opLabels := prometheus.Labels{"server": labels["server"], "operation": op}
				ch <- prometheus.MustNewConstMetric(monitorOpsInitDesc, prometheus.CounterValue, n, opLabels["server"], opLabels["operation"])
			}
		}

		// Completed operations
		if val, err := client.SearchMonitor(dn, "monitorOpCompleted"); err == nil {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				opLabels := prometheus.Labels{"server": labels["server"], "operation": op}
				ch <- prometheus.MustNewConstMetric(monitorOpsCompletedDesc, prometheus.CounterValue, n, opLabels["server"], opLabels["operation"])
			}
		}

		// Waiting operations
		if val, err := client.SearchMonitor(dn, "monitorOpWaiting"); err == nil {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				opLabels := prometheus.Labels{"server": labels["server"], "operation": op}
				ch <- prometheus.MustNewConstMetric(monitorOpsWaitingDesc, prometheus.GaugeValue, n, opLabels["server"], opLabels["operation"])
			}
		}
	}

	// Monitor: statistics
	stats := []string{"Bytes", "Entries", "Referrals", "Operations"}
	for _, stat := range stats {
		dn := "cn=" + stat + ",cn=Statistics,cn=Monitor"
		if val, err := client.SearchMonitor(dn, "monitorCounter"); err == nil {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				statLabels := prometheus.Labels{"server": labels["server"], "statistic": stat}
				ch <- prometheus.MustNewConstMetric(monitorStatDesc, prometheus.CounterValue, n, statLabels["server"], statLabels["statistic"])
			}
		}
	}

	// Monitor: thread pool statistics
	threadStates := []string{"Active", "Starting", "Backing", "Pausing", "Pending"}
	for _, state := range threadStates {
		dn := "cn=" + state + ",cn=Threads,cn=Monitor"
		if val, err := client.SearchMonitor(dn, "monitoredInfo"); err == nil {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				threadLabels := prometheus.Labels{"server": labels["server"], "state": state}
				ch <- prometheus.MustNewConstMetric(threadsDesc, prometheus.GaugeValue, n, threadLabels["server"], threadLabels["state"])
			}
		}
	}

	// Monitor: waiters
	if val, err := client.SearchMonitor("cn=Waiters,cn=Threads,cn=Monitor", "monitorCounter"); err == nil {
		if n, err := strconv.ParseFloat(val, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(waitersDesc, prometheus.GaugeValue, n, labels["server"])
		}
	}

	// Monitor: time metrics
	timeTypes := []string{"Start", "Current"}
	for _, timeType := range timeTypes {
		dn := "cn=" + timeType + ",cn=Time,cn=Monitor"
		if val, err := client.SearchMonitor(dn, "monitorTimestamp"); err == nil {
			// Convert timestamp to seconds since epoch
			if secs, err := parseLDAPTimestampToSeconds(val); err == nil {
				timeLabels := prometheus.Labels{"server": labels["server"], "type": timeType}
				ch <- prometheus.MustNewConstMetric(timeDesc, prometheus.GaugeValue, secs, timeLabels["server"], timeLabels["type"])
			}
		}
	}

	// Custom searches
	for _, cs := range c.config.CustomSearches {
		if count, err := client.SearchCount(cs.BaseDN, cs.Filter); err == nil {
			ch <- prometheus.MustNewConstMetric(customSearchDesc, prometheus.GaugeValue, float64(count), labels["server"], cs.Name)
		} else {
			c.logger.Warnf("Custom search '%s' failed: %v", cs.Name, err)
		}
	}

	// Collect new SSL/TLS related metrics
	if tlsStats, err := client.GetTLSStats(); err == nil {
		if val, ok := tlsStats["tls_connections"]; ok {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				ch <- prometheus.MustNewConstMetric(tlsConnectionsDesc, prometheus.CounterValue, n, labels["server"])
			}
		}
		
		if val, ok := tlsStats["tls_active_connections"]; ok {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				ch <- prometheus.MustNewConstMetric(tlsActiveConnectionsDesc, prometheus.GaugeValue, n, labels["server"])
			}
		}
		
		if val, ok := tlsStats["starttls_success_total"]; ok {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				ch <- prometheus.MustNewConstMetric(startTlsSuccessDesc, prometheus.CounterValue, n, labels["server"])
			}
		}
		
		if val, ok := tlsStats["starttls_failure_total"]; ok {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				ch <- prometheus.MustNewConstMetric(startTlsFailureDesc, prometheus.CounterValue, n, labels["server"])
			}
		}
	}

	// Collect replication status metrics
	if replStats, err := client.GetReplicationStatus(); err == nil {
		if provider, ok := replStats["provider"]; ok {
			// Set provider status based on whether we found a provider
			providerLabels := prometheus.Labels{
				"server": labels["server"],
				"provider": provider,
			}
			
			// We assume the provider is up if we can get its config
			ch <- prometheus.MustNewConstMetric(replicationProviderStatusDesc, prometheus.GaugeValue, 1.0, 
				providerLabels["server"], providerLabels["provider"])
			
			// Check for delay if available
			if delay, ok := replStats["delay"]; ok {
				if n, err := strconv.ParseFloat(delay, 64); err == nil {
					ch <- prometheus.MustNewConstMetric(replicationProviderDelayDesc, prometheus.GaugeValue, n, 
						providerLabels["server"], providerLabels["provider"])
				}
			}
		}
	}

	// Collect security related metrics
	if secStats, err := client.GetSecurityStats(); err == nil {
		// Authentication success/failure counts (these would typically come from logs or special counters)
		// For now, we'll use placeholder values if not available in monitor
		if val, ok := secStats["simple_bind_total"]; ok {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				ch <- prometheus.MustNewConstMetric(securitySimpleBindCountDesc, prometheus.CounterValue, n, labels["server"])
			}
		}
		
		if val, ok := secStats["sasl_bind_total"]; ok {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				ch <- prometheus.MustNewConstMetric(securitySaslBindCountDesc, prometheus.CounterValue, n, labels["server"])
			}
		}
	}

	// Collect performance related metrics
	if perfStats, err := client.GetPerformanceStats(); err == nil {
		// Performance metrics would be calculated as response times during actual operations
		// For now, we'll expose some counters from the monitor
		if val, ok := perfStats["read_ops_completed"]; ok {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				opLabels := prometheus.Labels{
					"server": labels["server"],
					"operation": "read",
				}
				ch <- prometheus.MustNewConstMetric(ldapOperationResponseTimeDesc, prometheus.CounterValue, n, 
					opLabels["server"], opLabels["operation"])
			}
		}
	}
}

// parseLDAPTimestampToSeconds converts LDAP timestamp format to seconds since epoch
func parseLDAPTimestampToSeconds(timestamp string) (float64, error) {
	// LDAP Generalized Time format: YYYYMMDDHHMMSS[.sss]Z or YYYYMMDDHHMMSS[.sss]+HHMM
	// For simplicity, we'll parse the basic format without milliseconds

	// Remove trailing Z or timezone info for basic parsing
	timestamp = timestamp[:len(timestamp)-1] // Remove last char (Z)

	// Parse format: YYYYMMDDHHMMSS
	if len(timestamp) >= 14 {
		year := timestamp[0:4]
		month := timestamp[4:6]
		day := timestamp[6:8]
		hour := timestamp[8:10]
		minute := timestamp[10:12]
		second := timestamp[12:14]

		dateStr := year + "-" + month + "-" + day + "T" + hour + ":" + minute + ":" + second + "Z"
		t, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			return 0, err
		}
		return float64(t.Unix()), nil
	}

	return 0, nil // fallback
}
