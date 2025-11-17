package collector

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
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

	monitorConnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "connections_total"),
		"Current number of connections.",
		[]string{"server"}, nil)

	monitorOpsInitDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "operations_initiated_total"),
		"Number of initiated operations.",
		[]string{"server", "operation"}, nil)

	customSearchDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "custom_search_result_count"),
		"Result count of custom LDAP search.",
		[]string{"server", "name"}, nil)
)

// OpenLDAPCollector implements the prometheus.Collector interface
type OpenLDAPCollector struct {
	config *config.Config
	logger *logrus.Logger
}

// New creates a new OpenLDAPCollector
func New(cfg *config.Config, logger *logrus.Logger) *OpenLDAPCollector {
	return &OpenLDAPCollector{
		config: cfg,
		logger: logger,
	}
}

// Describe implements the prometheus.Collector interface
func (c *OpenLDAPCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- upDesc
	ch <- entriesTotalDesc
	ch <- monitorConnDesc
	ch <- monitorOpsInitDesc
	ch <- customSearchDesc
}

// Collect implements the prometheus.Collector interface
func (c *OpenLDAPCollector) Collect(ch chan<- prometheus.Metric) {
	labels := prometheus.Labels{"server": c.config.LDAP.Server}

	// Create LDAP client
	client, err := NewLDAPClient(&c.config.LDAP, c.logger)
	if err != nil {
		c.logger.Errorf("Failed to connect to LDAP: %v", err)
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 0.0, labels["server"])
		return
	}
	defer client.Close()

	// Connection successful
	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 1.0, labels["server"])

	// Total entries
	if count, err := client.SearchCount("", "(objectClass=*)"); err == nil {
		ch <- prometheus.MustNewConstMetric(entriesTotalDesc, prometheus.GaugeValue, float64(count), labels["server"])
	} else {
		c.logger.Warnf("Failed to get total entries: %v", err)
	}

	// Monitor: connections
	if val, err := client.SearchMonitor("cn=Connections,cn=Monitor", "monitorCounter"); err == nil {
		if n, err := strconv.ParseFloat(val, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(monitorConnDesc, prometheus.GaugeValue, n, labels["server"])
		}
	}

	// Monitor: operations (initiated)
	ops := []string{"Bind", "Unbind", "Search", "Modify", "Add", "Delete"}
	for _, op := range ops {
		dn := "cn=" + op + ",cn=Operations,cn=Monitor"
		if val, err := client.SearchMonitor(dn, "monitorOpInitiated"); err == nil {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				opLabels := prometheus.Labels{"server": labels["server"], "operation": op}
				ch <- prometheus.MustNewConstMetric(monitorOpsInitDesc, prometheus.CounterValue, n, opLabels["server"], opLabels["operation"])
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
}