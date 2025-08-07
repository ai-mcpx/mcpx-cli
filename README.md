# mcpx-cli

[![Build Status](https://github.com/ai-mcpx/mcpx-cli/workflows/ci/badge.svg?branch=main&event=push)](https://github.com/ai-mcpx/mcpx-cli/actions?query=workflow%3Aci)
[![Go Report Card](https://goreportcard.com/badge/github.com/ai-mcpx/mcpx-cli)](https://goreportcard.com/report/github.com/ai-mcpx/mcpx-cli)
[![License](https://img.shields.io/github/license/ai-mcpx/mcpx-cli.svg)](https://github.com/ai-mcpx/mcpx-cli/blob/main/LICENSE)
[![Tag](https://img.shields.io/github/tag/ai-mcpx/mcpx-cli.svg)](https://github.com/ai-mcpx/mcpx-cli/tags)

A command-line interface for interacting with the mcpx (Model Context Protocol Extended) registry api. This CLI provides easy access to all the core mcpx api endpoints for managing MCP servers.

## Features

- **Health Check**: Verify api connectivity and status
- **Server Listing**: Browse available MCP servers with pagination
- **Server Details**: Get comprehensive information about specific servers
- **Server Publishing**: Publish new MCP servers to the registry (requires authentication)
- **JSON Output**: All responses are formatted for easy reading
- **Configurable Base URL**: Target different mcpx registry instances

## API Endpoints Supported

- `GET /v0/health` - Health check and status
- `GET /v0/servers` - List servers with optional pagination
- `GET /v0/servers/{id}` - Get detailed server information
- `POST /v0/publish` - Publish a new server (requires authentication)

## Installation

### Prerequisites

- Go 1.21 or later
- Access to an mcpx registry api instance (default: http://localhost:8080)

### Build from Source

```bash
# Clone or navigate to the project directory
cd mcpx-cli

# Build the binary
make build

# Or use go directly
go build -o mcpx-cli .
```

### Install for System-wide Use

```bash
make install
```

## Usage

### Basic Syntax

```bash
mcpx-cli [global flags] <command> [command flags]
```

### Global Flags

- `--base-url string`: Base url of the mcpx api (default: http://localhost:8080)

### Commands

#### Health Check

Check the health and status of the mcpx api:

```bash
mcpx-cli health
```

Example output:
```
=== Health Check ===
Status Code: 200
Status: ok
GitHub Client ID: your-github-client-id
```

#### List Servers

Browse available MCP servers:

```bash
# List servers with default pagination
mcpx-cli servers

# List with custom limit
mcpx-cli servers --limit 10

# Use pagination cursor
mcpx-cli servers --cursor "uuid-cursor-string" --limit 5
```

**Flags:**
- `--cursor string`: Pagination cursor for next page
- `--limit int`: Maximum number of servers to return (default: 30)

Example output:
```
=== List Servers ===
Status Code: 200
Total Servers: 2
Next Cursor: some-uuid-cursor

--- Server 1 ---
ID: a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1
Name: io.modelcontextprotocol/filesystem
Description: Node.js server implementing Model Context Protocol (MCP) for filesystem operations
Repository: https://github.com/modelcontextprotocol/servers (github)
Version: 1.0.2
Release Date: 2023-06-15T10:30:00Z
```

#### Get Server Details

Get comprehensive information about a specific server:

```bash
mcpx-cli server <server-id>
```

Example:
```bash
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1
```

Example output:
```
=== Get Server Details (ID: a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1) ===
Status Code: 200
ID: a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1
Name: io.modelcontextprotocol/filesystem
Description: Node.js server implementing Model Context Protocol (MCP) for filesystem operations
Repository: https://github.com/modelcontextprotocol/servers (github)
Version: 1.0.2
Release Date: 2023-06-15T10:30:00Z

Packages:
  Package 1:
    Registry: npm
    Name: @modelcontextprotocol/server-filesystem
    Version: 1.0.2
    Runtime Hint: npx
    Environment Variables:
      - LOG_LEVEL: Logging level (debug, info, warn, error)
```

#### Publish Server

Publish a new MCP server to the registry:

```bash
mcpx-cli publish <server.json> --token <auth-token>
```

**Flags:**
- `--token string`: Authentication token (required) - typically a GitHub token for `io.github.*` namespaced servers

Example:
```bash
mcpx-cli publish example-server.json --token ghp_your_github_token_here
```

Example output:
```
=== Publish Server (File: example-server.json) ===
Status Code: 201
Success: Server publication successful
Server ID: b1234567-8901-2345-6789-012345678901
```

### Targeting Different Environments

Use the `--base-url` flag to target different mcpx registry instances:

```bash
# Local development
mcpx-cli --base-url http://localhost:8080 servers

# Production instance
mcpx-cli --base-url https://registry.modelcontextprotocol.io servers

# Custom instance
mcpx-cli --base-url https://your-custom-registry.com servers
```

## Server JSON Format

When publishing servers, you need to provide a JSON file describing the server. Here's the structure:

```json
{
  "name": "io.github.example/test-server",
  "description": "A test MCP server for demonstration purposes",
  "status": "active",
  "repository": {
    "url": "https://github.com/example/test-server",
    "source": "github",
    "id": "example/test-server"
  },
  "version_detail": {
    "version": "1.0.0",
    "release_date": "2025-07-20T12:00:00Z",
    "is_latest": true
  },
  "packages": [
    {
      "registry_name": "npm",
      "name": "@example/test-server",
      "version": "1.0.0",
      "runtime_hint": "npx",
      "package_arguments": [
        {
          "type": "positional",
          "value_hint": "config_path",
          "description": "Path to configuration file",
          "default": "./config.json",
          "is_required": true
        }
      ],
      "environment_variables": [
        {
          "name": "API_KEY",
          "description": "API key for external service",
          "is_required": true,
          "is_secret": true
        }
      ]
    }
  ]
}
```

See `example-server.json` for a complete example.

## Authentication

Publishing servers requires authentication:

1. **GitHub Namespaced Servers** (`io.github.*`): Requires a GitHub token
   - Generate a token at: https://github.com/settings/tokens
   - Token needs appropriate repository permissions

2. **Other Namespaces**: May require different authentication methods or no authentication

## Development

### Building

```bash
# Build for current platform
make build

# Build for Linux
make build-linux

# Clean build artifacts
make clean
```

### Testing

```bash
# Run tests
make test

# Format code
make fmt

# Vet code
make vet
```

### Demo Commands

If you have an mcpx server running locally:

```bash
# Test health endpoint
make demo-health

# Test servers listing
make demo-servers

# Run all demos
make demo-all
```

## Examples

### Complete Workflow

```bash
# 1. Check api health
mcpx-cli health

# 2. List available servers
mcpx-cli servers --limit 5

# 3. Get details for a specific server
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1

# 4. Publish a new server (requires token)
mcpx-cli publish example-server.json --token ghp_your_token_here
```

### Working with Different Registries

```bash
# Development environment
mcpx-cli --base-url http://localhost:8080 health

# Staging environment
mcpx-cli --base-url https://staging-registry.example.com servers

# Production environment
mcpx-cli --base-url https://registry.modelcontextprotocol.io servers
```

## Error Handling

The CLI provides clear error messages for common issues:

- **Connection errors**: Network connectivity problems
- **Authentication errors**: Invalid or missing tokens
- **Validation errors**: Invalid server JSON format
- **Server errors**: API-specific error responses

All errors include HTTP status codes and detailed error messages when available.

## License

This project is licensed under the same license as the mcpx project.
