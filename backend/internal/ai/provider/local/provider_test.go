package local

import (
	"context"
	"testing"
)

func TestGenerateText(t *testing.T) {
	p := New("", "")
	response, err := p.GenerateText(context.Background(), "Say hello in one sentence.")
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if response == "" {
		t.Fatalf("Empty response")
	}
	// We don't check the exact response because it's non-deterministic.
	t.Logf("Response: %s", response)
}