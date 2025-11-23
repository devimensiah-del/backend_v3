package enrichment_test

import (
	"backend_v3/adapter/scraper"
	"backend_v3/config"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"
	"backend_v3/llm"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// TEST CONFIGURATION - SET THESE TO CONTROL TEST BEHAVIOR
// ============================================================================

const (
	// Set to your OpenRouter API key to test with real AI, or leave empty to use mock
	OPENROUTER_API_KEY = "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"

	// Set to true to use real scraper, false to mock it
	USE_REAL_SCRAPER = true

	// Model to use for testing (using a valid model)
	TEST_MODEL = "google/gemini-2.0-flash-001"
)

// ============================================================================
// MOCK IMPLEMENTATIONS
// ============================================================================

// MockRepository implements enrichment.Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, e *enrichment.Enrichment) error {
	args := m.Called(ctx, e)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*enrichment.Enrichment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*enrichment.Enrichment), args.Error(1)
}

func (m *MockRepository) GetBySubmissionID(ctx context.Context, id uuid.UUID) (*enrichment.Enrichment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*enrichment.Enrichment), args.Error(1)
}

func (m *MockRepository) UpdateSystem(ctx context.Context, e *enrichment.Enrichment) error {
	args := m.Called(ctx, e)
	return args.Error(0)
}

func (m *MockRepository) UpdateUser(ctx context.Context, e *enrichment.Enrichment) error {
	args := m.Called(ctx, e)
	return args.Error(0)
}

// MockSubmissionRepo implements submission.Repository interface
type MockSubmissionRepo struct {
	mock.Mock
}

func (m *MockSubmissionRepo) Create(ctx context.Context, s *submission.Submission) error {
	return m.Called(ctx, s).Error(0)
}

func (m *MockSubmissionRepo) GetByID(ctx context.Context, id uuid.UUID) (*submission.Submission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*submission.Submission), args.Error(1)
}

func (m *MockSubmissionRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return m.Called(ctx, id, status).Error(0)
}

func (m *MockSubmissionRepo) List(ctx context.Context, opts *submission.ListOptions) ([]*submission.Submission, int, error) {
	return nil, 0, nil
}

func (m *MockSubmissionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockSubmissionRepo) GetActiveCount(ctx context.Context) (int, error) {
	return 0, nil
}

func (m *MockSubmissionRepo) GetTotalCount(ctx context.Context) (int, error) {
	return 0, nil
}

func (m *MockSubmissionRepo) GetByEmail(ctx context.Context, email string) ([]*submission.Submission, error) {
	return nil, nil
}

func (m *MockSubmissionRepo) Update(ctx context.Context, s *submission.Submission) error {
	return nil
}

func (m *MockSubmissionRepo) GetCompletedCount(ctx context.Context) (int, error) {
	return 0, nil
}

// MockScraper implements the scraper interface
type MockScraper struct {
	mock.Mock
}

func (m *MockScraper) Scrape(ctx context.Context, url string) (scraper.MetaData, error) {
	args := m.Called(ctx, url)
	return args.Get(0).(scraper.MetaData), args.Error(1)
}

// ============================================================================
// MOCK AI RESPONSE - Valid JSON matching UnifiedProfile structure
// ============================================================================

func getMockAIResponse() string {
	return `{
  "profile_overview": {
    "legal_name": "Test Corporation Ltd.",
    "website": "https://testcorp.com",
    "foundation_year": "2015",
    "headquarters": "São Paulo, Brazil"
  },
  "market_position": {
    "sector": "Technology / SaaS",
    "target_audience": "B2B Enterprise Clients",
    "value_proposition": "Cloud-based business intelligence platform"
  },
  "financials": {
    "employees_range": "50-100",
    "revenue_estimate": "$5M-$10M USD annually",
    "business_model": "Subscription-based SaaS"
  },
  "competitive_landscape": {
    "competitors": ["Tableau", "Power BI", "Looker"],
    "market_share_status": "Emerging player in regional market"
  },
  "strategic_assessment": {
    "digital_maturity": 7,
    "strengths": [
      "Strong technical team",
      "Modern tech stack",
      "Growing customer base"
    ],
    "weaknesses": [
      "Limited market presence",
      "Small sales team",
      "Needs more marketing investment"
    ]
  }
}`
}

// ============================================================================
// MAIN TEST FUNCTION
// ============================================================================

func TestEnrichmentWorkflow_Integration(t *testing.T) {
	ctx := context.Background()

	// Setup test data
	submissionID := uuid.New()
	companyWebsite := "https://www.coimma.com.br"

	sub := &submission.Submission{
		ID:                submissionID,
		CompanyName:       "Coimma",
		CompanyWebsite:    &companyWebsite,
		Status:            "received",
		BusinessChallenge: "Sou CEO da Coimma. Somos fabricantes de equipamentos de manejo para gado, balanças bovinas eletronica, balanças rodoviárias, automação de balancas rodoviárias e balanças de fluxo por batelada para grãos. Atualmente 70% do nosso faturamento é proveniente do mercado pecuária. Os outros 30% sao provenientes de segmentos diversos destacando-se grãos e armazenamento. A prestação de serviços de calibração e manutenção representa 5% do faturamento.Nossas competências são fabricação de equipamentos, sistemas de pesagem e eletrônica elabore um plano de de negócios para nossa companhia em duas fases. A primeira fase mais imediata visa expandir as nossas vendas com nosso portfólio atual de produtos e aproveitando nossa estrutura atual de vendas e assistência técnica. Na segunda fase queremos desenvolver novos produtos para o segmento industrial que requer investimentos em estrutura de vendas e assistência técnica.",
	}

	// ========================================================================
	// PHASE 1: Setup Mocks
	// ========================================================================

	enrichRepo := new(MockRepository)
	subRepo := new(MockSubmissionRepo)

	// Mock repository expectations
	subRepo.On("GetByID", mock.Anything, submissionID).Return(sub, nil)
	enrichRepo.On("GetBySubmissionID", mock.Anything, submissionID).Return(nil, fmt.Errorf("not found"))
	enrichRepo.On("Create", mock.Anything, mock.AnythingOfType("*enrichment.Enrichment")).Return(nil)
	enrichRepo.On("UpdateSystem", mock.Anything, mock.AnythingOfType("*enrichment.Enrichment")).Return(nil)

	// ========================================================================
	// PHASE 2: Setup LLM Client (Real or Mock)
	// ========================================================================

	var llmClient *llm.Client
	var mockAIServer *httptest.Server

	if OPENROUTER_API_KEY == "" {
		// No API key - use mock AI server
		t.Log("⚠️  No API key provided - Using MOCK AI")

		mockAIServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"content": getMockAIResponse(),
						},
					},
				},
				"usage": map[string]interface{}{"total_tokens": 1500},
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer mockAIServer.Close()

		llmClient = llm.NewClientWithBaseURL("mock-key", mockAIServer.URL)
	} else {
		// Real API key provided - use real OpenRouter
		t.Log("✅ API key provided - Using REAL OpenRouter AI")
		llmClient = llm.NewClient(OPENROUTER_API_KEY)
	}

	// ========================================================================
	// PHASE 3: Setup Scraper (Real or Mock)
	// ========================================================================

	var mockWebServer *httptest.Server

	if USE_REAL_SCRAPER {
		t.Log("✅ Using REAL scraper (will use actual website)")
	} else {
		t.Log("⚠️  Using MOCK scraper")

		// Create mock web server for scraper to hit
		mockWebServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			html := `<!DOCTYPE html>
<html lang="en">
<head>
    <title>Test Corporation - Business Intelligence Platform</title>
    <meta name="description" content="Leading cloud-based business intelligence platform for enterprise clients">
    <meta name="keywords" content="business intelligence, analytics, SaaS, enterprise">
    <meta property="og:image" content="https://testcorp.com/og-image.png">
</head>
<body><h1>Test Corporation</h1></body>
</html>`
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(html))
		}))
		defer mockWebServer.Close()

		// Update submission to point to mock server
		sub.CompanyWebsite = &mockWebServer.URL
		t.Logf("📝 Mock website URL: %s", mockWebServer.URL)
	}

	// ========================================================================
	// PHASE 4: Create Service and Execute Workflow
	// ========================================================================

	cfg := config.FrameworkConfig{
		Model:       TEST_MODEL,
		Temperature: 0.5,
		MaxTokens:   4000,
	}

	// Create asynq client for job orchestration (dummy Redis address for testing)
	// The test doesn't actually call Approve() so this won't be used
	queueClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr: "localhost:6379",
	})
	defer queueClient.Close()

	svc := enrichment.NewService(enrichRepo, subRepo, llmClient, queueClient, cfg)

	t.Log("🚀 Starting enrichment workflow...")
	startTime := time.Now()

	result, err := svc.EnrichSubmission(ctx, submissionID)

	duration := time.Since(startTime)
	t.Logf("⏱️  Workflow completed in %v", duration)

	// ========================================================================
	// PHASE 5: Assertions - Validate JSON Structure
	// ========================================================================

	assert.NoError(t, err, "Enrichment workflow should not return an error")
	assert.NotNil(t, result, "Result should not be nil")

	if err != nil {
		t.Fatalf("❌ Workflow failed: %v", err)
	}

	// Check enrichment status
	assert.Equal(t, enrichment.StatusFinished, result.Status, "Status should be 'finished'")
	assert.Equal(t, 100, result.Progress, "Progress should be 100%")

	// Validate the JSON structure matches what the database expects
	t.Log("📋 Validating JSON structure...")

	enrichedData := result.EnrichedData
	assert.NotNil(t, enrichedData, "EnrichedData should not be nil")

	// Convert to UnifiedProfile to validate structure
	jsonBytes, err := json.Marshal(enrichedData)
	assert.NoError(t, err, "Should be able to marshal enriched data")

	var profile enrichment.UnifiedProfile
	err = json.Unmarshal(jsonBytes, &profile)
	assert.NoError(t, err, "Should be able to unmarshal into UnifiedProfile structure")

	// Validate ProfileOverview
	assert.NotEmpty(t, profile.ProfileOverview.LegalName, "Legal name should not be empty")
	assert.NotEmpty(t, profile.ProfileOverview.Website, "Website should not be empty")
	assert.NotEmpty(t, profile.ProfileOverview.Headquarters, "Headquarters should not be empty")

	// Validate MarketPosition
	assert.NotEmpty(t, profile.MarketPosition.Sector, "Sector should not be empty")
	assert.NotEmpty(t, profile.MarketPosition.TargetAudience, "Target audience should not be empty")
	assert.NotEmpty(t, profile.MarketPosition.ValueProposition, "Value proposition should not be empty")

	// Validate Financials
	assert.NotEmpty(t, profile.Financials.EmployeesRange, "Employees range should not be empty")
	assert.NotEmpty(t, profile.Financials.RevenueEstimate, "Revenue estimate should not be empty")
	assert.NotEmpty(t, profile.Financials.BusinessModel, "Business model should not be empty")

	// Validate CompetitiveLandscape
	assert.NotEmpty(t, profile.CompetitiveLandscape.Competitors, "Competitors list should not be empty")
	assert.NotEmpty(t, profile.CompetitiveLandscape.MarketShareStatus, "Market share status should not be empty")

	// Validate StrategicAssessment
	assert.GreaterOrEqual(t, profile.StrategicAssessment.DigitalMaturity, 0, "Digital maturity should be >= 0")
	assert.LessOrEqual(t, profile.StrategicAssessment.DigitalMaturity, 10, "Digital maturity should be <= 10")
	assert.NotEmpty(t, profile.StrategicAssessment.Strengths, "Strengths should not be empty")
	assert.NotEmpty(t, profile.StrategicAssessment.Weaknesses, "Weaknesses should not be empty")

	// Pretty print the result
	prettyJSON, _ := json.MarshalIndent(profile, "", "  ")
	t.Logf("\n✅ VALIDATED JSON STRUCTURE:\n%s\n", string(prettyJSON))

	// Verify sources were captured
	assert.NotNil(t, result.SourcesStatus, "SourcesStatus should not be nil")
	t.Logf("📊 Sources used: %v", result.SourcesStatus)

	t.Log("✅ ALL TESTS PASSED - JSON structure matches database expectations")
}
