package collector

import (
	"fmt"
	"net"

	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"github.com/go-ldap/ldap/v3"
	"github.com/sirupsen/logrus"
)

// CheckLDAPHealth 执行LDAP服务器的健康检查
func CheckLDAPHealth(cfg *config.LDAPConfig, logger *logrus.Logger) (bool, string) {
	if cfg.Server == "" {
		logger.Debug("LDAP server URL is empty")
		return false, "LDAP server URL is empty"
	}

	// Establish connection
	conn, err := ldap.DialURL(cfg.Server, ldap.DialWithDialer(&net.Dialer{Timeout: cfg.Timeout}))
	if err != nil {
		logger.Debugf("Health check failed to connect to LDAP server: %v", err)
		return false, fmt.Sprintf("Failed to connect to LDAP server: %v", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			logger.Debugf("Error closing LDAP connection: %v", closeErr)
		}
	}()

	// Start TLS if configured
	if cfg.StartTLS {
		err = conn.StartTLS(cfg.TLSConfig)
		if err != nil {
			logger.Debugf("Health check failed to start TLS: %v", err)
			return false, fmt.Sprintf("Failed to start TLS: %v", err)
		}
	}

	// Bind with credentials if provided
	if cfg.BindDN != "" {
		err = conn.Bind(cfg.BindDN, cfg.BindPassword)
		if err != nil {
			logger.Debugf("Health check failed to bind to LDAP server: %v", err)
			return false, fmt.Sprintf("Failed to bind to LDAP server: %v", err)
		}
	}

	// Perform a lightweight WhoAmI operation
	_, err = conn.WhoAmI(nil)
	if err != nil {
		logger.Debugf("Health check failed to perform WhoAmI operation: %v", err)
		return false, fmt.Sprintf("Failed to perform WhoAmI operation: %v", err)
	}

	logger.Debug("LDAP health check successful")
	return true, ""
}
