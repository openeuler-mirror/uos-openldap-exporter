package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitee.com/openeuler/uos-openldap-exporter/internal/collector"
	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLDAPClient is a mock implementation of LDAPClientInterface for testing
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

func TestNew(t *testing.T) {
	// Arrange
	cfg := &config.Config{
		LDAP: config.LDAPConfig{
			Server:       "localhost:389",
			BindDN:       "cn=admin,dc=example,dc=com",
			BindPassword: "password",
			Timeout:      30 * time.Second,
		},
	}
	logger := logrus.New()
	coll := collector.New(cfg, logger)

	// Act
	server := New(":8080", "/metrics", coll, logger)

	// Assert
	assert.NotNil(t, server)
	assert.Equal(t, ":8080", server.addr)
	assert.Equal(t, "/metrics", server.metricsPath)
	assert.Equal(t, coll, server.collector)
	assert.Equal(t, logger, server.logger)
}

func TestLoggingMiddleware(t *testing.T) {
	// Arrange
	logger := logrus.New()
	logger.SetOutput(bytes.NewBuffer([]byte{})) // Discard logs

	cfg := &config.Config{
		LDAP: config.LDAPConfig{
			Server:  "localhost:389",
			Timeout: 30 * time.Second,
		},
	}
	coll := collector.New(cfg, logger)
	server := New(":8080", "/metrics", coll, logger)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Act
	wrappedHandler := server.loggingMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	// Assert
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleHealthCheckSuccess(t *testing.T) {
	// Arrange
	logger := logrus.New()
	logger.SetOutput(bytes.NewBuffer([]byte{})) // Discard logs

	cfg := &config.Config{
		LDAP: config.LDAPConfig{
			Server:       "localhost:389",
			BindDN:       "cn=admin,dc=example,dc=com",
			BindPassword: "password",
			Timeout:      30 * time.Second,
		},
	}

	coll := collector.New(cfg, logger)

	// 使用测试辅助函数设置ldapClientCreator
	mockClient := new(MockLDAPClient)
	mockClient.On("Close").Return()
	mockClient.On("CheckHealth").Return(true, "")

	coll.SetLDAPClientCreatorForTest(func(cfg *config.LDAPConfig, logger *logrus.Logger) (collector.LDAPClientInterface, error) {
		return mockClient, nil
	})

	server := New(":8080", "/metrics", coll, logger)

	// Create request and response recorder
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()

	// Act
	server.handleHealthCheck(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response HealthResponse
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response.Status)
}

func TestHandleHealthCheckFailure(t *testing.T) {
	// Arrange
	logger := logrus.New()
	logger.SetOutput(bytes.NewBuffer([]byte{})) // Discard logs

	cfg := &config.Config{
		LDAP: config.LDAPConfig{
			Server:       "invalid-server:389",
			BindDN:       "cn=admin,dc=example,dc=com",
			BindPassword: "password",
			Timeout:      1 * time.Second, // Short timeout for faster test
		},
	}

	coll := collector.New(cfg, logger)

	// 使用测试辅助函数设置ldapClientCreator
	coll.SetLDAPClientCreatorForTest(func(cfg *config.LDAPConfig, logger *logrus.Logger) (collector.LDAPClientInterface, error) {
		return nil, errors.New("connection failed")
	})

	server := New(":8080", "/metrics", coll, logger)

	// Create request and response recorder
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()

	// Act
	server.handleHealthCheck(rec, req)

	// Assert
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var response HealthResponse
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "error", response.Status)
	assert.NotEmpty(t, response.LDAP)
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	// Arrange
	rec := httptest.NewRecorder()
	wrapped := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	// Act
	wrapped.WriteHeader(http.StatusInternalServerError)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, wrapped.statusCode)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestServer_Run(t *testing.T) {
	// 创建测试用的logger
	logger := logrus.New()
	logger.SetOutput(io.Discard) // Discard logs

	// 创建测试配置
	cfg := &config.Config{
		LDAP: config.LDAPConfig{
			Server:       "localhost:389",
			BindDN:       "cn=admin,dc=example,dc=com",
			BindPassword: "password",
			Timeout:      1 * time.Second, // 短超时时间
		},
		CustomSearches: []config.CustomSearch{},
	}

	// 创建collector
	coll := collector.New(cfg, logger)

	// 创建mock LDAP客户端
	mockClient := new(MockLDAPClient)
	mockClient.On("Close").Return()
	mockClient.On("CheckHealth").Return(true, "").Maybe()
	mockClient.On("SearchCount", "", "(objectClass=*)").Return(0, nil).Maybe()
	mockClient.On("SearchMonitor", mock.Anything, mock.Anything).Return("0", nil).Maybe()

	// 设置collector使用mock客户端
	coll.SetLDAPClientCreatorForTest(func(cfg *config.LDAPConfig, logger *logrus.Logger) (collector.LDAPClientInterface, error) {
		return mockClient, nil
	})

	// 创建listener来获取随机端口
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	// 创建服务器实例
	server := New(addr, "/metrics", coll, logger)

	// 在goroutine中启动服务器
	go func() {
		// 由于测试只需要验证服务器能正常启动，我们忽略错误
		_ = server.Run()
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 发送测试请求
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 测试/healthz端点
	healthzURL := fmt.Sprintf("http://%s/healthz", addr)
	resp, err := client.Get(healthzURL)
	if assert.NoError(t, err, "Should be able to make request to /healthz") {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return 200 OK")
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"), "Should return JSON content type")
		
		// 验证响应体
		var healthResp HealthResponse
		err = json.NewDecoder(resp.Body).Decode(&healthResp)
		assert.NoError(t, err, "Should be able to decode JSON response")
		assert.Equal(t, "ok", healthResp.Status, "Should return status 'ok'")
	}

	// 测试/metrics端点
	metricsURL := fmt.Sprintf("http://%s/metrics", addr)
	resp, err = client.Get(metricsURL)
	if assert.NoError(t, err, "Should be able to make request to /metrics") {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return 200 OK")
	}
}

/*
func TestServer_Run_Implementation(t *testing.T) {
	// 创建测试用的logger
	logger := logrus.New()
	logger.SetOutput(bytes.NewBuffer([]byte{})) // Discard logs

	// 创建测试配置
	cfg := &config.Config{
		Web: config.WebConfig{
			ListenAddress: ":0", // 使用随机端口
			MetricsPath:   "/metrics",
		},
		LDAP: config.LDAPConfig{
			Server:       "localhost:389",
			BindDN:       "cn=admin,dc=example,dc=com",
			BindPassword: "password",
			Timeout:      30 * time.Second,
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "text",
		},
		CustomSearches: []config.CustomSearch{},
	}

	// 创建collector
	coll := collector.New(cfg, logger)

	// 创建mock LDAP客户端
	mockClient := new(MockLDAPClient)
	mockClient.On("Close").Return()
	mockClient.On("CheckHealth").Return(true, "").Maybe()
	mockClient.On("SearchCount", "", "(objectClass=*)").Return(0, nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Current,cn=Connections,cn=Monitor", "monitorCounter").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Total,cn=Connections,cn=Monitor", "monitorCounter").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Max File Descriptors,cn=Connections,cn=Monitor", "monitorCounter").Return("100", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Operations,cn=Monitor", "monitorOpActive").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Operations,cn=Monitor", "monitorOpPending").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Bind,cn=Operations,cn=Monitor", "monitorOpInitiated").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Bind,cn=Operations,cn=Monitor", "monitorOpCompleted").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Bind,cn=Operations,cn=Monitor", "monitorOpWaiting").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Unbind,cn=Operations,cn=Monitor", "monitorOpInitiated").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Unbind,cn=Operations,cn=Monitor", "monitorOpCompleted").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Unbind,cn=Operations,cn=Monitor", "monitorOpWaiting").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Search,cn=Operations,cn=Monitor", "monitorOpInitiated").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Search,cn=Operations,cn=Monitor", "monitorOpCompleted").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Search,cn=Operations,cn=Monitor", "monitorOpWaiting").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Modify,cn=Operations,cn=Monitor", "monitorOpInitiated").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Modify,cn=Operations,cn=Monitor", "monitorOpCompleted").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Modify,cn=Operations,cn=Monitor", "monitorOpWaiting").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Add,cn=Operations,cn=Monitor", "monitorOpInitiated").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Add,cn=Operations,cn=Monitor", "monitorOpCompleted").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Add,cn=Operations,cn=Monitor", "monitorOpWaiting").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Delete,cn=Operations,cn=Monitor", "monitorOpInitiated").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Delete,cn=Operations,cn=Monitor", "monitorOpCompleted").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Delete,cn=Operations,cn=Monitor", "monitorOpWaiting").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Bytes,cn=Statistics,cn=Monitor", "monitorCounter").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Entries,cn=Statistics,cn=Monitor", "monitorCounter").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Referrals,cn=Statistics,cn=Monitor", "monitorCounter").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Operations,cn=Statistics,cn=Monitor", "monitorCounter").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Active,cn=Threads,cn=Monitor", "monitoredInfo").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Starting,cn=Threads,cn=Monitor", "monitoredInfo").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Backing,cn=Threads,cn=Monitor", "monitoredInfo").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Pausing,cn=Threads,cn=Monitor", "monitoredInfo").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Pending,cn=Threads,cn=Monitor", "monitoredInfo").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Waiters,cn=Threads,cn=Monitor", "monitorCounter").Return("0", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Start,cn=Time,cn=Monitor", "monitorTimestamp").Return("20230101000000Z", nil).Maybe()
	mockClient.On("SearchMonitor", "cn=Current,cn=Time,cn=Monitor", "monitorTimestamp").Return("20230101000000Z", nil).Maybe()

	// 设置collector使用mock客户端
	coll.SetLDAPClientCreatorForTest(func(cfg *config.LDAPConfig, logger *logrus.Logger) (collector.LDAPClientInterface, error) {
		return mockClient, nil
	})

	// 创建服务器实例
	server := New(":0", "/metrics", coll, logger)

	// 在goroutine中启动服务器
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Run()
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 创建测试客户端
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// 测试/healthz端点
	resp, err := client.Get("http://localhost:9330/healthz")
	if err != nil {
		// 尝试获取实际监听地址
		t.Logf("Failed to connect to default port, trying to find actual address...")
		
		// 由于我们使用了":0"，我们需要找到实际分配的端口
		// 这里简化处理，直接跳过网络测试部分
		t.Skip("Skipping network tests due to dynamic port allocation complexity")
		return
	}
	defer resp.Body.Close()

	// 验证响应状态码
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 验证响应内容类型
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// 验证响应体
	var healthResp HealthResponse
	err = json.NewDecoder(resp.Body).Decode(&healthResp)
	assert.NoError(t, err)
	assert.Equal(t, "ok", healthResp.Status)

	// 测试/metrics端点
	resp, err = client.Get("http://localhost:9330/metrics")
	assert.NoError(t, err)
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// 不需要等待服务器错误，因为我们只是想验证它能正常启动和响应请求
	// 在实际测试中，我们会在这里关闭服务器
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Server error: %v", err)
		}
	default:
		// 服务器仍在运行，这是预期的
	}
}
