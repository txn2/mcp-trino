# Go Library

**mcp-trino is the only composable Go library for Trino MCP.** All other Trino MCP implementations are standalone servers. This library lets you build custom MCP servers with Trino capabilities.

## Why Build a Custom MCP Server?

The standalone mcp-trino server works for most use cases. But enterprises and SaaS platforms need more:

| Requirement | Standalone Server | Custom with Library |
|-------------|------------------|---------------------|
| Basic Trino queries | Yes | Yes |
| Custom authentication | No | **OAuth, API keys, SSO** |
| Multi-tenant isolation | No | **Row-level security** |
| Audit logging | Basic | **SOC2, compliance** |
| Combine data sources | Trino only | **Trino + Postgres + APIs** |
| Custom business logic | No | **Domain-specific tools** |

## Use Cases

### Multi-Tenant SaaS

Build a data platform where each customer only sees their own data:

```go
// Add tenant isolation to all queries
toolkit.AddInterceptor(NewTenantFilterInterceptor("tenant_id"))
```

### Enterprise Authentication

Integrate with your existing auth system:

```go
// OAuth/OIDC authentication
toolkit.Use(NewOAuthMiddleware(authConfig))

// API key validation
toolkit.Use(NewAPIKeyMiddleware(keyStore))
```

### Compliance and Audit

Meet SOC2, HIPAA, or GDPR requirements:

```go
// Audit all queries for compliance
toolkit.AddInterceptor(NewAuditLogInterceptor(auditStore))

// Redact sensitive data (PII, PHI)
toolkit.AddTransformer(NewRedactionTransformer(sensitiveColumns))
```

### Unified Data Access

Combine multiple data sources in one MCP server:

```go
// Register Trino tools
trinoToolkit.RegisterAll(mcpServer)

// Add PostgreSQL tools
postgresToolkit.RegisterAll(mcpServer)

// Add custom API tools
registerCustomTools(mcpServer)
```

### Domain-Specific Tools

Create tools tailored to your business:

```go
// Custom "get_customer_360" tool
mcpServer.AddTool("get_customer_360", func(ctx context.Context, req CustomerRequest) (*CustomerData, error) {
    // Combine data from multiple Trino tables
    // Apply business logic
    // Return formatted result
})
```

## Library vs Server

| Feature | Use Standalone Server | Use Go Library |
|---------|----------------------|----------------|
| Quick setup | Use server | |
| Default security | Use server | |
| Custom auth | | Use library |
| Multi-tenant | | Use library |
| Combine tools | | Use library |
| Audit logging | | Use library |
| Custom tools | | Use library |

## Getting Started

### Install

```bash
go get github.com/txn2/mcp-trino
```

### Minimal Example

```go
package main

import (
    "github.com/modelcontextprotocol/go-sdk/server"
    "github.com/txn2/mcp-trino/pkg/client"
    "github.com/txn2/mcp-trino/pkg/tools"
)

func main() {
    // Create Trino client
    trinoClient, _ := client.New(client.FromEnv())
    defer trinoClient.Close()

    // Create toolkit
    toolkit := tools.NewToolkit(trinoClient, tools.DefaultConfig())

    // Create MCP server
    mcpServer := server.NewMCPServer("my-server", "1.0.0")

    // Register all Trino tools
    toolkit.RegisterAll(mcpServer)

    // Run
    server.ServeStdio(mcpServer)
}
```

This gives you a working MCP server with all Trino tools. From here, add middleware, interceptors, and transformers to meet your requirements.

## Next Steps

- [Quick Start](quickstart.md) - Working examples with progressive enhancement
- [Architecture](architecture.md) - Package structure and request flow
- [Extensibility](extensibility.md) - Middleware, interceptors, and transformers
