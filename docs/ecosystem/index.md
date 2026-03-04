---
hide:
  - navigation
  - title
---

# MCP Data Platform Ecosystem

mcp-trino is part of a broader suite of open-source MCP servers designed to work together as a composable data platform. Each component can run standalone or be combined to give AI assistants unified access to storage, query engines, and metadata catalogs.

---

## [mcp-data-platform](https://github.com/txn2/mcp-data-platform/)

The orchestration layer that ties the ecosystem together. mcp-data-platform provides a single MCP server that coordinates access across S3 storage, Trino query engines, and DataHub metadata catalogs. Rather than configuring each MCP server independently, mcp-data-platform presents a unified interface where AI assistants can discover datasets through the catalog, query them through Trino, and access the underlying files in S3, all from one connection. It handles connection routing, credential management, and cross-service context so that assistants can work with data end-to-end without switching between tools.

## [mcp-datahub](https://github.com/txn2/mcp-datahub/)

An MCP server for [DataHub](https://datahubproject.io/), the open-source metadata platform. mcp-datahub lets AI assistants search and browse the data catalog, inspect dataset schemas, trace column-level lineage, and look up business glossary terms. When paired with mcp-trino and mcp-s3, it provides the discovery layer: assistants can find the right dataset by name or description, understand its structure and ownership, then seamlessly query or retrieve the data. It supports tags, domains, data products, and quality scores, giving assistants the context they need to work with data responsibly.

## [mcp-s3](https://github.com/txn2/mcp-s3/)

An MCP server for [Amazon S3](https://aws.amazon.com/s3/), providing AI assistants with direct access to object storage. mcp-s3 lets assistants list buckets, browse prefixes, read objects, and generate presigned URLs for temporary access. When paired with mcp-datahub and mcp-trino, it provides the raw file access layer: assistants can discover datasets through the catalog, query structured data through Trino, and retrieve or inspect the underlying files in S3. It supports multi-server configurations for accessing storage across accounts and regions.
