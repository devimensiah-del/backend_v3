<objective>
Create the `domain/analysisbysteps/` package for IAH-2: step-by-step analysis flow with human editing capability.

This package enables human-in-the-loop analysis where AI generates output, humans can edit it, and the system tracks which version to display.
</objective>

<context>
Repository: backend_v3 (Go)
Branch: Create `feature/IAH-2-analysisbysteps` from `main`
Jira: IAH-2 - Domínio e modelo do fluxo por etapas

Existing patterns to follow:
- `domain/analysis/` for model/repository/service structure
- `domain/analysis/constants.go` for framework-related constants
- Migration naming: `v2_019_*.sql` (next after v2_018)
</context>

<requirements>
1. **Create branch** from main:
   ```
   git checkout main && git checkout -b feature/IAH-2-analysisbysteps
   ```

2. **Create package structure** at `domain/analysisbysteps/`:
   - `model.go` - AnalysisStep struct and related types
   - `repository.go` - Database operations
   - `service.go` - Empty placeholder (just package declaration)
   - `constants.go` - Framework order constants

3. **AnalysisStep model** in `model.go`:
   ```go
   type AnalysisStep struct {
       ID            string     `db:"id" json:"id"`
       AnalysisID    string     `db:"analysis_id" json:"analysis_id"`
       FrameworkCode string     `db:"framework_code" json:"framework_code"`
       StepNumber    int        `db:"step_number" json:"step_number"`
       AIOutput      *string    `db:"ai_output" json:"ai_output"`
       HumanEdited   *string    `db:"human_edited" json:"human_edited"`
       Visible       bool       `db:"visible" json:"visible"`
       Status        string     `db:"status" json:"status"`
       GeneratedAt   *time.Time `db:"generated_at" json:"generated_at"`
       ApprovedAt    *time.Time `db:"approved_at" json:"approved_at"`
       CreatedAt     time.Time  `db:"created_at" json:"created_at"`
       UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
   }
   ```

4. **GetEffectiveOutput() method** on AnalysisStep:
   - Returns `HumanEdited` if not nil, otherwise `AIOutput`
   - Mirrors SQL: `COALESCE(human_edited, ai_output)`

5. **Framework order constants** in `constants.go`:
   ```go
   // FrameworkMeta contains metadata for each analysis framework step
   type FrameworkMeta struct {
       Code         string
       Name         string
       GuidanceText string // Human checkpoint reflection text
   }

   // FrameworkOrder defines the 14-step analysis sequence (0-13)
   var FrameworkOrder = []FrameworkMeta{
       {Code: "challenge_refinement", Name: "Refinamento do Desafio", GuidanceText: "Revise se o desafio está claro e específico. Este é realmente o problema ou apenas um sintoma?"},
       {Code: "pestel", Name: "Análise PESTEL", GuidanceText: "Considere quais fatores externos realmente impactam este negócio. Algum fator foi ignorado ou superestimado?"},
       {Code: "porter", Name: "5 Forças de Porter", GuidanceText: "Avalie a intensidade competitiva. Os concorrentes listados estão corretos? Algum foi esquecido?"},
       {Code: "benchmarking", Name: "Benchmarking", GuidanceText: "Os players comparados são realmente relevantes? As métricas fazem sentido para este contexto?"},
       {Code: "swot", Name: "Análise SWOT", GuidanceText: "As forças listadas realmente geram valor? As fraquezas são críticas para o desafio?"},
       {Code: "swotcross", Name: "SWOT Cruzado", GuidanceText: "As estratégias cruzadas são viáveis? Priorize as que atacam o desafio diretamente."},
       {Code: "tam_sam_som", Name: "TAM-SAM-SOM", GuidanceText: "O dimensionamento de mercado está realista? O SOM é operacionalmente alcançável?"},
       {Code: "blue_ocean", Name: "Blue Ocean", GuidanceText: "A curva de valor proposta realmente diferencia? O cliente pagaria por isso?"},
       {Code: "growth_hacking", Name: "Growth Hacking", GuidanceText: "As táticas de crescimento são aplicáveis ao estágio atual da empresa?"},
       {Code: "scenarios", Name: "Cenários", GuidanceText: "Os cenários cobrem riscos relevantes? Há plano de contingência para o pior caso?"},
       {Code: "decision_matrix", Name: "Matriz de Decisão", GuidanceText: "Os critérios de decisão refletem as prioridades reais? Os pesos estão calibrados?"},
       {Code: "okrs", Name: "OKRs", GuidanceText: "Os objetivos são ambiciosos mas alcançáveis? Os key results são mensuráveis?"},
       {Code: "bsc", Name: "Balanced Scorecard", GuidanceText: "As perspectivas estão balanceadas? Os indicadores são acionáveis?"},
       {Code: "synthesis", Name: "Síntese Executiva", GuidanceText: "A síntese captura as principais conclusões? As recomendações são claras e priorizadas?"},
   }
   ```
   Also add helpers:
   - `GetStepNumber(frameworkCode string) int`
   - `GetFrameworkMeta(frameworkCode string) *FrameworkMeta`

6. **Migration** at `migrations/v2_019_analysis_steps_by_human.sql`:
   ```sql
   -- v2_019_analysis_steps_by_human.sql
   -- IAH-2: Step-by-step analysis with human editing

   CREATE TABLE IF NOT EXISTS analysis_steps_v2 (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       analysis_id UUID NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
       framework_code TEXT NOT NULL,
       step_number INTEGER NOT NULL,
       ai_output TEXT,
       human_edited TEXT,
       visible BOOLEAN NOT NULL DEFAULT true,  -- frameworks visible by default, user hides if needed
       status TEXT NOT NULL DEFAULT 'pending',
       generated_at TIMESTAMPTZ,
       approved_at TIMESTAMPTZ,
       created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
       updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
       UNIQUE(analysis_id, framework_code)
   );

   CREATE INDEX idx_analysis_steps_v2_analysis ON analysis_steps_v2(analysis_id);
   CREATE INDEX idx_analysis_steps_v2_status ON analysis_steps_v2(status);

   -- Rollback: DROP TABLE IF EXISTS analysis_steps_v2 CASCADE;
   ```

7. **Repository** in `repository.go`:
   - Basic CRUD: Create, GetByID, GetByAnalysisID, Update
   - GetByAnalysisAndFramework(analysisID, frameworkCode)
   - Use `*sqlx.DB` and `*sqlx.Tx` pattern from existing repos
</requirements>

<implementation>
Follow existing domain patterns:
- Look at `domain/analysis/model.go` for struct style
- Look at `domain/analysis/repository.go` for query patterns
- Use `github.com/jmoiron/sqlx` for database operations
- Status values: "pending", "generating", "generated", "approved", "failed"
</implementation>

<output>
Create/modify files:
- `./domain/analysisbysteps/model.go`
- `./domain/analysisbysteps/constants.go`
- `./domain/analysisbysteps/repository.go`
- `./domain/analysisbysteps/service.go` (empty, just package declaration)
- `./migrations/v2_019_analysis_steps_by_human.sql`
</output>

<verification>
Before completing:
1. Run `go build ./domain/analysisbysteps/...` - must compile without errors
2. Verify all 14 frameworks are in FrameworkOrder (0-13)
3. Verify GetEffectiveOutput() returns correct priority
4. Migration SQL is valid (check syntax)
5. Commit changes to the feature branch
</verification>

<success_criteria>
- Package compiles: `go build ./...` passes
- All files created with proper Go conventions
- Migration ready to run (includes rollback comment)
- Branch committed and ready for PR
</success_criteria>
