# Quick Start

Build a custom MCP server with Trino capabilities in minutes.

## Prerequisites

- Go 1.21+
- Access to a Trino server

## Step 1: Minimal Server

Create a basic MCP server with all Trino tools:

```go
// main.go
package main

import (
    "log"

    "github.com/modelcontextprotocol/go-sdk/server"
    "github.com/txn2/mcp-trino/pkg/client"
    "github.com/txn2/mcp-trino/pkg/tools"
)

func main() {
    // Create Trino client from environment
    trinoClient, err := client.New(client.FromEnv())
    if err != nil {
        log.Fatal(err)
    }
    defer trinoClient.Close()

    // Create toolkit with default config
    toolkit := tools.NewToolkit(trinoClient, tools.DefaultConfig())

    // Create MCP server
    mcpServer := server.NewMCPServer("my-trino-server", "1.0.0")

    // Register all Trino tools
    toolkit.RegisterAll(mcpServer)

    // Run server
    if err := server.ServeStdio(mcpServer); err != nil {
        log.Fatal(err)
    }
}
```

Build and run:

```bash
export TRINO_HOST=trino.example.com
export TRINO_USER=analyst
export TRINO_PASSWORD=secret
export TRINO_CATALOG=hive

go build -o my-server
./my-server
```

## Step 2: Add Logging

Add request logging to see what's happening:

```go
import (
    "os"

    "github.com/txn2/mcp-trino/pkg/extensions"
)

func main() {
    trinoClient, _ := client.New(client.FromEnv())
    toolkit := tools.NewToolkit(trinoClient, tools.DefaultConfig())

    // Add logging middleware
    toolkit.Use(extensions.NewLoggingMiddleware(os.Stderr))

    mcpServer := server.NewMCPServer("my-trino-server", "1.0.0")
    toolkit.RegisterAll(mcpServer)
    server.ServeStdio(mcpServer)
}
```

Now you'll see JSON logs for each request:

```json
{"event":"request_start","tool":"trino_query","request_id":"abc123"}
{"event":"request_success","tool":"trino_query","duration_ms":150}
```

## Step 3: Add Read-Only Protection

Block write operations (INSERT, UPDATE, DELETE):

```go
// Add logging and read-only protection
toolkit.Use(extensions.NewLoggingMiddleware(os.Stderr))
toolkit.Use(extensions.NewReadOnlyMiddleware())
```

Now dangerous queries are blocked:

```
Error: Write operations are not allowed (INSERT, UPDATE, DELETE, DROP, etc.)
```

## Step 4: Add Query Logging

Log all SQL queries for audit:

```go
// Middleware for request lifecycle
toolkit.Use(extensions.NewLoggingMiddleware(os.Stderr))
toolkit.Use(extensions.NewReadOnlyMiddleware())

// Interceptor for SQL queries
toolkit.AddInterceptor(extensions.NewQueryLogInterceptor(os.Stderr))
```

SQL queries are now logged:

```json
{"event":"query","sql":"SELECT * FROM users LIMIT 10","timestamp":"2024-01-15T10:30:00Z"}
```

## Step 5: Add Result Metadata

Include execution stats in query results:

```go
// Middleware
toolkit.Use(extensions.NewLoggingMiddleware(os.Stderr))
toolkit.Use(extensions.NewReadOnlyMiddleware())

// Interceptor
toolkit.AddInterceptor(extensions.NewQueryLogInterceptor(os.Stderr))

// Transformer
toolkit.AddTransformer(extensions.NewMetadataTransformer())
```

Results now include metadata:

```json
{
  "rows": [...],
  "metadata": {
    "row_count": 10,
    "execution_time_ms": 150,
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

## Step 6: Custom Authentication

Add your own authentication:

```go
type AuthMiddleware struct {
    validator TokenValidator
}

func (m *AuthMiddleware) Before(ctx *tools.ToolContext) error {
    token, ok := ctx.Get("auth_token")
    if !ok {
        return errors.New("authentication required")
    }

    user, err := m.validator.Validate(token.(string))
    if err != nil {
        return fmt.Errorf("invalid token: %w", err)
    }

    ctx.Set("user", user)
    return nil
}

func (m *AuthMiddleware) After(ctx *tools.ToolContext, result *mcp.CallToolResult, err error) {
    // Log access
    user, _ := ctx.Get("user")
    log.Printf("User %s called %s", user, ctx.ToolName)
}

// Use it
toolkit.Use(&AuthMiddleware{validator: myValidator})
```

## Step 7: Tenant Isolation

Filter queries by tenant:

```go
type TenantInterceptor struct {
    column string
}

func (i *TenantInterceptor) Intercept(ctx context.Context, sql string) (string, error) {
    tenant := ctx.Value("tenant")
    if tenant == nil {
        return "", errors.New("tenant not set")
    }

    // Wrap query with tenant filter
    return fmt.Sprintf(
        "SELECT * FROM (%s) WHERE %s = '%s'",
        sql, i.column, tenant,
    ), nil
}

// Use it
toolkit.AddInterceptor(&TenantInterceptor{column: "tenant_id"})
```

## Complete Example

Here's a production-ready server with all extensions:

```go
package main

import (
    "log"
    "os"
    "time"

    "github.com/modelcontextprotocol/go-sdk/server"

    "github.com/txn2/mcp-trino/pkg/client"
    "github.com/txn2/mcp-trino/pkg/extensions"
    "github.com/txn2/mcp-trino/pkg/tools"
)

func main() {
    // Create client
    trinoClient, err := client.New(client.FromEnv())
    if err != nil {
        log.Fatal(err)
    }
    defer trinoClient.Close()

    // Configure toolkit
    cfg := tools.Config{
        DefaultLimit:   1000,
        MaxLimit:       5000,
        DefaultTimeout: 60 * time.Second,
        MaxTimeout:     180 * time.Second,
    }
    toolkit := tools.NewToolkit(trinoClient, cfg)

    // Add middleware (executed in order)
    toolkit.Use(extensions.NewLoggingMiddleware(os.Stderr))
    toolkit.Use(extensions.NewReadOnlyMiddleware())

    // Add interceptors
    toolkit.AddInterceptor(extensions.NewQueryLogInterceptor(os.Stderr))

    // Add transformers
    toolkit.AddTransformer(extensions.NewMetadataTransformer())

    // Create and run server
    mcpServer := server.NewMCPServer("production-server", "1.0.0")
    toolkit.RegisterAll(mcpServer)

    if err := server.ServeStdio(mcpServer); err != nil {
        log.Fatal(err)
    }
}
```

## Multi-Server Configuration

Connect to multiple Trino clusters:

```go
// Create connection manager
manager := tools.NewManager()

// Add connections
prodClient, _ := client.New(client.Config{Host: "prod.trino.example.com", ...})
stagingClient, _ := client.New(client.Config{Host: "staging.trino.example.com", ...})

manager.Add("production", prodClient)
manager.Add("staging", stagingClient)
manager.SetDefault("production")

// Create toolkit with manager
toolkit := tools.NewToolkitWithManager(manager, tools.DefaultConfig())
```

Users can now specify which connection to use:

```json
{"tool": "trino_query", "arguments": {"sql": "SELECT * FROM users", "connection": "staging"}}
```

## Next Steps

- [Architecture](architecture.md) - Understand the package structure
- [Extensibility](extensibility.md) - Deep dive into middleware, interceptors, and transformers
