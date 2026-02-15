# Configuration Reference

Complete reference for all configuration options.

## Environment Variables

### Connection Settings

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `TRINO_HOST` | string | `localhost` | Trino server hostname |
| `TRINO_PORT` | integer | `443` (SSL) / `8080` (no SSL) | Server port |
| `TRINO_USER` | string | (required) | Authentication username |
| `TRINO_PASSWORD` | string | (empty) | Authentication password |
| `TRINO_CATALOG` | string | `memory` | Default catalog |
| `TRINO_SCHEMA` | string | `default` | Default schema |
| `TRINO_SSL` | boolean | `true` for remote | Enable HTTPS |
| `TRINO_SSL_VERIFY` | boolean | `true` | Verify SSL certificates |
| `TRINO_TIMEOUT` | integer | `120` | Query timeout (seconds) |
| `TRINO_SOURCE` | string | `mcp-trino` | Client identifier |

### Extension Settings

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `MCP_TRINO_EXT_READONLY` | boolean | `true` | Block write operations |
| `MCP_TRINO_EXT_ERRORS` | boolean | `true` | Add helpful hints to errors |
| `MCP_TRINO_EXT_LOGGING` | boolean | `false` | Enable request logging |
| `MCP_TRINO_EXT_METRICS` | boolean | `false` | Enable metrics collection |
| `MCP_TRINO_EXT_QUERYLOG` | boolean | `false` | Log all SQL queries |
| `MCP_TRINO_EXT_METADATA` | boolean | `false` | Add execution stats to results |

### Multi-Server Settings

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `TRINO_ADDITIONAL_SERVERS` | JSON | (empty) | Additional server configurations |

### File Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `MCP_TRINO_CONFIG` | string | (empty) | Path to YAML config file |

---

## YAML Configuration Schema

```yaml
# /etc/mcp-trino/config.yaml

# Trino connection settings
trino:
  host: trino.example.com       # Required
  port: 443                      # Optional, default: 443/8080
  user: ${TRINO_USER}           # Environment variable expansion
  password: ${TRINO_PASSWORD}   # Environment variable expansion
  catalog: hive                  # Optional, default: memory
  schema: default                # Optional, default: default
  ssl: true                      # Optional, default: true for remote
  ssl_verify: true               # Optional, default: true
  timeout: 120s                  # Optional, default: 120s
  source: mcp-trino              # Optional, default: mcp-trino

# Toolkit settings
toolkit:
  default_limit: 1000            # Optional, default: 1000
  max_limit: 10000               # Optional, default: 10000
  default_timeout: 120s          # Optional, default: 120s
  max_timeout: 300s              # Optional, default: 300s
  descriptions:                  # Optional, override tool descriptions
    trino_query: "Custom description for the query tool"
    trino_describe_table: "Custom description for describe"

# Extension settings
extensions:
  readonly: true                 # Optional, default: true
  logging: false                 # Optional, default: false
  metrics: false                 # Optional, default: false
  errors: true                   # Optional, default: true
  querylog: false                # Optional, default: false
  metadata: false                # Optional, default: false

# Additional servers (multi-server mode)
additional_servers:
  staging:
    host: staging.trino.example.com
    # Inherits user/password from primary
  dev:
    host: localhost
    port: 8080
    user: admin
    password: ""
    ssl: false
```

---

## Configuration Precedence

Configuration values are merged in this order (later overrides earlier):

1. **Defaults** - Built-in defaults
2. **Config file** - Values from YAML file
3. **Environment variables** - Always highest priority

This allows base settings in a config file with secrets from environment variables:

```yaml
# config.yaml - non-sensitive settings
trino:
  host: trino.example.com
  catalog: hive
  ssl: true
```

```bash
# Environment - secrets only
export TRINO_USER=service_account
export TRINO_PASSWORD=secret
export MCP_TRINO_CONFIG=/etc/mcp-trino/config.yaml
```

---

## Multi-Server Configuration

### Environment Variable Format

```bash
export TRINO_ADDITIONAL_SERVERS='{
  "staging": {
    "host": "staging.trino.example.com"
  },
  "dev": {
    "host": "localhost",
    "port": 8080,
    "ssl": false
  }
}'
```

### Server Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `host` | string | **Yes** | Server hostname |
| `port` | integer | No | Server port |
| `user` | string | No | Username (inherits from primary) |
| `password` | string | No | Password (inherits from primary) |
| `catalog` | string | No | Default catalog |
| `schema` | string | No | Default schema |
| `ssl` | boolean | No | Enable HTTPS |

### Credential Inheritance

Additional servers inherit credentials from the primary unless overridden:

```bash
# Primary credentials
export TRINO_USER=analyst
export TRINO_PASSWORD=secret

# Staging inherits analyst/secret
# Dev uses admin with no password
export TRINO_ADDITIONAL_SERVERS='{
  "staging": {
    "host": "staging.trino.example.com"
  },
  "dev": {
    "host": "localhost",
    "port": 8080,
    "user": "admin",
    "password": "",
    "ssl": false
  }
}'
```

---

## Boolean Values

Boolean environment variables accept:

| True | False |
|------|-------|
| `true`, `1`, `yes`, `on` | `false`, `0`, `no`, `off` |

```bash
export TRINO_SSL=true
export MCP_TRINO_EXT_READONLY=1
export MCP_TRINO_EXT_LOGGING=yes
```

---

## Duration Values

Duration values in YAML accept:

| Format | Example |
|--------|---------|
| Seconds | `120s` |
| Minutes | `2m` |
| Hours | `1h` |
| Combined | `1h30m` |

```yaml
trino:
  timeout: 2m
toolkit:
  default_timeout: 60s
  max_timeout: 5m
```

---

## Port Defaults

The default port depends on SSL setting:

| SSL | Default Port |
|-----|--------------|
| `true` | 443 |
| `false` | 8080 |

---

## SSL Auto-Detection

SSL is automatically enabled for remote hosts:

| Host | Default SSL |
|------|-------------|
| `localhost` | `false` |
| `127.0.0.1` | `false` |
| Any other | `true` |

Override with explicit `TRINO_SSL` setting.

---

## Toolkit Configuration (Go Library)

When using as a library, configure programmatically:

```go
cfg := tools.Config{
    DefaultLimit:   1000,
    MaxLimit:       10000,
    DefaultTimeout: 120 * time.Second,
    MaxTimeout:     300 * time.Second,
}

toolkit := tools.NewToolkit(trinoClient, cfg,
    tools.WithDescriptions(map[tools.ToolName]string{
        tools.ToolQuery:         "Query the retail analytics warehouse",
        tools.ToolDescribeTable: "Describe retail data tables",
    }),
)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `DefaultLimit` | int | 1000 | Default row limit |
| `MaxLimit` | int | 10000 | Maximum row limit |
| `DefaultTimeout` | Duration | 120s | Default query timeout |
| `MaxTimeout` | Duration | 300s | Maximum query timeout |
| `Descriptions` | map | (none) | Custom tool descriptions (via `WithDescriptions` option) |

---

## Client Configuration (Go Library)

```go
cfg := client.Config{
    Host:      "trino.example.com",
    Port:      443,
    User:      "analyst",
    Password:  "secret",
    Catalog:   "hive",
    Schema:    "default",
    SSL:       true,
    SSLVerify: true,
    Timeout:   120 * time.Second,
    Source:    "my-app",
}

trinoClient, err := client.New(cfg)
```

Or from environment:

```go
cfg := client.FromEnv()
trinoClient, err := client.New(cfg)
```

---

## Validation

Configuration is validated on startup:

| Check | Error |
|-------|-------|
| Missing `TRINO_USER` | `user is required` |
| Invalid port | `port must be between 1 and 65535` |
| Invalid timeout | `timeout must be positive` |
| Invalid JSON in `TRINO_ADDITIONAL_SERVERS` | `failed to parse additional servers` |
| Unknown connection name | `connection 'name' not found` |
