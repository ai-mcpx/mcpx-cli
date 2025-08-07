package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	defaultBaseURL = "http://localhost:8080"
)

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
	Servers  []Server `json:"servers"`
	Metadata Metadata `json:"metadata,omitempty"`
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
	InputWithVariables `json:",inline"`
	Name               string `json:"name"`
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
	RuntimeHint          string          `json:"runtime_hint,omitempty"`
	RuntimeArguments     []Argument      `json:"runtime_arguments,omitempty"`
	PackageArguments     []Argument      `json:"package_arguments,omitempty"`
	EnvironmentVariables []KeyValueInput `json:"environment_variables,omitempty"`
}

type Remote struct {
	TransportType string  `json:"transport_type"`
	URL           string  `json:"url"`
	Headers       []Input `json:"headers,omitempty"`
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
		httpClient: &http.Client{},
	}
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

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.httpClient.Do(req)
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

func (c *MCPXClient) ListServers(cursor string, limit int) error {
	var params []string

	fmt.Println("=== List Servers ===")

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

	fmt.Printf("Status Code: %d\n", resp.StatusCode)

	if resp.StatusCode == 200 {
		var serversResp ServersResponse
		if err := json.Unmarshal(body, &serversResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("Total Servers: %d\n", len(serversResp.Servers))
		if serversResp.Metadata.NextCursor != "" {
			fmt.Printf("Next Cursor: %s\n", serversResp.Metadata.NextCursor)
		}

		for i, server := range serversResp.Servers {
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
	} else {
		fmt.Printf("Error: %s\n", string(body))
	}

	return nil
}

func (c *MCPXClient) GetServer(id string) error {
	fmt.Printf("=== Get Server Details (ID: %s) ===\n", id)

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

	fmt.Printf("Status Code: %d\n", resp.StatusCode)

	if resp.StatusCode == 200 {
		var serverDetail ServerDetail
		if err := json.Unmarshal(body, &serverDetail); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

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
				if pkg.RuntimeHint != "" {
					fmt.Printf("    Runtime Hint: %s\n", pkg.RuntimeHint)
				}

				if len(pkg.EnvironmentVariables) > 0 {
					fmt.Printf("    Environment Variables:\n")
					for _, env := range pkg.EnvironmentVariables {
						fmt.Printf("      - %s: %s\n", env.Name, env.Description)
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
	} else {
		fmt.Printf("Error: %s\n", string(body))
	}

	return nil
}

func (c *MCPXClient) PublishServer(serverFile string, token string) error {
	fmt.Printf("=== Publish Server (File: %s) ===\n", serverFile)

	if token == "" {
		return fmt.Errorf("authentication token is required for publishing")
	}

	data, err := os.ReadFile(serverFile)
	if err != nil {
		return fmt.Errorf("failed to read server file: %w", err)
	}

	var serverDetail ServerDetail
	if err := json.Unmarshal(data, &serverDetail); err != nil {
		return fmt.Errorf("invalid JSON in server file: %w", err)
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

	if resp.StatusCode == 201 {
		var publishResp PublishResponse
		if err := json.Unmarshal(body, &publishResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("Success: %s\n", publishResp.Message)
		fmt.Printf("Server ID: %s\n", publishResp.ID)
	} else {
		fmt.Printf("Error: %s\n", string(body))
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
	fmt.Println("  --base-url string    Base url of the mcpx api (default: http://localhost:8080)")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  help                       Show this help message")
	fmt.Println("  health                     Check api health status")
	fmt.Println("  servers                    List all servers")
	fmt.Println("  server <id>                Get server details by ID")
	fmt.Println("  publish <server.json>      Publish a server to the registry")
	fmt.Println()
	fmt.Println("Server List Flags:")
	fmt.Println("  --cursor string      Pagination cursor")
	fmt.Println("  --limit int          Maximum number of servers to return (default: 30)")
	fmt.Println()
	fmt.Println("Publish Flags:")
	fmt.Println("  --token string       Authentication token (required)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mcpx-cli health")
	fmt.Println("  mcpx-cli servers --limit 10")
	fmt.Println("  mcpx-cli server a5e8a7f0-d4e4-4a1d-b12f-2896a23fd4f1")
	fmt.Println("  mcpx-cli publish server.json --token your_github_token")
	fmt.Println("  mcpx-cli --base-url http://prod.example.com servers")
}

func main() {
	if len(os.Args) >= 2 {
		arg := os.Args[1]
		if arg == "--help" || arg == "-h" || arg == "help" {
			printUsage()
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

	case "health":
		if err := client.Health(); err != nil {
			log.Fatalf("Health check failed: %v", err)
		}

	case "servers":
		var cursor string
		var limit int

		serversFlags := flag.NewFlagSet("servers", flag.ExitOnError)
		serversFlags.StringVar(&cursor, "cursor", "", "Pagination cursor")
		serversFlags.IntVar(&limit, "limit", 30, "Maximum number of servers to return")

		if err := serversFlags.Parse(args[1:]); err != nil {
			log.Fatalf("Error parsing servers flags: %v", err)
		}

		if err := client.ListServers(cursor, limit); err != nil {
			log.Fatalf("List servers failed: %v", err)
		}

	case "server":
		if len(args) < 2 {
			fmt.Println("Error: server ID is required")
			fmt.Println("Usage: mcpx-cli server <id>")
			os.Exit(1)
		}

		serverID := args[1]
		if err := client.GetServer(serverID); err != nil {
			log.Fatalf("Get server failed: %v", err)
		}

	case "publish":
		if len(args) < 2 {
			fmt.Println("Error: server file is required")
			fmt.Println("Usage: mcpx-cli publish <server.json> --token <token>")
			os.Exit(1)
		}

		var token string
		publishFlags := flag.NewFlagSet("publish", flag.ExitOnError)
		publishFlags.StringVar(&token, "token", "", "Authentication token (required)")

		if err := publishFlags.Parse(args[2:]); err != nil {
			log.Fatalf("Error parsing publish flags: %v", err)
		}

		serverFile := args[1]
		if err := client.PublishServer(serverFile, token); err != nil {
			log.Fatalf("Publish server failed: %v", err)
		}

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}
