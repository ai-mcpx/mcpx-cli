# mcpx-cli

A command-line interface for interacting with the mcpx registry api. This CLI provides easy access to all the core mcpx api endpoints for managing MCP servers with enhanced authentication support.

## Features

- **Authentication System**: Multiple authentication methods including GitHub OAuth, GitHub OIDC, and anonymous access
- **Repository Source Support**: Full support for GitHub, GitLab, and Gerrit repositories with automatic URL validation
- **Automatic Token Management**: Secure credential storage and automatic token refresh
- **Health Check**: Verify api connectivity and status
- **Server Listing**: Browse available MCP servers with pagination
- **Server Details**: Get comprehensive information about specific servers
- **Detailed Server Information**: Retrieve complete server data including packages and remotes
- **Server Publishing**: Publish new MCP servers to the registry
- **Server Updates**: Update existing MCP servers in the registry
- **Server Deletion**: Delete servers from the registry
- **Interactive Mode**: Create server configurations interactively with Node.js, Python PyPI, Python Wheel, and Binary templates
- **JSON Output**: All responses are formatted for easy reading with optional detailed information
- **Configurable Base URL**: Target different mcpx registry instances

## Authentication Methods

The mcpx-cli supports multiple authentication methods that match the backend capabilities:

- **Anonymous**: Basic access without GitHub authentication
- **GitHub OAuth**: Full GitHub OAuth flow for authenticated access
- **GitHub OIDC**: GitHub OpenID Connect for enterprise environments
- **DNS**: Domain-based authentication (future implementation)
- **HTTP**: HTTP-based authentication (future implementation)

## API Endpoints Supported

- `POST /api/auth/anonymous` - Anonymous authentication
- `POST /api/auth/github/oauth` - GitHub OAuth authentication
- `POST /api/auth/github/oidc` - GitHub OIDC authentication
- `GET /v0/health` - Health check and status
- `GET /v0/servers` - List servers with basic information and optional pagination
- `GET /v0/servers/{id}` - Get detailed server information including packages and remotes
- `POST /v0/publish` - Publish a new server
- `PUT /v0/servers/{id}` - Update an existing server
- `DELETE /v0/servers/{id}` - Delete a server from the registry

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

#### Authentication

##### Login

Authenticate with the mcpx registry using various methods:

```bash
# Login with anonymous authentication (default)
mcpx-cli login --method anonymous

# Login with GitHub OAuth
mcpx-cli login --method github-oauth

# Login with GitHub OIDC
mcpx-cli login --method github-oidc

# Default to anonymous if no method specified
mcpx-cli login
```

**Authentication Flags:**
- `--method string`: Authentication method (anonymous, github-oauth, github-oidc) (default: anonymous)

Authentication credentials are automatically saved to `~/.mcpx-cli-config.json` and used for subsequent API calls.

##### Logout

Clear stored authentication credentials:

```bash
mcpx-cli logout
```

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
          "registry_type": "npm",
          "identifier": "@modelcontextprotocol/server-filesystem",
          "version": "1.0.2",
          "runtime_hint": "npx",
          "transport_type": "stdio",
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
      "registry_type": "npm",
      "identifier": "@modelcontextprotocol/server-filesystem",
      "version": "1.0.2",
      "runtime_hint": "npx",
      "transport_type": "stdio",
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

#### Delete Server

Delete a server from the registry. Authentication is automatically handled through stored credentials or explicit tokens.

```bash
# Using stored authentication (recommended)
mcpx-cli delete <server-id>

# Using explicit token (legacy method)
mcpx-cli delete <server-id> --token <auth-token>

# With JSON output
mcpx-cli delete <server-id> --json

# Combined flags
mcpx-cli delete <server-id> --token <auth-token> --json
```

Example:
```bash
# Login first (authentication is stored automatically)
mcpx-cli login --method github-oauth

# Delete using stored authentication
mcpx-cli delete a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1

# Alternative: provide token explicitly
mcpx-cli delete a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 --token ghp_your_github_token_here

# Output result in JSON format
mcpx-cli delete a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 --json

# For anonymous deletion (if supported by registry policy)
mcpx-cli login --method anonymous
mcpx-cli delete a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1
```

**Flags:**
- `--token string`: Authentication token (optional if using stored authentication)
- `--json`: Output result in JSON format

**Important Notes:**
- **Irreversible operation**: Deleted servers cannot be recovered
- **Authentication**: Automatically uses stored credentials or provided token
- **Permission check**: You can only delete servers you have permission to modify
- **Confirmation prompt**: By default, the CLI will ask for confirmation before deletion

Example output:
```
=== Delete Server (ID: a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1) ===
Server Name: io.modelcontextprotocol/filesystem
Server Description: Node.js server implementing Model Context Protocol (MCP) for filesystem operations

Are you sure you want to delete this server? This action cannot be undone.
Type 'yes' to confirm: yes

Status Code: 200
Success: Server deleted successfully
```

JSON output example:
```json
{"message": "Server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 deleted successfully"}
```

#### Update Server

Update an existing MCP server in the registry. Authentication is automatically handled through stored credentials or explicit tokens.

```bash
# Using stored authentication (recommended)
mcpx-cli update <server-id> <server.json>

# Using explicit token (legacy method)
mcpx-cli update <server-id> <server.json> --token <auth-token>

# With JSON output
mcpx-cli update <server-id> <server.json> --json

# Combined flags
mcpx-cli update <server-id> <server.json> --token <auth-token> --json
```

Example:
```bash
# Login first (authentication is stored automatically)
mcpx-cli login --method github-oauth

# Update using stored authentication
mcpx-cli update a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 updated-server.json

# Alternative: provide token explicitly
mcpx-cli update a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 updated-server.json --token ghp_your_github_token_here

# Output result in JSON format
mcpx-cli update a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 updated-server.json --json

# For anonymous updates (if supported by registry policy)
mcpx-cli login --method anonymous
mcpx-cli update a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 updated-server.json
```

**Flags:**
- `--token string`: Authentication token (optional if using stored authentication)
- `--json`: Output result in JSON format

**Important Notes:**
- **Server configuration file**: The JSON file should contain the complete server configuration
- **Authentication**: Automatically uses stored credentials or provided token
- **Permission check**: You can only update servers you have permission to modify
- **Validation**: The server configuration will be validated before update

Example server configuration file (`updated-server.json`):
```json
{
  "name": "io.modelcontextprotocol/filesystem",
  "description": "Updated Node.js server implementing Model Context Protocol (MCP) for filesystem operations",
  "repository": {
    "url": "https://github.com/modelcontextprotocol/servers",
    "source": "github"
  },
  "version_detail": {
    "version": "1.0.3",
    "is_latest": true
  },
  "packages": [
    {
      "registry_type": "npm",
      "identifier": "@modelcontextprotocol/server-filesystem",
      "version": "1.0.3",
      "runtime_hint": "npx",
      "transport_type": "stdio",
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
```

Example output:
```
=== Update Server (ID: a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1) ===
Status Code: 200
Success: Server updated successfully
Updated Server Name: io.modelcontextprotocol/filesystem
Updated Version: 1.0.3
```

JSON output example:
```json
{
  "message": "Server updated successfully",
  "server": {
    "id": "a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1",
    "name": "io.modelcontextprotocol/filesystem",
    "version": "1.0.3"
  }
}
```

#### Publish Server

Publish a new MCP server to the registry. The CLI supports two modes and automatic authentication:

##### File-based Publishing

Publish using an existing server configuration file:

```bash
# Using stored authentication (recommended)
mcpx-cli publish <server.json>

# Using explicit token (legacy method)
mcpx-cli publish <server.json> --token <auth-token>
```

Example:
```bash
# Login first (authentication is stored automatically)
mcpx-cli login --method github-oauth

# Then publish (uses stored authentication)
mcpx-cli publish example-server.json

# Alternative: provide token explicitly (for GitHub namespaced servers)
mcpx-cli publish example-server.json --token ghp_your_github_token_here

# For non-GitHub namespaces with anonymous authentication
mcpx-cli login --method anonymous
mcpx-cli publish example-server.json
```

##### Interactive Publishing

Create and publish a server configuration interactively:

```bash
# Using stored authentication (recommended)
mcpx-cli publish --interactive

# Using explicit token (legacy method)
mcpx-cli publish --interactive --token <auth-token>
```

Examples:
```bash
# Login first with appropriate method
mcpx-cli login --method github-oauth

# Then use interactive mode (uses stored authentication)
mcpx-cli publish --interactive

# Alternative: provide token explicitly for GitHub namespaced servers
mcpx-cli publish --interactive --token ghp_your_github_token_here

# For non-GitHub projects with anonymous authentication
mcpx-cli login --method anonymous
mcpx-cli publish --interactive
```

The interactive mode will:
1. **Choose Runtime**: Select between Node.js, Python, Binary, and Gerrit server templates
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

### Node.js Template (`example-server-npm.json`)

The Node.js template is pre-configured with:
- NPM package registry settings (`registry_type: "npm"`)
- `npx` runtime hint for execution
- Standard transport type (`transport_type: "stdio"`)
- Standard environment variables (`MCP_HOST`, `MCP_PORT`)
- Positional arguments for configuration file path

### Python PyPI Template (`example-server-pypi.json`)

The Python PyPI template includes:
- PyPI package registry settings (`registry_type: "pypi"`)
- `python` runtime hint for execution
- Standard transport type (`transport_type: "stdio"`)
- Python-specific environment variables (`PYTHONPATH`, `MCP_HOST`, `MCP_PORT`)
- Option and positional arguments for script execution

### Python Wheel Template (`example-server-wheel.json`)

The Python Wheel template includes:
- Wheel package registry settings (`registry_type: "wheel"`)
- Direct wheel URL for package distribution
- `python` runtime hint for execution
- Standard transport type (`transport_type: "stdio"`)
- Python-specific environment variables

### Binary Template (`example-server-binary.json`)

The Binary template includes:
- Binary package registry settings (`registry_type: "binary"`)
- Direct binary URL for package distribution
- `binary` runtime hint for execution
- Standard transport type (`transport_type: "stdio"`)
- Configuration arguments for binary execution

### Gerrit Template (`example-server-gerrit.json`)

The Gerrit template includes:
- Gerrit repository source (`source: "gerrit"`)
- PyPI and OCI package registry support
- Python and Docker runtime hints
- Standard transport types (`transport_type: "stdio"` and `streamable-http`)
- Gerrit-specific repository URL format

All templates provide sensible defaults that can be customized during the interactive configuration process and support the new transport type specifications.

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

## Repository Sources

The mcpx-cli supports multiple repository sources when publishing servers. The repository information helps users and security experts inspect the source code of MCP servers for transparency.

### Supported Repository Sources

| Source | URL Format | Example | Description |
|--------|------------|---------|-------------|
| **GitHub** | `https://github.com/user/repo` | `https://github.com/microsoft/vscode` | Most common, supports GitHub authentication |
| **GitLab** | `https://gitlab.com/user/repo` | `https://gitlab.com/gitlab-org/gitlab` | GitLab-hosted repositories |
| **Gerrit** | `http://host:port/project/path` | `http://gerrit.example.com:8080/my-project` | Enterprise Gerrit installations |

### Repository Configuration Examples

#### GitHub Repository
```json
{
  "name": "io.github.example/my-server",
  "description": "My awesome MCP server",
  "repository": {
    "url": "https://github.com/example/my-server",
    "source": "github",
    "id": "example/my-server"
  }
}
```

#### GitLab Repository
```json
{
  "name": "io.modelcontextprotocol.anonymous/gitlab-server",
  "description": "MCP server hosted on GitLab",
  "repository": {
    "url": "https://gitlab.com/myorg/my-server",
    "source": "gitlab",
    "id": "myorg/my-server"
  }
}
```

#### Gerrit Repository
```json
{
  "name": "io.modelcontextprotocol.anonymous/enterprise-server",
  "description": "Enterprise MCP server from Gerrit",
  "repository": {
    "url": "http://gerrit.company.com:8080/plugins/gitiles/mcp-server/+/refs/heads/main",
    "source": "gerrit",
    "id": "mcp-server"
  }
}
```

### Repository Source Validation

The CLI automatically validates repository URLs based on the specified source:

- **GitHub**: Validates `github.com` domain and proper owner/repo format
- **GitLab**: Validates `gitlab.com` domain and proper owner/repo format
- **Gerrit**: Accepts flexible URL formats for various Gerrit installations

**Note**: Repository information is validated by the registry server. Ensure your repository URL is publicly accessible for transparency and security validation.

## Examples

### Authentication Workflows

#### GitHub-based Project Workflow

```bash
# 1. Login with GitHub OAuth
mcpx-cli login --method github-oauth

# 2. Check login status
mcpx-cli health

# 3. Publish/update/delete operations use stored authentication automatically
mcpx-cli publish example-server.json
mcpx-cli update server-id updated-server.json
mcpx-cli delete server-id

# 4. Logout when done
mcpx-cli logout
```

#### Anonymous Access Workflow

```bash
# 1. Login anonymously
mcpx-cli login --method anonymous

# 2. Browse and interact with the registry
mcpx-cli servers --json --detailed
mcpx-cli server server-id

# 3. Publish non-GitHub servers (if allowed by registry policy)
mcpx-cli publish example-server.json

# 4. Logout when done
mcpx-cli logout
```

#### Mixed Authentication Workflow

```bash
# 1. Start with anonymous access for browsing
mcpx-cli login --method anonymous
mcpx-cli servers --limit 10

# 2. Switch to GitHub authentication for publishing
mcpx-cli login --method github-oauth
mcpx-cli publish --interactive

# 3. Continue with authenticated operations
mcpx-cli update server-id updated-server.json
```

### Complete Workflow

```bash
# 0. Check CLI version
mcpx-cli --version

# 1. Authenticate (stored for subsequent commands)
mcpx-cli login --method github-oauth

# 2. Check api health
mcpx-cli health

# 3. List available servers
mcpx-cli servers --limit 5

# 4. List servers in JSON format (for programmatic processing)
mcpx-cli servers --json --limit 10

# 5. List servers with complete details including packages and remotes
mcpx-cli servers --json --detailed --limit 5

# 6. Get details for a specific server
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1

# 7. Get server details in JSON format (for programmatic processing)
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 --json

# 8. Publish a new server (file-based) - uses stored authentication
mcpx-cli publish example-server.json

# 9. Publish a new server (interactive mode) - uses stored authentication
mcpx-cli publish --interactive

# 10. Update an existing server - uses stored authentication
mcpx-cli update a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 updated-server.json

# 11. Delete a server - uses stored authentication
mcpx-cli delete a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1

# 12. Logout when done
mcpx-cli logout
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

# Extract package identifiers from all servers with detailed info
mcpx-cli servers --json --detailed | jq -r '.servers[].packages[].identifier'

# Find servers with specific package registry
mcpx-cli servers --json --detailed | jq '.servers[] | select(.packages[]?.registry_type == "npm")'

# Get all remote transport types
mcpx-cli servers --json --detailed | jq -r '.servers[].remotes[]?.transport_type' | sort -u

# Get detailed server information in JSON format
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 --json

# Extract specific fields from server details
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 --json | jq '.packages[].identifier'

# Get all package registries from a server
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 --json | jq -r '.packages[].registry_type'

# Extract remote URLs
mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 --json | jq -r '.remotes[].url'

# Batch operations with server management
# Find servers with deprecated status and extract their IDs
mcpx-cli servers --json --detailed | jq -r '.servers[] | select(.status == "deprecated") | .id'

# Delete multiple servers by filtering (example with careful scripting)
# Note: Always verify IDs before bulk deletion!
TOKEN="ghp_your_token_here"
mcpx-cli servers --json | jq -r '.servers[] | select(.name | contains("test")) | .id' | \
  while read id; do
    echo "Deleting server: $id"
    mcpx-cli delete "$id" --token "$TOKEN" --confirm --json | jq '.message'
  done

# Delete server with JSON output for automation/logging
mcpx-cli delete a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1 --json --token "$TOKEN" | jq '.message'
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

## Authentication Configuration

### Configuration File

Authentication credentials are automatically stored in `~/.mcpx-cli-config.json` when you use the `login` command. The file structure is:

```json
{
  "method": "github-oauth",
  "token": "gho_xxxxxxxxxxxx",
  "expires_at": 1693612800
}
```

### Security Notes

- **File Permissions**: The configuration file is created with `0600` permissions (readable only by the user)
- **Token Expiration**: Tokens are automatically checked for expiration before use
- **Automatic Cleanup**: Expired tokens are automatically removed from the configuration
- **Manual Cleanup**: Use `mcpx-cli logout` to clear stored credentials

### Supported Authentication Methods

| Method | Description | Use Case |
|--------|-------------|----------|
| `anonymous` | Basic anonymous access | Public browsing, non-GitHub servers |
| `github-oauth` | GitHub OAuth authentication | GitHub-namespaced servers, full access |
| `github-oidc` | GitHub OIDC authentication | Enterprise environments, CI/CD |
| `dns` | Domain-based authentication | Future implementation |
| `http` | HTTP-based authentication | Future implementation |

### Migration from Token-based Authentication

If you were previously using the `--token` flag, you can migrate to the new authentication system:

```bash
# Old way (still supported)
mcpx-cli publish server.json --token ghp_your_token_here

# New way (recommended)
mcpx-cli login --method github-oauth
mcpx-cli publish server.json
```

The token-based authentication is still supported for backward compatibility and CI/CD environments where interactive login is not possible.

## Error Handling

The CLI provides clear error messages for common issues:

- **Connection errors**: Network connectivity problems
- **Authentication errors**: Invalid or missing tokens
- **Validation errors**: Invalid server JSON format
- **Permission errors**: Attempting to delete servers you don't own
- **Not found errors**: Server ID not found for deletion
- **Server errors**: API-specific error responses

All errors include HTTP status codes and detailed error messages when available.

## License

This project is licensed under the same license as the mcpx project.
