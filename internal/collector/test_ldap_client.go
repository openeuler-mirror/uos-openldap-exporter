package collector

import (
	"fmt"
	"net"

	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"github.com/go-ldap/ldap/v3"
	"github.com/sirupsen/logrus"
)

// TestLDAPClientInterface 定义用于测试的LDAP客户端接口
type TestLDAPClientInterface interface {
	Connect() (*ldap.Conn, error)
	Bind(*ldap.Conn) error
	Search(*ldap.Conn, string, string, []string) ([]*ldap.Entry, error)
	Close(*ldap.Conn) error
}

// TestLDAPClient 实现TestLDAPClientInterface接口
type TestLDAPClient struct {
	config *config.LDAPConfig
	logger *logrus.Logger
}

// NewTestLDAPClient 创建一个新的用于测试的LDAP客户端
func NewTestLDAPClient(cfg *config.LDAPConfig) (*TestLDAPClient, error) {
	if cfg.Server == "" {
		return nil, fmt.Errorf("ldap.server is required")
	}

	return &TestLDAPClient{
		config: cfg,
		logger: logrus.New(),
	}, nil
}

// Connect 建立到LDAP服务器的新连接
func (c *TestLDAPClient) Connect() (*ldap.Conn, error) {
	// Create a dialer with timeout
	dialer := &net.Dialer{
		Timeout: c.config.Timeout,
	}

	// Establish connection
	conn, err := ldap.DialURL(c.config.Server, ldap.DialWithDialer(dialer))
	if err != nil {
		return nil, fmt.Errorf("failed to dial LDAP: %w", err)
	}

	// Start TLS if configured
	if c.config.StartTLS {
		err = conn.StartTLS(c.config.TLSConfig)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	return conn, nil
}

// Bind 使用配置的凭据对LDAP服务器进行身份验证
func (c *TestLDAPClient) Bind(conn *ldap.Conn) error {
	if c.config.BindDN != "" {
		err := conn.Bind(c.config.BindDN, c.config.BindPassword)
		if err != nil {
			return fmt.Errorf("bind failed: %w", err)
		}
	}
	return nil
}

// Search 执行LDAP搜索并返回条目
func (c *TestLDAPClient) Search(conn *ldap.Conn, baseDN, filter string, attributes []string) ([]*ldap.Entry, error) {
	req := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,                               // no size limit
		int(c.config.Timeout.Seconds()), // time limit in seconds
		false,
		filter,
		attributes,
		nil,
	)

	res, err := conn.Search(req)
	if err != nil {
		c.logger.Debugf("Search failed for baseDN=%s, filter=%s: %v", baseDN, filter, err)
		return nil, err
	}

	c.logger.Debugf("Search returned %d entries for baseDN=%s, filter=%s", len(res.Entries), baseDN, filter)
	return res.Entries, nil
}

// Close 关闭LDAP连接
func (c *TestLDAPClient) Close(conn *ldap.Conn) error {
	if conn != nil {
		return conn.Close()
	}
	return nil
}
