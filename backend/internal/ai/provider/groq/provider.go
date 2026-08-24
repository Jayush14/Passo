package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"passo/backend/internal/ai/provider/contract"
)

// groqProvider implements contract.TextGenerator using Groq's API.
type groqProvider struct {
	apiKey string
	model  string
}

// New returns a TextGenerator for Groq using the supplied API key.
// If model is empty, a default model will be used.
func New(apiKey, model string) contract.TextGenerator {
	if apiKey == "" {
		return &groqProvider{}
	}
	if model == "" {
		model = "mixtral-8x7b-32768" // Default fallback model
	}
	return &groqProvider{
		apiKey: apiKey,
		model:  model,
	}
}

// GenerateText calls Groq's chat completions API with the configured model.
func (p *groqProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	if p.apiKey == "" {
		return "", contract.ErrMissingAPIKey
	}
	if p.model == "" {
		return "", fmt.Errorf("groq model not configured")
	}

	payload := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"model": p.model,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(payloadBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq API error: %s", resp.Status)
	}

	var respData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", err
	}

	if choices, ok := respData["choices"].([]interface{}); ok && len(choices) > 0 {
		if firstChoice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := firstChoice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", contract.ErrEmptyResponse
}