package nvidia

import (
	"context"

	oai "github.com/openai/openai-go"
	opt "github.com/openai/openai-go/option"

	"passo/backend/internal/ai/provider/contract"
)

// nvidiaProvider implements contract.TextGenerator using NVIDIA's API via OpenAI-compatible endpoint.
type nvidiaProvider struct {
	client *oai.Client
	model  string
}

// New returns a TextGenerator for NVIDIA using the supplied API key.
// If model is empty, a default model will be used.
func New(apiKey, model string) contract.TextGenerator {
	if apiKey == "" {
		return &nvidiaProvider{}
	}
	if model == "" {
		model = "deepseek-ai/deepseek-v4-pro-0813" // Default fallback model
	}
	client := oai.NewClient(
		opt.WithAPIKey(apiKey),
		opt.WithBaseURL("https://integrate.api.nvidia.com"),
	)
	return &nvidiaProvider{
		client: &client,
		model:  model,
	}
}

// GenerateText calls NVIDIA's chat completions endpoint with the configured model.
func (p *nvidiaProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	if p.client == nil {
		return "", contract.ErrMissingAPIKey
	}
	if p.model == "" {
		return "", contract.ErrMissingAPIKey // Should not happen if New is called properly
	}

	resp, err := p.client.Chat.Completions.New(ctx, oai.ChatCompletionNewParams{
		Messages: []oai.ChatCompletionMessageParamUnion{
			oai.UserMessage(prompt),
		},
		Model: oai.ChatModel(p.model),
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", contract.ErrEmptyResponse
	}
	return resp.Choices[0].Message.Content, nil
}