<objective>
Implement Challenge-Centric Analysis improvements for IMENSIAH strategic analysis platform.

This addresses the cofounder feedback:
- "Análise ficou desconectada do desafio" (Analysis disconnected from challenge)
- "Tem desafios que o sistema não consegue ajudar" (Some challenges the system can't help with)

The goal is to:
1. Remove challenge types that our 11 frameworks cannot meaningfully analyze
2. Inject challenge context into ALL framework prompts so analysis is challenge-focused
3. Make the Synthesis explicitly address the user's stated challenge
</objective>

<context>
Read @CLAUDE.md and @backend_v3/CLAUDE.md for project conventions.

Current architecture problem:
- Challenge entity exists with category, type, and free-text (domain/challenge/model.go)
- Analysis workflow validates challengeID but NEVER loads the Challenge data
- ContextContainer has SubmissionData and CompanyData but NO ChallengeData
- Framework prompts don't receive challenge context
- Synthesis tries to GUESS the "central challenge" instead of knowing it

This is a FULL STACK change affecting both backend and frontend.
</context>

<challenge_types_to_remove>
Based on cofounder feedback, remove these challenge types that our 11 strategic frameworks cannot meaningfully analyze:

REMOVE ENTIRE CATEGORIES (all types gone):
- **transition** (entire category) - All types require specialized expertise we don't have
- **funding** (entire category) - All types require financial modeling expertise

SPECIFIC TYPES TO REMOVE:
- `transform_culture` - Requires organizational psychology, not market analysis
- `transform_operational` - Internal operations focus, not strategic frameworks

WHAT REMAINS (10 types across 3 categories):
- **growth** (5 types): growth_organic, growth_geographic, growth_segment, growth_product, growth_channel
- **transform** (2 types): transform_digital, transform_model
- **compete** (3 types): compete_differentiate, compete_defend, compete_reposition
</challenge_types_to_remove>

<requirements>
CRITICAL: Do NOT break existing functionality. All changes must be additive where possible.

================================================================================
PART 1: BACKEND - Remove Challenge Types (domain/challenge/model.go)
================================================================================

In `domain/challenge/model.go`:

1. Remove category constants:
```go
// REMOVE these lines (keep only Growth, Transform, Compete)
CategoryTransition ChallengeCategory = "transition"  // DELETE
CategoryFunding    ChallengeCategory = "funding"     // DELETE
```

2. Remove type constants:
```go
// DELETE entire Transform Culture and Operational types:
TypeTransformCulture     ChallengeType = "transform_culture"     // DELETE
TypeTransformOperational ChallengeType = "transform_operational" // DELETE

// DELETE entire Transition section:
TypeTransitionSuccession ChallengeType = "transition_succession" // DELETE
TypeTransitionExit       ChallengeType = "transition_exit"       // DELETE
TypeTransitionMerger     ChallengeType = "transition_merger"     // DELETE
TypeTransitionTurnaround ChallengeType = "transition_turnaround" // DELETE

// DELETE entire Funding section:
TypeFundingRaise ChallengeType = "funding_raise" // DELETE
TypeFundingDebt  ChallengeType = "funding_debt"  // DELETE
TypeFundingIPO   ChallengeType = "funding_ipo"   // DELETE
```

3. Update ValidCategories:
```go
var ValidCategories = []ChallengeCategory{
    CategoryGrowth, CategoryTransform, CategoryCompete,
    // REMOVED: CategoryTransition, CategoryFunding
}
```

4. Update ValidTypesByCategory:
```go
var ValidTypesByCategory = map[ChallengeCategory][]ChallengeType{
    CategoryGrowth:    {TypeGrowthOrganic, TypeGrowthGeographic, TypeGrowthSegment, TypeGrowthProduct, TypeGrowthChannel},
    CategoryTransform: {TypeTransformDigital, TypeTransformModel},
    // REMOVED: TypeTransformCulture, TypeTransformOperational
    CategoryCompete:   {TypeCompeteDifferentiate, TypeCompeteDefend, TypeCompeteReposition},
    // REMOVED: CategoryTransition entirely
    // REMOVED: CategoryFunding entirely
}
```

================================================================================
PART 2: BACKEND - Add ChallengeData to ContextContainer (domain/analysis/model.go)
================================================================================

In `domain/analysis/model.go`, find the ContextContainer struct (~line 400) and add:

```go
type ContextContainer struct {
    SubmissionData map[string]interface{}
    CompanyData    map[string]interface{}
    ChallengeData  map[string]interface{} // NEW: Challenge context for framework prompts
    // ... existing framework result pointers remain unchanged
}
```

================================================================================
PART 3: BACKEND - Load Challenge in Workflow (domain/analysis/service.go + workflow.go)
================================================================================

IMPORTANT: Use SETTER INJECTION pattern (like companyService), NOT constructor changes.
This is safer and matches the existing pattern in the codebase.

Step 3.1: Add import for challenge domain at top of workflow.go:
```go
import (
    // ... existing imports
    "backend_v3/domain/challenge"  // ADD THIS
)
```

Step 3.2: Add challenge repository field to Service struct (in service.go ~line 79):
```go
type Service struct {
    repo           Repository
    submissionRepo SubmissionRepository
    llm            LLMClient
    logger         zerolog.Logger
    queueClient    *asynq.Client
    companyService AnalysisCompanyServiceInterface
    challengeRepo  challenge.Repository  // ADD THIS LINE
    frameworks     map[string]config.FrameworkConfig
}
```

Step 3.3: Add setter method (in service.go, after SetCompanyService ~line 116):
```go
// SetChallengeRepo wires the challenge repository for loading challenge data during analysis
func (s *Service) SetChallengeRepo(repo challenge.Repository) {
    s.challengeRepo = repo
}
```

Step 3.4: Update main.go (~line 182, after SetCompanyService call):
```go
// Inject challenge repo for challenge data access during analysis
analysisSvc.SetChallengeRepo(challengeRepo)
log.Info().Msg("ChallengeRepo injected into analysis (challenge context enabled)")
```

Step 3.5: In RunAnalysis function (workflow.go), after loading company data (~line 76), add:
```go
// 2.5 FETCH CHALLENGE DATA
if s.challengeRepo == nil {
    s.logger.Error().Msg("challengeRepo not initialized - call SetChallengeRepo first")
    return nil, fmt.Errorf("challengeRepo not initialized")
}
challengeEntity, err := s.challengeRepo.GetByID(ctx, challengeID)
if err != nil {
    s.logger.Error().Err(err).Str("challenge_id", challengeID.String()).Msg("Failed to fetch challenge data")
    return nil, fmt.Errorf("failed to fetch challenge: %w", err)
}

s.logger.Info().
    Str("challenge_category", string(challengeEntity.ChallengeCategory)).
    Str("challenge_type", string(challengeEntity.ChallengeType)).
    Msg("Challenge data loaded successfully")
```

Step 3.6: Update ContextContainer initialization (~line 84):
```go
knowledge := &ContextContainer{
    SubmissionData: submissionData,
    CompanyData:    companyDataToMap(companyData),
    ChallengeData:  challengeToMap(challengeEntity),  // ADD THIS
}
```

Step 3.7: Add helper function at bottom of workflow.go (near submissionToMap):
```go
// challengeToMap converts a Challenge struct to a map for template injection
func challengeToMap(c *challenge.Challenge) map[string]interface{} {
    return map[string]interface{}{
        "challenge_id":        c.ID.String(),
        "challenge_category":  string(c.ChallengeCategory),
        "challenge_type":      string(c.ChallengeType),
        "business_challenge":  c.BusinessChallenge,
    }
}
```

================================================================================
PART 4: BACKEND - Inject Challenge into ALL Framework Runners (domain/analysis/workflow.go)
================================================================================

Update EVERY runXXX function to include challenge data in the data map.
There are 12 functions to update:

4.1 runPESTEL (~line 578):
```go
func (s *Service) runPESTEL(ctx context.Context, k *ContextContainer) (*PESTELAnalysis, error) {
    var res PESTELAnalysis
    data := map[string]interface{}{
        "company_data":       k.CompanyData,
        "macro_context":      s.extractMacroContext(),
        "challenge_context":  k.ChallengeData["business_challenge"],   // ADD
        "challenge_type":     k.ChallengeData["challenge_type"],       // ADD
        "challenge_category": k.ChallengeData["challenge_category"],   // ADD
    }
    // ... rest unchanged
}
```

4.2 runPorter (~line 589):
```go
func (s *Service) runPorter(ctx context.Context, k *ContextContainer) (*PorterAnalysis, error) {
    var res PorterAnalysis
    data := map[string]interface{}{
        "company_data":       k.CompanyData,
        "macro_context":      s.extractMacroContext(),
        "challenge_context":  k.ChallengeData["business_challenge"],   // ADD
        "challenge_type":     k.ChallengeData["challenge_type"],       // ADD
        "challenge_category": k.ChallengeData["challenge_category"],   // ADD
    }
    // ... rest unchanged
}
```

4.3 runTamSamSom (~line 600):
Add same 3 challenge fields to data map.

4.4 runSWOT (~line 611):
Add same 3 challenge fields to data map.

4.5 runBenchmarking (~line 633):
Add same 3 challenge fields to data map.

4.6 runBlueOcean (~line 651):
Add same 3 challenge fields to data map.

4.7 runGrowthHacking (~line 669):
Add same 3 challenge fields to data map.

4.8 runScenarios (~line 709):
Add same 3 challenge fields to data map.

4.9 runOKRs (~line 729):
Add same 3 challenge fields to data map.

4.10 runBSC (~line 783):
Add same 3 challenge fields to data map.

4.11 runDecisionMatrix (~line 815):
Add same 3 challenge fields to data map.

4.12 runSynthesis (~line 847):
Add same 3 challenge fields to data map. THIS IS ESPECIALLY IMPORTANT.

================================================================================
PART 5: BACKEND - Update All Framework Prompts (llm/prompts.go)
================================================================================

Add this block to the START of EVERY framework prompt (after the first line):

```
DESAFIO ESTRATÉGICO DO CLIENTE:
{{.challenge_context}}

TIPO DO DESAFIO: {{.challenge_type}} (Categoria: {{.challenge_category}})

INSTRUÇÃO CRÍTICA: Toda a análise deve ser direcionada para ajudar a resolver este desafio específico.
Conecte cada insight diretamente ao problema do cliente.
```

Update these 12 prompts in llm/prompts.go:
1. FrameworkPESTELPrompt (~line 85)
2. FrameworkPorterPrompt (~line 114)
3. FrameworkTamSamSomPrompt (~line 160)
4. FrameworkSWOTPrompt (~line 257)
5. FrameworkBenchmarkingPrompt (~line 299)
6. FrameworkBlueOceanPrompt (~line 316)
7. FrameworkGrowthHackingPrompt (~line 337)
8. FrameworkScenariosPrompt (~line 389)
9. FrameworkOKRsPrompt (~line 435)
10. FrameworkBSCPrompt (~line 564)
11. FrameworkDecisionMatrixPrompt (~line 582)
12. SynthesisPrompt (~line 687)

For SynthesisPrompt specifically, also update:
- Change line ~715 "Central Challenge: Identifique O desafio estratégico central" to:
  "Central Challenge: O desafio declarado pelo cliente é: {{.challenge_context}}. Use isso como base."
- Add to JSON output instruction: "O executive_summary DEVE endereçar diretamente como a análise resolve o desafio declarado."

================================================================================
PART 6: FRONTEND - Remove Challenge Types (frontend_v2/src/lib/config/challenges.ts)
================================================================================

Update the file to only include remaining categories and types:

```typescript
export type ChallengeCategory = 'growth' | 'transform' | 'compete'
// REMOVED: 'transition' | 'funding'

export const CHALLENGE_CATEGORIES: ChallengeCategoryInfo[] = [
  { code: 'growth', label: 'Crescimento', emoji: '🚀' },
  { code: 'transform', label: 'Transformação', emoji: '🔄' },
  { code: 'compete', label: 'Competitividade', emoji: '⚔️' },
  // REMOVED: transition, funding
]

export const CHALLENGE_TYPES: Record<ChallengeCategory, ChallengeTypeInfo[]> = {
  growth: [
    { code: 'growth_organic', category: 'growth', label: 'Crescimento Orgânico', description: 'Crescer com recursos próprios', emoji: '🌱' },
    { code: 'growth_geographic', category: 'growth', label: 'Expansão Geográfica', description: 'Expandir para novas regiões', emoji: '🗺️' },
    { code: 'growth_segment', category: 'growth', label: 'Novo Segmento', description: 'Entrar em novo segmento de mercado', emoji: '🎯' },
    { code: 'growth_product', category: 'growth', label: 'Novos Produtos', description: 'Lançar novos produtos/serviços', emoji: '📦' },
    { code: 'growth_channel', category: 'growth', label: 'Novos Canais', description: 'Novos canais de venda', emoji: '🛒' },
  ],
  transform: [
    { code: 'transform_digital', category: 'transform', label: 'Transformação Digital', description: 'Digitalização de processos', emoji: '💻' },
    { code: 'transform_model', category: 'transform', label: 'Modelo de Negócio', description: 'Mudar modelo de negócio', emoji: '🔧' },
    // REMOVED: transform_culture, transform_operational
  ],
  compete: [
    { code: 'compete_differentiate', category: 'compete', label: 'Diferenciação', description: 'Criar diferenciação', emoji: '⭐' },
    { code: 'compete_defend', category: 'compete', label: 'Defender Posição', description: 'Defender posição de mercado', emoji: '🛡️' },
    { code: 'compete_reposition', category: 'compete', label: 'Reposicionamento', description: 'Reposicionar marca', emoji: '📍' },
  ],
  // REMOVED: transition, funding categories entirely
}
```

================================================================================
PART 7: FRONTEND - Update Domain Types (frontend_v2/src/lib/types/domain.ts)
================================================================================

Update the ChallengeCategory and ChallengeType types:

```typescript
// ~line 126
export type ChallengeCategory = 'growth' | 'transform' | 'compete'
// REMOVED: 'transition' | 'funding'

// ~line 128-152
export type ChallengeType =
  // Growth
  | 'growth_organic'
  | 'growth_geographic'
  | 'growth_segment'
  | 'growth_product'
  | 'growth_channel'
  // Transform
  | 'transform_digital'
  | 'transform_model'
  // REMOVED: 'transform_culture', 'transform_operational'
  // Compete
  | 'compete_differentiate'
  | 'compete_defend'
  | 'compete_reposition'
  // REMOVED: All transition types
  // REMOVED: All funding types
```

Also update line ~48 in the Submission interface:
```typescript
challengeCategory?: 'growth' | 'transform' | 'compete'
// REMOVED: 'transition' | 'funding'
```

================================================================================
PART 8: FRONTEND - Update Validation Schema (frontend_v2/src/lib/validations/submission.ts)
================================================================================

1. Update challengeTypes object (~line 4-34):
```typescript
export const challengeTypes = {
  growth: [
    { value: 'growth_organic', label: 'Crescimento Orgânico' },
    { value: 'growth_geographic', label: 'Expansão Geográfica' },
    { value: 'growth_segment', label: 'Novo Segmento' },
    { value: 'growth_product', label: 'Novo Produto/Serviço' },
    { value: 'growth_channel', label: 'Novo Canal' },
  ],
  transform: [
    { value: 'transform_digital', label: 'Transformação Digital' },
    { value: 'transform_model', label: 'Mudança de Modelo de Negócio' },
    // REMOVED: transform_culture, transform_operational
  ],
  compete: [
    { value: 'compete_differentiate', label: 'Diferenciação Competitiva' },
    { value: 'compete_defend', label: 'Defesa de Posicionamento' },
    { value: 'compete_reposition', label: 'Reposicionamento' },
  ],
  // REMOVED: transition, funding categories
} as const
```

2. Update the Zod schema (~line 59):
```typescript
challengeCategory: z.enum(['growth', 'transform', 'compete'], {
  // REMOVED: 'transition', 'funding'
  required_error: 'Selecione uma categoria de desafio',
}),
```

================================================================================
PART 9: FRONTEND - Update Submit Form (frontend_v2/src/components/features/submission/submit-section.tsx)
================================================================================

1. Update CHALLENGE_CATEGORIES array (~line 31-37):
```typescript
const CHALLENGE_CATEGORIES = [
  { code: 'growth', label: 'Crescimento', description: 'Estratégias para expandir o negócio' },
  { code: 'transform', label: 'Transformação', description: 'Mudanças estruturais e modernização' },
  { code: 'compete', label: 'Competitividade', description: 'Diferenciação e posicionamento' },
  // REMOVED: transition, funding
] as const
```

2. Update CHALLENGE_TYPES object (~line 41-70):
```typescript
const CHALLENGE_TYPES: Record<ChallengeCategory, { code: string; label: string }[]> = {
  growth: [
    { code: 'growth_organic', label: 'Crescimento Orgânico' },
    { code: 'growth_geographic', label: 'Expansão Geográfica' },
    { code: 'growth_segment', label: 'Novo Segmento' },
    { code: 'growth_product', label: 'Novo Produto' },
    { code: 'growth_channel', label: 'Novo Canal' },
  ],
  transform: [
    { code: 'transform_digital', label: 'Transformação Digital' },
    { code: 'transform_model', label: 'Modelo de Negócio' },
    // REMOVED: transform_culture, transform_operational
  ],
  compete: [
    { code: 'compete_differentiate', label: 'Diferenciação' },
    { code: 'compete_defend', label: 'Defender Posição' },
    { code: 'compete_reposition', label: 'Reposicionamento' },
  ],
  // REMOVED: transition, funding categories
}
```

3. Update the Zod schema in the component (~line 78):
```typescript
challenge_category: z.enum(['growth', 'transform', 'compete'], {
  // REMOVED: 'transition', 'funding'
  required_error: 'Selecione uma categoria',
}),
```

4. Update the ChallengeCategory type (~line 39):
```typescript
type ChallengeCategory = 'growth' | 'transform' | 'compete'
// REMOVED: 'transition' | 'funding'
```

================================================================================
PART 10: BACKEND - Update Test Files
================================================================================

CRITICAL: Test files reference removed types and will FAIL if not updated.

10.1 Update domain/challenge/model_test.go:

Remove test cases for removed categories (lines ~139, ~141):
```go
// DELETE these test cases from TestIsValidCategory:
{"transition", true},  // DELETE
{"funding", true},     // DELETE
```

Remove test cases for removed types (lines ~172-190):
```go
// DELETE these test cases from TestIsValidType:
{"transform", "transform_culture", true},      // DELETE
{"transform", "transform_operational", true},  // DELETE
{"transition", "transition_succession", true}, // DELETE
{"transition", "transition_exit", true},       // DELETE
{"transition", "transition_merger", true},     // DELETE
{"transition", "transition_turnaround", true}, // DELETE
{"funding", "funding_raise", true},            // DELETE
{"funding", "funding_debt", true},             // DELETE
{"funding", "funding_ipo", true},              // DELETE
```

Remove test cases from TestIsValidTypeAny (lines ~213, ~215):
```go
// DELETE these test cases:
{"transition_exit", true},  // DELETE
{"funding_ipo", true},      // DELETE
```

Update count assertions (lines ~260-266):
```go
// CHANGE from:
// Check transition has 4 types
// Check funding has 3 types
// TO: Remove these assertions entirely since categories are gone
```

Update the test that checks ValidTypesByCategory to only expect 3 categories:
```go
// Should now expect only 3 categories: growth (5), transform (2), compete (3)
// Total: 10 types
```

10.2 Update domain/submission/service_test.go:

Change test cases using removed types to valid types (lines ~465-467):
```go
// CHANGE FROM:
{"transition", "transition_succession"},
{"funding", "funding_raise"},
// TO:
{"growth", "growth_organic"},
{"compete", "compete_differentiate"},
```

Update mismatch test case (line ~508):
```go
// CHANGE FROM:
{"transform", "funding_raise"},  // Invalid: funding_raise not in transform
// TO:
{"transform", "compete_defend"},  // Invalid: compete_defend not in transform
```

10.3 Verify domain/submission/repository_test.go:

The reference to "funding_stage" on line ~35 is about submission fields, NOT challenge types.
This does NOT need to change - funding_stage is a company attribute, not a challenge category.
</requirements>

<implementation_order>
Execute in this order to minimize risk:

1. BACKEND Part 1: Remove challenge types from domain/challenge/model.go
2. BACKEND Part 10: Update test files (do this RIGHT AFTER Part 1 so tests pass)
3. BACKEND Part 2: Add ChallengeData to ContextContainer
4. BACKEND Part 3: Wire challenge loading (service.go setter + main.go injection + workflow.go loading)
5. BACKEND Part 4: Update ALL 12 runXXX functions with challenge data
6. BACKEND Part 5: Update ALL 12 prompts with challenge context
7. FRONTEND Part 6: Update challenges.ts config
8. FRONTEND Part 7: Update domain.ts types
9. FRONTEND Part 8: Update submission.ts validation
10. FRONTEND Part 9: Update submit-section.tsx form

After each backend change, run `go build` to verify compilation.
After Part 10, run `make test` to verify tests pass.
</implementation_order>

<constraints>
- Do NOT modify database schema (challenge table already has all needed fields)
- Do NOT change the 11-framework structure (keep all frameworks running)
- Do NOT modify the layered execution order
- PRESERVE backward compatibility for existing analyses
- PRESERVE wizard mode functionality
- All framework outputs maintain existing JSON structure
- Use SETTER INJECTION pattern for challengeRepo (do NOT change NewService constructor)
</constraints>

<testing>
After implementation, verify:

1. Backend compilation:
   ```bash
   cd backend_v3
   go build
   ```

2. Backend tests:
   ```bash
   make test
   ```

3. Frontend build:
   ```bash
   cd frontend_v2
   npm run build
   ```

4. Manual verification:
   - Navigate to /submit page - only growth, transform, compete categories should appear
   - Selecting each category shows only its valid types
   - Submit form should reject removed types

5. API verification:
   - POST to /api/v1/submit with removed type should return validation error
</testing>

<files_to_modify>
Backend (Go) - 8 files:
- `./domain/challenge/model.go` - Remove types and categories
- `./domain/challenge/model_test.go` - Update tests for removed types
- `./domain/submission/service_test.go` - Update test cases using removed types
- `./domain/analysis/model.go` - Add ChallengeData to ContextContainer
- `./domain/analysis/service.go` - Add challengeRepo field + SetChallengeRepo() setter
- `./domain/analysis/workflow.go` - Load challenge + add challengeToMap() + update all 12 runXXX functions
- `./llm/prompts.go` - Add challenge context to all 12 prompts
- `./main.go` - Add SetChallengeRepo() call after line 182

Frontend (TypeScript) - 4 files:
- `../frontend_v2/src/lib/config/challenges.ts` - Remove categories and types
- `../frontend_v2/src/lib/types/domain.ts` - Update type unions
- `../frontend_v2/src/lib/validations/submission.ts` - Update validation
- `../frontend_v2/src/components/features/submission/submit-section.tsx` - Update form

Total: 12 files
</files_to_modify>

<verification>
Before declaring complete:
1. `go build` in backend_v3 - no errors
2. `make test` - all tests pass (especially model_test.go and service_test.go)
3. `npm run build` in frontend_v2 - no errors
4. Grep for "transition" and "funding" in modified files - should only appear in comments
5. Verify all 12 prompts contain "challenge_context"
6. Verify ContextContainer has ChallengeData field
7. Verify all 12 runXXX functions have challenge fields in data map
8. Verify SetChallengeRepo() method exists in service.go
9. Verify main.go calls SetChallengeRepo(challengeRepo)
10. Verify challengeToMap() function exists in workflow.go
</verification>

<success_criteria>
- Only growth, transform, compete categories remain
- Only 10 challenge types remain (5 growth + 2 transform + 3 compete)
- All 12 framework prompts receive challenge context
- Synthesis explicitly references the user's challenge
- Frontend form only shows valid categories/types
- All backend tests pass
- Frontend builds without errors
</success_criteria>