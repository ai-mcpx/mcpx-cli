package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Integration tests that test the CLI commands end-to-end

func TestCLIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Build the CLI binary for testing
	binaryPath := buildCLIBinary(t)
	defer func(name string) {
		_ = os.Remove(name)
	}(binaryPath)

	// Create mock server
	mockServer := createMockServer()
	defer mockServer.Close()

	t.Run("help command", func(t *testing.T) {
		output, err := runCLI(t, binaryPath, "--help")
		if err != nil {
			t.Fatalf("Help command failed: %v", err)
		}

		expectedStrings := []string{
			"mcpx-cli - A command-line client",
			"Commands:",
			"login",
			"logout",
			"health",
			"servers",
			"publish",
			"update",
			"delete",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(output, expected) {
				t.Errorf("Help output missing expected string: %s", expected)
			}
		}
	})

	t.Run("version command", func(t *testing.T) {
		output, err := runCLI(t, binaryPath, "--version")
		if err != nil {
			t.Fatalf("Version command failed: %v", err)
		}

		if !strings.Contains(output, "dev") {
			t.Errorf("Version output should contain version, got: %s", output)
		}
	})

	t.Run("health command", func(t *testing.T) {
		output, err := runCLI(t, binaryPath, "--base-url", mockServer.URL, "health")
		if err != nil {
			t.Fatalf("Health command failed: %v", err)
		}

		if !strings.Contains(output, "Health Check") {
			t.Errorf("Health output missing expected content, got: %s", output)
		}
	})

	t.Run("servers list command", func(t *testing.T) {
		output, err := runCLI(t, binaryPath, "--base-url", mockServer.URL, "servers", "--limit", "5")
		if err != nil {
			t.Fatalf("Servers command failed: %v", err)
		}

		expectedStrings := []string{
			"List Servers",
			"test-server-1",
			"test-server-2",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(output, expected) {
				t.Errorf("Servers output missing expected string: %s", expected)
			}
		}
	})

	t.Run("servers list json command", func(t *testing.T) {
		output, err := runCLI(t, binaryPath, "--base-url", mockServer.URL, "servers", "--json")
		if err != nil {
			t.Fatalf("Servers JSON command failed: %v", err)
		}

		// Should be valid JSON
		var response ServersResponse
		if err := json.Unmarshal([]byte(output), &response); err != nil {
			t.Errorf("Invalid JSON output: %v\nOutput: %s", err, output)
		}

		if len(response.Servers) == 0 {
			t.Errorf("Expected servers in JSON response")
		}
	})

	t.Run("server detail command", func(t *testing.T) {
		output, err := runCLI(t, binaryPath, "--base-url", mockServer.URL, "server", "test-server-1")
		if err != nil {
			t.Fatalf("Server detail command failed: %v", err)
		}

		expectedStrings := []string{
			"Server Details",
			"test-server-1",
			"io.test/server1",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(output, expected) {
				t.Errorf("Server detail output missing expected string: %s", expected)
			}
		}
	})

	t.Run("login anonymous command", func(t *testing.T) {
		// Create temp directory for this test
		tmpDir := t.TempDir()

		output, err := runCLIWithEnv(t, binaryPath, map[string]string{"HOME": tmpDir},
			"--base-url", mockServer.URL, "login", "--method", "anonymous")
		if err != nil {
			t.Fatalf("Login command failed: %v", err)
		}

		if !strings.Contains(output, "Successfully authenticated") {
			t.Errorf("Login output missing success message, got: %s", output)
		}

		// Verify config file was created
		configPath := filepath.Join(tmpDir, configFileName)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("Config file was not created at %s", configPath)
		}

		// Verify config content
		configData, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read config file: %v", err)
		}

		var config AuthConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			t.Fatalf("Failed to parse config file: %v", err)
		}

		if config.Method != AuthMethodAnonymous {
			t.Errorf("Expected method %s, got %s", AuthMethodAnonymous, config.Method)
		}
		if config.Token == "" {
			t.Errorf("Expected non-empty token")
		}
	})

	t.Run("logout command", func(t *testing.T) {
		// Create temp directory with existing config
		tmpDir := t.TempDir()
		config := AuthConfig{
			Method:    AuthMethodAnonymous,
			Token:     "test-token",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}
		configPath := filepath.Join(tmpDir, configFileName)
		configData, _ := json.MarshalIndent(config, "", "  ")
		_ = os.WriteFile(configPath, configData, 0600)

		output, err := runCLIWithEnv(t, binaryPath, map[string]string{"HOME": tmpDir}, "logout")
		if err != nil {
			t.Fatalf("Logout command failed: %v", err)
		}

		if !strings.Contains(output, "Successfully logged out") {
			t.Errorf("Logout output missing success message, got: %s", output)
		}

		// Verify config file was removed or cleared
		if _, err := os.Stat(configPath); err == nil {
			// File exists, check if it's empty
			configData, _ := os.ReadFile(configPath)
			var emptyConfig AuthConfig
			_ = json.Unmarshal(configData, &emptyConfig)
			if emptyConfig.Token != "" {
				t.Errorf("Config file should be cleared after logout")
			}
		}
	})

	t.Run("publish command", func(t *testing.T) {
		// Create temporary server file
		serverFile := createTempServerFile(t, exampleServerNPMJSON)
		defer func(name string) {
			_ = os.Remove(name)
		}(serverFile)

		output, err := runCLI(t, binaryPath, "--base-url", mockServer.URL, "publish", serverFile)
		if err != nil {
			t.Fatalf("Publish command failed: %v", err)
		}

		if !strings.Contains(output, "Publish Server") {
			t.Errorf("Publish output missing expected content, got: %s", output)
		}
	})

	t.Run("update command", func(t *testing.T) {
		// Create temporary server file
		serverFile := createTempServerFile(t, exampleServerNPMJSON)
		defer func(name string) {
			_ = os.Remove(name)
		}(serverFile)

		output, err := runCLI(t, binaryPath, "--base-url", mockServer.URL, "update", "test-server-1", serverFile)
		if err != nil {
			t.Fatalf("Update command failed: %v", err)
		}

		if !strings.Contains(output, "Update Server") {
			t.Errorf("Update output missing expected content, got: %s", output)
		}
	})

	t.Run("delete command", func(t *testing.T) {
		output, err := runCLI(t, binaryPath, "--base-url", mockServer.URL, "delete", "test-server-1")
		if err != nil {
			t.Fatalf("Delete command failed: %v", err)
		}

		if !strings.Contains(output, "Delete Server") {
			t.Errorf("Delete output missing expected content, got: %s", output)
		}
	})
}

func TestCLIErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	binaryPath := buildCLIBinary(t)
	defer func(name string) {
		_ = os.Remove(name)
	}(binaryPath)

	tests := []struct {
		name          string
		args          []string
		expectedError string
		shouldFail    bool
	}{
		{
			name:          "unknown command",
			args:          []string{"unknown-command"},
			expectedError: "Unknown command",
			shouldFail:    true,
		},
		{
			name:          "missing server ID for detail",
			args:          []string{"server"},
			expectedError: "server ID is required",
			shouldFail:    true,
		},
		{
			name:          "missing server ID for update",
			args:          []string{"update"},
			expectedError: "server ID is required",
			shouldFail:    true,
		},
		{
			name:          "missing server ID for delete",
			args:          []string{"delete"},
			expectedError: "server ID is required",
			shouldFail:    true,
		},
		{
			name:          "invalid server file for publish",
			args:          []string{"publish", "non-existent.json"},
			expectedError: "no such file or directory",
			shouldFail:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := runCLI(t, binaryPath, tt.args...)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected command to fail, but it succeeded. Output: %s", output)
				}
				if !strings.Contains(output, tt.expectedError) && !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error message to contain '%s', got output: %s, error: %v", tt.expectedError, output, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected command to succeed, but it failed: %v", err)
				}
			}
		})
	}
}

func TestCLIWithInvalidServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	binaryPath := buildCLIBinary(t)
	defer func(name string) {
		_ = os.Remove(name)
	}(binaryPath)

	// Test with unreachable server
	t.Run("unreachable server", func(t *testing.T) {
		output, err := runCLI(t, binaryPath, "--base-url", "http://localhost:9999", "health")
		if err == nil {
			t.Errorf("Expected health check to fail with unreachable server, but it succeeded: %s", output)
		}
	})
}

// Helper functions for integration tests

func buildCLIBinary(t *testing.T) string {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "mcpx-cli-test")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "." // Current directory where main.go is located

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI binary: %v\nOutput: %s", err, output)
	}

	return binaryPath
}

func runCLI(t *testing.T, binaryPath string, args ...string) (string, error) {
	return runCLIWithEnv(t, binaryPath, nil, args...)
}

func runCLIWithEnv(t *testing.T, binaryPath string, env map[string]string, args ...string) (string, error) {
	cmd := exec.Command(binaryPath, args...)

	// Set environment variables
	if env != nil {
		cmd.Env = os.Environ()
		for key, value := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Combine stdout and stderr for complete output
	output := stdout.String() + stderr.String()

	return output, err
}

// Additional test for authentication flow
func TestAuthenticationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	mockServer := createMockServer()
	defer mockServer.Close()

	binaryPath := buildCLIBinary(t)
	defer func(name string) {
		_ = os.Remove(name)
	}(binaryPath)

	tmpDir := t.TempDir()

	// Test login -> use authenticated command -> logout flow
	t.Run("full auth flow", func(t *testing.T) {
		// 1. Login
		loginOutput, err := runCLIWithEnv(t, binaryPath, map[string]string{"HOME": tmpDir},
			"--base-url", mockServer.URL, "login", "--method", "anonymous")
		if err != nil {
			t.Fatalf("Login failed: %v\nOutput: %s", err, loginOutput)
		}

		// 2. Use authenticated command (publish)
		serverFile := createTempServerFile(t, exampleServerNPMJSON)
		defer func(name string) {
			_ = os.Remove(name)
		}(serverFile)

		publishOutput, err := runCLIWithEnv(t, binaryPath, map[string]string{"HOME": tmpDir},
			"--base-url", mockServer.URL, "publish", serverFile)
		if err != nil {
			t.Fatalf("Publish with auth failed: %v\nOutput: %s", err, publishOutput)
		}

		// 3. Logout
		logoutOutput, err := runCLIWithEnv(t, binaryPath, map[string]string{"HOME": tmpDir}, "logout")
		if err != nil {
			t.Fatalf("Logout failed: %v\nOutput: %s", err, logoutOutput)
		}

		// Verify logout message
		if !strings.Contains(logoutOutput, "Successfully logged out") {
			t.Errorf("Logout should show success message")
		}
	})
}

// Performance test for CLI commands
func TestCLIPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	mockServer := createMockServer()
	defer mockServer.Close()

	binaryPath := buildCLIBinary(t)
	defer func(name string) {
		_ = os.Remove(name)
	}(binaryPath)

	t.Run("servers command performance", func(t *testing.T) {
		start := time.Now()

		_, err := runCLI(t, binaryPath, "--base-url", mockServer.URL, "servers", "--limit", "10")
		if err != nil {
			t.Fatalf("Servers command failed: %v", err)
		}

		elapsed := time.Since(start)
		if elapsed > time.Second*5 {
			t.Errorf("Servers command took too long: %v", elapsed)
		}
	})

	t.Run("health command performance", func(t *testing.T) {
		start := time.Now()

		_, err := runCLI(t, binaryPath, "--base-url", mockServer.URL, "health")
		if err != nil {
			t.Fatalf("Health command failed: %v", err)
		}

		elapsed := time.Since(start)
		if elapsed > time.Second*2 {
			t.Errorf("Health command took too long: %v", elapsed)
		}
	})
}
