# Installation

Install the mcp-trino server for your platform and AI client.

## Homebrew (macOS)

The easiest way to install on macOS:

```bash
brew install txn2/tap/mcp-trino
```

## Claude Desktop

Claude Desktop is the GUI application for chatting with Claude.

### Option 1: One-Click Install (Recommended)

Download the `.mcpb` bundle for your Mac from the [releases page](https://github.com/txn2/mcp-trino/releases):

| Mac Type | Chip | Download |
|----------|------|----------|
| MacBook Air/Pro (2020+), Mac Mini (2020+), iMac (2021+), Mac Studio | Apple M1/M2/M3/M4 (arm64) | `mcp-trino_*_darwin_arm64.mcpb` |
| MacBook Air/Pro (pre-2020), Mac Mini (pre-2020), iMac (pre-2021) | Intel (amd64) | `mcp-trino_*_darwin_amd64.mcpb` |

Double-click the `.mcpb` file to install. Configure your Trino connection in Claude Desktop settings.

!!! tip "Which chip do I have?"
    Click  → "About This Mac". Look for "Chip" (Apple Silicon) or "Processor" (Intel).

### Option 2: Manual Configuration

Add to your `claude_desktop_config.json` (Claude Desktop → Settings → Developer):

```json
{
  "mcpServers": {
    "trino": {
      "command": "/opt/homebrew/bin/mcp-trino",
      "env": {
        "TRINO_HOST": "trino.example.com",
        "TRINO_USER": "your_user",
        "TRINO_PASSWORD": "your_password",
        "TRINO_CATALOG": "hive",
        "TRINO_SCHEMA": "default"
      }
    }
  }
}
```

## Claude Code CLI

Claude Code is the terminal-based coding assistant.

```bash
# If installed via Homebrew
claude mcp add trino \
  -e TRINO_HOST=trino.example.com \
  -e TRINO_USER=your_user \
  -e TRINO_PASSWORD=your_password \
  -e TRINO_CATALOG=hive \
  -- mcp-trino
```

Or with a downloaded binary:

```bash
# Download
curl -L https://github.com/txn2/mcp-trino/releases/latest/download/mcp-trino_$(uname -s)_$(uname -m).tar.gz | tar xz

# Add to Claude Code
claude mcp add trino \
  -e TRINO_HOST=trino.example.com \
  -e TRINO_USER=your_user \
  -- ./mcp-trino
```

## Docker

```bash
docker run --rm -i \
  -e TRINO_HOST=trino.example.com \
  -e TRINO_USER=your_user \
  -e TRINO_PASSWORD=your_password \
  ghcr.io/txn2/mcp-trino:latest
```

For MCP clients that support Docker:

```json
{
  "mcpServers": {
    "trino": {
      "command": "docker",
      "args": ["run", "--rm", "-i",
        "-e", "TRINO_HOST=trino.example.com",
        "-e", "TRINO_USER=your_user",
        "ghcr.io/txn2/mcp-trino:latest"
      ]
    }
  }
}
```

## Go Install

If you have Go installed:

```bash
go install github.com/txn2/mcp-trino/cmd/mcp-trino@latest
```

## Binary Download

Download pre-built binaries from the [releases page](https://github.com/txn2/mcp-trino/releases):

```bash
# Linux/macOS
curl -LO https://github.com/txn2/mcp-trino/releases/latest/download/mcp-trino_Linux_x86_64.tar.gz
tar xzf mcp-trino_Linux_x86_64.tar.gz
chmod +x mcp-trino
sudo mv mcp-trino /usr/local/bin/
```

### Verify Download

All releases are signed. Verify with Cosign:

```bash
cosign verify-blob \
  --bundle mcp-trino_*.tar.gz.sigstore.json \
  mcp-trino_*.tar.gz
```

## Local Testing

To test with a local Trino instance:

```bash
# Start Trino in Docker
docker run -d -p 8080:8080 --name trino trinodb/trino

# Configure for local connection
export TRINO_HOST=localhost
export TRINO_PORT=8080
export TRINO_USER=admin
export TRINO_SSL=false
export TRINO_CATALOG=memory
export TRINO_SCHEMA=default

# Run the server
mcp-trino
```

## Next Steps

- [Configuration](configuration.md) - Configure your connection
- [Tools](tools.md) - Learn about available tools
