package collector

import (
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
)

// MockLDAPClient 是LDAP客户端的模拟实现
type MockLDAPClient struct {
	mock.Mock
}

func (m *MockLDAPClient) Close() {
	m.Called()
}

func (m *MockLDAPClient) SearchCount(baseDN, filter string) (int, error) {
	args := m.Called(baseDN, filter)
	return args.Int(0), args.Error(1)
}

func (m *MockLDAPClient) SearchMonitor(dn, attr string) (string, error) {
	args := m.Called(dn, attr)
	return args.String(0), args.Error(1)
}

func (m *MockLDAPClient) CheckHealth() (bool, string) {
	args := m.Called()
	return args.Bool(0), args.String(1)
}

func (m *MockLDAPClient) GetTLSStats() (map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockLDAPClient) GetReplicationStatus() (map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockLDAPClient) GetSecurityStats() (map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockLDAPClient) GetPerformanceStats() (map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]string), args.Error(1)
}

func TestOpenLDAPCollector_ConnectError(t *testing.T) {
	// Arrange
	cfg := &config.Config{
		LDAP: config.LDAPConfig{
			Server:       "localhost:389",
			BindDN:       "cn=admin,dc=example,dc=com",
			BindPassword: "password",
			Timeout:      5 * time.Second,
		},
	}
	log := logrus.New()

	// 创建collector
	collector := New(cfg, log)

	// 替换ldapClientCreator为总是返回错误的函数
	collector.ldapClientCreator = func(cfg *config.LDAPConfig, logger *logrus.Logger) (LDAPClientInterface, error) {
		return nil, errors.New("connection failed")
	}

	// Act & Assert
	ch := make(chan prometheus.Metric, 100) // 使用较大的channel避免阻塞
	go func() {
		defer close(ch)
		collector.Collect(ch)
	}()

	// 收集所有指标
	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	// 验证up指标为0
	assert.NotEmpty(t, metrics)
	foundUpMetric := false
	for _, metric := range metrics {
		dto := &dto.Metric{}
		if metric.Write(dto) == nil {
			// 检查指标名称是否为up，并且值为0
			if metric.Desc().String() == upDesc.String() {
				if dto.GetGauge().GetValue() == 0.0 {
					// 检查标签是否包含服务器地址
					foundServerLabel := false
					for _, label := range dto.GetLabel() {
						if label.GetName() == "server" {
							foundServerLabel = true
							break
						}
					}
					if foundServerLabel {
						foundUpMetric = true
						break
					}
				}
			}
		}
	}
	assert.True(t, foundUpMetric, "Expected to find up metric with value 0")
}

func TestOpenLDAPCollector_SuccessfulCollection(t *testing.T) {
	// 准备测试配置
	cfg := &config.Config{
		LDAP: config.LDAPConfig{
			Server: "ldap://localhost:389",
		},
		CustomSearches: []config.CustomSearch{
			{
				Name:   "test_search",
				BaseDN: "ou=people,dc=example,dc=com",
				Filter: "(objectClass=person)",
			},
		},
	}

	// 创建logger
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	// 创建mock客户端
	mockClient := new(MockLDAPClient)
	mockClient.On("Close").Return()
	mockClient.On("SearchCount", "", "(objectClass=*)").Return(10, nil)
	mockClient.On("SearchCount", "ou=people,dc=example,dc=com", "(objectClass=person)").Return(5, nil)
	mockClient.On("CheckHealth").Return(true, "")

	// cn=Current,cn=Connections,cn=Monitor
	mockClient.On("SearchMonitor", "cn=Current,cn=Connections,cn=Monitor", "monitorCounter").Return("5", nil)
	// cn=Total,cn=Connections,cn=Monitor
	mockClient.On("SearchMonitor", "cn=Total,cn=Connections,cn=Monitor", "monitorCounter").Return("15", nil)
	// cn=Max File Descriptors,cn=Connections,cn=Monitor
	mockClient.On("SearchMonitor", "cn=Max File Descriptors,cn=Connections,cn=Monitor", "monitorCounter").Return("100", nil)
	// cn=Operations,cn=Monitor
	mockClient.On("SearchMonitor", "cn=Operations,cn=Monitor", "monitorOpActive").Return("2", nil)
	mockClient.On("SearchMonitor", "cn=Operations,cn=Monitor", "monitorOpPending").Return("1", nil)

	// Operations initiated/completed/waiting - mock the specific calls that are made in the actual code
	opTypes := []string{"bind", "unbind", "search", "compare", "modify", "modrdn", "add", "delete", "abandon"}
	for _, opType := range opTypes {
		mockClient.On("SearchMonitor", "cn=Operations,cn=Monitor", "monitorOpInitiated-"+opType).Return("10", nil)
		mockClient.On("SearchMonitor", "cn=Operations,cn=Monitor", "monitorOpCompleted-"+opType).Return("9", nil)
		mockClient.On("SearchMonitor", "cn=Operations,cn=Monitor", "monitorOpWaiting-"+opType).Return("0", nil)
	}

	// Statistics - mock the specific call made in the actual code
	mockClient.On("SearchMonitor", "cn=Statistics,cn=Monitor", "monitorCounter").Return("100", nil)

	// Thread pool stats
	threadStates := []string{"active", "idle", "max", "starting", "rdn", "wakeup"}
	for _, state := range threadStates {
		mockClient.On("SearchMonitor", "cn=ThreadPool,cn=Monitor", "nBackload"+state).Return("1", nil)
	}

	// Waiters
	mockClient.On("SearchMonitor", "cn=Waiters,cn=Monitor", "monitorCounter").Return("0", nil)

	// Time metrics
	mockClient.On("SearchMonitor", "cn=Time,cn=Monitor", "monitorTimestamp-current").Return("20230101010000Z", nil)
	mockClient.On("SearchMonitor", "cn=Time,cn=Monitor", "monitorTimestamp-uptime").Return("20230101000000Z", nil)

	// STARTTLS metrics
	mockClient.On("SearchMonitor", "cn=Statistics,cn=Monitor", "monitorCounter-starttls_success").Return("25", nil)
	mockClient.On("SearchMonitor", "cn=Statistics,cn=Monitor", "monitorCounter-starttls_failure").Return("2", nil)

	// TLS Stats
	tlsStats := map[string]string{
		"tls_connections":        "50",
		"tls_active_connections": "5",
		"starttls_success_total": "25",
		"starttls_failure_total": "2",
	}
	mockClient.On("GetTLSStats").Return(tlsStats, nil)

	// Replication Stats
	replStats := map[string]string{
		"provider": "ldap://replica.example.com:389",
		"delay":    "10",
	}
	mockClient.On("GetReplicationStatus").Return(replStats, nil)

	// Security Stats
	secStats := map[string]string{
		"simple_bind_total": "100",
		"sasl_bind_total":   "50",
	}
	mockClient.On("GetSecurityStats").Return(secStats, nil)

	// Performance Stats
	perfStats := map[string]string{
		"read_ops_completed": "200",
	}
	mockClient.On("GetPerformanceStats").Return(perfStats, nil)

	// 创建collector
	collector := New(cfg, log)

	// 替换ldapClientCreator为返回mock客户端的函数
	collector.ldapClientCreator = func(cfg *config.LDAPConfig, logger *logrus.Logger) (LDAPClientInterface, error) {
		return mockClient, nil
	}

	// 创建一个测试注册表
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// 验证所有指标都被正确收集（只验证部分关键指标）
	metric, err := registry.Gather()
	assert.NoError(t, err)
	assert.NotEmpty(t, metric)

	// 验证mock被正确调用
	mockClient.AssertExpectations(t)
}
