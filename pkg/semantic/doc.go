// Package semantic provides interfaces and types for integrating external
// metadata catalogs (DataHub, Atlan, OpenMetadata, Alation, dbt docs) with
// mcp-trino.
//
// The Provider interface enables tools to be enriched with business
// context: descriptions, ownership, tags, glossary terms, data quality scores,
// deprecation status, and lineage information.
//
// # Design Principles
//
//   - Generic: Works with any metadata catalog
//   - Zero-overhead: No impact when not configured
//   - Cacheable: Built-in caching support
//   - Composable: Chain multiple providers
//   - Production-grade: Error handling, timeouts, retries
//
// # Basic Usage
//
// Configure a semantic provider with the toolkit:
//
//	provider := datahub.NewProvider(datahub.FromEnv())
//	toolkit := tools.NewToolkit(client, cfg,
//	    tools.WithProvider(provider),
//	    tools.WithSemanticCache(semantic.DefaultCacheConfig()),
//	)
//
// # Combining Providers
//
// Use ProviderChain to query multiple sources:
//
//	chain := semantic.NewProviderChain(
//	    customOverrides,  // Check custom overrides first
//	    datahubProvider,  // Fall back to DataHub
//	)
//	toolkit := tools.NewToolkit(client, cfg,
//	    tools.WithProvider(chain),
//	)
//
// # Custom Providers
//
// Implement Provider for custom metadata sources:
//
//	type MyProvider struct {
//	    // ...
//	}
//
//	func (p *MyProvider) Name() string { return "my-provider" }
//
//	func (p *MyProvider) GetTableContext(ctx context.Context, table TableIdentifier) (*TableContext, error) {
//	    // Fetch from your metadata source
//	    return &TableContext{
//	        Description: "My description",
//	        // ...
//	    }, nil
//	}
//
// # Function-Based Providers
//
// For simple cases, use ProviderFunc:
//
//	provider := semantic.ProviderFunc{
//	    NameFn: func() string { return "inline" },
//	    GetTableContextFn: func(ctx context.Context, table TableIdentifier) (*TableContext, error) {
//	        return &TableContext{Description: "inline description"}, nil
//	    },
//	}
package semantic
