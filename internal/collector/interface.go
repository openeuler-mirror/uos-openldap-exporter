package collector

// LDAPClientInterface 定义了LDAP客户端的接口，用于提高代码的可测试性
type LDAPClientInterface interface {
	// Close 关闭LDAP连接
	Close()

	// SearchCount 执行搜索并返回匹配条目数
	SearchCount(baseDN, filter string) (int, error)

	// SearchMonitor 获取cn=Monitor下指定属性的值
	SearchMonitor(dn, attr string) (string, error)

	// CheckHealth 执行健康检查
	CheckHealth() (bool, string)

	// GetTLSStats 获取TLS连接统计信息
	GetTLSStats() (map[string]string, error)

	// GetReplicationStatus 获取复制状态
	GetReplicationStatus() (map[string]string, error)

	// GetSecurityStats 获取安全相关统计
	GetSecurityStats() (map[string]string, error)

	// GetPerformanceStats 获取性能相关统计
	GetPerformanceStats() (map[string]string, error)
}