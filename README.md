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
  --web.listen-address=:9331
```

### 验证运行

访问以下端点验证服务是否正常运行：
- 指标端点: http://localhost:9330/metrics
- 健康检查: http://localhost:9330/healthz

## 收集的指标

- `openldap_up{}`：LDAP 服务是否可达（1=正常，0=异常）
- `openldap_entries_total{}`：目录中条目总数
- `openldap_monitor_connections_total{}`：当前连接数
- `openldap_monitor_operations_initiated_total{}`：各类操作发起次数（按操作类型分类）
- `openldap_custom_search_result_count{}`：自定义查询结果数量

## 参与贡献

1. Fork 本仓库
2. 创建功能分支 (git checkout -b feature/AmazingFeature)
3. 提交更改 (git commit -m 'Add some AmazingFeature')
4. 推送到分支 (git push origin feature/AmazingFeature)
5. 创建 Pull Request

## 许可证

请查看项目 LICENSE 文件了解详细信息。