# Architecture

Understanding the mcp-trino package structure and request flow.

## Package Structure

| Package | Description |
|---------|-------------|
| `cmd/mcp-trino` | Standalone server entry point |
| `pkg/client` | Trino client wrapper (connection, queries, configuration) |
| `pkg/tools` | MCP tool implementations (toolkit, manager, query, explain, schema) |
| `pkg/extensions` | Built-in extensions (middleware, interceptors, transformers) |
| `internal/server` | Private default server setup |

## Component Diagram

```mermaid
flowchart TB
    subgraph External["External"]
        AI["AI Assistant"]
    end

    subgraph MCP["Your MCP Server"]
        SDK["MCP SDK"]
        MW["Middleware"]
        TK["Toolkit"]
    end

    subgraph McpTrino["mcp-trino/pkg"]
        Client["client.Client"]
        Tools["tools.Toolkit"]
        INT["Interceptors"]
        TF["Transformers"]
    end

    subgraph Trino["Trino Cluster"]
        TS["Trino Server"]
    end

    AI -->|"MCP Protocol"| SDK
    SDK --> MW
    MW --> TK
    TK --> Tools
    Tools --> INT
    INT --> Client
    Client -->|"SQL"| TS
    TS -->|"Results"| Client
    Client --> TF
    TF --> Tools
```

## Request Flow

```mermaid
sequenceDiagram
    participant AI as AI Assistant
    participant MCP as MCP Server
    participant MW as Middleware Chain
    participant INT as Interceptor Chain
    participant Client as Trino Client
    participant TF as Transformer Chain
    participant Trino as Trino Server

    AI->>MCP: CallTool(trino_query)
    MCP->>MW: Before(context)

    alt Authentication Failed
        MW-->>MCP: Error: unauthorized
        MCP-->>AI: Error response
    end

    MW->>INT: Intercept(sql)

    alt SQL Blocked
        INT-->>MW: Error: blocked
        MW-->>MCP: Error response
        MCP-->>AI: Error response
    end

    INT->>Client: Query(modified_sql)
    Client->>Trino: Execute SQL
    Trino-->>Client: Results
    Client->>TF: Transform(results)
    TF-->>MW: After(context, results)
    MW-->>MCP: Final results
    MCP-->>AI: CallToolResult
```

## Components

### Client (`pkg/client`)

The Trino client wrapper handles connection management and query execution.

```go
// Create from environment
cfg := client.FromEnv()
trinoClient, err := client.New(cfg)

// Or configure explicitly
cfg := client.Config{
    Host:     "trino.example.com",
    Port:     443,
    User:     "analyst",
    Password: "secret",
    Catalog:  "hive",
    SSL:      true,
}
trinoClient, err := client.New(cfg)
```

Key responsibilities:

- Connection pooling via `database/sql`
- DSN generation for Trino driver
- Query execution with context/timeout support
- Resource cleanup

### Toolkit (`pkg/tools`)

The toolkit manages MCP tool registration and coordinates extensions.

```go
toolkit := tools.NewToolkit(trinoClient, tools.Config{
    DefaultLimit:   1000,
    MaxLimit:       10000,
    DefaultTimeout: 120 * time.Second,
    MaxTimeout:     300 * time.Second,
})

// Add extensions
toolkit.Use(middleware)
toolkit.AddInterceptor(interceptor)
toolkit.AddTransformer(transformer)

// Register tools on MCP server
toolkit.RegisterAll(mcpServer)
```

Key responsibilities:

- Tool registration with MCP server
- Extension chain management
- Configuration enforcement (limits, timeouts)
- Multi-server connection management

### Manager (`pkg/tools`)

Manages multiple Trino connections for multi-cluster deployments.

```go
manager := tools.NewManager()
manager.Add("prod", prodClient)
manager.Add("staging", stagingClient)
manager.SetDefault("prod")

toolkit := tools.NewToolkitWithManager(manager, cfg)
```

### Extensions (`pkg/extensions`)

Built-in extensions for common use cases:

| Extension | Type | Purpose |
|-----------|------|---------|
| `LoggingMiddleware` | Middleware | Request/response logging |
| `ReadOnlyMiddleware` | Middleware | Block write operations |
| `MetricsMiddleware` | Middleware | Collect metrics |
| `QueryLogInterceptor` | Interceptor | SQL audit logging |
| `MetadataTransformer` | Transformer | Add execution metadata |
| `ErrorHelpTransformer` | Transformer | Enhance error messages |

## Extension Execution Order

Extensions are executed in a specific order:

```mermaid
flowchart TB
    REQ[Request] --> MW_BEFORE

    subgraph MW_BEFORE["Middleware Before()"]
        direction TB
        M1B["Middleware 1"] --> M2B["Middleware 2"] --> M3B["Middleware 3"]
    end

    MW_BEFORE --> INT

    subgraph INT["Interceptors"]
        direction TB
        I1["Interceptor 1"] --> I2["Interceptor 2"] --> I3["Interceptor 3"]
    end

    INT --> EXEC["Execute Query"]
    EXEC --> TF

    subgraph TF["Transformers"]
        direction TB
        T1["Transformer 1"] --> T2["Transformer 2"] --> T3["Transformer 3"]
    end

    TF --> MW_AFTER

    subgraph MW_AFTER["Middleware After()"]
        direction TB
        M3A["Middleware 3"] --> M2A["Middleware 2"] --> M1A["Middleware 1"]
    end

    MW_AFTER --> RES[Response]
```

Note: Middleware `After()` is called in reverse order (LIFO).

## Tool Registration

The toolkit registers these MCP tools:

| Tool | Method | Description |
|------|--------|-------------|
| `trino_query` | `RegisterQuery` | Execute SQL queries |
| `trino_explain` | `RegisterExplain` | Analyze query plans |
| `trino_list_catalogs` | `RegisterListCatalogs` | List catalogs |
| `trino_list_schemas` | `RegisterListSchemas` | List schemas |
| `trino_list_tables` | `RegisterListTables` | List tables |
| `trino_describe_table` | `RegisterDescribeTable` | Describe table |
| `trino_list_connections` | `RegisterListConnections` | List connections |

Register all at once:

```go
toolkit.RegisterAll(mcpServer)
```

Or selectively:

```go
toolkit.RegisterQuery(mcpServer)
toolkit.RegisterExplain(mcpServer)
// Skip others
```

## Configuration Flow

Configuration sources (in priority order):

```mermaid
flowchart LR
    ENV["Environment Variables"]
    FILE["Config File (YAML)"]
    CODE["Go Code"]
    DEFAULTS["Defaults"]

    ENV -->|"Highest"| MERGED["Merged Config"]
    FILE -->|"Medium"| MERGED
    CODE -->|"Medium"| MERGED
    DEFAULTS -->|"Lowest"| MERGED
```

Example:

```go
// Defaults
cfg := tools.DefaultConfig()

// Override in code
cfg.MaxLimit = 5000

// File config overrides code
serverCfg, _ := extensions.FromFile("config.yaml")

// Environment overrides everything
// TRINO_HOST, TRINO_USER, etc.
```

## Thread Safety

All components are thread-safe:

- **Client**: Uses `database/sql` connection pool
- **Toolkit**: Thread-safe tool execution
- **ToolContext**: Uses `sync.Map` for values
- **Manager**: Thread-safe connection lookup

## Next Steps

- [Extensibility](extensibility.md) - Build custom middleware, interceptors, and transformers
