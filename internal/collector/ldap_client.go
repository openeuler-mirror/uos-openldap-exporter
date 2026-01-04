package collector

import (
	"fmt"
	"net"

	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"github.com/go-ldap/ldap/v3"
	"github.com/sirupsen/logrus"
)

// LDAPClient wraps an LDAP connection with configuration and logging
type LDAPClient struct {
	conn   *ldap.Conn
	config *config.LDAPConfig
	logger *logrus.Logger
}

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