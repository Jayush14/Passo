package main

import (
	"context"
	"fmt"
	"log"

	"passo/backend/internal/ai/provider/local"
)

func main() {
	p := local.New("", "")
	response, err := p.GenerateText(context.Background(), "Say hello in one sentence.")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Response: %s\n", response)
}