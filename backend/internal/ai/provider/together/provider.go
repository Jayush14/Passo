package together

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"passo/backend/internal/ai/provider/contract"
)

// togetherProvider implements contract.TextGenerator using Together AI's API.
type togetherProvider struct {
	apiKey string
	model  string
}

// New returns a TextGenerator for Together AI using the supplied API key.
// If model is empty, a default model will be used.
func New(apiKey, model string) contract.TextGenerator {
	if apiKey == "" {
		return &togetherProvider{}
	}
	if model == "" {
		model = "mistralai/Mixtral-8x7B-Instruct-v0.1" // Default fallback model
	}
	return &togetherProvider{
		apiKey: apiKey,
		model:  model,
	}
}

// GenerateText calls Together AI's API with the configured model.
func (p *togetherProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	if p.apiKey == "" {
		return "", contract.ErrMissingAPIKey
	}
	if p.model == "" {
		return "", fmt.Errorf("together model not configured")
	}

	payload := map[string]interface{}{
		"model":  p.model,
		"prompt": prompt,
		// Add reasonable defaults for other parameters
		"max_tokens": 512,
		"temperature": 0.7,
		"top_p": 0.7,
		"top_k": 50,
		"repetition_penalty": 1,
		"stop": []string{"<|eot_id|>", "<|eom_id|>"}, // Common stop tokens
		"stream": false,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.together.xyz/v1/completions", bytes.NewReader(payloadBytes))
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
		return "", fmt.Errorf("together API error: %s", resp.Status)
	}

	var respData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", err
	}

	if output, ok := respData["output"].(map[string]interface{}); ok {
		if choices, ok := output["choices"].([]interface{}); ok && len(choices) > 0 {
			if firstChoice, ok := choices[0].(map[string]interface{}); ok {
				if text, ok := firstChoice["text"].(string); ok {
					return text, nil
				}
			}
		}
	}

	return "", contract.ErrEmptyResponse
}