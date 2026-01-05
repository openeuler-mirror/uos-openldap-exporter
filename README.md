# uos-openldap-exporter

## 项目介绍

uos-openldap-exporter 是一个针对 OpenLDAP 的 Prometheus 监控指标导出器。它能够连接到 OpenLDAP 服务器，收集关键的服务指标，并以 Prometheus 标准格式通过 HTTP 接口暴露，方便集成到 Prometheus 监控系统中。

该导出器支持多种配置选项，包括 LDAP 连接参数、TLS/StartTLS 加密、绑定认证以及自定义 LDAP 查询等功能。

## 功能特性

- 连接 OpenLDAP 服务器并收集关键指标
- 支持 LDAP 和 LDAPS 协议
- 支持 StartTLS 加密连接
- 支持绑定认证和匿名连接
- 支持从 `cn=Monitor` 子树收集性能指标
- 支持自定义 LDAP 查询和计数
- 通过 `/metrics` 接口暴露 Prometheus 格式指标
- 提供 `/healthz` 健康检查接口，实际检测 LDAP 连接状态
- 支持 YAML 配置文件和命令行参数配置
- 结构化日志输出，支持多种日志级别和格式
- 支持版本信息查看
- 配置验证功能，确保配置项的有效性
- 支持日志轮转和文件输出
- 生产级架构设计，支持可测试性和可扩展性
- 更全面的指标收集，包括SSL/TLS、复制状态、性能和安全相关指标
- **插件架构**：允许用户开发自定义指标收集插件
- **指标过滤**：支持按需启用/禁用特定指标

## 软件架构

```
+------------------+
|   HTTP Server    | ← /metrics, /healthz
+------------------+
        ↑
+------------------+
|  Metrics Collector|
|  - Connect LDAP  |
|  - Query Monitor |
|  - Custom Search |
|  - Plugin System |
+------------------+
        ↑
+------------------+
| Config & Logging |
| (Viper + Logrus) |
+------------------+
```

### 架构特点

1. **模块化设计**：
   - `cmd/` - 命令行接口
   - `internal/collector/` - 指标收集器
   - `internal/config/` - 配置管理
   - `internal/logger/` - 日志系统
   - `internal/server/` - HTTP服务

2. **插件架构**：
   - 通过 `PluginCollector` 接口实现插件系统
   - 支持动态注册和启用/禁用插件
   - 提供 `PluginManager` 管理插件生命周期
   - 插件可独立开发、测试和部署

3. **接口抽象**：
   - 使用接口隔离具体实现，提高代码可测试性
   - LDAP客户端通过接口定义，便于模拟测试

4. **中间件支持**：
   - HTTP服务支持中间件，如日志记录
   - 便于扩展功能，如认证、限流等

5. **生产级特性**：
   - 完善的错误处理机制
   - 请求超时控制防止Slowloris攻击
   - 结构化日志和日志轮转
   - 健康检查和指标收集分离

## 安装教程

1. 确保已安装 Go 1.25.4 或更高版本
2. 克隆项目代码：
   ```bash
   git clone https://gitee.com/openeuler/uos-openldap-exporter.git
   ```
3. 进入项目目录并构建：
   ```bash
   cd uos-openldap-exporter
   go build -o uos-openldap-exporter ./cmd/
   ```

## 使用说明

### 配置文件方式

创建配置文件 `config.yaml`：

```yaml
ldap:
  server: "ldap://localhost:389"
  bind_dn: "cn=admin,dc=example,dc=com"
  bind_password: "secret"
  timeout: 10s
  start_tls: false
  insecure_skip_verify: false

web:
  listen_address: ":9330"
  metrics_path: "/metrics"

log:
  level: "info"
  format: "text"  # 可选值: "text" 或 "json"，默认为"text"
  output: "/var/log/openldap-exporter.log"  # 日志输出文件路径，默认为 stdout
  max_size: 100   # 每个日志文件最大大小(MB)，默认100
  max_age: 30     # 保留旧日志文件的最大天数，默认30
  max_backups: 3  # 保留旧日志文件的最大个数，默认3
  local_time: false  # 是否使用本地时间，默认false(UTC)
  compress: false    # 是否压缩轮转的日志文件，默认false

plugins:
  enabled:  # 指定启用的插件，留空表示启用所有插件
    # - "base_connection"
    # - "monitor_specific"
    # - "security"

custom_searches:
  - name: "user_count"
    base_dn: "ou=People,dc=example,dc=com"
    filter: "(objectClass=inetOrgPerson)"
  - name: "group_count"
    base_dn: "ou=Groups,dc=example,dc=com"
    filter: "(objectClass=groupOfNames)"
```

运行 exporter：
```bash
./uos-openldap-exporter --config.file=config.yaml
```

### 命令行方式

```bash
# 使用配置文件启动
./uos-openldap-exporter --config.file=config.yaml

# 使用命令行参数启动
./uos-openldap-exporter --ldap.server=ldaps://ldap.example.com:636 --web.listen-address=:9331
```

### 插件系统使用

#### 启用特定插件

在配置文件中指定启用的插件：

```yaml
plugins:
  enabled:
    - "base_connection"
    - "security"
```

#### 开发自定义插件

要开发自定义插件，需要实现 `PluginCollector` 接口：

```go
type MyPlugin struct {
    BasePluginCollector
    myMetricDesc *prometheus.Desc
}

func NewMyPlugin() *MyPlugin {
    return &MyPlugin{
        myMetricDesc: prometheus.NewDesc(
            prometheus.BuildFQName(namespace, "my_plugin", "my_metric"),
            "Description of my metric.",
            []string{"server"}, nil,
        ),
        BasePluginCollector: BasePluginCollector{enabled: true},
    }
}

func (p *MyPlugin) Name() string {
    return "my_plugin"
}

func (p *MyPlugin) Describe(ch chan<- *prometheus.Desc) {
    ch <- p.myMetricDesc
}

func (p *MyPlugin) Collect(ch chan<- prometheus.Metric, client LDAPClientInterface, server string) error {
    labels := prometheus.Labels{"server": server}
    ch <- prometheus.MustNewConstMetric(p.myMetricDesc, prometheus.GaugeValue, 42, labels["server"])
    return nil
}
```

在主程序中注册插件：

```go
myPlugin := NewMyPlugin()
collector.GetPluginManager().RegisterPlugin(myPlugin)
```

### 环境变量

以下环境变量可用于配置导出器：

- `OPENLDAP_EXPORTER_SERVER` - LDAP服务器地址
- `OPENLDAP_EXPORTER_BIND_DN` - 绑定DN
- `OPENLDAP_EXPORTER_BIND_PASSWORD` - 绑定密码
- `OPENLDAP_EXPORTER_LISTEN_ADDRESS` - 监听地址
- `OPENLDAP_EXPORTER_METRICS_PATH` - 指标路径
- `OPENLDAP_EXPORTER_LOG_LEVEL` - 日志级别

### 配置验证

配置文件必须包含以下要求：

- `ldap.server` 必须设置
- `web.listen_address` 和 `web.metrics_path` 必须设置
- `custom_searches` 中的每一项都必须包含 name、base_dn 和 filter 字段

### 健康检查

`/healthz` 端点提供真实的健康检查功能，它会：
1. 尝试连接到配置的 LDAP 服务器
2. 如果配置了 StartTLS，则启动 TLS
3. 如果配置了绑定凭据，则执行绑定操作
4. 执行轻量级的 WhoAmI 操作验证连接
5. 返回 JSON 格式的健康状态

健康的响应示例：
```json
{
  "status": "ok"
}
```

不健康的响应示例：
```json
{
  "status": "error",
  "ldap": "Failed to connect to LDAP server: ..."
}
```

健康检查的实现位于 `internal/collector/health_check.go`，它独立于指标收集逻辑，可以直接调用 LDAP 连接和验证功能。

### 配置重载

目前不支持运行时配置重载，需要重启服务才能使配置变更生效。

## 开发指南

### 项目结构

```
uos-openldap-exporter/
├── cmd/                    # 命令行接口
├── internal/
│   ├── collector/          # 指标收集器
│   ├── config/             # 配置管理
│   ├── logger/             # 日志系统
│   └── server/             # HTTP服务
├── scripts/                # 脚本
├── Dockerfile              # Docker配置
├── Makefile                # 构建脚本
├── README.md               # 项目文档
└── config.example.yaml     # 配置示例
```

### 插件开发

要开发插件，需要：

1. 实现 `PluginCollector` 接口
2. 继承 `BasePluginCollector` 以获得基本功能
3. 在 `Describe` 方法中定义指标描述符
4. 在 `Collect` 方法中收集指标数据
5. 使用 `Name` 方法返回插件唯一标识符

### 基础指标

| 指标名称 | 类型 | 含义 |
|---------|------|-----|
| openldap_up | Gauge | OpenLDAP服务器是否可达 |
| openldap_entries_total | Gauge | 目录中的条目总数 |

### 连接指标

| 指标名称 | 类型 | 含义 |
|---------|------|-----|
| openldap_monitor_current_connections | Gauge | 当前连接客户端数量 |
| openldap_monitor_total_connections | Counter | 服务器启动以来的总连接数 |
| openldap_monitor_max_connections | Gauge | 服务器配置允许的最大连接数 |

### 操作指标

| 指标名称 | 类型 | 含义 |
|---------|------|-----|
| openldap_monitor_active_operations | Gauge | 当前活跃操作数 |
| openldap_monitor_pending_operations | Gauge | 待处理操作数 |
| openldap_monitor_operations_initiated_total | Counter | 已发起的操作总数（按操作类型分类） |
| openldap_monitor_operations_completed_total | Counter | 已完成的操作总数（按操作类型分类） |
| openldap_monitor_operations_waiting | Gauge | 等待中的操作数（按操作类型分类） |

### 统计指标

| 指标名称 | 类型 | 含义 |
|---------|------|-----|
| openldap_monitor_statistics | Counter | 各类统计数据（按统计类型分类） |

### 线程池指标

| 指标名称 | 类型 | 含义 |
|---------|------|-----|
| openldap_monitor_threads | Gauge | 线程池统计信息（按线程状态分类） |
| openldap_monitor_waiters | Gauge | 等待资源的线程数 |

### 时间指标

| 指标名称 | 类型 | 含义 |
|---------|------|-----|
| openldap_monitor_time_seconds | Gauge | 系统时间指标（启动时间和当前时间） |

### 自定义搜索指标

| 指标名称 | 类型 | 含义 |
|---------|------|-----|
| openldap_custom_search_result_count | Gauge | 自定义LDAP搜索的结果计数 |

### SSL/TLS 相关指标

| 指标名称 | 类型 | 含义 |
|---------|------|-----|
| openldap_tls_connections_total | Counter | 建立的TLS连接总数 |
| openldap_tls_active_connections | Gauge | 当前活跃的TLS连接数 |
| openldap_tls_starttls_success_total | Counter | 成功的StartTLS操作总数 |
| openldap_tls_starttls_failure_total | Counter | 失败的StartTLS操作总数 |

### 复制状态指标

| 指标名称 | 类型 | 含义 |
|---------|------|-----|
| openldap_replication_provider_status | Gauge | 复制提供者状态（1=正常，0=异常） |
| openldap_replication_consumer_status | Gauge | 复制消费者状态（1=正常，0=异常） |
| openldap_replication_provider_delay_seconds | Gauge | 复制延迟（秒） |
| openldap_replication_provider_last_update_time_seconds | Gauge | 最后复制更新时间戳 |

### 性能指标

| 指标名称 | 类型 | 含义 |
|---------|------|-----|
| openldap_performance_operation_response_time_seconds | Counter | LDAP操作响应时间（秒） |

### 安全相关指标

| 指标名称 | 类型 | 含义 |
|---------|------|-----|
| openldap_security_authentication_success_total | Counter | 成功认证总数 |
| openldap_security_authentication_failure_total | Counter | 失败认证总数 |
| openldap_security_sasl_bind_total | Counter | SASL绑定操作总数 |
| openldap_security_simple_bind_total | Counter | 简单绑定操作总数 |
| openldap_security_strong_auth_total | Counter | 强认证操作总数 |

## 构建与测试

### 构建

```bash
# 本地构建
go build -o uos-openldap-exporter main.go

# 使用 Makefile 构建
make build

# 多平台构建
make release
```

### 测试

```bash
# 运行测试
make test

# 测试覆盖率
make test-cover

# 安全检查
make sec
```

### Docker 镜像

```bash
# 构建 Docker 镜像
make docker-build

# 推送镜像
make docker-push
```

## 部署

### Docker 部署

```bash
docker run -d -p 9330:9330 \
  -v /path/to/config.yaml:/etc/openldap-exporter/config.yaml \
  uos-openldap-exporter:<version>
```

### Kubernetes 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: openldap-exporter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: openldap-exporter
  template:
    metadata:
      labels:
        app: openldap-exporter
    spec:
      containers:
      - name: openldap-exporter
        image: uos-openldap-exporter:latest
        ports:
        - containerPort: 9330
        volumeMounts:
        - name: config
          mountPath: /etc/openldap-exporter
      volumes:
      - name: config
        configMap:
          name: openldap-exporter-config
```

## 贡献

欢迎提交 Issue 和 Pull Request 来改进项目。

## 许可证

本项目采用 MulanPSL-2.0 许可证。