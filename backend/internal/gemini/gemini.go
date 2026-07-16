package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GenerateText generates text using the Gemini API with the provided API key.
// It calls the Gemini API with the specified model and prompt, and returns the generated text.
func GenerateText(apiKey, prompt, model string) (string, error) {
	// Call Gemini API using provided API key and model
	fmt.Printf("Calling Gemini API with model: %s and prompt: %s\n", model, prompt) // Log the request for debugging purposes.
	geminiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

	// Prepare the request body according to Gemini API format
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
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

	resp, err := http.Post(geminiURL, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini API error: %s", string(bodyBytes))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no candidates returned from Gemini")
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
