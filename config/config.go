package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// FrameworkConfig holds model-specific configuration for each analysis framework
type FrameworkConfig struct {
	Model         string
	Temperature   float64
	MaxTokens     int
	FallbackModel string // Used if primary model fails (rate limit, unavailable, timeout)
}

// Config holds all application configuration
type Config struct {
	// Server configuration
	Port           string
	Environment    string
	AllowedOrigins string // Comma-separated origins for CORS

	// Database (Supabase Postgres)
	DatabaseURL string

	// Database Connection Pool Configuration
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration

	// AI/LLM Configuration (Simplified 4-Model Approach)
	OpenRouterAPIKey string

	// Primary model: Used for enrichment and all 11 analysis frameworks
	PrimaryModel    string
	PrimaryFallback string

	// Synthesis model: Premium model for executive summary
	SynthesisModel    string
	SynthesisFallback string

	// Shared settings
	AITemperature         float64
	MaxTokensEnrichment   int
	MaxTokensAnalysis     int
	MaxTokensSynthesis    int

	// Framework-specific model configurations (auto-generated from simplified config)
	Frameworks map[string]FrameworkConfig

	// Redis for background jobs
	RedisURL      string
	RedisPassword string

	// Worker configuration
	WorkerEnabled        bool
	WorkerConcurrency    int
	WorkerQueues         string
	EnrichmentTimeout    int // seconds
	AnalysisTimeout      int // seconds
	EnrichmentMaxRetries int
	AnalysisMaxRetries   int
	JobRetryInitialDelay int // seconds
	JobRetryMaxDelay     int // seconds
	DeadLetterQueueTTL   int // hours

	// External Services
	GotenbergURL       string
	SupabaseURL        string
	SupabaseAnonKey    string // Public key for Supabase Auth API calls
	SupabaseJWTSecret  string // Secret for validating JWT tokens (AuthMiddleware)
	SupabaseServiceKey string // Service role key for storage uploads
	SupabaseBucket     string // Storage bucket for reports
	SupabaseSignedTTL  int    // Signed URL TTL in seconds

	// Storage & Frontend
	StorageBasePath string
	StorageBaseURL  string
	FrontendURL     string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	_ = godotenv.Load() // Load .env if present

	cfg := &Config{
		Port:        getEnv("SERVER_PORT", "8080"),
		Environment: getEnv("ENV", "development"),
		// CRITICAL for Production: Update ALLOWED_ORIGINS env var to include your production frontend
		// Example: ALLOWED_ORIGINS="http://localhost:3000,https://yourdomain.vercel.app,https://yourdomain.com"
		// Multiple origins separated by commas. Default only allows localhost.
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:3000"),
		DatabaseURL:    getEnv("DATABASE_URL", ""),

		// Database Connection Pool (Railway PostgreSQL typically allows 50-100 connections)
		// Production values should be tuned based on: (API instances × max_connections) + worker_connections < DB_limit
		DBMaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25), // Maximum open connections
		DBMaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),  // Idle connections to keep alive
		DBConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 5)) * time.Minute,

		// AI Configuration (Simplified 4-Model Approach)
		OpenRouterAPIKey:  getEnv("OPENAI_API_KEY", ""),
		PrimaryModel:      getEnv("AI_PRIMARY_MODEL", "google/gemini-2.5-flash"),
		PrimaryFallback:   getEnv("AI_PRIMARY_FALLBACK", "openai/gpt-4.1-mini"),
		SynthesisModel:    getEnv("AI_SYNTHESIS_MODEL", "google/gemini-2.5-pro-preview"),
		SynthesisFallback: getEnv("AI_SYNTHESIS_FALLBACK", "openai/gpt-4.1"),
		AITemperature:     getEnvFloat("AI_TEMPERATURE", 0.5),
		MaxTokensEnrichment: getEnvInt("AI_MAX_TOKENS_ENRICHMENT", 10000),
		MaxTokensAnalysis:   getEnvInt("AI_MAX_TOKENS_ANALYSIS", 4000),
		MaxTokensSynthesis:  getEnvInt("AI_MAX_TOKENS_SYNTHESIS", 6000),

		// Redis & Worker
		// Parse Redis connection from REDIS_URL (Railway) or fall back to REDIS_ADDR (local)
		RedisURL:             "",
		RedisPassword:        "",
		WorkerEnabled:        getEnvBool("WORKER_ENABLED", true),
		WorkerConcurrency:    getEnvInt("ASYNQ_CONCURRENCY", 10),
		WorkerQueues:         "critical:6,default:3,low:1",
		EnrichmentTimeout:    getEnvInt("ENRICHMENT_TIMEOUT", 300),     // 5 minutes
		AnalysisTimeout:      getEnvInt("ANALYSIS_TIMEOUT", 900),       // 15 minutes
		EnrichmentMaxRetries: getEnvInt("ENRICHMENT_MAX_RETRIES", 3),   // 3 retries
		AnalysisMaxRetries:   getEnvInt("ANALYSIS_MAX_RETRIES", 2),     // 2 retries
		JobRetryInitialDelay: getEnvInt("JOB_RETRY_INITIAL_DELAY", 60), // 1 minute
		JobRetryMaxDelay:     getEnvInt("JOB_RETRY_MAX_DELAY", 3600),   // 1 hour
		DeadLetterQueueTTL:   getEnvInt("DLQ_TTL_HOURS", 168),          // 7 days

		// External Services
		// Note: Default Gotenberg URL assumes docker-compose service name "gotenberg"
		// For local dev without docker: use "http://localhost:3001" (avoid port 3000 conflict with frontend)
		GotenbergURL:       getEnv("GOTENBERG_URL", "http://gotenberg:3000"),
		SupabaseURL:        getEnv("SUPABASE_URL", ""),
		SupabaseAnonKey:    getEnv("SUPABASE_ANON_KEY", ""),
		SupabaseJWTSecret:  getEnv("SUPABASE_JWT_SECRET", ""),
		SupabaseServiceKey: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
		SupabaseBucket:     getEnv("SUPABASE_STORAGE_BUCKET", "reports-pdf"),
		SupabaseSignedTTL:  getEnvInt("SUPABASE_SIGNED_URL_TTL", 7*24*60*60), // default 7 days

		// Storage
		StorageBasePath: "./uploads",
		StorageBaseURL:  "http://localhost:8080/uploads",
		FrontendURL:     "http://localhost:3000",
	}

	// Setup logging AFTER environment is determined
	setupLogger(cfg.Environment)

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

			log.Info().
				Str("source", "REDIS_URL").
				Str("addr", addr).
				Bool("has_password", password != "").
				Msg("Redis configuration loaded")
			return addr, password
		}
		log.Warn().Err(err).Msg("Failed to parse REDIS_URL, falling back to REDIS_ADDR")
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

	log.Info().
		Str("source", "REDIS_ADDR").
		Str("addr", addr).
		Bool("has_password", password != "").
		Msg("Redis configuration loaded")
	return addr, password
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.OpenRouterAPIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	// CRITICAL SECURITY: Validate JWT secret strength
	if c.SupabaseJWTSecret == "" {
		return fmt.Errorf("SUPABASE_JWT_SECRET is required for JWT validation")
	}

	// Enforce minimum secret length (32 characters minimum for cryptographic security)
	if len(c.SupabaseJWTSecret) < 32 {
		return fmt.Errorf("SUPABASE_JWT_SECRET must be at least 32 characters long (current: %d chars)", len(c.SupabaseJWTSecret))
	}

	// Check for common default/example values that should never be used in production
	dangerousDefaults := []string{
		"super-secret", "secret", "example", "test", "changeme", "password",
		"jwt-secret", "your-secret-key", "default", "demo",
	}
	lowerSecret := strings.ToLower(c.SupabaseJWTSecret)
	for _, dangerous := range dangerousDefaults {
		if strings.Contains(lowerSecret, dangerous) {
			return fmt.Errorf("SUPABASE_JWT_SECRET appears to contain a default/example value (%s) - use a cryptographically random secret", dangerous)
		}
	}

	// In production, enforce additional security requirements
	if c.Environment == "production" {
		// Ensure ALLOWED_ORIGINS doesn't include localhost in production
		if strings.Contains(strings.ToLower(c.AllowedOrigins), "localhost") {
			return fmt.Errorf("ALLOWED_ORIGINS cannot contain 'localhost' in production environment")
		}

		// Ensure database uses SSL in production
		if !strings.Contains(c.DatabaseURL, "sslmode=require") &&
			!strings.Contains(c.DatabaseURL, "sslmode=verify-") {
			return fmt.Errorf("DATABASE_URL must use sslmode=require or stronger in production")
		}
	}

	if c.SupabaseAnonKey == "" {
		return fmt.Errorf("SUPABASE_ANON_KEY is required for auth API calls")
	}

	if c.SupabaseServiceKey == "" {
		return fmt.Errorf("SUPABASE_SERVICE_ROLE_KEY is required for storage uploads")
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

// loadFrameworkConfigs generates framework configurations from simplified 4-model approach
// All analysis frameworks use PrimaryModel, Synthesis uses SynthesisModel
func loadFrameworkConfigs() map[string]FrameworkConfig {
	configs := make(map[string]FrameworkConfig)

	// Load simplified 4-model configuration
	primaryModel := getEnv("AI_PRIMARY_MODEL", "google/gemini-2.5-flash")
	primaryFallback := getEnv("AI_PRIMARY_FALLBACK", "openai/gpt-4.1-mini")
	synthesisModel := getEnv("AI_SYNTHESIS_MODEL", "google/gemini-2.5-pro-preview")
	synthesisFallback := getEnv("AI_SYNTHESIS_FALLBACK", "openai/gpt-4.1")
	temperature := getEnvFloat("AI_TEMPERATURE", 0.5)
	tokensEnrichment := getEnvInt("AI_MAX_TOKENS_ENRICHMENT", 10000)
	tokensAnalysis := getEnvInt("AI_MAX_TOKENS_ANALYSIS", 4000)
	tokensSynthesis := getEnvInt("AI_MAX_TOKENS_SYNTHESIS", 6000)

	log.Info().
		Str("primary_model", primaryModel).
		Str("primary_fallback", primaryFallback).
		Str("synthesis_model", synthesisModel).
		Str("synthesis_fallback", synthesisFallback).
		Float64("temperature", temperature).
		Msg("Loading simplified 4-model AI configuration")

	// Enrichment Layer (Layer 0) - Discovery & Data Gathering
	configs["enrichment"] = FrameworkConfig{
		Model:         primaryModel,
		Temperature:   temperature,
		MaxTokens:     tokensEnrichment,
		FallbackModel: primaryFallback,
	}

	// All 11 Analysis Frameworks use Primary Model
	analysisFrameworks := []string{
		"pestel", "porter", "tam_sam_som",  // Layer 1: Environment
		"swot", "benchmarking",              // Layer 2: Positioning
		"blue_ocean", "growth_hacking", "scenarios", // Layer 3: Strategy
		"okrs", "bsc", "decision_matrix",    // Layer 4: Execution
	}

	for _, framework := range analysisFrameworks {
		configs[framework] = FrameworkConfig{
			Model:         primaryModel,
			Temperature:   temperature,
			MaxTokens:     tokensAnalysis,
			FallbackModel: primaryFallback,
		}
	}

	// Synthesis Layer - Final Executive Summary (Premium Model)
	configs["synthesis"] = FrameworkConfig{
		Model:         synthesisModel,
		Temperature:   temperature,
		MaxTokens:     tokensSynthesis,
		FallbackModel: synthesisFallback,
	}

	log.Info().Int("frameworks_loaded", len(configs)).Msg("Framework configurations loaded successfully")
	return configs
}

func setupLogger(environment string) {
	// Production: Use JSON logging for Railway/structured log parsing
	// Development: Use pretty console output with colors
	if environment == "production" {
		// JSON format for production - Railway can parse this correctly
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Caller().Logger()
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	} else {
		// Console format for development - easier to read locally
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Set global log level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if environment == "development" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}
