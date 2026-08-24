package openai

import (
	"context"

	oai "github.com/openai/openai-go"
	opt "github.com/openai/openai-go/option"

	"passo/backend/internal/ai/provider/contract"
)

// openAIProvider implements contract.TextGenerator using OpenAI's API.
type openAIProvider struct {
	client *oai.Client
	model  string
}

// New returns a TextGenerator for OpenAI using the supplied API key.
// If model is empty, GPT-4o will be used as default.
func New(apiKey, model string) contract.TextGenerator {
	if apiKey == "" {
		return &openAIProvider{}
	}
	if model == "" {
		model = string(oai.ChatModelGPT4o) // Default fallback model
	}
	client := oai.NewClient(opt.WithAPIKey(apiKey))
	return &openAIProvider{
		client: &client,
		model:  model,
	}
}

// GenerateText calls OpenAI's chat completions endpoint with the configured model.
func (p *openAIProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
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