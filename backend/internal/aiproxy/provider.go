package aiproxy

import (
	"context"
)

// AIProvider defines the interface for interacting with an AI model.
type AIProvider interface {
	Generate(ctx context.Context, apiKey string, prompt string) (string, error)
}

// GeminiProvider implements AIProvider using Google's Gemini API.
type GeminiProvider struct{}

// Generate calls the Gemini API (stub).
func (g *GeminiProvider) Generate(ctx context.Context, apiKey string, prompt string) (string, error) {
	// TODO: Implement actual call to generativelanguage.googleapis.com
	// For now, return a placeholder response.
	return "This is a placeholder AI response.", nil
}