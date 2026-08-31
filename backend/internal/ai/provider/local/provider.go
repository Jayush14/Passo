package local

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"passo/backend/internal/ai/provider/contract"
)

// localProvider implements contract.TextGenerator by invoking the Claude Code CLI.
// If the CLI is a server that listens on a port (like our local_claude.exe),
// it starts the server (if not already running) and communicates with it via HTTP.
type localProvider struct{}

// Package-level state for the server.
var (
	serverOnce sync.Once
	serverErr  error
	serverPort int
	serverHost string
	serverCmd  *exec.Cmd
)

// New returns a TextGenerator for the local provider.
// apiKey and model parameters are ignored (kept for compatibility with Factory).
func New(apiKey, model string) contract.TextGenerator {
	return &localProvider{}
}

// startServer checks if the port is already in use (server running) and if not,
// starts the local_claude server. It waits for the server to be ready.
// It is safe to call concurrently; the actual start happens only once.
func startServer() error {
	// Determine the host and port from environment variables.
	host := os.Getenv("CLAUDE_LOCAL_HOST")
	if host == "" {
		host = "localhost"
	}
	portStr := os.Getenv("CLAUDE_LOCAL_PORT")
	if portStr == "" {
		portStr = "8081"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid CLAUDE_LOCAL_PORT: %w", err)
	}

	// Check if the port is already in use (server running) by trying to connect.
	// We'll try for 5 seconds, every 200ms.
	start := time.Now()
	for time.Since(start) < 5*time.Second {
		ln, err := net.Listen("tcp", net.JoinHostPort(host, portStr))
		if err != nil {
			// Port might be in use. Try to dial to see if we can connect.
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, portStr), 200*time.Millisecond)
			if err == nil {
				// Port is in use and we can connect -> server is running.
				conn.Close()
				serverPort = port
				serverHost = host
				return nil
			}
			// If we cannot connect, then the port might be free or the server might not be ready yet.
			// We'll continue to retry.
			time.Sleep(200 * time.Millisecond)
			continue
		}
		// If we can listen, then the port is free.
		ln.Close()
		break
	}

	// If we exited the loop because we found the port is free (by being able to listen),
	// or because we timed out waiting to connect to a server, we will start the server.
	// Check again if the port is free by trying to listen.
	ln, err := net.Listen("tcp", net.JoinHostPort(host, portStr))
	if err != nil {
		return fmt.Errorf("port %d on host %s is unavailable: %w", port, host, err)
	}
	ln.Close()
	// Port is free, we will start the server below.

	// Determine the CLI executable path from env var, then try to look in PATH, then fallback to "claude".
	cliPath := os.Getenv("CLAUDE_CLI_PATH")
	if cliPath == "" {
		// Try to find local_claude in PATH (without extension for Unix, with .exe for Windows)
		cliPath, _ = exec.LookPath("local_claude")
		if cliPath == "" {
			cliPath, _ = exec.LookPath("local_claude.exe")
		}
	}
	if cliPath == "" {
		cliPath = "claude"
	}

	// Start the server process.
	cmd := exec.Command(cliPath) // no arguments -> starts the server
	// We want to see the server's output in the backend's stderr for debugging.
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	// Start the process.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start local claude server: %w", err)
	}
	serverCmd = cmd

	// Wait for the server to be ready by trying to connect to the port.
	// We'll try for 10 seconds, every 200ms.
	start = time.Now()
	for time.Since(start) < 10*time.Second {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, portStr), 200*time.Millisecond)
		if err == nil {
			conn.Close()
			// Server is ready.
			serverPort = port
			serverHost = host
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	// If we get here, the server did not start in time.
	// Stop the process and return an error.
	if serverCmd != nil {
		serverCmd.Process.Kill()
		serverCmd.Wait()
	}
	return fmt.Errorf("local claude server did not become ready within 10 seconds")
}

// GenerateText calls the local claude server and returns its output.
func (p *localProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", errors.New("empty prompt")
	}

	// Start the server (if not already running) once.
	serverOnce.Do(func() {
		serverErr = startServer()
	})
	if serverErr != nil {
		return "", serverErr
	}

	// Prepare the HTTP request.
	url := fmt.Sprintf("http://%s:%d/generate", serverHost, serverPort)
	body := map[string]string{
		"prompt": prompt,
		"mode":   "cli",
	}
	jsonBody, err := marshalBody(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to local claude server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("local claude server returned status %d", resp.StatusCode)
	}

	// Decode the response.
	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response from local claude server: %w", err)
	}

	return result.Response, nil
}

// marshalBody is a helper to marshal the request body.
// We define it here to avoid importing the json package in the global scope (we import it locally in GenerateText).
func marshalBody(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}