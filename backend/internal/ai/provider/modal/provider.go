package modal

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

// modalProvider implements contract.TextGenerator by calling the Modal GLM-5.3 endpoint.
type modalProvider struct {
	httpClient *http.Client
	apiKey     string
	model      string
	baseURL    string
}

// New returns a TextGenerator for the Modal provider.
// apiKey and model parameters are kept for compatibility with Factory but token is sourced from env.
func New(apiKey, model string) contract.TextGenerator {
	// Use environment variables for token ID and secret
	tokenID := os.Getenv("MODAL_PROXY_TOKEN_ID")
	tokenSecret := os.Getenv("MODAL_PROXY_TOKEN_SECRET")
	var key string
	if tokenID != "" && tokenSecret != "" {
		key = fmt.Sprintf("%s.%s", tokenID, tokenSecret)
	} else {
		// fallback to provided apiKey (should contain the combined token)
		key = apiKey
	}
	// Use provided model or fallback to default from example
	if model == "" {
		model = "zai-org/GLM-5.3"
	}
	baseURL := "https://littleramp7--ep-glm-5-3-server.us-west.modal.direct/v1/chat/completions"

	return &modalProvider{
		httpClient: &http.Client{Timeout: 200 * time.Second},
		apiKey:     key,
		model:      model,
		baseURL:    baseURL,
	}
}

// GenerateText calls the Modal API and returns the generated text.
func (p *modalProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	fmt.Printf("Modal provider: Generating text for prompt: %s\n", prompt)
	if prompt == "" {
		return "", errors.New("empty prompt")
	}

	// Build request payload matching OpenAI chat completions format
	reqBody := map[string]interface{}{
		"model":    p.model,
		"messages": []map[string]string{{"role": "user", "content": prompt}},
	}

	// fmt.Printf("Request payload: %+v\n", prompt)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json") // Prefer JSON response
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := readAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Check Content-Type to determine how to parse the response
	contentType := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		// Handle JSON response
		var modalResp ModalResponse
		if err := json.NewDecoder(resp.Body).Decode(&modalResp); err != nil {
			return "", fmt.Errorf("failed to decode JSON response: %w", err)
		}

		// Extract content from first choice message
		if len(modalResp.Choices) == 0 {
			return "", errors.New("empty choices in response")
		}
		choice := modalResp.Choices[0]
		if choice.Message.Content == "" {
			return "", errors.New("empty message content in response")
		}
		return choice.Message.Content, nil

	case strings.Contains(contentType, "text/event-stream"):
		// Handle Server-Sent Events (SSE) response
		return p.parseSSEResponse(resp.Body)

	default:
		// Fallback: try to parse as JSON first, then as SSE if JSON fails
		// Reset body to try again (note: we can't actually reset, so we'll try JSON and if it fails, assume SSE)
		// Since we already read the body in the status check above, we need to recreate the request
		// For simplicity, we'll try to parse as JSON from the current body position
		// Note: This is a bit risky but handles edge cases
		var modalResp ModalResponse
		if err := json.NewDecoder(resp.Body).Decode(&modalResp); err != nil {
			// If JSON parsing fails, try SSE
			// We need to reset the body to the beginning, but we can't easily do that
			// Instead, we'll return an error suggesting the content type is unexpected
			return "", fmt.Errorf("unexpected Content-Type: %s. Expected application/json or text/event-stream", contentType)
		}

		// Extract content from first choice message
		if len(modalResp.Choices) == 0 {
			return "", errors.New("empty choices in response")
		}
		choice := modalResp.Choices[0]
		if choice.Message.Content == "" {
			return "", errors.New("empty message content in response")
		}
		// fmt.Printf("Final generated text: %s\n", choice.Message.Content)
		return choice.Message.Content, nil
	}
}

// parseSSEResponse handles Server-Sent Events response format and accumulates the content.
func (p *modalProvider) parseSSEResponse(body io.ReadCloser) (string, error) {
	defer body.Close()
	var accumulatedContent strings.Builder
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		// fmt.Println("SSE stream completed", line)
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}
			var chunk map[string]interface{}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				// Skip malformed data lines
				continue
			}
			if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
				if choiceMap, ok := choices[0].(map[string]interface{}); ok {
					if delta, ok := choiceMap["delta"].(map[string]interface{}); ok {
						if content, ok := delta["content"].(string); ok {
							accumulatedContent.WriteString(content)
						}
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading SSE stream: %w", err)
	}
	return accumulatedContent.String(), nil
}

// readAll reads the entire body (helper because io.ReadAll may not be available in older Go)
func readAll(body io.ReadCloser) ([]byte, error) {
	defer body.Close()
	return io.ReadAll(body)
}

// ModalResponse represents the response from the Modal endpoint.
type ModalResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Need to import io
// We'll add import after package line.
