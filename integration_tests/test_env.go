package integration_tests

import (
	"backend_v3/adapter/scraper"
	"backend_v3/infrastructure"
	"backend_v3/integration_tests/mocks"
	"backend_v3/llm"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// TestEnvironment holds configuration for integration tests
type TestEnvironment struct {
	UseMocks           bool   // Use ALL mocks (LLM, Storage, Scraper)
	UseLLMMock         bool   // Use mock LLM only
	UseStorageMock     bool   // Use mock Supabase Storage only
	UseScraperMock     bool   // Use mock scraper only
	OpenRouterAPIKey   string
	SupabaseURL        string
	SupabaseAnonKey    string
	DatabaseURL        string
	RedisAddr          string
	RedisPassword      string
	GotenbergURL       string
}

// LoadTestEnv loads environment from .env file and determines if mocks should be used
func LoadTestEnv() *TestEnvironment {
	// Try to load .env file (ignore error if not found)
	_ = godotenv.Load("../.env")

	env := &TestEnvironment{
		OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
		SupabaseURL:      os.Getenv("SUPABASE_URL"),
		SupabaseAnonKey:  os.Getenv("SUPABASE_ANON_KEY"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		RedisAddr:        os.Getenv("REDIS_ADDR"),
		RedisPassword:    os.Getenv("REDIS_PASSWORD"),
		GotenbergURL:     os.Getenv("GOTENBERG_URL"),
	}

	// Determine individual mock settings
	env.UseLLMMock = env.shouldUseLLMMock()
	env.UseStorageMock = env.shouldUseStorageMock()
	env.UseScraperMock = true // Always use scraper mock for now

	// UseMocks is true if ANY service uses mocks
	env.UseMocks = env.UseLLMMock || env.UseStorageMock || env.UseScraperMock

	env.logConfiguration()

	return env
}

// LoadTestEnvWithOverrides loads environment with manual override settings
func LoadTestEnvWithOverrides(useLLMMock, useStorageMock bool) *TestEnvironment {
	env := LoadTestEnv()
	env.UseLLMMock = useLLMMock
	env.UseStorageMock = useStorageMock
	env.UseMocks = env.UseLLMMock || env.UseStorageMock || env.UseScraperMock

	env.logConfiguration()
	return env
}

// shouldUseLLMMock returns true if LLM credentials are missing or invalid
func (e *TestEnvironment) shouldUseLLMMock() bool {
	if e.OpenRouterAPIKey == "" || e.OpenRouterAPIKey == "sk-or-your-openrouter-key" {
		return true
	}
	return false
}

// shouldUseStorageMock returns true if Supabase credentials are missing or invalid
func (e *TestEnvironment) shouldUseStorageMock() bool {
	if e.SupabaseURL == "" || e.SupabaseURL == "https://your-project-id.supabase.co" {
		return true
	}
	if e.SupabaseAnonKey == "" {
		return true
	}
	return false
}

// logConfiguration logs the current test configuration
func (e *TestEnvironment) logConfiguration() {
	if e.UseMocks {
		log.Info().
			Bool("llm_mock", e.UseLLMMock).
			Bool("storage_mock", e.UseStorageMock).
			Bool("scraper_mock", e.UseScraperMock).
			Msg("Integration tests running with MIXED configuration")
	} else {
		log.Info().Msg("Integration tests running with ALL REAL services")
	}
}

// HasValidDatabase returns true if a valid database URL is configured
func (e *TestEnvironment) HasValidDatabase() bool {
	// Check if database URL exists and is not a placeholder
	if e.DatabaseURL == "" {
		return false
	}
	// Check for placeholder values that indicate .env.example
	if strings.Contains(e.DatabaseURL, "aws-0-region.pooler.supabase.com") {
		return false
	}
	if strings.Contains(e.DatabaseURL, "your-project") {
		return false
	}
	if strings.Contains(e.DatabaseURL, "xxxx") {
		return false
	}
	return true
}

// GetLLMClient returns either a real or mock LLM client based on environment
func (e *TestEnvironment) GetLLMClient() *llm.Client {
	if e.UseLLMMock {
		log.Debug().Msg("Using MOCK LLM client")
		return mocks.NewMockLLMClient()
	}
	log.Debug().Msg("Using REAL OpenRouter LLM client")
	return llm.NewClient(e.OpenRouterAPIKey)
}

// GetSupabaseStorageClient returns either a real or mock Supabase storage client
// Returns *mocks.SupabaseJSONMock for mock, *infrastructure.SupabaseStorageClient for real
func (e *TestEnvironment) GetSupabaseStorageClient() interface{} {
	if e.UseStorageMock {
		log.Debug().Msg("Using MOCK Supabase JSON storage (file-based)")
		return mocks.NewSupabaseJSONMockDefault()
	}
	log.Debug().Msg("Using REAL Supabase storage")
	return infrastructure.NewSupabaseStorageClient(
		e.SupabaseURL,
		"reports",
		e.SupabaseAnonKey,
	)
}

// GetScraperClient returns either a real or mock scraper client
func (e *TestEnvironment) GetScraperClient() *scraper.Client {
	if e.UseScraperMock {
		log.Debug().Msg("Using MOCK scraper client")
		return mocks.NewMockScraperClient()
	}
	log.Debug().Msg("Using REAL scraper client")
	return scraper.NewClient()
}

// SetMockResponses configures mock responses for testing
func (e *TestEnvironment) SetMockResponses(enrichmentJSON string, analysisJSON string) {
	if !e.UseLLMMock {
		return // Only applicable for LLM mocks
	}
	// This would configure the mock LLM client with specific responses
	// Implementation depends on mock client structure
}

// GetConfigSummary returns a human-readable configuration summary
func (e *TestEnvironment) GetConfigSummary() string {
	var parts []string

	if e.UseLLMMock {
		parts = append(parts, "LLM: MOCK")
	} else {
		parts = append(parts, "LLM: REAL (OpenRouter)")
	}

	if e.UseStorageMock {
		parts = append(parts, "Storage: MOCK (JSON)")
	} else {
		parts = append(parts, "Storage: REAL (Supabase)")
	}

	if e.UseScraperMock {
		parts = append(parts, "Scraper: MOCK")
	} else {
		parts = append(parts, "Scraper: REAL")
	}

	return strings.Join(parts, " | ")
}
