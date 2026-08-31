package provider

import (
	"context"
	"fmt"
	"strings"

	"passo/backend/internal/ai/provider/contract"
	"passo/backend/internal/ai/provider/gemini"
	"passo/backend/internal/ai/provider/huggingface"
	"passo/backend/internal/ai/provider/groq"
	"passo/backend/internal/ai/provider/local"
	"passo/backend/internal/ai/provider/openai"
	"passo/backend/internal/ai/provider/together"
)

// Factory returns a TextGenerator for the given provider name and API key.
// Returns an error if the provider is unknown or the key is missing.
// model parameter is optional; if empty, provider's default model will be used.
func Factory(ctx context.Context, providerName, apiKey, model string) (contract.TextGenerator, error) {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "gemini":
		return gemini.New(apiKey, model), nil
	case "huggingface", "hf":
		return huggingface.New(apiKey, model), nil
	case "groq":
		return groq.New(apiKey, model), nil
	case "local":
		return local.New(apiKey, model), nil
	case "openai":
		return openai.New(apiKey, model), nil
	case "together", "togetherai":
		return together.New(apiKey, model), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", providerName)
	}
}