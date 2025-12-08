package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// FrameworkConfig holds model-specific configuration for each analysis framework.
// Each framework is assigned a primary model, temperature, token limit, and fallback model
// for automatic retry on rate limits or failures.
type FrameworkConfig struct {
	Model         string  // Primary model to use (e.g., "google/gemini-2.5-flash")
	Temperature   float64 // Sampling temperature (0.0 = deterministic, 1.0 = creative)
	MaxTokens     int     // Maximum tokens to generate
	FallbackModel string  // Fallback model if primary fails (rate limit, timeout, etc.)
}

// Config holds all application configuration loaded from environment variables.
// Required variables are marked with `required:"true"` and will cause Load() to fail if missing.
// Optional variables have defaults defined via `default:"value"` tags.
//
// Configuration groups:
//   - Server: Port, environment, CORS settings
//   - Database: PostgreSQL connection and pool settings
//   - AI/LLM: OpenRouter models for enrichment, analysis, and synthesis
//   - Redis/Jobs: Background job queue and retry configuration
//   - Supabase: Auth and storage integration
//   - Storage/Frontend: File uploads and frontend URL
type Config struct {
	// ==================== SERVER CONFIGURATION ====================
	// Port: HTTP server listen port (default: 8080)
	Port string `envconfig:"SERVER_PORT" default:"8080"`

	// Environment: "development" or "production" - affects logging format and validation
	Environment string `envconfig:"ENV" default:"development"`

	// AllowedOrigins: Comma-separated list of allowed CORS origins
	// Example: "http://localhost:3000,https://app.example.com"
	AllowedOrigins string `envconfig:"ALLOWED_ORIGINS" default:"http://localhost:3000"`

	// ==================== DATABASE CONFIGURATION ====================
	// DatabaseURL: PostgreSQL connection string (REQUIRED)
	// Format: postgres://user:password@host:port/dbname?sslmode=require
	// Production must use sslmode=require or stronger
	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`

	// Database Connection Pool Settings
	DBMaxOpenConns       int           `envconfig:"DB_MAX_OPEN_CONNS" default:"25"`         // Maximum open connections (default: 25)
	DBMaxIdleConns       int           `envconfig:"DB_MAX_IDLE_CONNS" default:"5"`          // Maximum idle connections (default: 5)
	DBConnMaxLifetimeMin int           `envconfig:"DB_CONN_MAX_LIFETIME_MINUTES" default:"5"` // Connection max lifetime in minutes (default: 5)
	DBConnMaxLifetime    time.Duration `ignored:"true"`                                     // Computed from DBConnMaxLifetimeMin

	// ==================== AI/LLM CONFIGURATION ====================
	// OpenRouterAPIKey: API key for OpenRouter (REQUIRED)
	// Format: sk-or-v1-...
	// Defined as OPENAI_API_KEY for OpenAI SDK compatibility
	OpenRouterAPIKey string `envconfig:"OPENAI_API_KEY" required:"true"`

	// PreSearch Model: Perplexity for company enrichment (inline at company creation)
	// Used for: Company identification, data gathering
	PreSearchModel    string `envconfig:"AI_PRESEARCH_MODEL" default:"perplexity/sonar-pro"`
	PreSearchFallback string `envconfig:"AI_PRESEARCH_FALLBACK" default:"perplexity/sonar"`

	// Primary Model: Used for ALL 11 analysis frameworks
	// Frameworks: PESTEL, Porter, TAM-SAM-SOM, SWOT, Benchmarking, Blue Ocean,
	//             Growth Hacking, Scenarios, OKRs, BSC, Decision Matrix
	PrimaryModel    string `envconfig:"AI_PRIMARY_MODEL" default:"google/gemini-2.5-flash"`
	PrimaryFallback string `envconfig:"AI_PRIMARY_FALLBACK" default:"openai/gpt-4.1-mini"`

	// Synthesis Model: Premium model for executive summary synthesis
	// Used for: Final executive summary combining all framework outputs
	SynthesisModel    string `envconfig:"AI_SYNTHESIS_MODEL" default:"google/gemini-2.5-pro-preview"`
	SynthesisFallback string `envconfig:"AI_SYNTHESIS_FALLBACK" default:"openai/gpt-4.1"`

	// Shared AI Settings
	AITemperature       float64 `envconfig:"AI_TEMPERATURE" default:"0.5"`                // Temperature for analysis (0.0-1.0, default: 0.5)
	MaxTokensEnrichment int     `envconfig:"AI_MAX_TOKENS_ENRICHMENT" default:"2000"`    // Max tokens for enrichment (default: 2000)
	MaxTokensAnalysis   int     `envconfig:"AI_MAX_TOKENS_ANALYSIS" default:"8000"`      // Max tokens for analysis (default: 8000)
	MaxTokensSynthesis  int     `envconfig:"AI_MAX_TOKENS_SYNTHESIS" default:"6000"`     // Max tokens for synthesis (default: 6000)

	// Framework-specific configurations (auto-generated from above settings)
	Frameworks map[string]FrameworkConfig `ignored:"true"`

	// ==================== REDIS/JOBS CONFIGURATION ====================
	// RedisURL: Parsed from REDIS_URL or REDIS_ADDR (see parseRedisConfig)
	RedisURL      string `ignored:"true"`
	RedisPassword string `ignored:"true"`

	// Worker Settings
	WorkerEnabled     bool   `envconfig:"WORKER_ENABLED" default:"true"`                       // Enable background workers (default: true)
	WorkerConcurrency int    `envconfig:"ASYNQ_CONCURRENCY" default:"10"`                      // Concurrent job workers (default: 10)
	WorkerQueues      string `envconfig:"WORKER_QUEUES" default:"critical:6,default:3,low:1"` // Queue priority weights

	// Job Timeout Settings (seconds)
	EnrichmentTimeout int `envconfig:"ENRICHMENT_TIMEOUT" default:"300"` // Enrichment job timeout (default: 300s = 5min)
	AnalysisTimeout   int `envconfig:"ANALYSIS_TIMEOUT" default:"900"`   // Analysis job timeout (default: 900s = 15min)

	// Job Retry Settings
	EnrichmentMaxRetries int `envconfig:"ENRICHMENT_MAX_RETRIES" default:"3"` // Enrichment max retries (default: 3)
	AnalysisMaxRetries   int `envconfig:"ANALYSIS_MAX_RETRIES" default:"2"`   // Analysis max retries (default: 2)
	JobRetryInitialDelay int `envconfig:"JOB_RETRY_INITIAL_DELAY" default:"60"`   // Initial retry delay in seconds (default: 60s)
	JobRetryMaxDelay     int `envconfig:"JOB_RETRY_MAX_DELAY" default:"3600"`     // Max retry delay in seconds (default: 3600s = 1h)
	DeadLetterQueueTTL   int `envconfig:"DLQ_TTL_HOURS" default:"168"`            // Dead letter queue TTL in hours (default: 168h = 7 days)

	// ==================== MACRO SCHEDULER ====================
	// MacroSchedulerEnabled: Enable/disable macro data cron jobs (SELIC, IPCA, USD/BRL)
	MacroSchedulerEnabled bool `envconfig:"MACRO_SCHEDULER_ENABLED" default:"true"`

	// ==================== SUPABASE INTEGRATION ====================
	// SupabaseURL: Supabase project URL (REQUIRED)
	// Format: https://your-project-id.supabase.co
	SupabaseURL string `envconfig:"SUPABASE_URL" required:"true"`

	// SupabaseAnonKey: Public anon key for Auth API calls (REQUIRED)
	// Found in: Supabase -> Settings -> API -> anon public
	SupabaseAnonKey string `envconfig:"SUPABASE_ANON_KEY" required:"true"`

	// SupabaseJWTSecret: JWT secret for validating user tokens (REQUIRED)
	// Found in: Supabase -> Settings -> API -> JWT Secret
	// MUST be at least 32 characters and cryptographically random
	SupabaseJWTSecret string `envconfig:"SUPABASE_JWT_SECRET" required:"true"`

	// SupabaseServiceKey: Service role key for storage uploads (REQUIRED)
	// Found in: Supabase -> Settings -> API -> service_role
	SupabaseServiceKey string `envconfig:"SUPABASE_SERVICE_ROLE_KEY" required:"true"`

	// SupabaseSignedTTL: Signed URL TTL for downloads in seconds (default: 604800 = 7 days)
	SupabaseSignedTTL int `envconfig:"SUPABASE_SIGNED_URL_TTL" default:"604800"`

	// ==================== STORAGE & FRONTEND ====================
	// StorageBasePath: Local file storage path (default: ./uploads)
	StorageBasePath string `envconfig:"STORAGE_BASE_PATH" default:"./uploads"`

	// StorageBaseURL: Public URL for uploaded files (default: http://localhost:8080/uploads)
	StorageBaseURL string `envconfig:"STORAGE_BASE_URL" default:"http://localhost:8080/uploads"`

	// FrontendURL: Frontend application URL for CORS and redirects (default: http://localhost:3000)
	FrontendURL string `envconfig:"FRONTEND_URL" default:"http://localhost:3000"`
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	_ = godotenv.Load() // Load .env if present

	cfg := &Config{}

	// Use envconfig to parse all environment variables
	if err := envconfig.Process("", cfg); err != nil {
		return nil, fmt.Errorf("failed to process environment config: %w", err)
	}

	// Compute DBConnMaxLifetime from minutes
	cfg.DBConnMaxLifetime = time.Duration(cfg.DBConnMaxLifetimeMin) * time.Minute

	// Setup logging AFTER environment is determined
	setupLogger(cfg.Environment)

	// Parse Redis configuration
	cfg.RedisURL, cfg.RedisPassword = parseRedisConfig()
	log.Info().
		Str("redis_addr", cfg.RedisURL).
		Bool("has_password", cfg.RedisPassword != "").
		Msg("📡 Redis configuration parsed for worker and API")

	// Load framework-specific configurations using struct fields (no env var re-reads)
	cfg.Frameworks = cfg.loadFrameworkConfigs()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// parseRedisConfig parses Redis connection from REDIS_URL (Railway format)
// or falls back to REDIS_ADDR (local development)
func parseRedisConfig() (addr string, password string) {
	// Try Railway's REDIS_URL first (format: redis://default:password@host:port or rediss:// for TLS)
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		parsed, err := url.Parse(redisURL)
		if err == nil {
			// Extract host:port
			addr = parsed.Host

			// Extract password from URL
			if parsed.User != nil {
				password, _ = parsed.User.Password()
			}

			// Check if TLS is required (rediss:// scheme)
			useTLS := parsed.Scheme == "rediss"

			log.Info().
				Str("source", "REDIS_URL").
				Str("addr", addr).
				Bool("has_password", password != "").
				Bool("use_tls", useTLS).
				Str("scheme", parsed.Scheme).
				Msg("Redis configuration loaded")
			return addr, password
		}
		log.Warn().Err(err).Msg("Failed to parse REDIS_URL, falling back to REDIS_ADDR")
	}

	// Fall back to REDIS_ADDR (local development)
	addr = os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	password = os.Getenv("REDIS_PASSWORD")

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

// Validate performs comprehensive validation of all configuration values.
// It checks for required fields, validates ranges, and enforces security requirements.
func (c *Config) Validate() error {
	// ==================== SERVER VALIDATION ====================
	if c.Port == "" {
		return fmt.Errorf("SERVER_PORT cannot be empty")
	}
	// Validate port is numeric and in valid range
	if port, err := strconv.Atoi(c.Port); err != nil {
		return fmt.Errorf("SERVER_PORT must be a valid number: %w", err)
	} else if port < 1 || port > 65535 {
		return fmt.Errorf("SERVER_PORT must be between 1 and 65535 (got: %d)", port)
	}

	if c.Environment != "development" && c.Environment != "production" {
		return fmt.Errorf("ENV must be 'development' or 'production' (got: %s)", c.Environment)
	}

	// ==================== DATABASE VALIDATION ====================
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	// Validate connection pool settings
	if c.DBMaxOpenConns < 1 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be at least 1 (got: %d)", c.DBMaxOpenConns)
	}
	if c.DBMaxIdleConns < 0 {
		return fmt.Errorf("DB_MAX_IDLE_CONNS cannot be negative (got: %d)", c.DBMaxIdleConns)
	}
	if c.DBMaxIdleConns > c.DBMaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS (%d) cannot exceed DB_MAX_OPEN_CONNS (%d)", c.DBMaxIdleConns, c.DBMaxOpenConns)
	}
	if c.DBConnMaxLifetimeMin < 1 {
		return fmt.Errorf("DB_CONN_MAX_LIFETIME_MINUTES must be at least 1 (got: %d)", c.DBConnMaxLifetimeMin)
	}

	// ==================== AI/LLM VALIDATION ====================
	if c.OpenRouterAPIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}
	if !strings.HasPrefix(c.OpenRouterAPIKey, "sk-or-") {
		log.Warn().Msg("OPENAI_API_KEY does not start with 'sk-or-' - ensure you're using an OpenRouter key")
	}

	// Validate AI model names are non-empty
	if c.PreSearchModel == "" {
		return fmt.Errorf("AI_PRESEARCH_MODEL cannot be empty")
	}
	if c.PreSearchFallback == "" {
		return fmt.Errorf("AI_PRESEARCH_FALLBACK cannot be empty")
	}
	if c.PrimaryModel == "" {
		return fmt.Errorf("AI_PRIMARY_MODEL cannot be empty")
	}
	if c.PrimaryFallback == "" {
		return fmt.Errorf("AI_PRIMARY_FALLBACK cannot be empty")
	}
	if c.SynthesisModel == "" {
		return fmt.Errorf("AI_SYNTHESIS_MODEL cannot be empty")
	}
	if c.SynthesisFallback == "" {
		return fmt.Errorf("AI_SYNTHESIS_FALLBACK cannot be empty")
	}

	// Validate temperature range
	if c.AITemperature < 0.0 || c.AITemperature > 2.0 {
		return fmt.Errorf("AI_TEMPERATURE must be between 0.0 and 2.0 (got: %.2f)", c.AITemperature)
	}

	// Validate max tokens are positive and reasonable
	if c.MaxTokensEnrichment < 100 || c.MaxTokensEnrichment > 100000 {
		return fmt.Errorf("AI_MAX_TOKENS_ENRICHMENT must be between 100 and 100000 (got: %d)", c.MaxTokensEnrichment)
	}
	if c.MaxTokensAnalysis < 100 || c.MaxTokensAnalysis > 100000 {
		return fmt.Errorf("AI_MAX_TOKENS_ANALYSIS must be between 100 and 100000 (got: %d)", c.MaxTokensAnalysis)
	}
	if c.MaxTokensSynthesis < 100 || c.MaxTokensSynthesis > 100000 {
		return fmt.Errorf("AI_MAX_TOKENS_SYNTHESIS must be between 100 and 100000 (got: %d)", c.MaxTokensSynthesis)
	}

	// ==================== WORKER/JOBS VALIDATION ====================
	if c.WorkerConcurrency < 1 || c.WorkerConcurrency > 100 {
		return fmt.Errorf("ASYNQ_CONCURRENCY must be between 1 and 100 (got: %d)", c.WorkerConcurrency)
	}

	// Validate timeout values (must be positive and less than 1 hour)
	if c.EnrichmentTimeout < 10 || c.EnrichmentTimeout > 3600 {
		return fmt.Errorf("ENRICHMENT_TIMEOUT must be between 10 and 3600 seconds (got: %d)", c.EnrichmentTimeout)
	}
	if c.AnalysisTimeout < 10 || c.AnalysisTimeout > 3600 {
		return fmt.Errorf("ANALYSIS_TIMEOUT must be between 10 and 3600 seconds (got: %d)", c.AnalysisTimeout)
	}

	// Validate retry settings
	if c.EnrichmentMaxRetries < 0 || c.EnrichmentMaxRetries > 10 {
		return fmt.Errorf("ENRICHMENT_MAX_RETRIES must be between 0 and 10 (got: %d)", c.EnrichmentMaxRetries)
	}
	if c.AnalysisMaxRetries < 0 || c.AnalysisMaxRetries > 10 {
		return fmt.Errorf("ANALYSIS_MAX_RETRIES must be between 0 and 10 (got: %d)", c.AnalysisMaxRetries)
	}
	if c.JobRetryInitialDelay < 1 || c.JobRetryInitialDelay > 3600 {
		return fmt.Errorf("JOB_RETRY_INITIAL_DELAY must be between 1 and 3600 seconds (got: %d)", c.JobRetryInitialDelay)
	}
	if c.JobRetryMaxDelay < c.JobRetryInitialDelay {
		return fmt.Errorf("JOB_RETRY_MAX_DELAY (%d) must be >= JOB_RETRY_INITIAL_DELAY (%d)", c.JobRetryMaxDelay, c.JobRetryInitialDelay)
	}
	if c.DeadLetterQueueTTL < 1 {
		return fmt.Errorf("DLQ_TTL_HOURS must be at least 1 (got: %d)", c.DeadLetterQueueTTL)
	}

	// ==================== SUPABASE VALIDATION ====================
	if c.SupabaseURL == "" {
		return fmt.Errorf("SUPABASE_URL is required")
	}
	if c.SupabaseAnonKey == "" {
		return fmt.Errorf("SUPABASE_ANON_KEY is required for auth API calls")
	}
	if c.SupabaseServiceKey == "" {
		return fmt.Errorf("SUPABASE_SERVICE_ROLE_KEY is required for storage uploads")
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

	// Validate Supabase signed URL TTL
	if c.SupabaseSignedTTL < 60 {
		return fmt.Errorf("SUPABASE_SIGNED_URL_TTL must be at least 60 seconds (got: %d)", c.SupabaseSignedTTL)
	}

	// ==================== PRODUCTION-SPECIFIC VALIDATION ====================
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

	return nil
}

// loadFrameworkConfigs generates framework configurations using the simplified 3-model approach:
// 1. PreSearch (Perplexity) - Company enrichment, inline at company creation
// 2. Primary - All 11 analysis frameworks
// 3. Synthesis - Executive summary (premium model)
//
// This method uses values already parsed into the Config struct, avoiding redundant env var reads.
func (c *Config) loadFrameworkConfigs() map[string]FrameworkConfig {
	configs := make(map[string]FrameworkConfig)

	log.Info().
		Str("presearch", c.PreSearchModel).
		Str("primary", c.PrimaryModel).
		Str("synthesis", c.SynthesisModel).
		Msg("Loading 3-model AI configuration")

	// PreSearch/Enrichment - Perplexity for company data gathering (inline)
	configs["presearch"] = FrameworkConfig{
		Model:         c.PreSearchModel,
		Temperature:   0.3, // Low temperature for consistent enrichment
		MaxTokens:     c.MaxTokensEnrichment,
		FallbackModel: c.PreSearchFallback,
	}

	// Backward compatibility: enrichment config points to presearch
	configs["enrichment"] = configs["presearch"]

	// Primary config - generic fallback for any framework not explicitly configured
	// Also used as explicit lookup key in wizard service
	configs["primary"] = FrameworkConfig{
		Model:         c.PrimaryModel,
		Temperature:   c.AITemperature,
		MaxTokens:     c.MaxTokensAnalysis,
		FallbackModel: c.PrimaryFallback,
	}

	// All 12 Analysis Frameworks use Primary Model (including wizard's challenge_refinement)
	analysisFrameworks := []string{
		"challenge_refinement", // Wizard step 0 - refine the business challenge
		"pestel", "porter", "tam_sam_som",
		"swot", "benchmarking",
		"blue_ocean", "growth_hacking", "scenarios",
		"okrs", "bsc", "decision_matrix",
	}
	for _, framework := range analysisFrameworks {
		configs[framework] = FrameworkConfig{
			Model:         c.PrimaryModel,
			Temperature:   c.AITemperature,
			MaxTokens:     c.MaxTokensAnalysis,
			FallbackModel: c.PrimaryFallback,
		}
	}

	// Synthesis - Premium model for executive summary
	configs["synthesis"] = FrameworkConfig{
		Model:         c.SynthesisModel,
		Temperature:   c.AITemperature,
		MaxTokens:     c.MaxTokensSynthesis,
		FallbackModel: c.SynthesisFallback,
	}

	log.Info().Int("frameworks_loaded", len(configs)).Msg("Framework configurations loaded")
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
