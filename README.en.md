# uos-openldap-exporter

## Introduction

uos-openldap-exporter is a Prometheus exporter for OpenLDAP. It connects to OpenLDAP servers, collects key service metrics, and exposes them via HTTP interface in standard Prometheus format for easy integration into Prometheus monitoring systems.

The exporter supports various configuration options including LDAP connection parameters, TLS/StartTLS encryption, bind authentication, and custom LDAP queries.

## Features

- Connects to OpenLDAP server and collects key metrics
- Supports LDAP and LDAPS protocols
- Supports StartTLS encrypted connections
- Supports bind authentication and anonymous connections
- Collects performance metrics from `cn=Monitor` subtree
- Supports custom LDAP queries and counting
- Exposes Prometheus format metrics via `/metrics` endpoint
- Provides `/healthz` health check endpoint
- Supports configuration via YAML file or command-line parameters
- Structured logging with multiple log levels

## Software Architecture

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

## Installation

1. Ensure Go 1.25.4 or higher is installed
2. Clone the project:
   ```bash
   git clone https://gitee.com/openeuler/uos-openldap-exporter.git
   ```
3. Navigate to project directory and build:
   ```bash
   cd uos-openldap-exporter
   go build -o uos-openldap-exporter ./cmd/
   ```

## Usage

### Using Configuration File

Create a configuration file `config.yaml`:

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

Run the exporter:
```bash
./uos-openldap-exporter --config.file=config.yaml
```

### Using Command Line Arguments

```bash
./uos-openldap-exporter \
  --ldap.server=ldaps://ldap.example.com:636 \
  --ldap.bind-dn="cn=monitor,dc=example,dc=com" \
  --web.listen-address=:9331
```

### Verification

Verify the service is running correctly by accessing:
- Metrics endpoint: http://localhost:9330/metrics
- Health check: http://localhost:9330/healthz

## Collected Metrics

- `openldap_up{}`: Whether the LDAP server is reachable (1=OK, 0=Error)
- `openldap_entries_total{}`: Total number of entries in directory
- `openldap_monitor_connections_total{}`: Current connection count
- `openldap_monitor_operations_initiated_total{}`: Number of initiated operations (by operation type)
- `openldap_custom_search_result_count{}`: Results count of custom searches

## Contributing

1. Fork the repository
2. Create your feature branch (git checkout -b feature/AmazingFeature)
3. Commit your changes (git commit -m 'Add some AmazingFeature')
4. Push to the branch (git push origin feature/AmazingFeature)
5. Open a pull request

## License

See the LICENSE file in the project root for more information.
