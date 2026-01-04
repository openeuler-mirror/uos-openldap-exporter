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
}
