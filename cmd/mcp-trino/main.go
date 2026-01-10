// mcp-trino is an MCP server providing Trino query capabilities.
//
// This server exposes tools for executing SQL queries, explaining execution plans,
// and exploring database schemas via the Model Context Protocol.
//
// Usage:
//
//	# Set environment variables
//	export TRINO_HOST=trino.example.com
//	export TRINO_USER=your_user
//	export TRINO_PASSWORD=your_password
//	export TRINO_CATALOG=hive
//	export TRINO_SCHEMA=default
//
//	# Run the server (stdio transport for Claude Desktop)
//	./mcp-trino
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/txn2/mcp-trino/internal/server"
)

func main() {
	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	// Create server with default options (from environment)
	opts := server.DefaultOptions()
	mcpServer, mgr, err := server.New(opts)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer mgr.Close()

	// Test connection to default server
	defaultClient, err := mgr.Client("")
	if err != nil {
		log.Printf("Warning: Could not get default client: %v", err)
	} else if err := defaultClient.Ping(ctx); err != nil {
		log.Printf("Warning: Could not ping Trino server: %v", err)
	}

	// Log startup info
	infos := mgr.ConnectionInfos()
	var defaultHost string
	for _, info := range infos {
		if info.IsDefault {
			defaultHost = info.Host
			break
		}
	}
	log.Printf("mcp-trino %s starting (%d connection(s), default: %s)",
		server.Version,
		mgr.ConnectionCount(),
		defaultHost,
	)

	// Run server with stdio transport
	if err := mcpServer.Run(ctx, &mcp.StdioTransport{}); err != nil {
		if ctx.Err() != nil {
			// Context canceled, normal shutdown
			log.Println("Server stopped")
			return
		}
		log.Fatalf("Server error: %v", err)
	}
}
