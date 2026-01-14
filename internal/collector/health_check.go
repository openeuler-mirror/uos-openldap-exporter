package collector

import (
	"fmt"
	"net"
	"time"

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

// EnhancedCheckLDAPHealth 执行增强的健康检查，提供更详细的诊断信息
func EnhancedCheckLDAPHealth(cfg *config.LDAPConfig, logger *logrus.Logger) (bool, string) {
	if cfg.Server == "" {
		logger.Debug("LDAP server URL is empty")
		return false, "LDAP server URL is empty"
	}

	// 记录开始时间
	startTime := time.Now()
	
	// 测试连接
	logger.Debug("Attempting to connect to LDAP server...")
	conn, err := ldap.DialURL(cfg.Server, ldap.DialWithDialer(&net.Dialer{Timeout: cfg.Timeout}))
	if err != nil {
		logger.Debugf("Enhanced health check failed to connect to LDAP server: %v", err)
		return false, fmt.Sprintf("Failed to connect to LDAP server: %v", err)
	}
	connectionTime := time.Since(startTime)
	logger.Debugf("Connected to LDAP server in %v", connectionTime)
	
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			logger.Debugf("Error closing LDAP connection: %v", closeErr)
		}
	}()

	// 测试 StartTLS (如果配置)
	var tlsTime time.Duration
	tlsStartTime := time.Now()
	if cfg.StartTLS {
		logger.Debug("Attempting to start TLS...")
		err = conn.StartTLS(cfg.TLSConfig)
		if err != nil {
			logger.Debugf("Enhanced health check failed to start TLS: %v", err)
			return false, fmt.Sprintf("Failed to start TLS: %v", err)
		}
		tlsTime = time.Since(tlsStartTime)
		logger.Debugf("Started TLS in %v", tlsTime)
	}

	// 测试绑定认证
	var bindTime time.Duration
	bindStartTime := time.Now()
	if cfg.BindDN != "" {
		logger.Debugf("Attempting to bind with DN: %s", cfg.BindDN)
		err = conn.Bind(cfg.BindDN, cfg.BindPassword)
		if err != nil {
			logger.Debugf("Enhanced health check failed to bind to LDAP server: %v", err)
			return false, fmt.Sprintf("Failed to bind to LDAP server: %v", err)
		}
		bindTime = time.Since(bindStartTime)
		logger.Debugf("Bind successful in %v", bindTime)
	}

	// 执行WhoAmI操作
	whoAmIStartTime := time.Now()
	whoamiResult, err := conn.WhoAmI(nil)
	if err != nil {
		logger.Debugf("Enhanced health check failed to perform WhoAmI operation: %v", err)
		return false, fmt.Sprintf("Failed to perform WhoAmI operation: %v", err)
	}
	whoAmITime := time.Since(whoAmIStartTime)
	logger.Debugf("WhoAmI operation successful in %v, result: %s", whoAmITime, whoamiResult)

	// 尝试获取基本监控信息
	monitorStartTime := time.Now()
	searchReq := ldap.NewSearchRequest(
		"cn=Monitor",
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0, 0, false,
		"(objectClass=*)",
		[]string{"monitorServerVersion", "monitorRuntimeConfig"},
		nil,
	)
	
	sr, err := conn.Search(searchReq)
	monitorTime := time.Since(monitorStartTime)
	if err != nil {
		logger.Debugf("Could not access monitor backend: %v (this may be normal depending on your LDAP server configuration)", err)
	} else if len(sr.Entries) > 0 {
		logger.Debugf("Monitor search successful in %v, found %d entries", monitorTime, len(sr.Entries))
	}

	totalTime := time.Since(startTime)
	
	details := fmt.Sprintf(
		"Connection: %v, Bind: %v, WhoAmI: %v, Total: %v",
		connectionTime,
		bindTime,
		whoAmITime,
		totalTime,
	)
	
	if cfg.StartTLS {
		details += fmt.Sprintf(", TLS: %v", tlsTime)
	}
	
	if sr != nil && len(sr.Entries) > 0 {
		details += ", Monitor access: OK"
	} else {
		details += ", Monitor access: Not available"
	}

	logger.Debug("Enhanced LDAP health check successful")
	return true, details
}