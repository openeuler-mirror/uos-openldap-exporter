package collector

// LDAP Monitor DNs
const (
	// Monitor base DN
	MonitorBaseDN = "cn=Monitor"

	// Connection monitor DNs
	MonitorConnectionsDN = "cn=Connections,cn=Monitor"
	MonitorCurrentDN     = "cn=Current," + MonitorConnectionsDN
	MonitorTotalDN       = "cn=Total," + MonitorConnectionsDN
	MonitorMaxFDDN       = "cn=Max File Descriptors," + MonitorConnectionsDN

	// Operations monitor DNs
	MonitorOperationsDN = "cn=Operations,cn=Monitor"

	// Thread pool monitor DNs
	MonitorThreadPoolDN = "cn=ThreadPool,cn=Monitor"
	MonitorWaitersDN    = "cn=Waiters,cn=Monitor"

	// Time monitor DN
	MonitorTimeDN = "cn=Time,cn=Monitor"

	// Statistics monitor DN
	MonitorStatisticsDN = "cn=Statistics,cn=Monitor"

	// TLS monitor DN
	MonitorTLSDN = "cn=TLS,cn=Monitor"

	// Replication monitor DNs
	MonitorSyncProvidersDN = "cn=Sync,cn=Providers,cn=Monitor"
	MonitorSyncReplDN      = "cn=SyncRepl,cn=config"

	// Security monitor DNs
	MonitorAuthenticationDN = "cn=Authentication,cn=Monitor"
	MonitorSecurityDN       = "cn=Security,cn=Monitor"
)

// LDAP Monitor Attributes
const (
	// Connection attributes
	MonitorCounterAttr = "monitorCounter"

	// Operations attributes
	MonitorOpActiveAttr     = "monitorOpActive"
	MonitorOpPendingAttr    = "monitorOpPending"
	MonitorOpInitiatedAttr  = "monitorOpInitiated"
	MonitorOpCompletedAttr  = "monitorOpCompleted"
	MonitorOpWaitingAttr    = "monitorOpWaiting"
	MonitorTimestampAttr    = "monitorTimestamp"
	MonitorServerVersionAttr = "monitorServerVersion"
	MonitorRuntimeConfigAttr = "monitorRuntimeConfig"

	// Thread pool attributes
	ThreadPoolBackloadPrefix = "nBackload"

	// Replication status values
	ReplicationStatusAvailable = "available"
)

// Prometheus Namespace
const (
	PrometheusNamespace = "openldap"
)

// Default Values
const (
	DefaultPoolSize = 5
)