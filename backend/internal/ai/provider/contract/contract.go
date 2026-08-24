package contract

import (
	"context"
	"errors"
)

// TextGenerator is the common contract for any AI text-generation service.
type TextGenerator interface {
	// GenerateText returns the generated text for the given prompt.
	// ctx can be used for cancellation/timeouts.
	GenerateText(ctx context.Context, prompt string) (string, error)
}

var (
	ErrMissingAPIKey = errors.New("missing API key")
	ErrEmptyResponse = errors.New("empty response from provider")
)