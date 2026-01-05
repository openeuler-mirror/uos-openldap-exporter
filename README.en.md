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
- **Plugin Architecture**: Allows users to develop custom metric collection plugins
- **Metric Filtering**: Supports enabling/disabling specific metrics on demand

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
|  - Plugin System |
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
  format: "text"  # Optional values: "text" or "json", default is "text"

plugins:
  enabled:  # Specify enabled plugins, leave empty to enable all plugins
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

### Using Command Line Parameters

```bash
# Start with configuration file
./uos-openldap-exporter --config.file=config.yaml

# Start with command line parameters
./uos-openldap-exporter --ldap.server=ldaps://ldap.example.com:636 --web.listen-address=:9331
```

### Plugin System Usage

#### Enable Specific Plugins

Specify plugins to enable in the configuration file:

```yaml
plugins:
  enabled:
    - "base_connection"
    - "security"
```

#### Developing Custom Plugins

To develop a custom plugin, you need to implement the `PluginCollector` interface:

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

Register the plugin in the main program:

```go
myPlugin := NewMyPlugin()
collector.GetPluginManager().RegisterPlugin(myPlugin)
```

### Environment Variables

The following environment variables can be used to configure the exporter:

- `OPENLDAP_EXPORTER_SERVER` - LDAP server address
- `OPENLDAP_EXPORTER_BIND_DN` - Bind DN
- `OPENLDAP_EXPORTER_BIND_PASSWORD` - Bind password
- `OPENLDAP_EXPORTER_LISTEN_ADDRESS` - Listen address
- `OPENLDAP_EXPORTER_METRICS_PATH` - Metrics path
- `OPENLDAP_EXPORTER_LOG_LEVEL` - Log level

### Configuration Validation

The configuration file must contain the following requirements:

- `ldap.server` must be set
- `web.listen_address` and `web.metrics_path` must be set
- Each item in `custom_searches` must contain name, base_dn, and filter fields

### Health Check

The `/healthz` endpoint provides real health check functionality, which:
1. Attempts to connect to the configured LDAP server
2. Starts TLS if StartTLS is configured
3. Performs bind operation if bind credentials are configured
4. Performs a lightweight WhoAmI operation to verify the connection
5. Returns JSON-formatted health status

Example of a healthy response:
```json
{
  "status": "ok"
}
```

Example of an unhealthy response:
```json
{
  "status": "error",
  "ldap": "Failed to connect to LDAP server: ..."
}
```

### Configuration Reload

Currently, runtime configuration reload is not supported. The service needs to be restarted for configuration changes to take effect.

## Development Guide

### Project Structure

```
uos-openldap-exporter/
├── cmd/                    # Command line interface
├── internal/
│   ├── collector/          # Metrics collector
│   ├── config/             # Configuration management
│   ├── logger/             # Logging system
│   └── server/             # HTTP service
├── scripts/                # Scripts
├── Dockerfile              # Docker configuration
├── Makefile                # Build script
├── README.md               # Project documentation
└── config.example.yaml     # Configuration example
```

### Plugin Development

To develop plugins, you need to:

1. Implement the `PluginCollector` interface
2. Inherit from `BasePluginCollector` to get basic functionality
3. Define metric descriptors in the `Describe` method
4. Collect metric data in the `Collect` method
5. Use the `Name` method to return the plugin's unique identifier

## Build and Test

### Build

```bash
# Local build
go build -o uos-openldap-exporter main.go

# Build using Makefile
make build

# Multi-platform build
make release
```

### Test

```bash
# Run tests
make test

# Test coverage
make test-cover

# Security check
make sec
```

### Docker Image

```bash
# Build Docker image
make docker-build

# Push image
make docker-push
```

## Deployment

### Docker Deployment

```bash
docker run -d -p 9330:9330 \
  -v /path/to/config.yaml:/etc/openldap-exporter/config.yaml \
  uos-openldap-exporter:<version>
```

### Kubernetes Deployment

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

## Contributing

Feel free to submit issues and pull requests to improve the project.

## License

This project is licensed under the MulanPSL-2.0 license.