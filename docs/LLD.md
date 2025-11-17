# OpenLDAP Exporter 低阶设计文档（LLD）

**项目名称**：OpenLDAP Exporter  
**模块路径**：`gitee.com/openeuler/uos-openldap-exporter`  
**目标语言**：Go 1.25.4  
**文档版本**：v1.0  
**撰写日期**：2025年11月17日  
**作者**：AI Assistant  

---

## 1. 文档目的

本文档提供 **OpenLDAP Exporter** 的详细实现设计，包括包结构、关键函数签名、数据结构定义、错误处理策略及单元测试要点，指导开发人员进行编码。

---

## 2. 项目结构

```
uos-openldap-exporter/
├── cmd/
│   └── root.go                 # CLI 入口
├── internal/
│   ├── config/
│   │   └── config.go           # 配置加载与结构体定义
│   ├── logger/
│   │   └── logger.go           # logrus 封装
│   ├── collector/
│   │   ├── collector.go        # Prometheus Collector 实现
│   │   └── ldap_client.go      # LDAP 连接与查询封装
│   └── server/
│       └── server.go           # HTTP 服务启动
├── pkg/
│   └── metrics/                # （可选）指标常量定义
├── go.mod
├── go.sum
├── README.md
└── config.example.yaml
```

---

## 3. 模块详细设计

### 3.1 `cmd/root.go`

#### 功能
- 定义命令行标志
- 加载配置
- 初始化日志、Collector、HTTP Server
- 启动服务

#### 关键代码结构
```go
var (
	cfgFile string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config.file", "", "Path to config file")
	// Web flags
	rootCmd.Flags().String("web.listen-address", ":9330", "Address to listen on")
	rootCmd.Flags().String("web.metrics-path", "/metrics", "Path under which to expose metrics")
	// LDAP flags
	rootCmd.Flags().String("ldap.server", "", "LDAP server URL (e.g., ldap://localhost:389)")
	rootCmd.Flags().String("ldap.bind-dn", "", "Bind DN for authentication")
	rootCmd.Flags().String("ldap.bind-password", "", "Bind password")
	// 绑定 viper
	viper.BindPFlag("web.listen_address", rootCmd.Flags().Lookup("web.listen-address"))
	// ... 其他绑定
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "uos-openldap-exporter",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load(cfgFile)
		log := logger.New(cfg.Log.Level)
		collector := collector.New(cfg, log)
		srv := server.New(cfg.Web.ListenAddress, cfg.Web.MetricsPath, collector, log)
		return srv.Run()
	},
}
```

---

### 3.2 `internal/config/config.go`

#### 结构体定义
```go
package config

import (
	"crypto/tls"
	"time"
)

type Config struct {
	Web            WebConfig        `mapstructure:"web"`
	LDAP           LDAPConfig       `mapstructure:"ldap"`
	Log            LogConfig        `mapstructure:"log"`
	CustomSearches []CustomSearch   `mapstructure:"custom_searches"`
}

type WebConfig struct {
	ListenAddress string `mapstructure:"listen_address"`
	MetricsPath   string `mapstructure:"metrics_path"`
}

type LDAPConfig struct {
	Server             string        `mapstructure:"server"`
	BindDN             string        `mapstructure:"bind_dn"`
	BindPassword       string        `mapstructure:"bind_password"`
	Timeout            time.Duration `mapstructure:"timeout"` // 默认 10s
	StartTLS           bool          `mapstructure:"start_tls"`
	InsecureSkipVerify bool          `mapstructure:"insecure_skip_verify"`
	tlsConfig          *tls.Config   // 私有字段，运行时构建
}

type LogConfig struct {
	Level string `mapstructure:"level"` // info, debug, warn, error
}

type CustomSearch struct {
	Name    string `mapstructure:"name"`
	BaseDN  string `mapstructure:"base_dn"`
	Filter  string `mapstructure:"filter"`
}
```

#### 加载逻辑
```go
func Load(configFile string) *Config {
	viper.SetConfigFile(configFile)
	viper.SetEnvPrefix("OPENLDAP_EXPORTER")
	viper.AutomaticEnv()

	// 设置默认值
	viper.SetDefault("web.listen_address", ":9330")
	viper.SetDefault("web.metrics_path", "/metrics")
	viper.SetDefault("ldap.timeout", "10s")
	viper.SetDefault("log.level", "info")

	if configFile != "" {
		_ = viper.ReadInConfig()
	}

	var cfg Config
	err := viper.Unmarshal(&cfg)
	if err != nil {
		panic(fmt.Errorf("failed to parse config: %w", err))
	}

	// 构建 TLS 配置
	if cfg.LDAP.InsecureSkipVerify {
		cfg.LDAP.tlsConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		cfg.LDAP.tlsConfig = &tls.Config{}
	}

	return &cfg
}
```

> **注意**：密码可通过环境变量 `OPENLDAP_EXPORTER_LDAP_BIND_PASSWORD` 注入。

---

### 3.3 `internal/logger/logger.go`

```go
package logger

import (
	"github.com/sirupsen/logrus"
)

func New(levelStr string) *logrus.Logger {
	log := logrus.New()
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return log
}
```

---

### 3.4 `internal/collector/ldap_client.go`

#### 功能
封装 LDAP 连接、认证、搜索操作。

```go
package collector

import (
	"context"
	"time"

	"github.com/go-ldap/ldap/v3"
	"uos-openldap-exporter/internal/config"
)

type LDAPClient struct {
	conn   *ldap.Conn
	config *config.LDAPConfig
	logger *logrus.Logger
}

func NewLDAPClient(cfg *config.LDAPConfig, logger *logrus.Logger) (*LDAPClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	var conn *ldap.Conn
	var err error

	if cfg.Server == "" {
		return nil, fmt.Errorf("ldap.server is required")
	}

	conn, err = ldap.DialURL(cfg.Server, ldap.DialWithTimeout(cfg.Timeout))
	if err != nil {
		return nil, fmt.Errorf("failed to dial LDAP: %w", err)
	}

	if cfg.StartTLS {
		err = conn.StartTLS(cfg.tlsConfig)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	if cfg.BindDN != "" {
		err = conn.Bind(cfg.BindDN, cfg.BindPassword)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("bind failed: %w", err)
		}
	}

	return &LDAPClient{conn: conn, config: cfg, logger: logger}, nil
}

func (c *LDAPClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// SearchCount 执行搜索并返回匹配条目数（不返回具体条目）
func (c *LDAPClient) SearchCount(baseDN, filter string) (int, error) {
	req := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, // no size limit for count (but risk large result)
		0,
		false,
		filter,
		[]string{"dn"}, // minimal attr
		nil,
	)

	res, err := c.conn.Search(req)
	if err != nil {
		return 0, err
	}
	return len(res.Entries), nil
}

// SearchMonitor 获取 cn=Monitor 下指定属性的值（如 monitorCounter）
func (c *LDAPClient) SearchMonitor(dn, attr string) (string, error) {
	res, err := c.conn.Search(ldap.NewSearchRequest(
		dn,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1, 0, false,
		"(objectClass=*)",
		[]string{attr},
		nil,
	))
	if err != nil || len(res.Entries) == 0 {
		return "", err
	}
	vals := res.Entries[0].GetAttributeValues(attr)
	if len(vals) == 0 {
		return "", fmt.Errorf("attribute %s not found in %s", attr, dn)
	}
	return vals[0], nil
}
```

> **优化建议**：未来可增加 `SizeLimit` 控制防爆内存，但计数需完整结果。

---

### 3.5 `internal/collector/collector.go`

#### 实现 `prometheus.Collector`

```go
package collector

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"uos-openldap-exporter/internal/config"
)

const (
	namespace = "openldap"
)

var (
	upDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Whether the OpenLDAP server is reachable.",
		[]string{"server"}, nil)

	entriesTotalDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "entries_total"),
		"Total number of entries in the directory.",
		[]string{"server"}, nil)

	monitorConnDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "connections_total"),
		"Current number of connections.",
		[]string{"server"}, nil)

	monitorOpsInitDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "monitor", "operations_initiated_total"),
		"Number of initiated operations.",
		[]string{"server", "operation"}, nil)

	customSearchDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "custom_search_result_count"),
		"Result count of custom LDAP search.",
		[]string{"server", "name"}, nil)
)

type OpenLDAPCollector struct {
	config *config.Config
	logger *logrus.Logger
}

func New(cfg *config.Config, logger *logrus.Logger) *OpenLDAPCollector {
	return &OpenLDAPCollector{config: cfg, logger: logger}
}

func (c *OpenLDAPCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- upDesc
	ch <- entriesTotalDesc
	ch <- monitorConnDesc
	ch <- monitorOpsInitDesc
	ch <- customSearchDesc
}

func (c *OpenLDAPCollector) Collect(ch chan<- prometheus.Metric) {
	labels := prometheus.Labels{"server": c.config.LDAP.Server}

	client, err := NewLDAPClient(&c.config.LDAP, c.logger)
	up := 1.0
	if err != nil {
		c.logger.Errorf("Failed to connect to LDAP: %v", err)
		up = 0.0
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, up, labels["server"])
		return // 无法继续采集
	}
	defer client.Close()

	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, up, labels["server"])

	// Total entries
	if count, err := client.SearchCount("", "(objectClass=*)"); err == nil {
		ch <- prometheus.MustNewConstMetric(entriesTotalDesc, prometheus.GaugeValue, float64(count), labels["server"])
	} else {
		c.logger.Warnf("Failed to get total entries: %v", err)
	}

	// Monitor: connections
	if val, err := client.SearchMonitor("cn=Connections,cn=Monitor", "monitorCounter"); err == nil {
		if n, err := strconv.ParseFloat(val, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(monitorConnDesc, prometheus.GaugeValue, n, labels["server"])
		}
	}

	// Monitor: operations (initiated)
	ops := []string{"Bind", "Unbind", "Search", "Modify", "Add", "Delete"}
	for _, op := range ops {
		dn := "cn=" + op + ",cn=Operations,cn=Monitor"
		if val, err := client.SearchMonitor(dn, "monitorOpInitiated"); err == nil {
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				opLabels := prometheus.Labels{"server": labels["server"], "operation": op}
				ch <- prometheus.MustNewConstMetric(monitorOpsInitDesc, prometheus.CounterValue, n, opLabels["server"], opLabels["operation"])
			}
		}
	}

	// Custom searches
	for _, cs := range c.config.CustomSearches {
		if count, err := client.SearchCount(cs.BaseDN, cs.Filter); err == nil {
			ch <- prometheus.MustNewConstMetric(customSearchDesc, prometheus.GaugeValue, float64(count), labels["server"], cs.Name)
		} else {
			c.logger.Warnf("Custom search '%s' failed: %v", cs.Name, err)
		}
	}
}
```

---

### 3.6 `internal/server/server.go`

```go
package server

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"uos-openldap-exporter/internal/collector"
)

type Server struct {
	addr        string
	metricsPath string
	collector   *collector.OpenLDAPCollector
	logger      *logrus.Logger
}

func New(addr, metricsPath string, coll *collector.OpenLDAPCollector, logger *logrus.Logger) *Server {
	return &Server{
		addr:        addr,
		metricsPath: metricsPath,
		collector:   coll,
		logger:      logger,
	}
}

func (s *Server) Run() error {
	prometheus.MustRegister(s.collector)

	mux := http.NewServeMux()
	mux.Handle(s.metricsPath, promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	s.logger.Infof("Starting server on %s", s.addr)
	return http.ListenAndServe(s.addr, mux)
}
```

---

## 4. 错误处理策略

| 场景 | 处理方式 |
|------|--------|
| 配置加载失败 | `panic`（启动期致命错误） |
| LDAP 连接失败 | `openldap_up = 0`，记录 error，返回 |
| 单个指标查询失败 | 记录 warn，跳过该指标，不影响其他 |
| 自定义搜索失败 | 同上，不影响核心指标 |
| HTTP 服务启动失败 | 返回 error，进程退出 |

---

## 5. 单元测试要点

| 模块 | 测试重点 |
|------|--------|
| `config` | 默认值、环境变量覆盖、YAML 解析 |
| `ldap_client` | Dial、Bind、SearchCount、SearchMonitor（需 mock ldap.Conn） |
| `collector` | 指标标签正确性、错误路径覆盖（使用 mock client） |
| `server` | 路由注册、/healthz 返回 |

> **Mock 建议**：使用 `github.com/stretchr/testify/mock` 或接口抽象 `LDAPClientInterface`。

---

## 6. 部署与验证步骤

1. 编译：
   ```bash
   GOOS=linux GOARCH=amd64 go build -o uos-openldap-exporter ./cmd/
   ```
2. 准备配置文件 `config.yaml`
3. 启动：
   ```bash
   ./uos-openldap-exporter --config.file=config.yaml
   ```
4. 验证：
   ```bash
   curl http://localhost:9330/metrics | grep openldap
   curl http://localhost:9330/healthz
   ```

---

## 7. 附录：指标清单（v1.0）

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `openldap_up` | Gauge | `server` | 1=正常，0=异常 |
| `openldap_entries_total` | Gauge | `server` | 目录总条目数 |
| `openldap_monitor_connections_total` | Gauge | `server` | 当前连接数 |
| `openldap_monitor_operations_initiated_total` | Counter | `server`, `operation` | 各类操作发起次数（Bind/Search等） |
| `openldap_custom_search_result_count` | Gauge | `server`, `name` | 自定义查询结果数量 |

---

> **备注**：本 LLD 对应 PRD/HLD 中 v1.0 MVP 功能。所有代码应遵循 Go 最佳实践，包含注释和错误处理。