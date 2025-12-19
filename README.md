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
- 提供 `/healthz` 健康检查接口
- 支持 YAML 配置文件和命令行参数配置
- 结构化日志输出，支持多种日志级别
- 支持版本信息查看
- 配置验证功能，确保配置项的有效性

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
+------------------+
        ↑
+------------------+
| Config & Logging |
| (Viper + Logrus) |
+------------------+
```

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

custom_searches:
  - name: "user_count"
    base_dn: "ou=People,dc=example,dc=com"
    filter: "(objectClass=inetOrgPerson)"
```

运行 exporter：
```bash
./uos-openldap-exporter --config.file=config.yaml
```

### 命令行参数方式

```bash
./uos-openldap-exporter \
  --ldap.server=ldaps://ldap.example.com:636 \
  --ldap.bind-dn="cn=monitor,dc=example,dc=com" \
  --ldap.bind-password="secret" \
  --ldap.timeout=30s \
  --web.listen-address=:9331 \
  --log.level=debug
```

### 查看版本信息

```bash
./uos-openldap-exporter version
```

### 配置优先级

配置项的优先级顺序如下（从高到低）：
1. 命令行参数
2. 环境变量
3. 配置文件
4. 默认值

环境变量命名规则：将配置文件中的键名中的点（`.`）替换为下划线（`_`），并加上前缀`OPENLDAP_EXPORTER_`。
例如：`ldap.server` 对应环境变量 `OPENLDAP_EXPORTER_LDAP_SERVER`

### 配置验证

程序会在启动时对配置进行验证，如果配置不合法会输出错误信息并退出。验证规则包括：
- `ldap.server` 必须设置
- `ldap.timeout` 必须为正数
- `log.level` 必须是 debug/info/warn/error 之一
- `log.format` 必须是 text/json 之一
- `web.listen_address` 和 `web.metrics_path` 必须设置
- `custom_searches` 中的每一项都必须包含 name、base_dn 和 filter 字段

### 验证运行

访问以下端点验证服务是否正常运行：
- 指标端点: http://localhost:9330/metrics
- 健康检查: http://localhost:9330/healthz

## 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| --config.file | "" | 配置文件路径 |
| --web.listen-address | :9330 | 监听地址 |
| --web.metrics-path | /metrics | 指标暴露路径 |
| --ldap.server | "" | LDAP服务器地址 |
| --ldap.bind-dn | "" | 绑定DN |
| --ldap.bind-password | "" | 绑定密码 |
| --ldap.timeout | 10s | LDAP连接超时时间 |
| --ldap.start-tls | false | 是否启用StartTLS |
| --ldap.insecure-skip-verify | false | 是否跳过LDAP服务器证书验证（生产环境不推荐） |
| --log.level | info | 日志级别（debug, info, warn, error） |
| --log.format | text | 日志格式（text, json） |

## 收集的指标

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

## 许可证

本项目采用 Apache License 2.0 许可证。详情请见 [LICENSE](LICENSE) 文件。