package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

//go:embed example-server-npm.json
var exampleServerNPMJSON []byte

//go:embed example-server-pypi.json
var exampleServerPyPiJSON []byte

//go:embed example-server-wheel.json
var exampleServerWheelJSON []byte

//go:embed example-server-binary.json
var exampleServerBinaryJSON []byte

const (
	defaultBaseURL = "http://localhost:8080"
	configFileName = ".mcpx-cli-config.json"

	// Authentication methods (matching backend)
	AuthMethodGitHubOAuth = "github-oauth"
	AuthMethodGitHubOIDC  = "github-oidc"
	AuthMethodAnonymous   = "anonymous"
	AuthMethodDNS         = "dns"
	AuthMethodHTTP        = "http"
)

var version = "dev"

// Auth configuration structure
type AuthConfig struct {
	Token     string `json:"token"`
	Method    string `json:"method"`
	Domain    string `json:"domain,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

// Token response from auth endpoints
type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

type HealthResponse struct {
	Status         string `json:"status"`
	GitHubClientID string `json:"github_client_id"`
}

type Repository struct {
	URL    string `json:"url"`
	Source string `json:"source"`
	ID     string `json:"id"`
}

type VersionDetail struct {
	Version     string `json:"version"`
	ReleaseDate string `json:"release_date"`
	IsLatest    bool   `json:"is_latest"`
}

type Server struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Status        string        `json:"status,omitempty"`
	Repository    Repository    `json:"repository"`
	VersionDetail VersionDetail `json:"version_detail"`
}

type Metadata struct {
	NextCursor string `json:"next_cursor,omitempty"`
	Count      int    `json:"count,omitempty"`
	Total      int    `json:"total,omitempty"`
}

type ServersResponse struct {
	Servers  []ServerWrapper `json:"servers"`
	Metadata Metadata        `json:"metadata,omitempty"`
}

type DetailedServersResponse struct {
	Servers  []ServerDetailWrapper `json:"servers"`
	Metadata Metadata              `json:"metadata,omitempty"`
}

// New wrapper types for the API format
type ServerWrapper struct {
	Server       Server                 `json:"server"`
	RegistryMeta map[string]interface{} `json:"x-io.modelcontextprotocol.registry,omitempty"`
}

type ServerDetailWrapper struct {
	Server       ServerDetail           `json:"server"`
	RegistryMeta map[string]interface{} `json:"x-io.modelcontextprotocol.registry,omitempty"`
}

// Legacy response types for backward compatibility
type LegacyServersResponse struct {
	Servers  []Server `json:"servers"`
	Metadata Metadata `json:"metadata,omitempty"`
}

type LegacyDetailedServersResponse struct {
	Servers  []ServerDetail `json:"servers"`
	Metadata Metadata       `json:"metadata,omitempty"`
}

type Input struct {
	Description string   `json:"description,omitempty"`
	IsRequired  bool     `json:"is_required,omitempty"`
	Format      string   `json:"format,omitempty"`
	Value       string   `json:"value,omitempty"`
	IsSecret    bool     `json:"is_secret,omitempty"`
	Default     string   `json:"default,omitempty"`
	Choices     []string `json:"choices,omitempty"`
}

type InputWithVariables struct {
	Input     `json:",inline"`
	Variables map[string]Input `json:"variables,omitempty"`
}

type KeyValueInput struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	IsRequired  bool             `json:"is_required,omitempty"`
	Format      string           `json:"format,omitempty"`
	Value       string           `json:"value,omitempty"`
	IsSecret    bool             `json:"is_secret,omitempty"`
	Default     string           `json:"default,omitempty"`
	Choices     []string         `json:"choices,omitempty"`
	Variables   map[string]Input `json:"variables,omitempty"`
}

type Argument struct {
	InputWithVariables `json:",inline"`
	Type               string `json:"type"`
	Name               string `json:"name,omitempty"`
	IsRepeated         bool   `json:"is_repeated,omitempty"`
	ValueHint          string `json:"value_hint,omitempty"`
}

type Package struct {
	RegistryName         string          `json:"registry_name"`
	Name                 string          `json:"name"`
	Version              string          `json:"version"`
	WheelURL             string          `json:"wheel_url,omitempty"`
	BinaryURL            string          `json:"binary_url,omitempty"`
	RuntimeHint          string          `json:"runtime_hint,omitempty"`
	RuntimeArguments     []Argument      `json:"runtime_arguments,omitempty"`
	PackageArguments     []Argument      `json:"package_arguments,omitempty"`
	EnvironmentVariables []KeyValueInput `json:"environment_variables,omitempty"`
}

type Remote struct {
	TransportType string          `json:"transport_type"`
	URL           string          `json:"url"`
	Headers       []KeyValueInput `json:"headers,omitempty"`
}

type ServerDetail struct {
	Server   `json:",inline"`
	Packages []Package `json:"packages,omitempty"`
	Remotes  []Remote  `json:"remotes,omitempty"`
}

type PublishResponse struct {
	Message string `json:"message"`
	ID      string `json:"id"`
}

// PublishRequest represents the request format for the /v0/publish endpoint
type PublishRequest struct {
	Server     ServerDetail           `json:"server"`
	XPublisher map[string]interface{} `json:"x-publisher,omitempty"`
}

type MCPXClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewMCPXClient(baseURL string) *MCPXClient {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &MCPXClient{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Authentication helper methods
func (c *MCPXClient) saveAuthConfig(config AuthConfig) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := fmt.Sprintf("%s/%s", homeDir, configFileName)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, data, 0600)
}

func (c *MCPXClient) loadAuthConfig() (AuthConfig, error) {
	var config AuthConfig
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := fmt.Sprintf("%s/%s", homeDir, configFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil // No config file is OK
		}
		return config, fmt.Errorf("failed to read config: %w", err)
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Check if token is expired
	if config.ExpiresAt > 0 && time.Now().Unix() > config.ExpiresAt {
		return AuthConfig{}, nil // Return empty config if expired
	}

	return config, nil
}

func (c *MCPXClient) clearAuthConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := fmt.Sprintf("%s/%s", homeDir, configFileName)
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config file: %w", err)
	}

	return nil
}

func (c *MCPXClient) makeRequest(method, endpoint string, body []byte, token string) (*http.Response, error) {
	url := c.baseURL + endpoint

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Use provided token or auto-load from config
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		config, _ := c.loadAuthConfig()
		if config.Token != "" {
			req.Header.Set("Authorization", "Bearer "+config.Token)
		}
	}

	req.Header.Set("User-Agent", "mcpx-cli/1.0")

	return c.httpClient.Do(req)
}

// Authentication commands
func (c *MCPXClient) login(authMethod string) error {
	switch authMethod {
	case AuthMethodGitHubOAuth:
		return c.loginGitHubOAuth()
	case AuthMethodGitHubOIDC:
		return c.loginGitHubOIDC()
	case AuthMethodAnonymous:
		return c.loginAnonymous()
	default:
		return fmt.Errorf("unsupported authentication method: %s", authMethod)
	}
}

func (c *MCPXClient) loginGitHubOAuth() error {
	// Implement GitHub OAuth flow
	fmt.Println("GitHub OAuth authentication not yet implemented")
	return nil
}

func (c *MCPXClient) loginGitHubOIDC() error {
	// Implement GitHub OIDC flow
	fmt.Println("GitHub OIDC authentication not yet implemented")
	return nil
}

func (c *MCPXClient) loginAnonymous() error {
	resp, err := c.makeRequest("POST", "/api/auth/anonymous", nil, "")
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status: %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	// Use provided expiration or default to 1 hour from now
	expiresAt := tokenResp.ExpiresAt
	if expiresAt == 0 {
		expiresAt = time.Now().Add(time.Hour).Unix()
	}

	config := AuthConfig{
		Method:    AuthMethodAnonymous,
		Token:     tokenResp.Token,
		ExpiresAt: expiresAt,
	}

	if err := c.saveAuthConfig(config); err != nil {
		return fmt.Errorf("failed to save auth config: %w", err)
	}

	fmt.Println("Successfully authenticated as anonymous user")
	return nil
}

func (c *MCPXClient) logout() error {
	if err := c.clearAuthConfig(); err != nil {
		return fmt.Errorf("failed to clear authentication: %w", err)
	}

	fmt.Println("Successfully logged out")
	return nil
}

func (c *MCPXClient) Health() error {
	fmt.Println("=== Health Check ===")

	resp, err := c.makeRequest("GET", "/v0/health", nil, "")
	if err != nil {
		return fmt.Errorf("health request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("Status Code: %d\n", resp.StatusCode)

	if resp.StatusCode == 200 {
		var healthResp HealthResponse
		if err := json.Unmarshal(body, &healthResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		fmt.Printf("Status: %s\n", healthResp.Status)
		if healthResp.GitHubClientID != "" {
			fmt.Printf("GitHub Client ID: %s\n", healthResp.GitHubClientID)
		}
	} else {
		fmt.Printf("Error: %s\n", string(body))
	}

	return nil
}

func (c *MCPXClient) ListServers(cursor string, limit int, jsonOutput bool, detailed bool) error {
	var params []string

	if !jsonOutput {
		fmt.Println("=== List Servers ===")
	}

	endpoint := "/v0/servers"

	if cursor != "" {
		params = append(params, "cursor="+cursor)
	}

	if limit > 0 {
		params = append(params, "limit="+strconv.Itoa(limit))
	}

	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}

	resp, err := c.makeRequest("GET", endpoint, nil, "")
	if err != nil {
		return fmt.Errorf("list servers request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if !jsonOutput {
		fmt.Printf("Status Code: %d\n", resp.StatusCode)
	}

	if resp.StatusCode == 200 {
		// First try to unmarshal and check what format we have
		var rawResponse map[string]interface{}
		if err := json.Unmarshal(body, &rawResponse); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		var servers []Server
		var metadata Metadata

		// Check if response has 'servers' array with wrapper format
		if serversArray, ok := rawResponse["servers"].([]interface{}); ok && len(serversArray) > 0 {
			if firstServer, ok := serversArray[0].(map[string]interface{}); ok {
				if _, hasServerField := firstServer["server"]; hasServerField {
					// New wrapper format
					var serversResp ServersResponse
					if err := json.Unmarshal(body, &serversResp); err == nil {
						for _, wrapper := range serversResp.Servers {
							server := wrapper.Server
							// Extract ID from registry metadata if not in server
							if server.ID == "" {
								if wrapper.RegistryMeta != nil {
									if id, ok := wrapper.RegistryMeta["id"].(string); ok {
										server.ID = id
									}
								}
							}
							servers = append(servers, server)
						}
						metadata = serversResp.Metadata
					}
				} else {
					// Legacy format
					var legacyResp LegacyServersResponse
					if err := json.Unmarshal(body, &legacyResp); err == nil {
						servers = legacyResp.Servers
						metadata = legacyResp.Metadata
					}
				}
			}
		} else {
			// Fallback: try legacy format
			var legacyResp LegacyServersResponse
			if err := json.Unmarshal(body, &legacyResp); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			servers = legacyResp.Servers
			metadata = legacyResp.Metadata
		}

		if detailed && jsonOutput {
			var detailedServers []ServerDetail
			for _, server := range servers {
				detailResp, err := c.makeRequest("GET", "/v0/servers/"+server.ID, nil, "")
				if err != nil {
					return fmt.Errorf("failed to get details for server %s: %w", server.ID, err)
				}
				detailBody, err := io.ReadAll(detailResp.Body)
				_ = detailResp.Body.Close()
				if err != nil {
					return fmt.Errorf("failed to read detail response for server %s: %w", server.ID, err)
				}
				if detailResp.StatusCode == 200 {
					var serverDetail ServerDetail
					// Try new wrapper format first
					var detailWrapper ServerDetailWrapper
					if err := json.Unmarshal(detailBody, &detailWrapper); err == nil && (detailWrapper.Server.ID != "" || detailWrapper.RegistryMeta != nil) {
						serverDetail = detailWrapper.Server
						// Extract ID from registry metadata if not in server
						if serverDetail.ID == "" && detailWrapper.RegistryMeta != nil {
							if id, ok := detailWrapper.RegistryMeta["id"].(string); ok {
								serverDetail.ID = id
							}
						}
					} else {
						// Try legacy format
						if err := json.Unmarshal(detailBody, &serverDetail); err != nil {
							return fmt.Errorf("failed to parse detail response for server %s: %w", server.ID, err)
						}
					}
					detailedServers = append(detailedServers, serverDetail)
				} else {
					serverDetail := ServerDetail{
						Server: server,
					}
					detailedServers = append(detailedServers, serverDetail)
				}
			}
			detailedResp := LegacyDetailedServersResponse{
				Servers:  detailedServers,
				Metadata: metadata,
			}
			prettyJSON, err := json.MarshalIndent(detailedResp, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to format JSON: %w", err)
			}
			fmt.Println(string(prettyJSON))
		} else if jsonOutput {
			// Convert back to legacy format for output
			legacyResp := LegacyServersResponse{
				Servers:  servers,
				Metadata: metadata,
			}
			prettyJSON, err := json.MarshalIndent(legacyResp, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to format JSON: %w", err)
			}
			fmt.Println(string(prettyJSON))
		} else {
			fmt.Printf("Total Servers: %d\n", len(servers))
			if metadata.NextCursor != "" {
				fmt.Printf("Next Cursor: %s\n", metadata.NextCursor)
			}
			for i, server := range servers {
				fmt.Printf("\n--- Server %d ---\n", i+1)
				fmt.Printf("ID: %s\n", server.ID)
				fmt.Printf("Name: %s\n", server.Name)
				fmt.Printf("Description: %s\n", server.Description)
				if server.Status != "" {
					fmt.Printf("Status: %s\n", server.Status)
				}
				fmt.Printf("Repository: %s (%s)\n", server.Repository.URL, server.Repository.Source)
				fmt.Printf("Version: %s\n", server.VersionDetail.Version)
				if server.VersionDetail.ReleaseDate != "" {
					fmt.Printf("Release Date: %s\n", server.VersionDetail.ReleaseDate)
				}
			}
		}
	} else {
		if jsonOutput {
			fmt.Println(string(body))
		} else {
			fmt.Printf("Error: %s\n", string(body))
		}
	}

	return nil
}

func (c *MCPXClient) GetServer(id string, jsonOutput bool) error {
	if !jsonOutput {
		fmt.Printf("=== Get Server Details (ID: %s) ===\n", id)
	}

	endpoint := "/v0/servers/" + id

	resp, err := c.makeRequest("GET", endpoint, nil, "")
	if err != nil {
		return fmt.Errorf("get server request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if !jsonOutput {
		fmt.Printf("Status Code: %d\n", resp.StatusCode)
	}

	if resp.StatusCode == 200 {
		var serverDetail ServerDetail

		// Try new wrapper format first
		var detailWrapper ServerDetailWrapper
		if err := json.Unmarshal(body, &detailWrapper); err == nil && (detailWrapper.Server.ID != "" || detailWrapper.RegistryMeta != nil) {
			serverDetail = detailWrapper.Server
			// Extract ID from registry metadata if not in server
			if serverDetail.ID == "" && detailWrapper.RegistryMeta != nil {
				if id, ok := detailWrapper.RegistryMeta["id"].(string); ok {
					serverDetail.ID = id
				}
			}
		} else {
			// Try legacy format
			if err := json.Unmarshal(body, &serverDetail); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
		}

		if jsonOutput {
			prettyJSON, err := json.MarshalIndent(serverDetail, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to format JSON: %w", err)
			}
			fmt.Println(string(prettyJSON))
		} else {
			fmt.Printf("ID: %s\n", serverDetail.ID)
			fmt.Printf("Name: %s\n", serverDetail.Name)
			fmt.Printf("Description: %s\n", serverDetail.Description)
			if serverDetail.Status != "" {
				fmt.Printf("Status: %s\n", serverDetail.Status)
			}
			fmt.Printf("Repository: %s (%s)\n", serverDetail.Repository.URL, serverDetail.Repository.Source)
			fmt.Printf("Version: %s\n", serverDetail.VersionDetail.Version)
			if serverDetail.VersionDetail.ReleaseDate != "" {
				fmt.Printf("Release Date: %s\n", serverDetail.VersionDetail.ReleaseDate)
			}
			if len(serverDetail.Packages) > 0 {
				fmt.Printf("\nPackages:\n")
				for i, pkg := range serverDetail.Packages {
					fmt.Printf("  Package %d:\n", i+1)
					fmt.Printf("    Registry: %s\n", pkg.RegistryName)
					fmt.Printf("    Name: %s\n", pkg.Name)
					fmt.Printf("    Version: %s\n", pkg.Version)
					if pkg.WheelURL != "" {
						fmt.Printf("    Wheel URL: %s\n", pkg.WheelURL)
					}
					if pkg.BinaryURL != "" {
						fmt.Printf("    Binary URL: %s\n", pkg.BinaryURL)
					}
					if pkg.RuntimeHint != "" {
						fmt.Printf("    Runtime Hint: %s\n", pkg.RuntimeHint)
					}
					if len(pkg.EnvironmentVariables) > 0 {
						fmt.Printf("    Environment Variables:\n")
						for _, env := range pkg.EnvironmentVariables {
							required := "optional"
							if env.IsRequired {
								required = "required"
							}
							fmt.Printf("      - %s: %s (%s)\n", env.Name, env.Description, required)
						}
					}
					if len(pkg.RuntimeArguments) > 0 {
						fmt.Printf("    Runtime Arguments:\n")
						for _, arg := range pkg.RuntimeArguments {
							required := "optional"
							if arg.IsRequired {
								required = "required"
							}
							nameInfo := arg.Type
							if arg.Name != "" {
								nameInfo = fmt.Sprintf("%s:%s", arg.Type, arg.Name)
							}
							fmt.Printf("      - %s (%s): %s\n", nameInfo, required, arg.Description)
						}
					}
				}
			}
			if len(serverDetail.Remotes) > 0 {
				fmt.Printf("\nRemotes:\n")
				for i, remote := range serverDetail.Remotes {
					fmt.Printf("  Remote %d:\n", i+1)
					fmt.Printf("    Transport: %s\n", remote.TransportType)
					fmt.Printf("    URL: %s\n", remote.URL)
				}
			}
		}
	} else {
		if jsonOutput {
			fmt.Println(string(body))
		} else {
			fmt.Printf("Error: %s\n", string(body))
		}
	}

	return nil
}

func (c *MCPXClient) PublishServer(serverFile string, token string) error {
	fmt.Printf("=== Publish Server (File: %s) ===\n", serverFile)

	data, err := os.ReadFile(serverFile)
	if err != nil {
		return fmt.Errorf("failed to read server file: %w", err)
	}

	// Try to parse as PublishRequest first (new format)
	var publishReq PublishRequest
	if err := json.Unmarshal(data, &publishReq); err == nil && publishReq.Server.Name != "" {
		// It's a PublishRequest format, check server name for GitHub namespace
		if strings.HasPrefix(publishReq.Server.Name, "io.github.") && token == "" {
			return fmt.Errorf("authentication token is required for GitHub namespaced servers (io.github.*)")
		}
	} else {
		// Try to parse as legacy ServerDetail format
		var serverDetail ServerDetail
		if err := json.Unmarshal(data, &serverDetail); err != nil {
			return fmt.Errorf("invalid JSON in server file: %w", err)
		}

		if strings.HasPrefix(serverDetail.Name, "io.github.") && token == "" {
			return fmt.Errorf("authentication token is required for GitHub namespaced servers (io.github.*)")
		}

		// Convert legacy format to PublishRequest format
		publishReq = PublishRequest{
			Server: serverDetail,
		}

		// Re-marshal as PublishRequest format
		data, err = json.Marshal(publishReq)
		if err != nil {
			return fmt.Errorf("failed to convert to publish format: %w", err)
		}
	}

	resp, err := c.makeRequest("POST", "/v0/publish", data, token)
	if err != nil {
		return fmt.Errorf("publish request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("Status Code: %d\n", resp.StatusCode)

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		// Try to parse as PublishResponse first
		var publishResp PublishResponse
		if err := json.Unmarshal(body, &publishResp); err == nil && publishResp.Message != "" {
			fmt.Printf("✅ Success: %s\n", publishResp.Message)
			fmt.Printf("Server ID: %s\n", publishResp.ID)
		} else {
			// Try new wrapper format
			var serverWrapper ServerDetailWrapper
			if err := json.Unmarshal(body, &serverWrapper); err == nil && serverWrapper.Server.ID != "" {
				fmt.Printf("✅ Server published successfully\n")
				fmt.Printf("Server ID: %s\n", serverWrapper.Server.ID)
			} else {
				// Try legacy Server response (200 case)
				var serverResp Server
				if err := json.Unmarshal(body, &serverResp); err == nil && serverResp.ID != "" {
					fmt.Printf("✅ Server published successfully\n")
					fmt.Printf("Server ID: %s\n", serverResp.ID)
				} else {
					// Fallback: just show the response
					fmt.Printf("✅ Success\n")
					fmt.Printf("Response: %s\n", string(body))
				}
			}
		}
	} else {
		fmt.Printf("❌ Error: %s\n", string(body))
	}

	return nil
}

func promptUser(prompt string, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" && defaultValue != "" {
		return defaultValue
	}

	return input
}

func promptChoice(prompt string, choices []string, defaultChoice string) string {
	fmt.Printf("%s\n", prompt)

	for i, choice := range choices {
		marker := " "
		if choice == defaultChoice {
			marker = "*"
		}
		fmt.Printf("  %s %d) %s\n", marker, i+1, choice)
	}

	for {
		input := promptUser("Enter choice (1-"+strconv.Itoa(len(choices))+")", "")
		if input == "" && defaultChoice != "" {
			return defaultChoice
		}
		choice, err := strconv.Atoi(input)
		if err == nil && choice >= 1 && choice <= len(choices) {
			return choices[choice-1]
		}
		fmt.Printf("Invalid choice. Please enter a number between 1 and %d.\n", len(choices))
	}
}

func createInteractiveServer() (*ServerDetail, error) {
	fmt.Println("=== Interactive Server Configuration ===")
	fmt.Println()

	runtime := promptChoice("Select server runtime:", []string{"node", "python-pypi", "python-wheel", "binary"}, "node")

	var data []byte
	switch runtime {
	case "node":
		data = exampleServerNPMJSON
	case "python-pypi":
		data = exampleServerPyPiJSON
	case "python-wheel":
		data = exampleServerWheelJSON
	case "binary":
		data = exampleServerBinaryJSON
	}

	var server ServerDetail
	if err := json.Unmarshal(data, &server); err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// Interactive prompts
	fmt.Println()
	server.Name = promptUser("Server name", server.Name)
	server.Description = promptUser("Server description", server.Description)

	fmt.Println("\n--- Repository Information ---")
	server.Repository.URL = promptUser("Repository URL", server.Repository.URL)
	server.Repository.ID = promptUser("Repository ID (e.g., username/repo)", server.Repository.ID)

	fmt.Println("\n--- Version Information ---")
	server.VersionDetail.Version = promptUser("Version", server.VersionDetail.Version)

	server.VersionDetail.ReleaseDate = time.Now().Format(time.RFC3339)

	if len(server.Packages) > 0 {
		fmt.Println("\n--- Package Information ---")
		for pkgIndex := range server.Packages {
			pkg := &server.Packages[pkgIndex]
			fmt.Printf("\nConfiguring package %d (%s):\n", pkgIndex+1, pkg.RegistryName)

			switch pkg.RegistryName {
			case "npm":
				pkg.Name = promptUser("NPM package name", pkg.Name)
			case "pypi":
				pkg.Name = promptUser("PyPI package name", pkg.Name)
				if pkg.WheelURL != "" {
					pkg.WheelURL = promptUser("Wheel URL", pkg.WheelURL)
				}
			case "wheel":
				pkg.Name = promptUser("Wheel package name", pkg.Name)
				if pkg.WheelURL != "" {
					pkg.WheelURL = promptUser("Wheel URL", pkg.WheelURL)
				}
			case "binary":
				pkg.Name = promptUser("Binary package name", pkg.Name)
				if pkg.BinaryURL != "" {
					pkg.BinaryURL = promptUser("Binary download URL", pkg.BinaryURL)
				}
			case "docker":
				pkg.Name = promptUser("Docker image name", pkg.Name)
			default:
				pkg.Name = promptUser("Package name", pkg.Name)
			}
			pkg.Version = promptUser("Package version", pkg.Version)

			if len(pkg.EnvironmentVariables) > 0 {
				fmt.Printf("\n--- Environment Variables (%s) ---\n", pkg.RegistryName)
				for i := range pkg.EnvironmentVariables {
					env := &pkg.EnvironmentVariables[i]
					fmt.Printf("\nConfiguring environment variable: %s\n", env.Name)
					env.Default = promptUser(fmt.Sprintf("%s default value", env.Name), env.Default)
					requiredChoice := "false"
					if env.IsRequired {
						requiredChoice = "true"
					}
					requiredStr := promptChoice(fmt.Sprintf("Is %s required?", env.Name), []string{"true", "false"}, requiredChoice)
					env.IsRequired = requiredStr == "true"
				}
			}
			if len(pkg.RuntimeArguments) > 0 {
				fmt.Printf("\n--- Runtime Arguments (%s) ---\n", pkg.RegistryName)
				for i := range pkg.RuntimeArguments {
					arg := &pkg.RuntimeArguments[i]
					argIdentifier := arg.Description
					if arg.Name != "" {
						argIdentifier = fmt.Sprintf("%s (%s)", arg.Description, arg.Name)
					}
					fmt.Printf("\nConfiguring runtime argument: %s\n", argIdentifier)
					if arg.Name != "" {
						arg.Name = promptUser("Argument name", arg.Name)
					}
					if arg.Default != "" {
						arg.Default = promptUser(fmt.Sprintf("%s default value", arg.Description), arg.Default)
					}
					requiredChoice := "false"
					if arg.IsRequired {
						requiredChoice = "true"
					}
					requiredStr := promptChoice("Is this argument required?", []string{"true", "false"}, requiredChoice)
					arg.IsRequired = requiredStr == "true"
				}
			}
		}
	}

	if len(server.Remotes) > 0 {
		fmt.Println("\n--- Remote Configuration ---")
		remote := &server.Remotes[0]
		remote.URL = promptUser("Server URL", remote.URL)
	}

	return &server, nil
}

func (c *MCPXClient) PublishServerInteractive(token string) error {
	fmt.Println("=== Interactive Publish Server ===")

	server, err := createInteractiveServer()
	if err != nil {
		return fmt.Errorf("failed to create server config: %w", err)
	}

	if strings.HasPrefix(server.Name, "io.github.") && token == "" {
		return fmt.Errorf("authentication token is required for GitHub namespaced servers (io.github.*)")
	}

	// Create PublishRequest wrapper
	publishReq := PublishRequest{
		Server: *server,
		XPublisher: map[string]interface{}{
			"tool":    "mcpx-cli",
			"version": version,
			"build_info": map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
			},
		},
	}

	data, err := json.MarshalIndent(publishReq, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal server config: %w", err)
	}

	saveConfig := promptChoice("Save configuration to file?", []string{"yes", "no"}, "yes")
	if saveConfig == "yes" {
		filename := promptUser("Filename", "server-config.json")
		if !strings.HasSuffix(filename, ".json") {
			filename += ".json"
		}
		if err := os.WriteFile(filename, data, 0644); err != nil {
			fmt.Printf("Warning: Failed to save config to %s: %v\n", filename, err)
		} else {
			fmt.Printf("Configuration saved to %s\n", filename)
		}
	}

	fmt.Println("\n=== Server Configuration Preview ===")
	fmt.Printf("Name: %s\n", server.Name)
	fmt.Printf("Description: %s\n", server.Description)
	fmt.Printf("Version: %s\n", server.VersionDetail.Version)
	fmt.Printf("Repository: %s\n", server.Repository.URL)

	publish := promptChoice("Proceed with publishing?", []string{"yes", "no"}, "no")
	if publish != "yes" {
		fmt.Println("Publishing cancelled.")
		return nil
	}

	resp, err := c.makeRequest("POST", "/v0/publish", data, token)
	if err != nil {
		return fmt.Errorf("publish request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("Status Code: %d\n", resp.StatusCode)

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		// Try to parse as PublishResponse first
		var publishResp PublishResponse
		if err := json.Unmarshal(body, &publishResp); err == nil && publishResp.Message != "" {
			fmt.Printf("✅ Success: %s\n", publishResp.Message)
			fmt.Printf("Server ID: %s\n", publishResp.ID)
		} else {
			// If not a PublishResponse, it might be a Server response (200 case)
			var serverResp Server
			if err := json.Unmarshal(body, &serverResp); err == nil && serverResp.ID != "" {
				fmt.Printf("✅ Server published successfully\n")
				fmt.Printf("Server ID: %s\n", serverResp.ID)
			} else {
				// Fallback: just show the response
				fmt.Printf("✅ Success\n")
				fmt.Printf("Response: %s\n", string(body))
			}
		}
	} else {
		fmt.Printf("❌ Error: %s\n", string(body))
	}

	return nil
}

func (c *MCPXClient) UpdateServer(serverID, serverFile, token string, jsonOutput bool) error {
	if !jsonOutput {
		fmt.Printf("=== Update Server %s ===\n", serverID)
	}

	data, err := os.ReadFile(serverFile)
	if err != nil {
		return fmt.Errorf("failed to read server file: %w", err)
	}

	// Try to detect if this is a PublishRequest format and unwrap it
	var rawData map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return fmt.Errorf("invalid JSON in server file: %w", err)
	}

	var serverDetail ServerDetail

	// Check if this is a PublishRequest format with "server" wrapper
	if serverData, hasServerWrapper := rawData["server"]; hasServerWrapper {
		// Unwrap the server object from PublishRequest format
		serverBytes, err := json.Marshal(serverData)
		if err != nil {
			return fmt.Errorf("failed to marshal server data: %w", err)
		}
		if err := json.Unmarshal(serverBytes, &serverDetail); err != nil {
			return fmt.Errorf("invalid server data in PublishRequest: %w", err)
		}
		// Use the unwrapped server data
		data = serverBytes
	} else {
		// Direct ServerDetail format
		if err := json.Unmarshal(data, &serverDetail); err != nil {
			return fmt.Errorf("invalid JSON in server file: %w", err)
		}
	}

	if strings.HasPrefix(serverDetail.Name, "io.github.") && token == "" {
		return fmt.Errorf("authentication token is required for GitHub namespaced servers (io.github.*)")
	}

	endpoint := "/v0/servers/" + serverID

	resp, err := c.makeRequest("PUT", endpoint, data, token)
	if err != nil {
		return fmt.Errorf("update server request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if !jsonOutput {
		fmt.Printf("Status Code: %d\n", resp.StatusCode)
	}

	if resp.StatusCode == 200 {
		if jsonOutput {
			fmt.Println(string(body))
		} else {
			var updateResp map[string]string
			if err := json.Unmarshal(body, &updateResp); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			fmt.Printf("✅ %s\n", updateResp["message"])
			fmt.Printf("Server ID: %s\n", updateResp["id"])
		}
	} else {
		if jsonOutput {
			fmt.Println(string(body))
		} else {
			fmt.Printf("❌ Update failed: %s\n", string(body))
		}
	}

	return nil
}

func (c *MCPXClient) DeleteServer(serverID, token string, jsonOutput bool) error {
	if !jsonOutput {
		fmt.Printf("=== Delete Server %s ===\n", serverID)
	}

	endpoint := "/v0/servers/" + serverID

	response, err := c.makeRequest("DELETE", endpoint, nil, token)
	if err != nil {
		return fmt.Errorf("delete server request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if response.StatusCode == http.StatusNotFound {
		return fmt.Errorf("server not found: %s", serverID)
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete server failed with status %d: %s", response.StatusCode, string(body))
	}

	if jsonOutput {
		fmt.Printf("{\"message\": \"Server %s deleted successfully\"}\n", serverID)
	} else {
		fmt.Printf("✅ Server '%s' deleted successfully\n", serverID)
	}

	return nil
}

func printUsage() {
	fmt.Println("mcpx-cli - A command-line client for the mcpx registry api")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  mcpx-cli [global flags] <command> [command flags]")
	fmt.Println()
	fmt.Println("Global Flags:")
	fmt.Println("  --base-url=string    Base url of the mcpx api (default: http://localhost:8080)")
	fmt.Println("  --version            Show version information")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  help                                Show this help message")
	fmt.Println("  version                             Show version information")
	fmt.Println("  login [--method]                    Login with specified method (anonymous, github-oauth, github-oidc)")
	fmt.Println("  logout                              Logout and clear stored credentials")
	fmt.Println("  health                              Check api health status")
	fmt.Println("  servers                             List all servers")
	fmt.Println("  server <id> [--json]                Get server details by ID")
	fmt.Println("  update <id> <server.json> [--token] [--json]  Update a server by ID")
	fmt.Println("  delete <id> [--token] [--json]      Delete a server by ID (token optional)")
	fmt.Println("  publish <server.json>               Publish a server to the registry")
	fmt.Println("  publish --interactive               Interactive mode to create and publish a server (supports npm, PyPI, wheel, binary)")
	fmt.Println()
	fmt.Println("Authentication Flags:")
	fmt.Println("  --method string      Authentication method (anonymous, github-oauth, github-oidc) (default: anonymous)")
	fmt.Println()
	fmt.Println("Server List Flags:")
	fmt.Println("  --cursor string      Pagination cursor")
	fmt.Println("  --limit int          Maximum number of servers to return (default: 30)")
	fmt.Println("  --json               Output servers details in JSON format")
	fmt.Println("  --detailed           Include packages and remotes in JSON output (requires --json)")
	fmt.Println()
	fmt.Println("Server Detail Flags:")
	fmt.Println("  --json               Output server details in JSON format")
	fmt.Println()
	fmt.Println("Update Flags:")
	fmt.Println("  --token string       Authentication token (required for io.github.* servers)")
	fmt.Println("  --json               Output result in JSON format")
	fmt.Println()
	fmt.Println("Publish Flags:")
	fmt.Println("  --token string       Authentication token (required for io.github.* servers)")
	fmt.Println("  --interactive        Interactive mode to create server configuration")
	fmt.Println()
	fmt.Println("Delete Flags:")
	fmt.Println("  --token string       Authentication token (optional)")
	fmt.Println("  --json               Output result in JSON format")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mcpx-cli login --method anonymous                           # Login with anonymous authentication")
	fmt.Println("  mcpx-cli login --method github-oauth                       # Login with GitHub OAuth")
	fmt.Println("  mcpx-cli logout                                             # Logout and clear credentials")
	fmt.Println("  mcpx-cli health")
	fmt.Println("  mcpx-cli servers --limit 10")
	fmt.Println("  mcpx-cli servers --json --detailed")
	fmt.Println("  mcpx-cli server <id> [--json]")
	fmt.Println("  mcpx-cli update <id> server.json --token your_token         # With authentication")
	fmt.Println("  mcpx-cli update <id> server.json                            # Without authentication")
	fmt.Println("  mcpx-cli update <id> server.json --json                     # JSON output")
	fmt.Println("  mcpx-cli delete <id> --token your_token                     # With authentication")
	fmt.Println("  mcpx-cli delete <id>                                        # Without authentication")
	fmt.Println("  mcpx-cli delete <id> --json                                 # JSON output")
	fmt.Println("  mcpx-cli publish server.json --token your_github_token      # GitHub projects")
	fmt.Println("  mcpx-cli publish server.json                                # Non-GitHub projects")
	fmt.Println("  mcpx-cli publish --interactive --token your_github_token    # GitHub projects")
	fmt.Println("  mcpx-cli publish --interactive                              # Non-GitHub projects")
	fmt.Println("  mcpx-cli --base-url=http://localhost:8080 servers")
}

func main() {
	if len(os.Args) >= 2 {
		arg := os.Args[1]
		if arg == "--help" || arg == "-h" || arg == "help" {
			printUsage()
			os.Exit(0)
		}
		if arg == "--version" || arg == "version" {
			fmt.Println(version)
			os.Exit(0)
		}
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var baseURL string
	var globalFlags = flag.NewFlagSet("global", flag.ContinueOnError)
	globalFlags.StringVar(&baseURL, "base-url", defaultBaseURL, "Base url of the mcpx api")

	args := os.Args[1:]
	for i, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			if err := globalFlags.Parse(args[:i]); err != nil {
				fmt.Printf("Error parsing global flags: %v\n", err)
				os.Exit(1)
			}
			args = args[i:]
			break
		}
	}

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	client := NewMCPXClient(baseURL)
	command := args[0]

	switch command {
	case "help", "--help", "-h":
		printUsage()
	case "login":
		var authMethod string
		loginFlags := flag.NewFlagSet("login", flag.ExitOnError)
		loginFlags.StringVar(&authMethod, "method", AuthMethodAnonymous, "Authentication method (anonymous, github-oauth, github-oidc)")
		if err := loginFlags.Parse(args[1:]); err != nil {
			log.Fatalf("Error parsing login flags: %v", err)
		}
		if err := client.login(authMethod); err != nil {
			log.Fatalf("Login failed: %v", err)
		}
	case "logout":
		if err := client.logout(); err != nil {
			log.Fatalf("Logout failed: %v", err)
		}
	case "health":
		if err := client.Health(); err != nil {
			log.Fatalf("Health check failed: %v", err)
		}
	case "servers":
		var cursor string
		var limit int
		var jsonOutput bool
		var detailed bool
		serversFlags := flag.NewFlagSet("servers", flag.ExitOnError)
		serversFlags.StringVar(&cursor, "cursor", "", "Pagination cursor")
		serversFlags.IntVar(&limit, "limit", 30, "Maximum number of servers to return")
		serversFlags.BoolVar(&jsonOutput, "json", false, "Output servers details in JSON format")
		serversFlags.BoolVar(&detailed, "detailed", false, "Include packages and remotes in JSON output (requires --json)")
		if err := serversFlags.Parse(args[1:]); err != nil {
			log.Fatalf("Error parsing servers flags: %v", err)
		}
		if detailed && !jsonOutput {
			fmt.Println("Error: --detailed flag requires --json flag")
			os.Exit(1)
		}
		if err := client.ListServers(cursor, limit, jsonOutput, detailed); err != nil {
			log.Fatalf("List servers failed: %v", err)
		}
	case "server":
		var jsonOutput bool
		serverFlags := flag.NewFlagSet("server", flag.ExitOnError)
		serverFlags.BoolVar(&jsonOutput, "json", false, "Output server details in JSON format")
		var serverID string
		var flagArgs []string
		for i, arg := range args[1:] {
			if strings.HasPrefix(arg, "-") {
				flagArgs = args[i+1:]
				break
			} else {
				serverID = arg
			}
		}
		if serverID == "" {
			fmt.Println("Error: server ID is required")
			fmt.Println("Usage: mcpx-cli server <id> [--json]")
			os.Exit(1)
		}
		if err := serverFlags.Parse(flagArgs); err != nil {
			log.Fatalf("Error parsing server flags: %v", err)
		}
		if err := client.GetServer(serverID, jsonOutput); err != nil {
			log.Fatalf("Get server failed: %v", err)
		}
	case "update":
		var token string
		var jsonOutput bool
		updateFlags := flag.NewFlagSet("update", flag.ExitOnError)
		updateFlags.StringVar(&token, "token", "", "Authentication token (required for io.github.* servers)")
		updateFlags.BoolVar(&jsonOutput, "json", false, "Output result in JSON format")
		var serverID string
		var serverFile string
		var flagArgs []string

		// Parse serverID and serverFile from positional arguments
		argIndex := 0
		for i, arg := range args[1:] {
			if strings.HasPrefix(arg, "-") {
				flagArgs = args[i+1:]
				break
			} else {
				switch argIndex {
				case 0:
					serverID = arg
				case 1:
					serverFile = arg
				}
				argIndex++
			}
		}

		if serverID == "" {
			fmt.Println("Error: server ID is required")
			fmt.Println("Usage: mcpx-cli update <id> <server.json> [--token <token>] [--json]")
			os.Exit(1)
		}
		if serverFile == "" {
			fmt.Println("Error: server file is required")
			fmt.Println("Usage: mcpx-cli update <id> <server.json> [--token <token>] [--json]")
			os.Exit(1)
		}
		if err := updateFlags.Parse(flagArgs); err != nil {
			log.Fatalf("Error parsing update flags: %v", err)
		}
		if err := client.UpdateServer(serverID, serverFile, token, jsonOutput); err != nil {
			log.Fatalf("Update server failed: %v", err)
		}
	case "publish":
		var token string
		var interactive bool
		publishFlags := flag.NewFlagSet("publish", flag.ExitOnError)
		publishFlags.StringVar(&token, "token", "", "Authentication token (optional)")
		publishFlags.BoolVar(&interactive, "interactive", false, "Interactive mode to create server configuration")
		flagArgs := args[1:]
		var serverFile string
		// If interactive flag is provided or no server file is given, use interactive mode
		if len(args) == 1 || (len(args) > 1 && strings.HasPrefix(args[1], "-")) {
			if err := publishFlags.Parse(flagArgs); err != nil {
				log.Fatalf("Error parsing publish flags: %v", err)
			}
			interactive = true
		} else {
			serverFile = args[1]
			if err := publishFlags.Parse(args[2:]); err != nil {
				log.Fatalf("Error parsing publish flags: %v", err)
			}
		}
		if interactive {
			if err := client.PublishServerInteractive(token); err != nil {
				log.Fatalf("Interactive publish failed: %v", err)
			}
		} else {
			if serverFile == "" {
				fmt.Println("Error: server file is required in non-interactive mode")
				fmt.Println("Usage: mcpx-cli publish <server.json> [--token <token>]")
				fmt.Println("   or: mcpx-cli publish --interactive [--token <token>]")
				fmt.Println("Note: --token is required only for GitHub namespaced servers (io.github.*)")
				os.Exit(1)
			}
			if err := client.PublishServer(serverFile, token); err != nil {
				log.Fatalf("Publish server failed: %v", err)
			}
		}
	case "delete":
		var token string
		var jsonOutput bool
		deleteFlags := flag.NewFlagSet("delete", flag.ExitOnError)
		deleteFlags.StringVar(&token, "token", "", "Authentication token (optional)")
		deleteFlags.BoolVar(&jsonOutput, "json", false, "Output result in JSON format")
		var serverID string
		var flagArgs []string
		for i, arg := range args[1:] {
			if strings.HasPrefix(arg, "-") {
				flagArgs = args[i+1:]
				break
			} else {
				serverID = arg
			}
		}
		if serverID == "" {
			fmt.Println("Error: server ID is required")
			fmt.Println("Usage: mcpx-cli delete <id> [--token <token>] [--json]")
			os.Exit(1)
		}
		if err := deleteFlags.Parse(flagArgs); err != nil {
			log.Fatalf("Error parsing delete flags: %v", err)
		}
		if err := client.DeleteServer(serverID, token, jsonOutput); err != nil {
			log.Fatalf("Delete server failed: %v", err)
		}
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}
