package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// FrameworkConfig holds model-specific configuration for each analysis framework
type FrameworkConfig struct {
	Model       string
	Temperature float64
	MaxTokens   int
}

// Config holds all application configuration
type Config struct {
	// Server configuration
	Port           string
	Environment    string
	AllowedOrigins string // Comma-separated origins for CORS

	// Database (Supabase Postgres)
	DatabaseURL string

	// AI/LLM Configuration
	OpenRouterAPIKey string
	EnrichmentModel  string // Gemini 2.0 Flash (deprecated, use Frameworks["enrichment"])
	AnalysisModel    string // Gemini 2.0 Pro (deprecated, use Frameworks[framework])
	SynthesisModel   string // Claude 3.5 Sonnet (deprecated, use Frameworks["synthesis"])

	// Framework-specific model configurations (heterogeneous approach)
	Frameworks map[string]FrameworkConfig

	// Redis for background jobs
	RedisURL      string
	RedisPassword string

	// Worker configuration
	WorkerEnabled     bool
	WorkerConcurrency int
	WorkerQueues      string

	// External Services
	GotenbergURL      string
	SupabaseURL       string
	SupabaseAnonKey   string // Public key for Supabase Auth API calls
	SupabaseJWTSecret string // Secret for validating JWT tokens (AuthMiddleware)

	// Storage & Frontend
	StorageBasePath string
	StorageBaseURL  string
	FrontendURL     string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	_ = godotenv.Load() // Load .env if present
	setupLogger()

	cfg := &Config{
		Port:           getEnv("SERVER_PORT", "8080"),
		Environment:    getEnv("ENV", "development"),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:3000"),
		DatabaseURL:    getEnv("DATABASE_URL", ""),

		// AI Configuration
		OpenRouterAPIKey: getEnv("OPENAI_API_KEY", ""),
		EnrichmentModel:  getEnv("AI_ENRICHMENT_MODEL", "google/gemini-2.0-flash-001"),
		AnalysisModel:    getEnv("AI_ANALYSIS_MODEL", "google/gemini-2.0-pro-exp-02-05"),
		SynthesisModel:   getEnv("AI_SYNTHESIS_MODEL", "anthropic/claude-3.5-sonnet"),

		// Redis & Worker
		// Parse Redis connection from REDIS_URL (Railway) or fall back to REDIS_ADDR (local)
		RedisURL:          "",
		RedisPassword:     "",
		WorkerEnabled:     getEnvBool("WORKER_ENABLED", true),
		WorkerConcurrency: getEnvInt("ASYNQ_CONCURRENCY", 10),
		WorkerQueues:      "critical:6,default:3,low:1",

		// External Services
		GotenbergURL:      getEnv("GOTENBERG_URL", "http://localhost:3000"),
		SupabaseURL:       getEnv("SUPABASE_URL", ""),
		SupabaseAnonKey:   getEnv("SUPABASE_ANON_KEY", ""),
		SupabaseJWTSecret: getEnv("SUPABASE_JWT_SECRET", ""),

		// Storage
		StorageBasePath: "./uploads",
		StorageBaseURL:  "http://localhost:8080/uploads",
		FrontendURL:     "http://localhost:3000",
	}

	// Parse Redis configuration
	cfg.RedisURL, cfg.RedisPassword = parseRedisConfig()

	// Load framework-specific configurations
	cfg.Frameworks = loadFrameworkConfigs()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// parseRedisConfig parses Redis connection from REDIS_URL (Railway format)
// or falls back to REDIS_ADDR (local development)
func parseRedisConfig() (addr string, password string) {
	// Try Railway's REDIS_URL first (format: redis://default:password@host:port)
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		parsed, err := url.Parse(redisURL)
		if err == nil {
			// Extract host:port
			addr = parsed.Host

			// Extract password from URL
			if parsed.User != nil {
				password, _ = parsed.User.Password()
			}

			log.Info().Str("addr", addr).Bool("has_password", password != "").Msg("Using REDIS_URL")
			return addr, password
		}
		log.Warn().Err(err).Str("url", redisURL).Msg("Failed to parse REDIS_URL, falling back to REDIS_ADDR")
	}

	// Fall back to REDIS_ADDR (local development)
	addr = getEnv("REDIS_ADDR", "localhost:6379")
	password = getEnv("REDIS_PASSWORD", "")

	// Handle REDIS_ADDR that might include protocol
	if strings.HasPrefix(addr, "redis://") {
		parsed, err := url.Parse(addr)
		if err == nil {
			addr = parsed.Host
			if parsed.User != nil {
				password, _ = parsed.User.Password()
			}
		}
	}

	log.Info().Str("addr", addr).Bool("has_password", password != "").Msg("Using REDIS_ADDR")
	return addr, password
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.OpenRouterAPIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}
	if c.SupabaseJWTSecret == "" {
		return fmt.Errorf("SUPABASE_JWT_SECRET is required for JWT validation")
	}
	if c.SupabaseAnonKey == "" {
		return fmt.Errorf("SUPABASE_ANON_KEY is required for auth API calls")
	}
	return nil
}

// Helper functions
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		return value == "true" || value == "1"
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if value, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return fallback
}

// loadFrameworkConfigs loads heterogeneous model configurations for all analysis frameworks
// Based on Strategic Cascade Optimization Report (Nov 2025)
func loadFrameworkConfigs() map[string]FrameworkConfig {
	configs := make(map[string]FrameworkConfig)

	// Enrichment Layer (Layer 0)
	configs["enrichment"] = FrameworkConfig{
		Model:       getEnv("AI_ENRICHMENT_MODEL", "google/gemini-2.0-flash-001"),
		Temperature: getEnvFloat("AI_ENRICHMENT_TEMP", 0.5),
		MaxTokens:   getEnvInt("AI_ENRICHMENT_MAX_TOKENS", 8000),
	}

	// Layer 1: Environment Scanning
	configs["pestel"] = FrameworkConfig{
		Model:       getEnv("AI_PESTEL_MODEL", "openai/o3-mini"),
		Temperature: getEnvFloat("AI_PESTEL_TEMP", 0.2),
		MaxTokens:   getEnvInt("AI_PESTEL_MAX_TOKENS", 300),
	}

	configs["porter"] = FrameworkConfig{
		Model:       getEnv("AI_PORTER_MODEL", "anthropic/claude-3.5-sonnet"),
		Temperature: getEnvFloat("AI_PORTER_TEMP", 0.3),
		MaxTokens:   getEnvInt("AI_PORTER_MAX_TOKENS", 250),
	}

	configs["tam_sam_som"] = FrameworkConfig{
		Model:       getEnv("AI_TAM_MODEL", "openai/o3-mini"),
		Temperature: getEnvFloat("AI_TAM_TEMP", 0.1),
		MaxTokens:   getEnvInt("AI_TAM_MAX_TOKENS", 350),
	}

	// Layer 2: Positioning
	configs["swot"] = FrameworkConfig{
		Model:       getEnv("AI_SWOT_MODEL", "google/gemini-2.0-pro-exp-02-05"),
		Temperature: getEnvFloat("AI_SWOT_TEMP", 0.4),
		MaxTokens:   getEnvInt("AI_SWOT_MAX_TOKENS", 200),
	}

	configs["benchmarking"] = FrameworkConfig{
		Model:       getEnv("AI_BENCHMARKING_MODEL", "google/gemini-2.0-pro-exp-02-05"),
		Temperature: getEnvFloat("AI_BENCHMARKING_TEMP", 0.35),
		MaxTokens:   getEnvInt("AI_BENCHMARKING_MAX_TOKENS", 220),
	}

	// Layer 3: Strategy
	configs["blue_ocean"] = FrameworkConfig{
		Model:       getEnv("AI_BLUE_OCEAN_MODEL", "anthropic/claude-3.5-sonnet"),
		Temperature: getEnvFloat("AI_BLUE_OCEAN_TEMP", 0.7),
		MaxTokens:   getEnvInt("AI_BLUE_OCEAN_MAX_TOKENS", 300),
	}

	configs["growth_hacking"] = FrameworkConfig{
		Model:       getEnv("AI_GROWTH_HACKING_MODEL", "anthropic/claude-3.5-sonnet"),
		Temperature: getEnvFloat("AI_GROWTH_HACKING_TEMP", 0.6),
		MaxTokens:   getEnvInt("AI_GROWTH_HACKING_MAX_TOKENS", 250),
	}

	configs["scenarios"] = FrameworkConfig{
		Model:       getEnv("AI_SCENARIOS_MODEL", "google/gemini-3.0-pro"),
		Temperature: getEnvFloat("AI_SCENARIOS_TEMP", 0.6),
		MaxTokens:   getEnvInt("AI_SCENARIOS_MAX_TOKENS", 400),
	}

	// Layer 4: Execution
	configs["okrs"] = FrameworkConfig{
		Model:       getEnv("AI_OKRS_MODEL", "openai/o3-mini"),
		Temperature: getEnvFloat("AI_OKRS_TEMP", 0.25),
		MaxTokens:   getEnvInt("AI_OKRS_MAX_TOKENS", 200),
	}

	configs["bsc"] = FrameworkConfig{
		Model:       getEnv("AI_BSC_MODEL", "anthropic/claude-3.5-sonnet"),
		Temperature: getEnvFloat("AI_BSC_TEMP", 0.35),
		MaxTokens:   getEnvInt("AI_BSC_MAX_TOKENS", 220),
	}

	configs["decision_matrix"] = FrameworkConfig{
		Model:       getEnv("AI_DECISION_MATRIX_MODEL", "openai/o3-mini"),
		Temperature: getEnvFloat("AI_DECISION_MATRIX_TEMP", 0.2),
		MaxTokens:   getEnvInt("AI_DECISION_MATRIX_MAX_TOKENS", 300),
	}

	// Synthesis Layer
	configs["synthesis"] = FrameworkConfig{
		Model:       getEnv("AI_SYNTHESIS_MODEL", "anthropic/claude-3.5-sonnet"),
		Temperature: getEnvFloat("AI_SYNTHESIS_TEMP", 0.4),
		MaxTokens:   getEnvInt("AI_SYNTHESIS_MAX_TOKENS", 3000),
	}

	return configs
}

func setupLogger() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}
