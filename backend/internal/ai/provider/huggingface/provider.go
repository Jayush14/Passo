package huggingface

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"passo/backend/internal/ai/provider/contract"
)

// huggingfaceProvider implements contract.TextGenerator using Hugging Face's API.
type huggingfaceProvider struct {
	apiKey  string
	model   string
	baseURL string
}

// New returns a TextGenerator for Hugging Face using the supplied API key.
// If model is empty, a default model will be used.
func New(apiKey, model string) contract.TextGenerator {
	if apiKey == "" {
		return &huggingfaceProvider{}
	}
	if model == "" {
		model = "google/flan-t5-large" // Default fallback model
	}
	return &huggingfaceProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api-inference.huggingface.co/models/",
	}
}

// GenerateText calls Hugging Face's inference API with the configured model.
func (p *huggingfaceProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	if p.apiKey == "" {
		return "", contract.ErrMissingAPIKey
	}
	if p.model == "" {
		return "", fmt.Errorf("huggingface model not configured")
	}

	payload := map[string]interface{}{
		"inputs": prompt,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+p.model, bytes.NewReader(payloadBytes))
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
		return "", fmt.Errorf("huggingface API error: %s", resp.Status)
	}

	var result []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result) == 0 {
		return "", contract.ErrEmptyResponse
	}

	// Assuming the first result contains the generated text
	if text, ok := result[0]["generated_text"].(string); ok {
		return text, nil
	}

	return "", fmt.Errorf("unexpected response format from huggingface")
}