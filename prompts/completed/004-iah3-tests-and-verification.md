<objective>
Write unit tests for the AnalysisBySteps service and handlers, then verify the complete IAH-3 implementation works end-to-end.

This is Part 4 of IAH-3: After implementing service (001), handlers (002), and router (003), write tests and verify the full flow.
</objective>

<context>
IAH-3 implementation complete in previous prompts:
- Service layer with 5 methods
- 6 HTTP handlers
- Routes wired in router.go

Testing patterns in the codebase:
- `*_test.go` files alongside source code
- `github.com/DATA-DOG/go-sqlmock` for DB mocking
- `github.com/stretchr/testify/assert` for assertions
- `github.com/stretchr/testify/mock` for service mocking
- Contract tests in `*_contract_test.go` files
</context>

<research>
Read these test files for patterns:
- `@domain/submission/service_test.go` - Service testing patterns
- `@api/auth_handlers_test.go` - Handler testing with gin
- `@api/submission_handlers_contract_test.go` - Contract testing patterns
</research>

<requirements>
## 1. Service Unit Tests

Create `domain/analysisbysteps/service_test.go`:

### Test Cases

```go
// TestService_StartAnalysisBySteps tests analysis creation
func TestService_StartAnalysisBySteps_Success(t *testing.T)
func TestService_StartAnalysisBySteps_InvalidChallenge(t *testing.T)
func TestService_StartAnalysisBySteps_AlreadyExists(t *testing.T)

// TestService_GenerateStep tests AI generation
func TestService_GenerateStep_Success(t *testing.T)
func TestService_GenerateStep_InvalidStepNumber(t *testing.T)
func TestService_GenerateStep_PreviousNotApproved(t *testing.T)
func TestService_GenerateStep_PreservesHumanEdited(t *testing.T)

// TestService_SaveHumanEdit tests human editing
func TestService_SaveHumanEdit_Success(t *testing.T)
func TestService_SaveHumanEdit_StepNotFound(t *testing.T)

// TestService_ApproveAndAdvance tests approval flow
func TestService_ApproveAndAdvance_Success(t *testing.T)
func TestService_ApproveAndAdvance_NoContent(t *testing.T)
func TestService_ApproveAndAdvance_LastStep(t *testing.T)

// TestService_GetStepState tests state retrieval
func TestService_GetStepState_Success(t *testing.T)
func TestService_GetStepState_AnalysisNotFound(t *testing.T)
```

### Mock Setup Pattern

```go
type mockDependencies struct {
    repo          *MockRepository
    analysisRepo  *MockAnalysisRepository
    companyService *MockCompanyService
    challengeRepo *MockChallengeRepository
    llmClient     *MockLLMClient
}

func setupServiceMocks(t *testing.T) (*Service, *mockDependencies) {
    mocks := &mockDependencies{
        repo:          new(MockRepository),
        analysisRepo:  new(MockAnalysisRepository),
        companyService: new(MockCompanyService),
        challengeRepo: new(MockChallengeRepository),
        llmClient:     new(MockLLMClient),
    }

    svc := NewService(
        mocks.repo,
        mocks.analysisRepo,
        mocks.companyService,
        mocks.challengeRepo,
        mocks.llmClient,
        map[string]config.FrameworkConfig{},
        zerolog.Nop(),
    )

    return svc, mocks
}
```

## 2. Handler Tests

Create `api/analysisbysteps_handlers_test.go`:

### Test Cases

```go
// Test route handlers with gin test context
func TestHandler_StartAnalysisBySteps_Success(t *testing.T)
func TestHandler_StartAnalysisBySteps_InvalidRequest(t *testing.T)
func TestHandler_StartAnalysisBySteps_Unauthorized(t *testing.T)

func TestHandler_GenerateStep_Success(t *testing.T)
func TestHandler_GenerateStep_InvalidStepNumber(t *testing.T)

func TestHandler_SaveHumanEdit_Success(t *testing.T)
func TestHandler_SaveHumanEdit_EmptyContent(t *testing.T)

func TestHandler_ApproveAndAdvance_Success(t *testing.T)
func TestHandler_GetStepState_Success(t *testing.T)
func TestHandler_GetAnalysisSteps_Success(t *testing.T)
```

### Gin Test Setup Pattern

```go
func setupTestRouter() (*gin.Engine, *MockAnalysisByStepsService) {
    gin.SetMode(gin.TestMode)

    mockSvc := new(MockAnalysisByStepsService)
    handlers := NewAnalysisByStepsHandlers(mockSvc, zerolog.Nop())

    router := gin.New()

    // Add routes
    protected := router.Group("/api/v1")
    // Simulate auth middleware
    protected.Use(func(c *gin.Context) {
        c.Set("userID", "test-user-id")
        c.Next()
    })

    analysisSteps := protected.Group("/analyses")
    analysisSteps.POST("/steps/start", handlers.StartAnalysisBySteps)
    analysisSteps.GET("/:id/steps", handlers.GetAnalysisSteps)
    analysisSteps.GET("/:id/steps/state", handlers.GetStepState)
    analysisSteps.POST("/:id/steps/:step/generate", handlers.GenerateStep)
    analysisSteps.PUT("/:id/steps/:step/edit", handlers.SaveHumanEdit)
    analysisSteps.POST("/:id/steps/:step/approve", handlers.ApproveAndAdvance)

    return router, mockSvc
}

func TestHandler_StartAnalysisBySteps_Success(t *testing.T) {
    router, mockSvc := setupTestRouter()

    // Setup mock expectation
    mockSvc.On("StartAnalysisBySteps", mock.Anything, mock.Anything).
        Return(&analysisbysteps.StartResponse{
            AnalysisID: "test-analysis-id",
            TotalSteps: 14,
        }, nil)

    // Create request
    body := `{"challenge_id":"550e8400-e29b-41d4-a716-446655440000"}`
    req, _ := http.NewRequest("POST", "/api/v1/analyses/steps/start", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    // Execute
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    // Assert
    assert.Equal(t, http.StatusOK, w.Code)
    mockSvc.AssertExpectations(t)
}
```

## 3. Create Mock Interfaces

Create `domain/analysisbysteps/mocks_test.go`:

```go
package analysisbysteps

import (
    "context"
    "github.com/stretchr/testify/mock"
)

type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, step *AnalysisStep) error {
    args := m.Called(ctx, step)
    return args.Error(0)
}

// ... implement all Repository interface methods

type MockLLMClient struct {
    mock.Mock
}

func (m *MockLLMClient) GenerateStructuredWithOptions(ctx context.Context, opts llm.GenerationOptions, prompt string, data interface{}, target interface{}) error {
    args := m.Called(ctx, opts, prompt, data, target)
    return args.Error(0)
}
```

## 4. Run Tests and Verify

```bash
# Run unit tests for the domain
npm run test:unit -- -run "TestAnalysisBySteps" -v

# Run all tests
npm run test:all

# Check test coverage
npm run test:coverage

# Verify build
go build ./...

# Run server and test endpoints manually
go run main.go &
curl -X GET http://localhost:8080/health
```

## 5. Manual End-to-End Verification

Use curl or Postman to test the complete flow:

```bash
# 1. Login to get JWT token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}' | jq -r '.access_token')

# 2. Start analysis
curl -X POST http://localhost:8080/api/v1/analyses/steps/start \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"challenge_id":"<your-challenge-id>"}'

# 3. Get step state
curl -X GET "http://localhost:8080/api/v1/analyses/<analysis-id>/steps/state" \
  -H "Authorization: Bearer $TOKEN"

# 4. Generate first step
curl -X POST "http://localhost:8080/api/v1/analyses/<analysis-id>/steps/0/generate" \
  -H "Authorization: Bearer $TOKEN"

# 5. Edit step
curl -X PUT "http://localhost:8080/api/v1/analyses/<analysis-id>/steps/0/edit" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"edited_content":"{\"refined_challenge\":\"Updated challenge\"}"}'

# 6. Approve and advance
curl -X POST "http://localhost:8080/api/v1/analyses/<analysis-id>/steps/0/approve" \
  -H "Authorization: Bearer $TOKEN"
```
</requirements>

<output>
Create these test files:

1. `domain/analysisbysteps/service_test.go` - Service unit tests
2. `domain/analysisbysteps/mocks_test.go` - Mock implementations
3. `api/analysisbysteps_handlers_test.go` - Handler tests

Run verification:
```bash
npm run test:unit -- -run "TestAnalysisBySteps" -v
go build ./...
```
</output>

<verification>
Before completing, verify:
- [ ] All service methods have tests (happy path + error cases)
- [ ] All handlers have tests (valid + invalid input)
- [ ] Mocks properly implement interfaces
- [ ] Tests pass: `npm run test:unit -- -run "TestAnalysisBySteps"`
- [ ] No data races: `go test -race ./domain/analysisbysteps/...`
- [ ] Coverage is reasonable (>70% for new code)
- [ ] Manual E2E test shows full flow works
</verification>

<success_criteria>
- All tests pass
- Test coverage >70% for new code
- No data races detected
- Manual flow (start → generate → edit → approve) works end-to-end
- All endpoints return 200 on happy path
- Error responses are appropriate (400, 404, 500)
</success_criteria>
