# Security Policy

## Supported Versions

We release patches for security vulnerabilities in the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability within mcp-trino, please report it responsibly.

### How to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of the following methods:

1. **GitHub Security Advisories** (Preferred): Use [GitHub's private vulnerability reporting](https://github.com/txn2/mcp-trino/security/advisories/new) to report the vulnerability directly.

2. **Email**: Send an email to security@txn2.com with:
   - A description of the vulnerability
   - Steps to reproduce the issue
   - Potential impact of the vulnerability
   - Any suggested fixes (optional)

### What to Expect

- **Acknowledgment**: We will acknowledge receipt of your vulnerability report within 48 hours.
- **Communication**: We will keep you informed about the progress of fixing the vulnerability.
- **Timeline**: We aim to release a fix within 90 days of the initial report, depending on complexity.
- **Credit**: We will credit you in the release notes (unless you prefer to remain anonymous).

### Security Best Practices for Users

When deploying mcp-trino:

1. **Credentials Management**
   - Never commit credentials to version control
   - Use environment variables or secret managers for sensitive configuration
   - Rotate credentials regularly

2. **Network Security**
   - Always use SSL/TLS when connecting to Trino (`TRINO_SSL=true`)
   - Verify SSL certificates in production (`TRINO_SSL_VERIFY=true`)
   - Deploy behind a firewall or VPN when possible

3. **Access Control**
   - Use Trino's role-based access control (RBAC)
   - Grant minimal necessary permissions to the Trino user
   - Consider using read-only credentials

4. **Query Safety**
   - mcp-trino enforces row limits (default: 1000, max: 10000) to prevent data exfiltration
   - Query timeouts are enforced (default: 120s, max: 300s) to prevent runaway queries
   - Only SELECT queries are supported (no write operations)

5. **Logging and Monitoring**
   - Monitor query patterns for unusual activity
   - Set up alerts for failed authentication attempts
   - Log access to sensitive data

## Security Features

mcp-trino includes several security features by default:

- **Query Limits**: Configurable row limits prevent excessive data transfer
- **Timeouts**: Query timeouts prevent long-running queries
- **Read-Only**: Only SELECT queries are supported
- **SSL Support**: Full TLS/SSL support with certificate verification
- **Connection Pooling**: Limits on concurrent connections

## Security Updates

Security updates are released as patch versions and announced via:

- GitHub Security Advisories
- Release notes
- The project README

We recommend always running the latest version of mcp-trino.
