# OpenLDAP Exporter 高阶设计文档（HLD）

**项目名称**：OpenLDAP Exporter  
**模块路径**：`gitee.com/openeuler/uos-openldap-exporter`  
**目标语言**：Go 1.25.4  
**文档版本**：v1.0  
**撰写日期**：2025年11月17日  


---

## 1. 文档目的

本文档旨在定义 **OpenLDAP Exporter** 的整体架构、模块划分、数据流、接口规范及关键设计决策，为后续详细开发（LLD）和代码实现提供技术蓝图。

---

## 2. 系统上下文

### 2.1 系统定位
OpenLDAP Exporter 是一个 **Prometheus Exporter** 类型的监控代理，运行在与 OpenLDAP 服务同网络可达的主机上（通常为同一节点或监控代理节点），定期采集 LDAP 服务状态与性能指标，并以 Prometheus 标准格式暴露 HTTP 接口供 Prometheus Server 抓取。

### 2.2 外部依赖
| 组件 | 作用 |
|------|------|
| OpenLDAP Server (≥2.4) | 数据源，需启用 `cn=Monitor` 子树 |
| Prometheus Server | 指标消费者，通过 HTTP 抓取 `/metrics` |
| （可选）Grafana | 可视化展示 |

### 2.3 部署拓扑
```
+------------------+       HTTP        +---------------------+
| Prometheus       | <---------------- | OpenLDAP Exporter   |
| (scrapes /metrics)|                   | (runs on host)      |
+------------------+                   +----------+----------+
                                                  |
                                                  | LDAP/TLS
                                                  v
                                         +---------------------+
                                         | OpenLDAP Server     |
                                         | (with cn=Monitor)   |
                                         +---------------------+
```

---

## 3. 架构概览

系统采用 **单进程、事件驱动、无状态** 设计，核心由以下模块组成：

```
┌───────────────────────────────────────────────────────┐
│                uos-openldap-exporter                  │
├─────────────┬───────────────┬──────────────┬──────────┤
│  CLI &      │ Configuration │   Collector  │  HTTP    │
│  Entrypoint │   (Viper)     │   (Metrics)  │  Server  │
└──────┬──────┴───────┬───────┴───────┬──────┴─────┬────┘
       │              │               │            │
       ▼              ▼               ▼            ▼
   cobra/viper    config struct   ldap.Client   promhttp
```

- **CLI & Entrypoint**：解析命令行参数，启动主流程。
- **Configuration**：统一管理配置来源（文件/命令行/环境变量）。
- **Collector**：核心逻辑模块，负责连接 LDAP、执行查询、生成指标。
- **HTTP Server**：暴露 `/metrics` 和 `/healthz`。

---

## 4. 模块详细设计

### 4.1 CLI 与启动入口（`cmd/`）

- 使用 `cobra` 构建命令行结构。
- 支持子命令（未来扩展用），当前仅 `rootCmd`。
- 参数示例：
  ```go
  --config.file string
  --web.listen-address string (default ":9330")
  --web.metrics-path string (default "/metrics")
  --log.level string (default "info")
  --ldap.server string
  --ldap.bind-dn string
  --ldap.bind-password string
  ```

### 4.2 配置管理（`internal/config/`）

- 使用 `viper` 绑定配置结构体。
- 支持 YAML 文件 + 命令行覆盖。
- 敏感字段（如密码）可通过 `LDAP_BIND_PASSWORD` 环境变量注入。
- 配置结构体定义：
  ```go
  type Config struct {
      Web struct {
          ListenAddress string `mapstructure:"listen_address"`
          MetricsPath   string `mapstructure:"metrics_path"`
      }
      LDAP struct {
          Server             string        `mapstructure:"server"`
          BindDN             string        `mapstructure:"bind_dn"`
          BindPassword       string        `mapstructure:"bind_password"`
          Timeout            time.Duration `mapstructure:"timeout"`
          StartTLS           bool          `mapstructure:"start_tls"`
          InsecureSkipVerify bool          `mapstructure:"insecure_skip_verify"`
          TLSConfig          *tls.Config   // 运行时构建
      }
      Log struct {
          Level string `mapstructure:"level"`
      }
      CustomSearches []CustomSearch `mapstructure:"custom_searches"`
  }

  type CustomSearch struct {
      Name    string `mapstructure:"name"`
      BaseDN  string `mapstructure:"base_dn"`
      Filter  string `mapstructure:"filter"`
  }
  ```

### 4.3 指标采集器（`internal/collector/`）

#### 4.3.1 Collector 接口
实现 `prometheus.Collector` 接口，支持动态指标注册。

```go
type OpenLDAPCollector struct {
    config *config.Config
    logger *logrus.Logger
}
func (c *OpenLDAPCollector) Describe(ch chan<- *prometheus.Desc)
func (c *OpenLDAPCollector) Collect(ch chan<- prometheus.Metric)
```

#### 4.3.2 采集流程
1. **建立连接**：
   - 解析 `ldap://` 或 `ldaps://`
   - 支持 StartTLS（若配置）
   - 设置超时（默认 10s）
2. **绑定认证**：
   - 使用 `BindDN` + `BindPassword`
   - 匿名绑定（若未配置）
3. **执行查询**：
   - **基础连通性**：简单搜索 `(objectClass=*)` 限制返回 1 条 → `openldap_up`
   - **总条目数**：搜索 `(objectClass=*)` 不限制 → `openldap_entries_total`
   - **Monitor 指标**：读取 `cn=Monitor` 下子项（如 `cn=Connections`, `cn=Operations`）
   - **自定义搜索**：遍历 `custom_searches` 列表，执行计数查询
4. **错误处理**：
   - 任何步骤失败 → `openldap_up = 0`
   - 记录 error 日志，但不中断其他指标（部分成功仍上报）

#### 4.3.3 关键指标定义（Prometheus Desc）
```go
var (
    upDesc = prometheus.NewDesc(
        "openldap_up",
        "Whether the OpenLDAP server is reachable.",
        []string{"server"}, nil)

    entriesTotalDesc = prometheus.NewDesc(
        "openldap_entries_total",
        "Total number of entries in the directory.",
        []string{"server"}, nil)

    monitorConnDesc = prometheus.NewDesc(
        "openldap_monitor_connections_total",
        "Current number of connections.",
        []string{"server"}, nil)

    customSearchDesc = prometheus.NewDesc(
        "openldap_custom_search_result_count",
        "Result count of custom LDAP search.",
        []string{"server", "name"}, nil)
)
```

### 4.4 HTTP 服务（`internal/server/`）

- 使用 `prometheus/client_golang` 的 `promhttp` 工具。
- 注册自定义 Collector。
- 同时暴露：
  - `/metrics`：指标端点
  - `/healthz`：返回 `{ "status": "ok" }` 或 503
- 自动包含 Go runtime 指标（`go_goroutines`, `process_cpu_seconds_total` 等）

### 4.5 日志系统（`internal/logger/`）

- 封装 `logrus`，支持结构化日志。
- 日志级别由配置控制。
- 关键事件记录：
  - 启动参数
  - LDAP 连接成功/失败
  - 采集耗时（debug 级别）
  - 自定义搜索执行

---

## 5. 安全设计

| 风险 | 缓解措施 |
|------|--------|
| 密码明文泄露 | - 禁止在日志中打印密码<br>- 建议通过环境变量传入<br>- 配置文件权限应为 600 |
| 中间人攻击 | - 强制验证 TLS 证书（默认）<br>- `insecure_skip_verify` 需显式开启 |
| 未授权访问 | - Exporter 本身无认证，应部署在可信网络<br>- 建议配合防火墙或 reverse proxy 增加 basic auth（外部方案） |
| LDAP 查询 DoS | - 所有搜索均设置 `SizeLimit=1`（除计数场景）<br>- 超时控制（默认 10s） |

---

## 6. 错误处理与可观测性

- **Exporter 自身健康**：`/healthz` 返回 200 即表示进程存活。
- **LDAP 不可用**：`openldap_up{server="..."} 0`，同时记录 error。
- **部分指标失败**：不影响其他指标采集（如 Monitor 不可用，仍可采集 entries）。
- **采集延迟**：通过 `promhttp_metric_handler_requests_in_flight` 监控。

---

## 7. 性能与资源

- **并发模型**：单线程轮询（Prometheus 拉模型天然串行）。
- **连接复用**：每次 `/metrics` 请求新建 LDAP 连接（避免长连接状态不一致），但可后续优化为连接池（v1.1）。
- **内存占用**：仅缓存配置和临时结果，无持久状态。
- **CPU 开销**：低，仅在抓取时活跃。

---

## 8. 扩展性设计

- **多目标支持**：当前为单实例，未来可通过 `target` URL 参数或配置列表支持多 LDAP 实例。
- **插件化采集**：Collector 内部可拆分为多个子采集器（MonitorCollector, EntryCounter 等）。
- **指标标签扩展**：预留 `labels map[string]string` 字段用于注入 DC、环境等元数据。

---

## 9. 构建与部署

- **构建命令**：
  ```bash
  go build -o uos-openldap-exporter ./cmd/
  ```
- **依赖管理**：Go Modules，锁定版本。
- **二进制兼容性**：静态编译，适配 openEuler/UOS glibc 环境。
- **systemd 示例**：
  ```ini
  [Unit]
  Description=OpenLDAP Exporter
  After=network.target

  [Service]
  ExecStart=/usr/bin/uos-openldap-exporter --config.file=/etc/openldap-exporter/config.yaml
  Restart=always
  User=prometheus

  [Install]
  WantedBy=multi-user.target
  ```

---

## 10. 附录：关键数据结构与接口

### 10.1 主函数流程
```go
func main() {
    cfg := config.Load()
    logger := logger.New(cfg.Log.Level)
    collector := collector.New(cfg, logger)
    server := server.New(cfg.Web, collector, logger)
    server.Run()
}
```

### 10.2 LDAP 连接伪代码
```go
func connectLDAP(cfg *LDAPConfig) (*ldap.Conn, error) {
    conn, err := ldap.DialURL(cfg.Server, ldap.DialWithTimeout(cfg.Timeout))
    if cfg.StartTLS {
        err = conn.StartTLS(cfg.TLSConfig)
    }
    if cfg.BindDN != "" {
        err = conn.Bind(cfg.BindDN, cfg.BindPassword)
    }
    return conn, err
}
```

---

> **备注**：本 HLD 聚焦于 v1.0 MVP 功能。后续可根据 PRD 中的扩展需求进行架构演进。