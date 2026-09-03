package local

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"passo/backend/internal/ai/provider/contract"
)

// localProvider implements contract.TextGenerator by calling the Free Claude Code API directly.
type localProvider struct {
	baseURL    string
	model      string
	apiKey     string
	httpClient *http.Client
}

// FCCRequest represents the request payload for the Free Claude Code API.
type FCCRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// Message represents a message in the conversation.
type Message struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// FCCResponse represents the response from the Free Claude Code API.
type FCCResponse struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"` // usually "message"
	Role    string         `json:"role"` // "assistant"
	Content []ContentBlock `json:"content"`
	Model   string         `json:"model"`
	Usage   Usage          `json:"usage,omitempty"`
}

// ContentBlock represents a block in the response content.
type ContentBlock struct {
	Type     string `json:"type"`               // "thinking" or "text"
	Thinking string `json:"thinking,omitempty"` // for thinking blocks
	Text     string `json:"text,omitempty"`     // for text blocks
}

// Usage represents token usage information.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// New returns a TextGenerator for the local provider.
// apiKey and model parameters are kept for compatibility with Factory but can be overridden by env vars or defaults.
func New(apiKey, model string) contract.TextGenerator {
	// Use environment variables for configuration with fallbacks
	baseURL := getEnv("FCC_BASE_URL", "http://127.0.0.1:8082")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8082"
	}

	// Use provided model or fallback to default from logs
	if model == "" {
		model = getEnv("FCC_MODEL", "nvidia_nim/nvidia/nemotron-3-super-120b-a12b")
	}

	// Use provided apiKey or fallback to default
	// if apiKey == "" {
	apiKey = getEnv("FCC_API_KEY", "freecc")
	// }

	return &localProvider{
		baseURL:    baseURL,
		model:      model,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 200 * time.Second},
	}
}

// GenerateText calls the Free Claude Code API and returns the generated text.
// func (p *localProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
// 	if prompt == "" {
// 		return "", errors.New("empty prompt")
// 	}

// 	// Build request payload
// 	reqBody := FCCRequest{
// 		Model: p.model,
// 		Messages: []Message{
// 			{
// 				Role:    "user",
// 				Content: prompt,
// 			},
// 		},
// 		MaxTokens:   1000,
// 		Temperature: 0.7,
// 		TopP:        0.9,
// 		Stream:      false, // Critical for non-streaming response
// 	}

// 	// Marshal JSON
// 	jsonData, err := json.Marshal(reqBody)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to marshal request: %w", err)
// 	}

// 	// Create HTTP request
// 	req, err := http.NewRequestWithContext(ctx, "POST",
// 		p.baseURL+"/v1/messages", bytes.NewBuffer(jsonData))
// 	if err != nil {
// 		return "", fmt.Errorf("failed to create request: %w", err)
// 	}

// 	// Set headers
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("x-api-key", p.apiKey) // Default FCC token

// 	// Execute request
// 	resp, err := p.httpClient.Do(req)
// 	if err != nil {
// 		return "", fmt.Errorf("HTTP request failed: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	// Check status code
// 	if resp.StatusCode != http.StatusOK {
// 		bodyBytes, _ := io.ReadAll(resp.Body)
// 		return "", fmt.Errorf("API returned status %d: %s",
// 			resp.StatusCode, string(bodyBytes))
// 	}

// 	// Decode response
// 	fmt.Printf("Response status: %s\n", resp)
// 	var fccResp FCCResponse
// 	if err := json.NewDecoder(resp.Body).Decode(&fccResp); err != nil {
// 		return "", fmt.Errorf("failed to decode response: %w", err)
// 	}

// 	// Extract text content (skip thinking blocks)
// 	var textParts []string
// 	for _, block := range fccResp.Content {
// 		if block.Type == "text" && block.Text != "" {
// 			textParts = append(textParts, block.Text)
// 		}
// 	}

// 	return strings.Join(textParts, "\n"), nil
// }

func (p *localProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", errors.New("empty prompt")
	}

	// Build request payload
	reqBody := FCCRequest{
		Model: p.model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Marshal request body
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Same URL that works in Postman
	url := p.baseURL + "/v1/messages?beta=true"

	// Create HTTP request
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Debug response status
	fmt.Printf("Response status: %s\n", resp.Status)
	fmt.Printf("Response content-type: %s\n", resp.Header.Get("Content-Type"))

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf(
				"API returned status %d and failed to read response body: %w",
				resp.StatusCode,
				err,
			)
		}

		return "", fmt.Errorf(
			"API returned status %d: %s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}

	// The API returns Server-Sent Events (SSE), even when stream=false.
	//
	// Example:
	//
	// event: content_block_delta
	// data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Hello"}}
	//
	// We read the response line by line and collect only text_delta events.

	scanner := bufio.NewScanner(resp.Body)

	// Increase Scanner buffer size in case the API sends a large event.
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	var textParts []string

	for scanner.Scan() {
		line := scanner.Text()

		// Ignore empty lines.
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Ignore SSE event lines such as:
		//
		// event: message_start
		// event: content_block_delta
		// event: message_stop
		//
		// We only need the "data:" lines.
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Decode the SSE data.
		var event struct {
			Type  string `json:"type"`
			Delta *struct {
				Type     string `json:"type"`
				Text     string `json:"text"`
				Thinking string `json:"thinking"`
			} `json:"delta"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return "", fmt.Errorf(
				"failed to decode SSE event: %w\nbody: %s",
				err,
				data,
			)
		}

		// We only want the actual response text.
		//
		// Ignore:
		//   thinking_delta
		//
		// Collect:
		//   text_delta
		if event.Type == "content_block_delta" &&
			event.Delta != nil &&
			event.Delta.Type == "text_delta" &&
			event.Delta.Text != "" {

			textParts = append(textParts, event.Delta.Text)
		}
	}

	// Check for scanner errors.
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed reading SSE response: %w", err)
	}

	// Combine all text chunks into one response.
	result := strings.Join(textParts, "")

	if result == "" {
		return "", errors.New("API returned no text content")
	}

	fmt.Printf("Final generated text: %s\n", result)
	return result, nil
}

// getEnv gets an environment variable or returns a fallback value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
