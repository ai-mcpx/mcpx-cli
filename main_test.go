package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Mock HTTP server for testing
func createMockServer() *httptest.Server {
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/v0/health", func(w http.ResponseWriter, r *http.Request) {
		response := HealthResponse{
			Status:         "ok",
			GitHubClientID: "test-client-id",
		}
		_ = json.NewEncoder(w).Encode(response)
	})

	// Auth endpoints
	mux.HandleFunc("/v0/auth/none", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		response := TokenResponse{
			RegistryToken: "test-anonymous-token",
			ExpiresAt:     time.Now().Add(time.Hour).Unix(),
		}
		_ = json.NewEncoder(w).Encode(response)
	})

	// Servers list endpoint
	mux.HandleFunc("/v0/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// Mock server list
			servers := []ServerWrapper{
				{
					Server: Server{
						ID:          "test-server-1",
						Name:        "io.test/server1",
						Description: "Test server 1",
						Status:      "active",
						Repository: Repository{
							URL:    "https://github.com/test/server1",
							Source: "github",
							ID:     "test/server1",
						},
						VersionDetail: VersionDetail{
							Version:     "1.0.0",
							ReleaseDate: "2023-01-01T00:00:00Z",
							IsLatest:    true,
						},
					},
				},
				{
					Server: Server{
						ID:          "test-server-2",
						Name:        "io.test/server2",
						Description: "Test server 2",
						Status:      "active",
						Repository: Repository{
							URL:    "https://github.com/test/server2",
							Source: "github",
							ID:     "test/server2",
						},
						VersionDetail: VersionDetail{
							Version:     "2.0.0",
							ReleaseDate: "2023-02-01T00:00:00Z",
							IsLatest:    true,
						},
					},
				},
			}
			response := ServersResponse{
				Servers: servers,
				Metadata: Metadata{
					Count: 2,
					Total: 2,
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	})

	// Individual server endpoint
	mux.HandleFunc("/v0/servers/", func(w http.ResponseWriter, r *http.Request) {
		serverID := strings.TrimPrefix(r.URL.Path, "/v0/servers/")

		switch r.Method {
		case "GET":
			// Mock server detail
			server := ServerDetailWrapper{
				Server: ServerDetail{
					Server: Server{
						ID:          serverID,
						Name:        "io.test/server1",
						Description: "Test server 1 detailed",
						Status:      "active",
						Repository: Repository{
							URL:    "https://github.com/test/server1",
							Source: "github",
							ID:     "test/server1",
						},
						VersionDetail: VersionDetail{
							Version:     "1.0.0",
							ReleaseDate: "2023-01-01T00:00:00Z",
							IsLatest:    true,
						},
					},
					Packages: []Package{
						{
							Name:         "@test/server1",
							Version:      "1.0.0",
							RegistryName: "npm",
						},
					},
					Remotes: []Remote{
						{
							TransportType: "stdio",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(server)
		case "PUT":
			// Mock server update
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `{"message": "Server %s updated successfully"}`, serverID)
		case "DELETE":
			// Mock server deletion
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `{"message": "Server %s deleted successfully"}`, serverID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Publish endpoint
	mux.HandleFunc("/v0/publish", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, `{"message": "Server published successfully", "id": "new-server-id"}`)
	})

	return httptest.NewServer(mux)
}

// Test helper to create a temporary config file
func createTempConfig(t *testing.T, config AuthConfig) string {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, configFileName)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	err = os.WriteFile(configPath, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Set HOME to temp directory for test
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
	})

	return configPath
}

// Test helper to create a temporary server JSON file
func createTempServerFile(t *testing.T, content []byte) string {
	tmpFile, err := os.CreateTemp("", "server-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func(tmpFile *os.File) {
		_ = tmpFile.Close()
	}(tmpFile)

	_, err = tmpFile.Write(content)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	return tmpFile.Name()
}

func TestNewMCPXClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{
			name:    "default URL",
			baseURL: "",
			want:    defaultBaseURL,
		},
		{
			name:    "custom URL",
			baseURL: "https://custom.example.com",
			want:    "https://custom.example.com",
		},
		{
			name:    "URL with trailing slash",
			baseURL: "https://example.com/",
			want:    "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewMCPXClient(tt.baseURL)
			if client.baseURL != tt.want {
				t.Errorf("NewMCPXClient() baseURL = %v, want %v", client.baseURL, tt.want)
			}
		})
	}
}

func TestAuthConfig(t *testing.T) {
	mockServer := createMockServer()
	defer mockServer.Close()

	client := NewMCPXClient(mockServer.URL)

	t.Run("save and load auth config", func(t *testing.T) {
		config := AuthConfig{
			Method:    AuthMethodAnonymous,
			Token:     "test-token",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		// Create temp config
		createTempConfig(t, config)

		// Test loading config
		loadedConfig, err := client.loadAuthConfig()
		if err != nil {
			t.Fatalf("Failed to load auth config: %v", err)
		}

		if loadedConfig.Method != config.Method {
			t.Errorf("Method = %v, want %v", loadedConfig.Method, config.Method)
		}
		if loadedConfig.Token != config.Token {
			t.Errorf("Token = %v, want %v", loadedConfig.Token, config.Token)
		}
	})

	t.Run("expired token cleanup", func(t *testing.T) {
		expiredConfig := AuthConfig{
			Method:    AuthMethodAnonymous,
			Token:     "expired-token",
			ExpiresAt: time.Now().Add(-time.Hour).Unix(), // Expired
		}

		createTempConfig(t, expiredConfig)

		// Should return empty config for expired token
		loadedConfig, err := client.loadAuthConfig()
		if err != nil {
			t.Fatalf("Failed to load auth config: %v", err)
		}

		if loadedConfig.Token != "" {
			t.Errorf("Expected empty token for expired config, got %v", loadedConfig.Token)
		}
	})
}

func TestHealth(t *testing.T) {
	mockServer := createMockServer()
	defer mockServer.Close()

	client := NewMCPXClient(mockServer.URL)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := client.Health()
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	_ = w.Close()
	os.Stdout = oldStdout

	out, _ := io.ReadAll(r)
	output := string(out)

	if !strings.Contains(output, "Health Check") {
		t.Errorf("Expected output to contain 'Health Check', got %v", output)
	}
	if !strings.Contains(output, "200") {
		t.Errorf("Expected output to contain status code 200, got %v", output)
	}
}

func TestListServers(t *testing.T) {
	mockServer := createMockServer()
	defer mockServer.Close()

	client := NewMCPXClient(mockServer.URL)

	tests := []struct {
		name     string
		cursor   string
		limit    int
		json     bool
		detailed bool
		wantErr  bool
	}{
		{
			name:     "basic list",
			cursor:   "",
			limit:    10,
			json:     false,
			detailed: false,
			wantErr:  false,
		},
		{
			name:     "json output",
			cursor:   "",
			limit:    10,
			json:     true,
			detailed: false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := client.ListServers(tt.cursor, tt.limit, tt.json, tt.detailed)

			_ = w.Close()
			os.Stdout = oldStdout

			if (err != nil) != tt.wantErr {
				t.Errorf("ListServers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			out, _ := io.ReadAll(r)
			output := string(out)

			if tt.json {
				// Should contain JSON output
				if !strings.Contains(output, "{") {
					t.Errorf("Expected JSON output, got %v", output)
				}
			} else {
				// Should contain formatted output
				if !strings.Contains(output, "List Servers") {
					t.Errorf("Expected formatted output to contain 'List Servers', got %v", output)
				}
			}
		})
	}
}

func TestGetServer(t *testing.T) {
	mockServer := createMockServer()
	defer mockServer.Close()

	client := NewMCPXClient(mockServer.URL)

	tests := []struct {
		name     string
		serverID string
		json     bool
		wantErr  bool
	}{
		{
			name:     "get server detail",
			serverID: "test-server-1",
			json:     false,
			wantErr:  false,
		},
		{
			name:     "get server json",
			serverID: "test-server-1",
			json:     true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := client.GetServer(tt.serverID, tt.json)

			_ = w.Close()
			os.Stdout = oldStdout

			if (err != nil) != tt.wantErr {
				t.Errorf("GetServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			out, _ := io.ReadAll(r)
			output := string(out)

			if tt.json {
				// Should contain JSON output
				if !strings.Contains(output, "{") {
					t.Errorf("Expected JSON output, got %v", output)
				}
			} else {
				// Should contain formatted output
				if !strings.Contains(output, "Server Details") {
					t.Errorf("Expected formatted output to contain 'Server Details', got %v", output)
				}
			}
		})
	}
}

func TestPublishServer(t *testing.T) {
	mockServer := createMockServer()
	defer mockServer.Close()

	client := NewMCPXClient(mockServer.URL)

	// Create temp server file
	serverFile := createTempServerFile(t, exampleServerNPMJSON)
	defer func(name string) {
		_ = os.Remove(name)
	}(serverFile)

	tests := []struct {
		name       string
		serverFile string
		token      string
		wantErr    bool
	}{
		{
			name:       "publish with token",
			serverFile: serverFile,
			token:      "test-token",
			wantErr:    false,
		},
		{
			name:       "publish without token",
			serverFile: serverFile,
			token:      "",
			wantErr:    false,
		},
		{
			name:       "publish non-existent file",
			serverFile: "non-existent.json",
			token:      "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := client.PublishServer(tt.serverFile, tt.token)

			_ = w.Close()
			os.Stdout = oldStdout

			if (err != nil) != tt.wantErr {
				t.Errorf("PublishServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				out, _ := io.ReadAll(r)
				output := string(out)
				if !strings.Contains(output, "Publish Server") {
					t.Errorf("Expected output to contain 'Publish Server', got %v", output)
				}
			}
		})
	}
}

func TestUpdateServer(t *testing.T) {
	mockServer := createMockServer()
	defer mockServer.Close()

	client := NewMCPXClient(mockServer.URL)

	// Create temp server file
	serverFile := createTempServerFile(t, exampleServerNPMJSON)
	defer func(name string) {
		_ = os.Remove(name)
	}(serverFile)

	tests := []struct {
		name       string
		serverID   string
		serverFile string
		token      string
		json       bool
		wantErr    bool
	}{
		{
			name:       "update server",
			serverID:   "test-server-1",
			serverFile: serverFile,
			token:      "test-token",
			json:       false,
			wantErr:    false,
		},
		{
			name:       "update server json output",
			serverID:   "test-server-1",
			serverFile: serverFile,
			token:      "test-token",
			json:       true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := client.UpdateServer(tt.serverID, tt.serverFile, tt.token, tt.json)

			_ = w.Close()
			os.Stdout = oldStdout

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			out, _ := io.ReadAll(r)
			output := string(out)

			if tt.json {
				// Should contain JSON output
				if !strings.Contains(output, "{") {
					t.Errorf("Expected JSON output, got %v", output)
				}
			} else {
				// Should contain formatted output
				if !strings.Contains(output, "Update Server") {
					t.Errorf("Expected formatted output to contain 'Update Server', got %v", output)
				}
			}
		})
	}
}

func TestDeleteServer(t *testing.T) {
	mockServer := createMockServer()
	defer mockServer.Close()

	client := NewMCPXClient(mockServer.URL)

	tests := []struct {
		name     string
		serverID string
		token    string
		json     bool
		wantErr  bool
	}{
		{
			name:     "delete server",
			serverID: "test-server-1",
			token:    "test-token",
			json:     false,
			wantErr:  false,
		},
		{
			name:     "delete server json output",
			serverID: "test-server-1",
			token:    "test-token",
			json:     true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := client.DeleteServer(tt.serverID, tt.token, tt.json)

			_ = w.Close()
			os.Stdout = oldStdout

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			out, _ := io.ReadAll(r)
			output := string(out)

			if tt.json {
				// Should contain JSON output
				if !strings.Contains(output, "{") {
					t.Errorf("Expected JSON output, got %v", output)
				}
			} else {
				// Should contain formatted output
				if !strings.Contains(output, "Delete Server") {
					t.Errorf("Expected formatted output to contain 'Delete Server', got %v", output)
				}
			}
		})
	}
}

func TestLoginAnonymous(t *testing.T) {
	mockServer := createMockServer()
	defer mockServer.Close()

	client := NewMCPXClient(mockServer.URL)

	// Create temp directory for config
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func(key, value string) {
		_ = os.Setenv(key, value)
	}("HOME", oldHome)

	err := client.loginAnonymous()
	if err != nil {
		t.Fatalf("loginAnonymous() error = %v", err)
	}

	// Verify config was saved
	config, err := client.loadAuthConfig()
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if config.Method != AuthMethodAnonymous {
		t.Errorf("Expected method %v, got %v", AuthMethodAnonymous, config.Method)
	}
	if config.Token == "" {
		t.Errorf("Expected non-empty token")
	}
}

func TestLogout(t *testing.T) {
	// Create temp config
	config := AuthConfig{
		Method:    AuthMethodAnonymous,
		Token:     "test-token",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	createTempConfig(t, config)

	client := NewMCPXClient("http://localhost:8080")

	// Verify config exists
	loadedConfig, err := client.loadAuthConfig()
	if err != nil {
		t.Fatalf("Failed to load auth config: %v", err)
	}
	if loadedConfig.Token == "" {
		t.Fatalf("Expected token to exist before logout")
	}

	// Logout
	err = client.logout()
	if err != nil {
		t.Fatalf("logout() error = %v", err)
	}

	// Verify config was cleared
	loadedConfig, err = client.loadAuthConfig()
	if err != nil {
		t.Fatalf("Failed to load auth config after logout: %v", err)
	}
	if loadedConfig.Token != "" {
		t.Errorf("Expected empty token after logout, got %v", loadedConfig.Token)
	}
}

func TestMakeRequestWithAuth(t *testing.T) {
	mockServer := createMockServer()
	defer mockServer.Close()

	client := NewMCPXClient(mockServer.URL)

	// Test with explicit token
	t.Run("with explicit token", func(t *testing.T) {
		resp, err := client.makeRequest("GET", "/v0/health", nil, "explicit-token")
		if err != nil {
			t.Fatalf("makeRequest() error = %v", err)
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %v", resp.StatusCode)
		}
	})

	// Test with stored auth
	t.Run("with stored auth", func(t *testing.T) {
		config := AuthConfig{
			Method:    AuthMethodAnonymous,
			Token:     "stored-token",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}
		createTempConfig(t, config)

		resp, err := client.makeRequest("GET", "/v0/health", nil, "")
		if err != nil {
			t.Fatalf("makeRequest() error = %v", err)
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %v", resp.StatusCode)
		}
	})
}

// Benchmark tests
func BenchmarkNewMCPXClient(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewMCPXClient("https://example.com")
	}
}

func BenchmarkAuthConfigLoad(b *testing.B) {
	// Setup
	config := AuthConfig{
		Method:    AuthMethodAnonymous,
		Token:     "test-token",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, configFileName)
	data, _ := json.MarshalIndent(config, "", "  ")
	_ = os.WriteFile(configPath, data, 0600)

	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func(key, value string) {
		_ = os.Setenv(key, value)
	}("HOME", oldHome)

	client := NewMCPXClient("http://localhost:8080")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.loadAuthConfig()
	}
}
