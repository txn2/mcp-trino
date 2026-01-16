# v0.2.0 - Semantic Layer

A backward-compatible release introducing a **Semantic Layer** that enriches AI assistant understanding with organizational context beyond basic table structures.

## Features

### Semantic Layer Capabilities
Adds eight metadata categories to enrich table and column descriptions:
- **Business descriptions** - Human-readable context for data assets
- **Ownership details** - Data owners and stewards
- **Tags** - Classification and categorization labels
- **Glossary terms** - Business terminology associations
- **Data quality scores** - Quality indicators and metrics
- **Sensitivity markers** - PII and data classification
- **Deprecation alerts** - Lifecycle status warnings
- **Lineage information** - Data flow and dependencies

### Provider Options
Three metadata sources are supported:
- **DataHub** - Enterprise metadata catalogs via GraphQL API
- **Static** - YAML/JSON files with hot-reload capability
- **Custom** - Implement the `semantic.Provider` interface for custom sources

### Performance Characteristics
- **Zero Overhead** - No performance impact when semantic layer is unconfigured
- **Built-in Caching** - Reduces latency with configurable TTL and entry limits
- **Provider Chaining** - Combine multiple providers with fallback logic
- **ProviderFunc** - Simple inline providers for lightweight customization

## Configuration

### Environment Variables

**DataHub Provider:**
```bash
export DATAHUB_ENDPOINT=https://datahub.example.com
export DATAHUB_TOKEN=your_token
```

**Static Provider:**
```bash
export SEMANTIC_STATIC_FILE=/path/to/metadata.yaml
```

### Programmatic Configuration
```go
toolkit := tools.NewToolkit(trinoClient, tools.DefaultConfig(),
    tools.WithSemanticProvider(provider),
    tools.WithSemanticCache(5*time.Minute, 1000),
)
```

## Installation

### Homebrew (macOS)
```bash
brew upgrade txn2/tap/mcp-trino
```

### Claude Desktop
Download the `.mcpb` bundle for your platform from the [releases page](https://github.com/txn2/mcp-trino/releases/tag/v0.2.0):
- macOS Apple Silicon: `mcp-trino_0.2.0_darwin_arm64.mcpb`
- macOS Intel: `mcp-trino_0.2.0_darwin_amd64.mcpb`
- Windows: `mcp-trino_0.2.0_windows_amd64.mcpb`

### Claude Code CLI
```bash
claude mcp add trino \
  -e TRINO_HOST=trino.example.com \
  -e TRINO_USER=your_user \
  -e TRINO_PASSWORD=your_password \
  -- mcp-trino
```

### Docker
```bash
docker pull ghcr.io/txn2/mcp-trino:v0.2.0
```

### Go Install
```bash
go install github.com/txn2/mcp-trino/cmd/mcp-trino@v0.2.0
```

## Additional Changes

- Removed pre-built documentation directories from repository
- Expanded README documentation
- Added Docker Compose testing infrastructure (e2e)
- Added extensibility guides for library users

## Verification

All artifacts are signed with Cosign (keyless). Verify with:
```bash
cosign verify-blob --bundle mcp-trino_0.2.0_linux_amd64.tar.gz.sigstore.json \
  mcp-trino_0.2.0_linux_amd64.tar.gz
```
