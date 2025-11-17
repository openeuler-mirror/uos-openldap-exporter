# OpenLDAP Exporter 产品需求文档（PRD）

**项目名称**：OpenLDAP Exporter  
**模块路径**：`gitee.com/openeuler/uos-openldap-exporter`  
**目标语言**：Go 1.25.4  
**文档版本**：v1.0  
**撰写日期**：2025年11月17日  

---

## 1. 背景与目标

### 1.1 背景
OpenLDAP 是广泛使用的开源 LDAP（轻量级目录访问协议）实现，常用于集中身份认证、用户/组信息管理等场景。在企业运维和监控体系中，需要对 OpenLDAP 的运行状态、性能指标进行可观测性采集，以便集成到 Prometheus + Grafana 监控栈中。

目前社区虽有部分 LDAP exporter，但存在以下问题：
- 不支持最新 Go 版本或依赖过旧；
- 缺乏对国产操作系统（如 openEuler/UOS）的适配验证；
- 功能单一，无法灵活配置监控项；
- 缺少完善的日志、错误处理和安全机制。

### 1.2 目标
开发一个轻量、安全、可配置的 **OpenLDAP Exporter**，用于：
- 定期从 OpenLDAP 服务拉取关键指标；
- 暴露标准 Prometheus 格式的 `/metrics` 接口；
- 支持 TLS/StartTLS、绑定认证、连接池等企业级特性；
- 适配 openEuler / UOS 环境，作为系统级监控组件集成。

---

## 2. 功能需求

### 2.1 核心功能

| 功能 | 描述 |
|------|------|
| **LDAP 连接管理** | 支持通过配置文件或命令行参数指定 LDAP 服务器地址、端口、协议（ldap/ldaps）、绑定 DN 和密码 |
| **指标采集** | 采集以下核心指标（可扩展）：<br>- `openldap_up{}`：LDAP 服务是否可达（1=正常，0=异常）<br>- `openldap_entries_total{}`：目录中条目总数<br>- `openldap_monitor_*`：从 `cn=Monitor` 子树提取的性能指标（如操作数、连接数、线程使用等）<br>- 自定义搜索查询结果计数（如用户数、组数） |
| **Prometheus 暴露** | 启动 HTTP 服务，默认监听 `:9330`，提供 `/metrics` 接口 |
| **配置管理** | 支持 YAML 配置文件 + 命令行参数（优先级：命令行 > 配置文件） |
| **日志输出** | 使用结构化日志（logrus），支持日志级别控制（debug/info/warn/error） |
| **健康检查** | 提供 `/healthz` 接口返回 exporter 自身健康状态 |

### 2.2 可选/扩展功能（v1.1+）
- 支持多 LDAP 实例监控（targets 列表）
- 支持自定义 LDAP 查询模板（如统计特定 OU 下的用户）
- 指标标签注入（如 `instance`, `dc` 等）
- 支持 SASL 认证（GSSAPI 等）

---

## 3. 非功能需求

| 类别 | 要求 |
|------|------|
| **性能** | 单次采集延迟 < 500ms（默认配置下）；内存占用 < 20MB |
| **安全性** | - 密码不得明文打印日志<br>- 支持 TLS 证书验证（可选跳过）<br>- 绑定凭证建议通过环境变量或文件传入 |
| **兼容性** | - 支持 OpenLDAP 2.4+（含 cn=Monitor）<br>- 适配 openEuler 22.03 LTS / UOS V20<br>- Go 1.25.4 编译通过 |
| **可观测性** | exporter 自身暴露 `go_*`, `process_*`, `promhttp_*` 等标准指标 |
| **部署方式** | 支持二进制直接运行、systemd 服务、容器化（Dockerfile 可选） |

---

## 4. 技术方案

### 4.1 依赖清单
```go
require (
	github.com/prometheus/client_golang v1.23.2   // Prometheus client
	github.com/spf13/cobra v1.10.1                // CLI 命令解析
	github.com/spf13/viper v1.21.0                // 配置管理
	github.com/sirupsen/logrus v1.9.3             // 结构化日志
	github.com/go-ldap/ldap/v3 v3.4.12            // LDAP 客户端
)
```

### 4.2 架构设计
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

### 4.3 关键指标说明

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `openldap_up` | gauge | `server` | LDAP 服务连通性 |
| `openldap_entries_total` | gauge | `server` | 总条目数（通过 `(objectClass=*)` 查询） |
| `openldap_monitor_connections_total` | gauge | `server` | 当前连接数 |
| `openldap_monitor_operations_initiated_total` | counter | `server`, `operation` | 各类操作发起次数 |
| `openldap_custom_search_result_count` | gauge | `server`, `name` | 自定义查询结果数量 |

> 注：`cn=Monitor` 是 OpenLDAP 的监控子系统，需在 slapd.conf 中启用。

---

## 5. 配置示例

### 5.1 config.yaml
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
  - name: "group_count"
    base_dn: "ou=Groups,dc=example,dc=com"
    filter: "(objectClass=groupOfNames)"
```

### 5.2 启动命令
```bash
./uos-openldap-exporter --config.file=config.yaml
# 或
./uos-openldap-exporter \
  --ldap.server=ldaps://ldap.example.com:636 \
  --ldap.bind-dn="cn=monitor,dc=example,dc=com" \
  --web.listen-address=:9331
```

---

## 6. 开发计划（MVP）

| 阶段 | 任务 | 交付物 |
|------|------|--------|
| Week 1 | 搭建项目结构、CLI 框架、配置加载 | 可解析配置的骨架程序 |
| Week 2 | 实现 LDAP 连接、基础指标采集（up, entries） | 能输出基本指标 |
| Week 3 | 集成 cn=Monitor 指标、自定义搜索 | 完整指标集 |
| Week 4 | 日志优化、错误处理、测试用例、文档 | 可发布 v1.0 |

---

## 7. 测试策略

- **单元测试**：覆盖配置解析、指标生成逻辑
- **集成测试**：启动本地 OpenLDAP 容器，验证指标准确性
- **端到端测试**：配合 Prometheus 抓取，验证数据正确性
- **安全测试**：确保密码不泄露、TLS 行为符合预期

---

## 8. 附录

### 8.1 OpenLDAP Monitor 启用方法
在 `slapd.conf` 中添加：
```
database monitor
access to *
  by dn.exact="cn=monitor,dc=example,dc=com" read
  by * none
```

### 8.2 参考项目
- https://github.com/tomcz/openldap_exporter
- https://github.com/prometheus-community/postgres_exporter （架构参考）

---

> **备注**：本 PRD 聚焦 MVP（最小可行产品），后续可根据实际运维需求迭代增强功能。