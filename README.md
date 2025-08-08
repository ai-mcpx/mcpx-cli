# mcpx-cli

A command-line interface for interacting with the mcpx registry api. This CLI provides easy access to all the core mcpx api endpoints for managing MCP servers.

## Features

- **Health Check**: Verify api connectivity and status
- **Server Listing**: Browse available MCP servers with pagination
- **Server Details**: Get comprehensive information about specific servers
- **Detailed Server Information**: Retrieve complete server data including packages and remotes
- **Server Publishing**: Publish new MCP servers to the registry (requires authentication)
- **Interactive Mode**: Create server configurations interactively with Node.js and Python templates
- **JSON Output**: All responses are formatted for easy reading with optional detailed information
- **Configurable Base URL**: Target different mcpx registry instances

## API Endpoints Supported

- `GET /v0/health` - Health check and status
- `GET /v0/servers` - List servers with basic information and optional pagination
- `GET /v0/servers/{id}` - Get detailed server information including packages and remotes
- `POST /v0/publish` - Publish a new server (requires authentication)

**Note**: The API follows a common pattern where the list endpoint (`/v0/servers`) returns basic server metadata for efficient browsing, while the detail endpoint (`/v0/servers/{id}`) provides complete information including packages and remotes. The CLI's `--detailed` flag bridges this gap by automatically fetching detailed information for all servers in a list.

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

- `--base-url=string`: Base url of the mcpx api (default: http://localhost:8080)
- `--version`: Show version information

### Commands

#### Version Information

Check the CLI version:

```bash
mcpx-cli --version
# or
mcpx-cli version
```

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

# Output in JSON format
mcpx-cli servers --json

# Get detailed server information including packages and remotes
mcpx-cli servers --json --detailed

# Combine JSON with pagination and detailed info
mcpx-cli servers --json --limit 10 --detailed
```

**Flags:**
- `--cursor string`: Pagination cursor for next page
- `--limit int`: Maximum number of servers to return (default: 30)
- `--json`: Output servers details in JSON format
- `--detailed`: Include packages and remotes in JSON output (requires --json)

**Note**: The `--detailed` flag makes individual API calls for each server to retrieve complete information. For large server lists, consider using `--limit` to reduce the number of requests and improve performance.

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

Example JSON output (with `--json` flag):
```json
{
  "servers": [
    {
      "id": "a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1",
      "name": "io.modelcontextprotocol/filesystem",
      "description": "Node.js server implementing Model Context Protocol (MCP) for filesystem operations",
      "repository": {
        "url": "https://github.com/modelcontextprotocol/servers",
        "source": "github",
        "id": "modelcontextprotocol/servers"
      },
      "version_detail": {
        "version": "1.0.2",
        "release_date": "2023-06-15T10:30:00Z",
        "is_latest": true
      }
    }
  ],
  "metadata": {
    "next_cursor": "some-uuid-cursor",
    "count": 1,
    "total": 2
  }
}
```

**Note**: The basic `--json` output only includes server metadata. For complete server information including packages and remotes, use `--detailed` with `--json`:

```bash
mcpx-cli servers --json --detailed
```

Example detailed JSON output (with `--detailed` flag):
```json
{
  "servers": [
    {
      "id": "a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1",
      "name": "io.modelcontextprotocol/filesystem",
      "description": "Node.js server implementing Model Context Protocol (MCP) for filesystem operations",
      "repository": {
        "url": "https://github.com/modelcontextprotocol/servers",
        "source": "github",
        "id": "modelcontextprotocol/servers"
      },
      "version_detail": {
        "version": "1.0.2",
        "release_date": "2023-06-15T10:30:00Z",
        "is_latest": true
      },
      "packages": [
        {
          "registry_name": "npm",
          "name": "@modelcontextprotocol/server-filesystem",
          "version": "1.0.2",
          "runtime_hint": "npx",
          "runtime_arguments": [
            {
              "type": "positional",
              "name": "target_dir",
              "description": "Path to access",
              "format": "string",
              "is_required": true,
              "default": "/Users/username/Desktop",
              "value_hint": "target_dir"
            }
          ],
          "environment_variables": [
            {
              "name": "LOG_LEVEL",
              "description": "Logging level (debug, info, warn, error)",
              "format": "string",
              "is_required": false,
              "default": "info"
            }
          ]
        }
      ],
      "remotes": [
        {
          "transport_type": "stdio",
          "url": "npx @modelcontextprotocol/server-filesystem"
        }
      ]
    }
  ],
  "metadata": {
    "next_cursor": "some-uuid-cursor",
    "count": 1,
    "total": 2
  }
}
```

#### Get Server Details

Get comprehensive information about a specific server:

```bash
mcpx-cli server <server-id>

# Output in JSON format
mcpx-cli server <server-id> --json
```

Example:
```bash
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1

# Or with JSON output
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 --json
```

**Flags:**
- `--json`: Output server details in JSON format

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

Example JSON output (with `--json` flag):
```json
{
  "id": "a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1",
  "name": "io.modelcontextprotocol/filesystem",
  "description": "Node.js server implementing Model Context Protocol (MCP) for filesystem operations",
  "repository": {
    "url": "https://github.com/modelcontextprotocol/servers",
    "source": "github",
    "id": "modelcontextprotocol/servers"
  },
  "version_detail": {
    "version": "1.0.2",
    "release_date": "2023-06-15T10:30:00Z",
    "is_latest": true
  },
  "packages": [
    {
      "registry_name": "npm",
      "name": "@modelcontextprotocol/server-filesystem",
      "version": "1.0.2",
      "runtime_hint": "npx",
      "runtime_arguments": [
        {
          "type": "named",
          "name": "--module",
          "description": "Run as Node.js module",
          "format": "string",
          "is_required": true,
          "default": "-m",
          "value_hint": "-m"
        }
      ],
      "environment_variables": [
        {
          "name": "LOG_LEVEL",
          "description": "Logging level (debug, info, warn, error)",
          "format": "string",
          "is_required": false,
          "default": "info"
        }
      ]
    }
  ],
  "remotes": [
    {
      "transport_type": "stdio",
      "url": "npx @modelcontextprotocol/server-filesystem"
    }
  ]
}
```

#### Publish Server

Publish a new MCP server to the registry. The CLI supports two modes:

##### File-based Publishing

Publish using an existing server configuration file:

```bash
mcpx-cli publish <server.json> --token <auth-token>
```

Example:
```bash
# For GitHub namespaced servers (token required)
mcpx-cli publish example-server.json --token ghp_your_github_token_here

# For other namespaces (token optional)
mcpx-cli publish example-server.json
```

##### Interactive Publishing

Create and publish a server configuration interactively:

```bash
mcpx-cli publish --interactive --token <auth-token>
```

Examples:
```bash
# For GitHub namespaced servers (token required)
mcpx-cli publish --interactive --token ghp_your_github_token_here

# For other namespaces (token optional)
mcpx-cli publish --interactive
```

The interactive mode will:
1. **Choose Runtime**: Select between Node.js and Python server templates
2. **Configure Server**: Set name, description, and repository information
3. **Set Version**: Specify package version and details
4. **Environment Setup**: Configure environment variables and runtime settings
5. **Save & Publish**: Optionally save the configuration file and publish to registry

Example interactive session:
```bash
mcpx-cli publish --interactive --token ghp_your_token_here

=== Interactive Server Configuration ===

Select server runtime:
  * 1) node
    2) python
Enter choice (1-2): 1

Server name [io.github.example/test-server-node]: my-awesome-server
Server description [A test mcp server in node]: My awesome MCP server for file operations

--- Repository Information ---
Repository URL [https://github.com/example/test-server-node]: https://github.com/myuser/awesome-server
Repository ID (e.g., username/repo) [example/test-server-node]: myuser/awesome-server

--- Version Information ---
Version [1.0.0]: 1.1.0

--- Package Information ---
NPM package name [@example/test-server-node]: @myuser/awesome-server
Package version [1.1.0]: 1.1.0

--- Environment Variables ---
MCP_HOST default value [0.0.0.0]:
MCP_PORT default value [8000]: 3000

--- Remote Configuration ---
Server URL [http://localhost:8000]: http://localhost:3000

Save configuration to file?
  * 1) yes
    2) no
Enter choice (1-2): 1
Filename [server-config.json]: my-server-config.json
Configuration saved to my-server-config.json

=== Server Configuration Preview ===
Name: my-awesome-server
Description: My awesome MCP server for file operations
Version: 1.1.0
Repository: https://github.com/myuser/awesome-server

Proceed with publishing?
    1) yes
  * 2) no
Enter choice (1-2): 1

Status Code: 201
Success: Server publication successful
Server ID: b1234567-8901-2345-6789-012345678901
```

**Flags:**
- `--token string`: Authentication token (required for `io.github.*` namespaced servers, optional for others)
- `--interactive`: Enable interactive mode to create server configuration

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
mcpx-cli --base-url=http://localhost:8080 servers

# Production instance
mcpx-cli --base-url=https://registry.modelcontextprotocol.io servers

# Custom instance
mcpx-cli --base-url=https://your-custom-registry.com servers
```

## Server JSON Format

When publishing servers, you need to provide a JSON file describing the server.

See `example-server-node.json` and `example-server-python.json` for complete examples.

## Interactive Mode Templates

The interactive mode uses built-in templates for different server runtimes:

### Node.js Template (`example-server-node.json`)

The Node.js template is pre-configured with:
- NPM package registry settings
- `npx` runtime hint for execution
- Standard environment variables (`MCP_HOST`, `MCP_PORT`)
- HTTP transport configuration
- Positional arguments for configuration file path

### Python Template (`example-server-python.json`)

The Python template includes:
- PyPI package registry settings
- `python` runtime hint for execution
- Python-specific environment variables (`PYTHONPATH`, `MCP_HOST`, `MCP_PORT`)
- HTTP transport configuration
- Option and positional arguments for script execution

Both templates provide sensible defaults that can be customized during the interactive configuration process.

## Authentication

Publishing servers may require authentication depending on the namespace:

1. **GitHub Namespaced Servers** (`io.github.*`): **Requires** a GitHub token
   - Generate a token at: https://github.com/settings/tokens
   - Token needs appropriate repository permissions
   - Use: `--token ghp_your_github_token_here`

2. **Other Namespaces**: Authentication is **optional**
   - Some registries may not require authentication
   - Others may use different authentication methods
   - Check with your registry administrator for specific requirements

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
# 0. Check CLI version
mcpx-cli --version

# 1. Check api health
mcpx-cli health

# 2. List available servers
mcpx-cli servers --limit 5

# 3. List servers in JSON format (for programmatic processing)
mcpx-cli servers --json --limit 10

# 3b. List servers with complete details including packages and remotes
mcpx-cli servers --json --detailed --limit 5

# 4. Get details for a specific server
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1

# 4b. Get server details in JSON format (for programmatic processing)
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 --json

# 5. Publish a new server (file-based)
mcpx-cli publish example-server.json --token ghp_your_token_here  # GitHub projects
mcpx-cli publish example-server.json  # Non-GitHub projects

# 6. Publish a new server (interactive mode)
mcpx-cli publish --interactive --token ghp_your_token_here  # GitHub projects
mcpx-cli publish --interactive  # Non-GitHub projects
```

### JSON Output for Automation

The `--json` flag is perfect for scripting and automation:

```bash
# Get servers as JSON and process with jq
mcpx-cli servers --json | jq '.servers[].name'

# Find servers by name pattern
mcpx-cli servers --json | jq '.servers[] | select(.name | contains("filesystem"))'

# Extract server IDs for batch processing
mcpx-cli servers --json | jq -r '.servers[].id'

# Get all servers with detailed information (packages and remotes)
mcpx-cli servers --json --detailed

# Extract package names from all servers with detailed info
mcpx-cli servers --json --detailed | jq -r '.servers[].packages[].name'

# Find servers with specific package registry
mcpx-cli servers --json --detailed | jq '.servers[] | select(.packages[]?.registry_name == "npm")'

# Get all remote transport types
mcpx-cli servers --json --detailed | jq -r '.servers[].remotes[]?.transport_type' | sort -u

# Get detailed server information in JSON format
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 --json

# Extract specific fields from server details
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 --json | jq '.packages[].name'

# Get all package registries from a server
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 --json | jq -r '.packages[].registry_name'

# Extract remote URLs
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 --json | jq -r '.remotes[].url'
```

### Working with Different Registries

```bash
# Development environment
mcpx-cli --base-url=http://localhost:8080 health

# Staging environment
mcpx-cli --base-url=https://staging-registry.example.com servers

# Production environment
mcpx-cli --base-url=https://registry.modelcontextprotocol.io servers
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
