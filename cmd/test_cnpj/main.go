package main

import (
	"context"
	"fmt"
	"time"

	"backend_v3/jina"
)

func main() {
	client := jina.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test with Natura CNPJ
	cnpj := "71.673.990/0001-77"
	fmt.Printf("Fetching CNPJ data for: %s\n", cnpj)

	data, err := client.FetchCNPJData(ctx, cnpj)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\n✓ CNPJ: %s\n", data.CNPJ)
	fmt.Printf("Content length: %d bytes\n", len(data.Content))
	fmt.Printf("\n--- First 2000 chars ---\n%s\n", data.Content[:min(2000, len(data.Content))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
