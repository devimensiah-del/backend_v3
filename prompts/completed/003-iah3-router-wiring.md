<objective>
Wire the AnalysisBySteps handlers into the router and update handler composition. This completes the HTTP API implementation for IAH-3.

This is Part 3 of IAH-3: After implementing service (001) and handlers (002), wire everything together in the router.
</objective>

<context>
Previous prompts created:
- `domain/analysisbysteps/service.go` - Full service with 5 methods
- `domain/analysisbysteps/types.go` - Response types
- `api/analysisbysteps_handlers.go` - 6 HTTP handlers

Now we need to:
1. Update router.go to instantiate AnalysisByStepsHandlers
2. Add AnalysisByStepsHandlers to main Handler struct
3. Register routes in the protected API group
4. Pass required dependencies through the composition chain

Existing patterns in router.go (lines 50-120):
- Handler structs are instantiated with their dependencies
- Handlers are composed into main Handler struct
- Routes are registered in groups (public, protected, admin)
</context>

<research>
Read these files:
- `@api/router.go` - Full router setup
- `@api/handlers.go` - Main Handler struct
- `@main.go` - Application bootstrap and service instantiation
</research>

<requirements>
## 1. Update api/handlers.go

Add AnalysisByStepsHandlers to the main Handler struct:

```go
type Handler struct {
    // ... existing fields ...

    // Add this field
    AnalysisByStepsHandlers *AnalysisByStepsHandlers
}
```

Update NewHandler to accept AnalysisByStepsHandlers (add after analysisHandlers, before authHandlers):

```go
func NewHandler(
    adminHandlers *AdminHandlers,
    analysisHandlers *AnalysisHandlers,
    analysisByStepsHandlers *AnalysisByStepsHandlers,  // ADD THIS (after analysisHandlers)
    authHandlers *AuthHandlers,
    companyHandlers *CompanyHandlers,
    submissionHandlers *SubmissionHandlers,
    userHandlers *UserHandlers,
    submissionResponseBuilder *SubmissionResponseBuilder,
    db *sqlx.DB,
    redisClient *redis.Client,
    logger zerolog.Logger,
    supabaseURL string,
    supabaseAnonKey string,
    supabaseJWTSecret string,
) *Handler {
    return &Handler{
        db:                        db,
        redisClient:               redisClient,
        logger:                    logger.With().Str("component", "api").Logger(),
        supabaseURL:               supabaseURL,
        supabaseAnonKey:           supabaseAnonKey,
        supabaseJWTSecret:         supabaseJWTSecret,
        AdminHandlers:             adminHandlers,
        AnalysisHandlers:          analysisHandlers,
        AnalysisByStepsHandlers:   analysisByStepsHandlers,  // ADD THIS
        AuthHandlers:              authHandlers,
        CompanyHandlers:           companyHandlers,
        SubmissionHandlers:        submissionHandlers,
        UserHandlers:              userHandlers,
        SubmissionResponseBuilder: submissionResponseBuilder,
    }
}
```

## 2. Update api/router.go

### Add Import

```go
import (
    // ... existing imports ...
    domainanalysisbysteps "backend_v3/domain/analysisbysteps"
)
```

### Update SetupRouter Signature

Add the analysisbysteps service to the parameter list:

```go
func SetupRouter(
    // ... existing params ...
    analysisByStepsSvc *domainanalysisbysteps.Service,  // ADD THIS
) *gin.Engine
```

### Instantiate Handler

After existing handler instantiations (around line 94), add:

```go
// Analysis by Steps handlers (IAH-3)
analysisByStepsHandlers := NewAnalysisByStepsHandlers(
    analysisByStepsSvc,
    logger,
)
```

### Update NewHandler Call

Add the new handlers to the composition (insert after analysisHandlers, before authHandlers):

```go
mainHandler := NewHandler(
    adminHandlers,
    analysisHandlers,
    analysisByStepsHandlers,  // ADD THIS (after analysisHandlers)
    authHandlers,
    companyHandlers,
    submissionHandlers,
    userHandlers,
    submissionResponseBuilder,
    db,
    redisClient,
    logger,
    supabaseURL,
    supabaseAnonKey,
    supabaseJWTSecret,
)
```

**IMPORTANT**: The parameter order must match exactly. Insert `analysisByStepsHandlers` between `analysisHandlers` and `authHandlers`.

### Register Routes

Add these routes in the protectedAPI group (after line 203, before the closing brace):

```go
// Analysis by Steps (IAH-3) - Human-in-the-loop step-by-step analysis
// These endpoints enable controlled analysis flow with human editing
analysisSteps := protectedAPI.Group("/analyses")
{
    // Start new step-by-step analysis
    analysisSteps.POST("/steps/start", mainHandler.AnalysisByStepsHandlers.StartAnalysisBySteps)

    // Step operations (requires analysis ID and step number)
    analysisSteps.GET("/:id/steps", mainHandler.AnalysisByStepsHandlers.GetAnalysisSteps)
    analysisSteps.GET("/:id/steps/state", mainHandler.AnalysisByStepsHandlers.GetStepState)
    analysisSteps.POST("/:id/steps/:step/generate", mainHandler.AnalysisByStepsHandlers.GenerateStep)
    analysisSteps.PUT("/:id/steps/:step/edit", mainHandler.AnalysisByStepsHandlers.SaveHumanEdit)
    analysisSteps.POST("/:id/steps/:step/approve", mainHandler.AnalysisByStepsHandlers.ApproveAndAdvance)
}
```

## 3. Update main.go (if needed)

The main.go file creates services and calls SetupRouter. You'll need to:

1. Instantiate the AnalysisBySteps service with its dependencies
2. Pass it to SetupRouter

Find where services are created and add:

```go
// Analysis by Steps service (IAH-3)
analysisByStepsRepo := analysisbysteps.NewRepository(db)
analysisByStepsSvc := analysisbysteps.NewService(
    analysisByStepsRepo,
    analysisRepo,      // analysis.Repository interface - use the concrete repo that implements it
    companySvc,        // *company.Service for GetByID
    challengeRepo,     // challenge.Repository interface
    llmClient,         // *llm.Client for AI generation
    cfg.Frameworks,    // map[string]config.FrameworkConfig for model routing
    logger,
)
```

**Note**: Check what variables already exist in main.go for:
- `analysisRepo` - likely a `*analysis.SQLRepository` which implements `analysis.Repository`
- `companySvc` - likely a `*company.Service`
- `challengeRepo` - likely implements `challenge.Repository`
- `llmClient` - `*llm.Client`
- `cfg.Frameworks` - framework configs from `config.Load()`

Then update the SetupRouter call to include `analysisByStepsSvc`.
</requirements>

<implementation>
### Route Group Structure

The routes follow RESTful conventions:
- `/analyses/steps/start` - Create new (POST to collection)
- `/analyses/:id/steps` - List steps for analysis
- `/analyses/:id/steps/state` - Get current state
- `/analyses/:id/steps/:step/generate` - Generate specific step
- `/analyses/:id/steps/:step/edit` - Edit specific step
- `/analyses/:id/steps/:step/approve` - Approve specific step

### Middleware Stack

All routes inherit from protectedAPI group middleware:
- `AuthMiddleware(jwtSecret, db)` - JWT validation

### TODO Comment Removal

Remove the TODO comment that was in router.go line 205:
```go
// TODO: IAH-3 - Add analysisbysteps API handlers for human editing
```
</implementation>

<output>
Modify these files:

1. `api/handlers.go` - Add AnalysisByStepsHandlers field and update constructor
2. `api/router.go` - Import, instantiate handler, update NewHandler call, register routes
3. `main.go` - Create service and pass to SetupRouter (if main.go handles DI)

Note: If dependency injection is handled elsewhere (e.g., wire), locate that file instead.
</output>

<verification>
Before completing, verify:
- [ ] Handler struct has AnalysisByStepsHandlers field
- [ ] NewHandler accepts new parameter
- [ ] SetupRouter accepts service parameter
- [ ] All 6 routes are registered in correct group
- [ ] Routes follow RESTful naming conventions
- [ ] TODO comment removed from router.go
- [ ] Application compiles: `go build ./...`
- [ ] Application starts without errors: `go run main.go` (then Ctrl+C)
</verification>

<success_criteria>
- All routes accessible via HTTP
- Protected routes require JWT authentication
- Route paths match specification from Jira
- Application starts and endpoints respond
- No compilation errors or runtime panics
</success_criteria>
