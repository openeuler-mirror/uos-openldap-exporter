package collector

import (
	"fmt"
	"net"
	"strings"
	"time"

	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"github.com/go-ldap/ldap/v3"
	"github.com/sirupsen/logrus"
)

// NewLDAPClient creates a new LDAP client and establishes a connection
func NewLDAPClient(cfg *config.LDAPConfig, logger *logrus.Logger) (*LDAPClient, error) {
	if cfg.Server == "" {
		return nil, fmt.Errorf("ldap.server is required")
	}

	// Create a dialer with timeout
	dialer := &net.Dialer{
		Timeout: cfg.Timeout,
	}

	// Establish connection
	conn, err := ldap.DialURL(cfg.Server, ldap.DialWithDialer(dialer))
	if err != nil {
		return nil, fmt.Errorf("failed to dial LDAP: %w", err)
	}

	// Start TLS if configured
	if cfg.StartTLS {
		err = conn.StartTLS(cfg.TLSConfig)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Bind with credentials if provided
	if cfg.BindDN != "" {
		err = conn.Bind(cfg.BindDN, cfg.BindPassword)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("bind failed: %w", err)
		}
	}

	return &LDAPClient{
		conn:   conn,
		config: cfg,
		logger: logger,
	}, nil
}

// LDAPClient wraps an LDAP connection with configuration and logging
type LDAPClient struct {
	conn   *ldap.Conn
	config *config.LDAPConfig
	logger *logrus.Logger
}

// Close closes the LDAP connection
func (c *LDAPClient) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// SearchCount executes a search and returns the count of matching entries
func (c *LDAPClient) SearchCount(baseDN, filter string) (int, error) {
	req := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, // no size limit for count
		0, // no time limit
		false,
		filter,
		[]string{"dn"}, // minimal attributes
		nil,
	)

	res, err := c.conn.Search(req)
	if err != nil {
		c.logger.Debugf("SearchCount failed for baseDN=%s, filter=%s: %v", baseDN, filter, err)
		return 0, err
	}

	c.logger.Debugf("SearchCount returned %d entries for baseDN=%s, filter=%s", len(res.Entries), baseDN, filter)
	return len(res.Entries), nil
}

// SearchMonitor retrieves the value of a specific attribute from a monitor DN
func (c *LDAPClient) SearchMonitor(dn, attr string) (string, error) {
	req := ldap.NewSearchRequest(
		dn,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1, // size limit
		0, // time limit
		false,
		"(objectClass=*)",
		[]string{attr},
		nil,
	)

	res, err := c.conn.Search(req)
	if err != nil || len(res.Entries) == 0 {
		c.logger.Debugf("SearchMonitor failed for dn=%s, attr=%s: %v", dn, attr, err)
		return "", err
	}

	vals := res.Entries[0].GetAttributeValues(attr)
	if len(vals) == 0 {
		err := fmt.Errorf("attribute %s not found in %s", attr, dn)
		c.logger.Debugf("SearchMonitor failed: %v", err)
		return "", err
	}

	c.logger.Debugf("SearchMonitor returned value %s for dn=%s, attr=%s", vals[0], dn, attr)
	return vals[0], nil
}

// CheckHealth performs a health check on the LDAP connection
func (c *LDAPClient) CheckHealth() (bool, string) {
	// Perform a lightweight WhoAmI operation
	startTime := time.Now()
	_, err := c.conn.WhoAmI(nil)
	duration := time.Since(startTime)

	if err != nil {
		c.logger.Debugf("Health check failed to perform WhoAmI operation: %v", err)
		return false, fmt.Sprintf("Failed to perform WhoAmI operation: %v", err)
	}

	c.logger.Debugf("LDAP health check successful, took %v", duration)
	return true, ""
}

// searchAndExtractAttributes performs a search and extracts attributes matching a filter function
func (c *LDAPClient) searchAndExtractAttributes(baseDN string, scope int, filter string, attributes []string, attrFilter func(string) bool) (map[string]string, error) {
	result := make(map[string]string)

	req := ldap.NewSearchRequest(
		baseDN,
		scope,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		attributes,
		nil,
	)

	res, err := c.conn.Search(req)
	if err != nil || len(res.Entries) == 0 {
		c.logger.Debugf("Search failed for baseDN=%s: %v", baseDN, err)
		return result, err
	}

	for _, entry := range res.Entries {
		for _, attr := range entry.Attributes {
			if attrFilter(attr.Name) && len(attr.Values) > 0 {
				result[attr.Name] = attr.Values[0]
			}
		}
	}

	return result, nil
}

// GetTLSStats 获取TLS连接统计信息
func (c *LDAPClient) GetTLSStats() (map[string]string, error) {
	stats := make(map[string]string)

	// We can't directly check if the connection is TLS in go-ldap, so we'll check config
	if c.config.StartTLS {
		stats["tls_connections"] = "1"
		stats["tls_active_connections"] = "1"
	} else {
		stats["tls_connections"] = "0"
		stats["tls_active_connections"] = "0"
	}

	// Try to get TLS-specific stats from monitor
	if val, err := c.SearchMonitor("cn=TLS,cn=Monitor", "monitorCounter"); err == nil {
		stats["tls_connections_total"] = val
	} else {
		// If cn=TLS,cn=Monitor is not available, default to connection status
		c.logger.Debugf("Could not get TLS stats from monitor: %v", err)
	}

	// Check for StartTLS statistics
	if val, err := c.SearchMonitor("cn=StartTLS,cn=Operations,cn=Monitor", "monitorOpCompleted"); err == nil {
		stats["starttls_success_total"] = val
	}

	if val, err := c.SearchMonitor("cn=StartTLS,cn=Operations,cn=Monitor", "monitorOpUnwillingToPerform"); err == nil {
		stats["starttls_failure_total"] = val
	}

	return stats, nil
}

// GetReplicationStatus 获取复制状态
func (c *LDAPClient) GetReplicationStatus() (map[string]string, error) {
	status := make(map[string]string)

	// Define attribute filter for replication-related attributes
	replicationAttrFilter := func(attrName string) bool {
		lowerName := strings.ToLower(attrName)
		return strings.Contains(lowerName, "status") ||
			strings.Contains(lowerName, "state") ||
			strings.Contains(lowerName, "delay")
	}

	// Check for syncrepl provider status
	result, err := c.searchAndExtractAttributes(
		"cn=Sync,cn=Providers,cn=Monitor",
		ldap.ScopeWholeSubtree,
		"(objectClass=*)",
		[]string{"*"},
		replicationAttrFilter,
	)
	if err == nil {
		for k, v := range result {
			status[k] = v
		}
	} else {
		c.logger.Debugf("Could not get replication provider status from monitor: %v", err)
	}

	// Try to get replication provider information from cn=SyncRepl
	syncReplReq := ldap.NewSearchRequest(
		"cn=SyncRepl,cn=config",
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		"(objectClass=olcSyncreplConfig)",
		[]string{"olcDatabase", "olcSyncRepl"},
		nil,
	)

	syncReplRes, err := c.conn.Search(syncReplReq)
	if err == nil && len(syncReplRes.Entries) > 0 {
		for _, entry := range syncReplRes.Entries {
			for _, attr := range entry.Attributes {
				if attr.Name == "olcSyncRepl" && len(attr.Values) > 0 {
					for _, value := range attr.Values {
						// Parse replication configuration to extract provider info
						if strings.Contains(value, "provider=") {
							parts := strings.Split(value, " ")
							for _, part := range parts {
								if strings.HasPrefix(part, "provider=") {
									provider := strings.TrimPrefix(part, "provider=")
									status["provider"] = provider
								} else if strings.HasPrefix(part, "binddn=") {
									binddn := strings.TrimPrefix(part, "binddn=")
									status["binddn"] = binddn
								}
							}
						}
					}
				}
			}
		}
	} else {
		c.logger.Debugf("Could not get replication config from cn=config: %v", err)
	}

	return status, nil
}

// GetSecurityStats 获取安全相关统计
func (c *LDAPClient) GetSecurityStats() (map[string]string, error) {
	stats := make(map[string]string)

	// Define attribute filter for security-related attributes
	securityAttrFilter := func(attrName string) bool {
		lowerName := strings.ToLower(attrName)
		return strings.Contains(lowerName, "auth") ||
			strings.Contains(lowerName, "bind") ||
			strings.Contains(lowerName, "sasl") ||
			strings.Contains(lowerName, "strong")
	}

	// Get authentication statistics from monitor
	authStats := []string{
		"cn=Authentication,cn=Monitor",
		"cn=Security,cn=Monitor",
	}

	for _, baseDN := range authStats {
		result, err := c.searchAndExtractAttributes(
			baseDN,
			ldap.ScopeBaseObject,
			"(objectClass=*)",
			[]string{"*"},
			securityAttrFilter,
		)
		if err == nil {
			for k, v := range result {
				stats[k] = v
			}
		}
	}

	// Specific security stats
	if val, err := c.SearchMonitor("cn=Simple Bind,cn=Operations,cn=Monitor", "monitorOpCompleted"); err == nil {
		stats["simple_bind_total"] = val
	} else {
		c.logger.Debugf("Could not get simple bind stats: %v", err)
	}

	if val, err := c.SearchMonitor("cn=SASL,cn=Operations,cn=Monitor", "monitorOpCompleted"); err == nil {
		stats["sasl_bind_total"] = val
	} else {
		c.logger.Debugf("Could not get SASL bind stats: %v", err)
	}

	return stats, nil
}

// GetPerformanceStats 获取性能相关统计
func (c *LDAPClient) GetPerformanceStats() (map[string]string, error) {
	stats := make(map[string]string)

	// Define performance metrics to collect
	perfMetrics := []struct {
		dn       string
		attr     string
		metricKey string
	}{
		{"cn=Read,cn=Operations,cn=Monitor", "monitorOpCompleted", "read_ops_completed"},
		{"cn=Compare,cn=Operations,cn=Monitor", "monitorOpCompleted", "compare_ops_completed"},
		{"cn=Time,cn=Monitor", "monitorTimestamp", "current_time"},
	}

	// Collect performance metrics
	for _, metric := range perfMetrics {
		if val, err := c.SearchMonitor(metric.dn, metric.attr); err == nil {
			stats[metric.metricKey] = val
		} else {
			c.logger.Debugf("Could not get performance stats for %s: %v", metric.dn, err)
		}
	}

	return stats, nil
}
