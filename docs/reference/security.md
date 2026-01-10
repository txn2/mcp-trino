# Security Reference

Security features, verification methods, and best practices for mcp-trino.

## Secure Defaults

mcp-trino ships with secure defaults that protect against common issues:

| Default | Value | Protection |
|---------|-------|------------|
| Read-only mode | Enabled | Blocks INSERT, UPDATE, DELETE |
| Row limit | 1000 | Prevents excessive data retrieval |
| Query timeout | 120s | Prevents runaway queries |
| SSL | Enabled | Encrypts data in transit |
| SSL verify | Enabled | Validates server certificates |

---

## Read-Only Mode

Read-only mode blocks write operations at the MCP server level, regardless of Trino permissions.

### Blocked Statements

- `INSERT`
- `UPDATE`
- `DELETE`
- `DROP`
- `CREATE`
- `ALTER`
- `TRUNCATE`
- `MERGE`

### Configuration

Enabled by default. To disable:

```bash
export MCP_TRINO_EXT_READONLY=false
```

Or in YAML:

```yaml
extensions:
  readonly: false
```

### When to Disable

Only disable read-only mode when:

- Users need to create/modify data
- Trino permissions are properly configured
- Audit logging is enabled

---

## Query Limits

### Row Limits

| Parameter | Default | Maximum |
|-----------|---------|---------|
| `limit` | 1000 | 10000 |

The row limit:

- Prevents accidentally retrieving millions of rows
- Can be overridden per-query up to maximum
- Returns `truncated: true` if more rows exist

### Timeout Limits

| Parameter | Default | Maximum |
|-----------|---------|---------|
| `timeout_seconds` | 120 | 300 |

The timeout:

- Cancels queries that run too long
- Prevents resource exhaustion
- Returns error on timeout

### Customizing Limits

```go
cfg := tools.Config{
    DefaultLimit:   500,    // Lower default
    MaxLimit:       5000,   // Lower maximum
    DefaultTimeout: 60 * time.Second,
    MaxTimeout:     180 * time.Second,
}
```

---

## SSL/TLS

### Default Behavior

| Host | SSL | SSL Verify |
|------|-----|------------|
| `localhost` | Off | - |
| `127.0.0.1` | Off | - |
| Remote | On | On |

### Configuration

```bash
# Force SSL
export TRINO_SSL=true
export TRINO_SSL_VERIFY=true

# Disable SSL (not recommended)
export TRINO_SSL=false
```

### Certificate Verification

When `TRINO_SSL_VERIFY=true`:

- Server certificate must be valid
- Certificate chain must be trusted
- Hostname must match certificate

To disable verification (testing only):

```bash
export TRINO_SSL_VERIFY=false
```

⚠️ Never disable SSL verification in production.

---

## Authentication

### Basic Authentication

```bash
export TRINO_USER=analyst
export TRINO_PASSWORD=secret
```

The password is sent via HTTP Basic Auth over SSL.

### Custom Authentication (Library)

For OAuth, API keys, or SSO, use middleware:

```go
type OAuthMiddleware struct {
    config OAuthConfig
}

func (m *OAuthMiddleware) Before(ctx *ToolContext) error {
    token, err := m.getAccessToken(ctx)
    if err != nil {
        return fmt.Errorf("authentication failed: %w", err)
    }
    ctx.Set("access_token", token)
    return nil
}
```

---

## Release Verification

All releases are signed and include provenance attestations.

### Checksum Verification

Each release includes a `checksums.txt` file:

```bash
# Download
curl -LO https://github.com/txn2/mcp-trino/releases/latest/download/mcp-trino_Linux_x86_64.tar.gz
curl -LO https://github.com/txn2/mcp-trino/releases/latest/download/checksums.txt

# Verify
sha256sum --check checksums.txt
```

### Cosign Signature Verification

Releases are signed with Cosign keyless signatures:

```bash
# Install cosign
brew install cosign

# Download signature bundle
curl -LO https://github.com/txn2/mcp-trino/releases/latest/download/mcp-trino_Linux_x86_64.tar.gz.sigstore.json

# Verify
cosign verify-blob \
  --bundle mcp-trino_Linux_x86_64.tar.gz.sigstore.json \
  --certificate-identity-regexp="https://github.com/txn2/mcp-trino" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  mcp-trino_Linux_x86_64.tar.gz
```

### SLSA Provenance

All releases include SLSA Level 3 provenance attestations:

```bash
# Download provenance
curl -LO https://github.com/txn2/mcp-trino/releases/latest/download/mcp-trino_Linux_x86_64.tar.gz.intoto.jsonl

# Verify with slsa-verifier
slsa-verifier verify-artifact \
  --provenance-path mcp-trino_Linux_x86_64.tar.gz.intoto.jsonl \
  --source-uri github.com/txn2/mcp-trino \
  mcp-trino_Linux_x86_64.tar.gz
```

SLSA Level 3 guarantees:

- Build process is tamper-resistant
- Provenance is non-falsifiable
- Source is version-controlled

---

## Best Practices

### Production Checklist

- [ ] Enable read-only mode
- [ ] Configure appropriate row limits
- [ ] Enable SSL with verification
- [ ] Use separate credentials per environment
- [ ] Enable audit logging
- [ ] Verify binary signatures before deployment

### Credential Management

**Do:**

- Store credentials in environment variables
- Use secrets management (Vault, K8s Secrets)
- Use separate credentials per environment
- Rotate credentials regularly

**Don't:**

- Hardcode credentials in code
- Commit credentials to version control
- Share credentials between environments
- Use production credentials for development

### Network Security

**Do:**

- Enable SSL/TLS
- Use private networks between MCP server and Trino
- Configure firewalls to limit access
- Use VPN for remote connections

**Don't:**

- Expose Trino directly to the internet
- Disable SSL verification
- Use unencrypted connections

### Audit Logging

Enable audit logging for compliance:

```bash
export MCP_TRINO_EXT_LOGGING=true
export MCP_TRINO_EXT_QUERYLOG=true
```

Or use custom audit interceptor:

```go
toolkit.AddInterceptor(&AuditLogInterceptor{
    output: auditFile,
    fields: []string{"user", "query", "timestamp"},
})
```

---

## Threat Model

### What mcp-trino Protects Against

| Threat | Protection |
|--------|------------|
| Accidental data modification | Read-only mode |
| Excessive data retrieval | Row limits |
| Resource exhaustion | Query timeouts |
| Man-in-the-middle | SSL/TLS |
| Tampered binaries | Cosign signatures |
| Supply chain attacks | SLSA provenance |

### What Requires Additional Protection

| Threat | Required Measure |
|--------|------------------|
| Unauthorized access | Trino authentication + custom middleware |
| SQL injection | Parameterized queries in application |
| Data exfiltration | Trino column masking + data governance |
| Privilege escalation | Trino role-based access control |

---

## Compliance

### SOC 2

For SOC 2 compliance:

- Enable audit logging (`MCP_TRINO_EXT_LOGGING=true`)
- Enable query logging (`MCP_TRINO_EXT_QUERYLOG=true`)
- Implement access controls via middleware
- Retain logs per retention policy

### HIPAA

For HIPAA compliance:

- Enable SSL with verification
- Implement data redaction transformer
- Configure audit logging
- Ensure proper access controls in Trino

### GDPR

For GDPR compliance:

- Implement data redaction for PII
- Configure tenant isolation
- Enable audit logging
- Ensure data residency requirements
