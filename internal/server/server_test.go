package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	mockClient.On("GetTLSStats").Return(map[string]string{}, nil).Maybe()
	mockClient.On("GetReplicationStatus").Return(map[string]string{}, nil).Maybe()
	mockClient.On("GetSecurityStats").Return(map[string]string{}, nil).Maybe()
	mockClient.On("GetPerformanceStats").Return(map[string]string{}, nil).Maybe()

	// 设置collector使用mock客户端
	coll.SetLDAPClientCreatorForTest(func(cfg *config.LDAPConfig, logger *logrus.Logger) (collector.LDAPClientInterface, error) {
		return mockClient, nil
	})

	// 创建listener来获取随机端口
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().String()

	// 创建服务器实例
	server := New(addr, "/metrics", coll, logger)

	// 在goroutine中启动服务器
	go func() {
		// 由于测试只需要验证服务器能正常启动，我们忽略错误
		_ = server.RunWithListener(listener)
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
