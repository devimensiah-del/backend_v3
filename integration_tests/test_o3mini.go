package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"backend_v3/config"
	"backend_v3/llm"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Info().Msg("🧪 Testing o3-mini model availability")

	// Test with higher token limit
	cfg := config.FrameworkConfig{
		Model:       "openai/o3-mini",
		Temperature: 0.2,
		MaxTokens:   2000, // Much higher than the 300 we used
	}

	client := llm.NewClient("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")

	// Simple test: Generate PESTEL analysis
	data := map[string]interface{}{
		"enrichment_data": map[string]interface{}{
			"overview": map[string]interface{}{
				"description": "Coimma é fabricante de equipamentos de manejo para gado, balanças bovinas eletrônicas, balanças rodoviárias e sistemas de pesagem",
			},
			"marketPosition": map[string]interface{}{
				"industry":    "Equipamentos Agropecuários / Sistemas de Pesagem",
				"competitors": []string{"Toledo do Brasil", "Balanças Açores", "Alfa Instrumentos"},
			},
		},
	}

	type PESTELTest struct {
		Political      []string `json:"political"`
		Economic       []string `json:"economic"`
		Social         []string `json:"social"`
		Technological  []string `json:"technological"`
		Environmental  []string `json:"environmental"`
		Legal          []string `json:"legal"`
		Summary        string   `json:"summary"`
	}

	var result PESTELTest
	opts := llm.NewGenerationOptions(cfg)

	log.Info().
		Str("model", cfg.Model).
		Int("max_tokens", cfg.MaxTokens).
		Float64("temperature", cfg.Temperature).
		Msg("Calling o3-mini...")

	err := client.GenerateStructuredWithOptions(context.Background(), opts, llm.FrameworkPESTELPrompt, data, &result)

	if err != nil {
		log.Error().Err(err).Msg("❌ o3-mini call FAILED")
		os.Exit(1)
	}

	log.Info().Msg("✅ o3-mini call SUCCEEDED")

	// Print result
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println("\n📊 PESTEL Analysis Result:")
	fmt.Println(string(resultJSON))
}
