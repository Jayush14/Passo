package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"passo/backend/internal/ai/provider/contract"
)

// geminiGenerator implements contract.TextGenerator using Gemini's API.
type geminiGenerator struct {
	apiKey       string
	defaultModel string
}

// New returns a TextGenerator for Gemini using the supplied API key.
// If model is empty, a default model will be used.
func New(apiKey, model string) contract.TextGenerator {
	if apiKey == "" {
		return &geminiGenerator{} // Will return ErrMissingAPIKey in GenerateText
	}
	if model == "" {
		model = "gemini-2.0-flash" // Default fallback model
	}
	return &geminiGenerator{
		apiKey:       apiKey,
		defaultModel: model,
	}
}

// GenerateText calls Gemini's API with the configured model.
func (g *geminiGenerator) GenerateText(ctx context.Context, prompt string) (string, error) {
	if g.apiKey == "" {
		return "", contract.ErrMissingAPIKey
	}
	if g.defaultModel == "" {
		return "", fmt.Errorf("gemini model not configured")
	}

	// Build Gemini-specific request
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.defaultModel, g.apiKey)

	// Prepare the request body according to Gemini API format
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.7,
		},
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to prepare request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gemini API error: %s", string(bodyBytes))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", contract.ErrEmptyResponse
	}

	// Return the first candidate's first part's text
	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// GeminiResponse represents the response from the Gemini API.
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}