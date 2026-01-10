# Troubleshooting

Solutions to common issues with mcp-trino.

## Connection Issues

### Connection Refused

**Symptoms:**
```
Error: dial tcp 127.0.0.1:8080: connect: connection refused
```

**Causes:**
- Trino server not running
- Wrong host or port
- Firewall blocking connection

**Solutions:**

1. Verify Trino is running:
   ```bash
   curl http://trino.example.com:8080/v1/info
   ```

2. Check host and port configuration:
   ```bash
   echo $TRINO_HOST $TRINO_PORT
   ```

3. Test network connectivity:
   ```bash
   nc -zv trino.example.com 443
   ```

### Connection Timeout

**Symptoms:**
```
Error: connection timeout after 30s
```

**Causes:**
- Network latency
- Firewall timeout
- DNS resolution issues

**Solutions:**

1. Check DNS resolution:
   ```bash
   nslookup trino.example.com
   ```

2. Test with IP address:
   ```bash
   export TRINO_HOST=10.0.0.1
   ```

3. Increase connection timeout (if supported)

## Authentication Errors

### Invalid Credentials

**Symptoms:**
```
Error: Authentication failed: Invalid credentials
```

**Solutions:**

1. Verify credentials:
   ```bash
   echo "User: $TRINO_USER"
   # Don't echo password, but verify it's set
   [ -n "$TRINO_PASSWORD" ] && echo "Password is set"
   ```

2. Test credentials directly:
   ```bash
   trino --server https://trino.example.com \
         --user $TRINO_USER \
         --password
   ```

### User Not Found

**Symptoms:**
```
Error: User 'unknown' does not exist
```

**Solutions:**

1. Verify username:
   ```bash
   export TRINO_USER=correct_username
   ```

2. Check user exists in Trino's authentication system

## SSL Certificate Problems

### Certificate Verification Failed

**Symptoms:**
```
Error: x509: certificate signed by unknown authority
```

**Solutions:**

1. Install the CA certificate:
   ```bash
   # Linux
   sudo cp ca.crt /usr/local/share/ca-certificates/
   sudo update-ca-certificates

   # macOS
   sudo security add-trusted-cert -d -r trustRoot \
     -k /Library/Keychains/System.keychain ca.crt
   ```

2. For development only, disable verification:
   ```bash
   export TRINO_SSL_VERIFY=false
   ```

### Certificate Expired

**Symptoms:**
```
Error: x509: certificate has expired or is not yet valid
```

**Solutions:**

1. Renew the certificate on the Trino server
2. Check system clock is correct:
   ```bash
   date
   ```

## Query Timeout Errors

### Query Exceeded Timeout

**Symptoms:**
```
Error: Query exceeded timeout of 120s
```

**Solutions:**

1. Increase timeout:
   ```bash
   export TRINO_TIMEOUT=300
   ```

2. Optimize the query:
   - Add filters to reduce data scanned
   - Use LIMIT to reduce result size
   - Check query execution plan

3. Request specific timeout:
   ```json
   {
     "sql": "SELECT ...",
     "timeout_seconds": 300
   }
   ```

### Query Killed

**Symptoms:**
```
Error: Query killed by Trino coordinator
```

**Causes:**
- Query exceeded Trino resource limits
- Coordinator memory pressure

**Solutions:**

1. Reduce query complexity
2. Contact Trino administrator about resource limits

## Permission Denied

### Table Access Denied

**Symptoms:**
```
Error: Access Denied: Cannot select from table hive.default.users
```

**Solutions:**

1. Verify user has SELECT permission
2. Check Trino access control rules
3. Use a different catalog/schema with access

### Catalog Not Found

**Symptoms:**
```
Error: Catalog 'unknown' does not exist
```

**Solutions:**

1. List available catalogs:
   ```sql
   SHOW CATALOGS
   ```

2. Update configuration:
   ```bash
   export TRINO_CATALOG=existing_catalog
   ```

### Schema Not Found

**Symptoms:**
```
Error: Schema 'unknown' does not exist
```

**Solutions:**

1. List schemas in catalog:
   ```sql
   SHOW SCHEMAS IN catalog_name
   ```

2. Update configuration:
   ```bash
   export TRINO_SCHEMA=existing_schema
   ```

## Write Operation Blocked

**Symptoms:**
```
Error: Write operation blocked: INSERT statements are not allowed in read-only mode
```

**Causes:**
- Read-only mode is enabled (default)

**Solutions:**

1. If you need write access, disable read-only mode:
   ```bash
   export MCP_TRINO_EXT_READONLY=false
   ```

2. Or use SELECT instead of INSERT/UPDATE/DELETE

## Query Syntax Errors

**Symptoms:**
```
Error: line 1:15: mismatched input 'FORM'. Expecting: 'FROM', ...
```

**Solutions:**

1. Check SQL syntax
2. Verify table and column names
3. Use DESCRIBE to check column names:
   ```sql
   DESCRIBE catalog.schema.table
   ```

## Verbose Logging

Enable detailed logging for debugging:

```bash
# Enable request logging
export MCP_TRINO_EXT_LOGGING=true

# Enable query logging
export MCP_TRINO_EXT_QUERYLOG=true

# Run with logs to stderr
mcp-trino 2>mcp-trino.log
```

## Debug Mode

### Test Connection

```bash
# Simple connection test
TRINO_HOST=trino.example.com \
TRINO_USER=user \
TRINO_PASSWORD=pass \
mcp-trino --test-connection
```

### Verify Configuration

```bash
# Print effective configuration
mcp-trino --show-config
```

### Direct Trino Query

Test with the Trino CLI to isolate issues:

```bash
trino --server https://trino.example.com \
      --user user \
      --password \
      --execute "SELECT 1"
```

## Docker Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs mcp-trino

# Run interactively
docker run -it --rm \
  -e TRINO_HOST=trino.example.com \
  -e TRINO_USER=user \
  ghcr.io/txn2/mcp-trino:latest
```

### Network Issues in Docker

```bash
# Use host network for debugging
docker run --network=host \
  -e TRINO_HOST=localhost \
  ghcr.io/txn2/mcp-trino:latest
```

## Getting Help

If you can't resolve an issue:

1. Check the [GitHub Issues](https://github.com/txn2/mcp-trino/issues)
2. Open a new issue with:
   - mcp-trino version
   - Configuration (without credentials)
   - Error message
   - Steps to reproduce
