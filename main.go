package main

import (
	"bufio"
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
	"time"
)

const (
	defaultBaseURL = "http://localhost:8080"
)

var version = "dev"

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

type DetailedServersResponse struct {
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
		var serversResp ServersResponse
		if err := json.Unmarshal(body, &serversResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
		if detailed && jsonOutput {
			var detailedServers []ServerDetail
			for _, server := range serversResp.Servers {
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
					if err := json.Unmarshal(detailBody, &serverDetail); err != nil {
						return fmt.Errorf("failed to parse detail response for server %s: %w", server.ID, err)
					}
					detailedServers = append(detailedServers, serverDetail)
				} else {
					serverDetail := ServerDetail{
						Server: server,
					}
					detailedServers = append(detailedServers, serverDetail)
				}
			}
			detailedResp := DetailedServersResponse{
				Servers:  detailedServers,
				Metadata: serversResp.Metadata,
			}
			prettyJSON, err := json.MarshalIndent(detailedResp, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to format JSON: %w", err)
			}
			fmt.Println(string(prettyJSON))
		} else if jsonOutput {
			prettyJSON, err := json.MarshalIndent(serversResp, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to format JSON: %w", err)
			}
			fmt.Println(string(prettyJSON))
		} else {
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
		if jsonOutput {
			var serverDetail ServerDetail
			if err := json.Unmarshal(body, &serverDetail); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			prettyJSON, err := json.MarshalIndent(serverDetail, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to format JSON: %w", err)
			}
			fmt.Println(string(prettyJSON))
		} else {
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

	var serverDetail ServerDetail
	if err := json.Unmarshal(data, &serverDetail); err != nil {
		return fmt.Errorf("invalid JSON in server file: %w", err)
	}

	if strings.HasPrefix(serverDetail.Name, "io.github.") && token == "" {
		return fmt.Errorf("authentication token is required for GitHub namespaced servers (io.github.*)")
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

	runtime := promptChoice("Select server runtime:", []string{"node", "python"}, "node")

	var templateFile string
	if runtime == "node" {
		templateFile = "example-server-node.json"
	} else {
		templateFile = "example-server-python.json"
	}

	data, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load template %s: %w", templateFile, err)
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
		pkg := &server.Packages[0]
		if runtime == "node" {
			pkg.Name = promptUser("NPM package name", pkg.Name)
		} else {
			pkg.Name = promptUser("PyPI package name", pkg.Name)
		}
		pkg.Version = promptUser("Package version", server.VersionDetail.Version)
		if len(pkg.EnvironmentVariables) > 0 {
			fmt.Println("\n--- Environment Variables ---")
			for i := range pkg.EnvironmentVariables {
				env := &pkg.EnvironmentVariables[i]
				env.Default = promptUser(fmt.Sprintf("%s default value", env.Name), env.Default)
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

	data, err := json.MarshalIndent(server, "", "  ")
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
	fmt.Println("  --base-url=string    Base url of the mcpx api (default: http://localhost:8080)")
	fmt.Println("  --version            Show version information")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  help                       Show this help message")
	fmt.Println("  version                    Show version information")
	fmt.Println("  health                     Check api health status")
	fmt.Println("  servers                    List all servers")
	fmt.Println("  server <id> [--json]       Get server details by ID")
	fmt.Println("  publish <server.json>      Publish a server to the registry")
	fmt.Println("  publish --interactive      Interactive mode to create and publish a server")
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
	fmt.Println("Publish Flags:")
	fmt.Println("  --token string       Authentication token (required for io.github.* servers)")
	fmt.Println("  --interactive        Interactive mode to create server configuration")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  mcpx-cli health")
	fmt.Println("  mcpx-cli servers --limit 10")
	fmt.Println("  mcpx-cli servers --json --detailed")
	fmt.Println("  mcpx-cli server <id> [--json]")
	fmt.Println("  mcpx-cli publish server.json --token your_github_token    # GitHub projects")
	fmt.Println("  mcpx-cli publish server.json                              # Non-GitHub projects")
	fmt.Println("  mcpx-cli publish --interactive --token your_github_token  # GitHub projects")
	fmt.Println("  mcpx-cli publish --interactive                            # Non-GitHub projects")
	fmt.Println("  mcpx-cli --base-url=http://prod.example.com servers")
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
	case "publish":
		var token string
		var interactive bool
		publishFlags := flag.NewFlagSet("publish", flag.ExitOnError)
		publishFlags.StringVar(&token, "token", "", "Authentication token (required)")
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
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}
