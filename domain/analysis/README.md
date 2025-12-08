# Analysis Domain

## Purpose
Executes 11 strategic analysis frameworks using enrichment data, producing comprehensive business intelligence. Supports both batch execution and step-by-step wizard mode for human-in-the-loop validation.

## Execution Modes

### 1. Direct Analysis Mode (Default)
**File**: `workflow.go` → `RunAnalysis()`

**Execution Strategy**: Maximum parallel execution respecting layer dependencies
- **Parallel execution WITHIN layers**: All frameworks in a layer run concurrently using goroutines
- **Sequential execution BETWEEN layers**: Each layer waits for all parallel tasks before proceeding
- **Checkpoint saves**: After each layer for crash recovery

**Performance**:
- Layer 1 (3 frameworks): ~60-90s total (fastest framework wins, not 3x60s)
- Layer 2 (2 frameworks): ~60-90s total
- Layer 3 (3 frameworks): ~60-90s total
- Layer 3.5 (1 framework): ~60-90s
- Layer 4 (2 frameworks): ~60-90s total
- Synthesis: ~60-90s
- **Total**: ~6-9 minutes (NOT 11x60s = 11 minutes)

**When to use**: Background batch processing, API-triggered analysis, automated workflows

### 2. Wizard Mode (Human-in-the-loop)
**File**: `wizard/service.go` → `WizardService`

**Execution Strategy**: Fully sequential with human approval gates
- **Step-by-step**: One framework at a time, wait for human review
- **Refinement pattern**: "Add context → regenerate" (no direct edits)
- **Versioning**: Each refinement creates audit trail snapshot
- **No going back**: Once approved, step cannot be regenerated

**Performance**:
- 12 total steps (11 frameworks + synthesis)
- Each step: ~60-90s LLM generation + human review time
- **Total**: Depends on human review speed (hours to days)

**When to use**: High-value analyses, expert validation, learning mode

## Data Flow (Strategic Cascade - Direct Analysis Mode)
```
Background Worker                    Domain                         LLM
      │                                │                              │
      ├── Process job ────────────────►│                              │
      │                                │                              │
      │    ┌───────────────────────────┴──────────────────────────┐   │
      │    │     LAYER 1: ENVIRONMENT (3 frameworks in PARALLEL)  │   │
      │    │  ┌─────────┐  ┌────────┐  ┌─────────────┐           │   │
      │    │  │ PESTEL  │  │ Porter │  │ TAM-SAM-SOM │ ──────────┬───►│
      │    │  │ 60-90s  │  │ 60-90s │  │   60-90s    │           │   │
      │    │  └─────────┘  └────────┘  └─────────────┘           │   │
      │    │  Layer duration: max(60-90s), NOT sum(180-270s)     │   │
      │    └──────────────────────────────────────────────────────┘   │
      │                                │                              │
      │    ┌───────────────────────────┴──────────────────────────┐   │
      │    │    LAYER 2: POSITIONING (2 frameworks in PARALLEL)   │   │
      │    │  ┌──────┐  ┌──────────────┐                          │   │
      │    │  │ SWOT │  │ Benchmarking │ ────────────────────────────►│
      │    │  └──────┘  └──────────────┘                          │   │
      │    └──────────────────────────────────────────────────────┘   │
      │                                │                              │
      │    ┌───────────────────────────┴──────────────────────────┐   │
      │    │     LAYER 3: STRATEGY (3 frameworks in PARALLEL)     │   │
      │    │  ┌────────────┐  ┌───────────────┐  ┌───────────┐   │   │
      │    │  │ Blue Ocean │  │ Growth Hacking│  │ Scenarios │ ─────►│
      │    │  └────────────┘  └───────────────┘  └───────────┘   │   │
      │    └──────────────────────────────────────────────────────┘   │
      │                                │                              │
      │    ┌───────────────────────────┴──────────────────────────┐   │
      │    │     LAYER 3.5: DECISION (1 framework, sequential)    │   │
      │    │  ┌─────────────────┐                                 │   │
      │    │  │ Decision Matrix │ ──────────────────────────────────►│
      │    │  └─────────────────┘                                 │   │
      │    └──────────────────────────────────────────────────────┘   │
      │                                │                              │
      │    ┌───────────────────────────┴──────────────────────────┐   │
      │    │    LAYER 4: EXECUTION (2 frameworks in PARALLEL)     │   │
      │    │  ┌──────┐  ┌─────┐                                   │   │
      │    │  │ OKRs │  │ BSC │ ────────────────────────────────────►│
      │    │  └──────┘  └─────┘                                   │   │
      │    └──────────────────────────────────────────────────────┘   │
      │                                │                              │
      │    ┌───────────────────────────┴──────────────────────────┐   │
      │    │         SYNTHESIS (premium model, sequential)        │   │
      │    │  ┌───────────────────────────────────────────────┐   │   │
      │    │  │ Executive Summary + Cross-Framework Validation │──────►│
      │    │  └───────────────────────────────────────────────┘   │   │
      │    └──────────────────────────────────────────────────────┘   │
      │                                │                              │
      │◄── Analysis complete ──────────┤                              │
```

## Performance Characteristics

**Bottleneck**: LLM response streaming (30-90s per call), NOT Go execution
- Go goroutine overhead: <1ms
- Database checkpoint saves: ~10-50ms
- LLM API calls: 30,000-90,000ms (99.9%+ of total time)

**Parallel execution benefit**:
- Layer 1: 3 frameworks in parallel = ~60-90s (vs 180-270s sequential)
- Layer 2: 2 frameworks in parallel = ~60-90s (vs 120-180s sequential)
- Layer 3: 3 frameworks in parallel = ~60-90s (vs 180-270s sequential)
- Layer 4: 2 frameworks in parallel = ~60-90s (vs 120-180s sequential)
- **Total savings**: ~3-5 minutes compared to fully sequential execution

**Why not more parallelization?**
- Layer dependencies prevent cross-layer parallelization (SWOT needs PESTEL+Porter)
- Some frameworks could theoretically run earlier (Scenarios only needs PESTEL from Layer 1)
- Current layer-by-layer approach prioritizes simplicity and maintainability
- Future optimization: Dependency graph approach (see "Potential Optimizations" below)

## Business Rules (INVARIANTS)

### MUST
- **Challenge ID is REQUIRED**: Every analysis must link to a specific business challenge
- Enrichment MUST be completed before analysis starts
- Layers execute in order: 1 → 2 → 3 → 3.5 → 4 → Synthesis
- Decision Matrix MUST complete before OKRs (OKRs use its recommendations)
- Synthesis MUST be last (uses all framework summaries)
- Save checkpoints after each layer (for recovery)
- All framework results stored in `framework_results` JSONB

### WIZARD MODE RULES
- **No going back**: Once a step is approved, it cannot be regenerated
- **Add context → regenerate**: Human provides context, AI regenerates (no direct edits)
- **Versioning on refinement**: Each refinement creates a version snapshot for audit trail
- **Forward-only progression**: Wizard only moves forward through 12 steps (11 frameworks + synthesis)

### NEVER
- Skip layers or reorder them
- Run OKRs before Decision Matrix
- Start analysis without enrichment data
- Remove synthesis (it's the executive summary)
- Store individual framework columns (use `framework_results`)
- Allow analysis without challenge_id (links to business problem)

## Status State Machine
```
    ┌─────────┐
    │ pending │ ←── Initial state
    └────┬────┘
         │ (worker starts)
         ▼
    ┌────────────┐
    │ processing │ ←── Frameworks executing
    └─────┬──────┘
          │
    ┌─────┴─────┐
    ▼           ▼
┌───────────┐  ┌─────────┐
│ completed │  │ failed  │
└───────────┘  └─────────┘
```

## 11 Frameworks

| Framework | Layer | Purpose | Key Output |
|-----------|-------|---------|------------|
| PESTEL | 1 | Macro environment | 6 dimensions + summary |
| Porter | 1 | Industry forces | 7 forces (5+2 modern) + implications |
| TAM-SAM-SOM | 1 | Market sizing | 3-tier scenarios + CAGR |
| SWOT | 2 | Internal/external | Items with confidence + source |
| Benchmarking | 2 | Competitive gaps | Best practices + gaps |
| Blue Ocean | 3 | Strategy canvas | ERRC framework |
| Growth Hacking | 3 | Growth loops | LEAP + SCALE loops |
| Scenarios | 3 | Future planning | Optimistic/Realist/Pessimistic |
| Decision Matrix | 3.5 | Priority recommendations | Ranked recommendations |
| OKRs | 4 | 90-day plan | Monthly milestones |
| BSC | 4 | Balanced scorecard | 4 perspectives |
| **Synthesis** | Final | Executive summary | Key findings + roadmap |

## Wizard Mode (v2)
Human-in-the-loop validation system for step-by-step analysis.

**THE MVP FEATURE** - Step-by-step framework execution with human approval at each step.

### Core Pattern: "Add Context → Regenerate"
The wizard does NOT allow direct edits to AI output. Instead:
1. Human reviews AI-generated framework output
2. If not satisfied, human provides additional context (feedback)
3. AI uses previous output + human context to regenerate
4. Refinement enriches, never replaces - builds on previous output

### Wizard Rules
- **12 Total Steps**: 11 frameworks + synthesis
- **No Going Back**: Once a step is approved, it cannot be modified
- **Versioning**: Each refinement creates a version snapshot (audit trail)
- **Forward-Only**: Wizard only moves forward (validated in GenerateStep)

```
┌──────────────────────────────────────────────────────────────────────┐
│  WIZARD MODE (wizard_mode = true)                                    │
│                                                                      │
│  Analysis.current_step tracks position (0-11)                        │
│  Analysis.steps_completed tracks approved frameworks                 │
│                                                                      │
│  AnalysisStep table tracks each framework:                           │
│  ├── status: pending → generating → generated → approved             │
│  ├── output: AI-generated framework result                           │
│  ├── human_context: Additional context for regeneration              │
│  ├── clarifying_questions/human_answers: Q&A for refinement          │
│  └── iteration_count: Number of regenerations                        │
│                                                                      │
│  Flow:                                                               │
│  1. GenerateStep() → AI generates framework → status=generated       │
│  2. Human reviews → ApproveStep() OR RefineStep(context)             │
│  3. If RefineStep: version snapshot created, iteration_count++       │
│  4. ApproveStep() advances to next step (no going back)              │
└──────────────────────────────────────────────────────────────────────┘
```

### Wizard Service Methods
| Method | Purpose |
|--------|---------|
| `StartWizard(subID)` | Initialize wizard, create Analysis with wizard_mode=true |
| `GetWizardState(analysisID)` | Get current wizard state (step, progress) |
| `GenerateStep(analysisID)` | Generate next framework (validates no going back) |
| `ApproveStep(analysisID)` | Approve current step and advance |
| `RefineStep(analysisID, context)` | Add context and regenerate (with versioning) |
| `GetWizardSummary(analysisID)` | Get summary of all completed frameworks |

## Key Types
| Type | Purpose |
|------|---------|
| `Analysis` | Main entity with `framework_results` JSONB |
| `AnalysisStep` | Individual framework step (wizard mode) |
| `ContextContainer` | Holds all framework results during execution |
| `Service` | Orchestrates framework execution |
| `Workflow` | Manages parallel/sequential execution |
| Framework structs | `PESTELAnalysis`, `SWOTAnalysis`, etc. |

## Framework Results Storage
```go
// Post-migration 034-036: All frameworks in single JSONB
analysis.FrameworkResults = map[string]json.RawMessage{
    "pestel":          {...},
    "porter":          {...},
    "swot":            {...},
    "tam_sam_som":     {...},
    "benchmarking":    {...},
    "blue_ocean":      {...},
    "growth_hacking":  {...},
    "scenarios":       {...},
    "decision_matrix": {...},
    "okrs":            {...},
    "bsc":             {...},
    "synthesis":       {...},
}

// Helper methods
analysis.GetFramework("pestel", &pestelResult)
analysis.SetFramework("pestel", pestelData)
analysis.HasFramework("swot")
```

## Visibility & Access
| Field | Purpose |
|-------|---------|
| `is_visible_to_user` | Admin must enable for user to see |
| `is_blurred` | Premium frameworks blurred for paywall |
| `is_public` | Accessible via access code without login |
| `access_code` | Shareable link for public access |

## Detailed Timing Logs

The workflow now includes comprehensive timing metrics for performance analysis:

### Log Levels and Output
```
INFO: Starting Strategic Cascade Analysis - Direct Analysis Mode (Parallel within layers)
INFO: 🚀 Layer started - frameworks will run in PARALLEL | layer="Layer 1: Environment"
INFO: ⚡ PESTEL started | framework="PESTEL"
INFO: ⚡ Porter started | framework="Porter"
INFO: ⚡ TAM-SAM-SOM started | framework="TAM-SAM-SOM"
INFO: ✅ PESTEL completed | framework="PESTEL" duration_ms=65432
INFO: ✅ Porter completed | framework="Porter" duration_ms=72156
INFO: ✅ TAM-SAM-SOM completed | framework="TAM-SAM-SOM" duration_ms=58921
INFO: ✅ Layer completed - all parallel frameworks finished | layer="Layer 1: Environment" duration_ms=72156 duration_seconds=72.156
```

### Metrics Captured
- **Per-framework timing**: Individual LLM call duration
- **Per-layer timing**: Total layer duration (max of parallel tasks)
- **Total analysis timing**: End-to-end processing time
- **Execution mode**: Direct Analysis vs Wizard Mode identification

### Using Logs for Optimization
1. Check which frameworks are slowest (bottleneck identification)
2. Verify parallel execution is working (layer time ≈ max(framework times), not sum)
3. Identify if LLM API is rate-limiting (unusually long durations)
4. Track total processing time trends over multiple runs

## Potential Optimizations

### 1. Dependency Graph Approach
**Current**: Layer-by-layer execution (5 sequential stages)
**Proposed**: Framework-level dependency graph

```
Layer 1 (parallel):
  PESTEL, Porter, TAM-SAM-SOM

After PESTEL+Porter complete → Start SWOT
After TAM-SAM-SOM complete → Start Benchmarking
After Porter complete → Start BlueOcean
After PESTEL complete → Start Scenarios
After SWOT+TAM-SAM-SOM complete → Start GrowthHacking
After Scenarios complete → Start DecisionMatrix
After DecisionMatrix complete → Start OKRs, BSC
After all complete → Start Synthesis
```

**Benefit**: Could save 1-2 minutes by not waiting for full layer completion
**Complexity**: Higher code complexity, harder to reason about, more error-prone
**Recommendation**: Current layer approach is good balance of simplicity vs performance

### 2. LLM Request Batching
**Current**: Individual LLM calls per framework
**Proposed**: Batch multiple frameworks in single LLM request
**Benefit**: Reduce network round-trips
**Downside**: Longer timeout windows, all-or-nothing failure mode
**Recommendation**: Not worth the trade-offs for streaming responses

### 3. Caching Layer
**Proposed**: Cache framework results for repeated company+challenge combinations
**Benefit**: Near-instant re-analysis
**Downside**: Cache invalidation complexity, stale data risk
**Recommendation**: Consider for high-traffic scenarios

## Related Domains
- **Enrichment**: Source data for all frameworks
- **Submission**: Parent entity (via enrichment)
- **Company**: Linked for company-centric views
- **Framework**: Dynamic framework configuration (v2+)
- **Wizard**: Human-in-the-loop execution mode

## AI Agent Warnings

### DO NOT
- Change layer execution order (breaks dependencies)
- Remove Decision Matrix → OKRs dependency (OKRs need recommendations)
- Skip validation at the end (critical frameworks check prevents incomplete reports)
- Remove checkpoint saves (needed for crash recovery)
- Change the 4-layer architecture (well-tested and battle-hardened)
- Use individual framework columns (deprecated - use framework_results JSONB)
- Break parallel execution within layers (performance regression)

### SAFE TO MODIFY
- Add new frameworks (follow layer pattern, update dependencies)
- Modify individual framework prompts (via llm/prompts.go)
- Adjust parallel execution within layers (maintain sync.WaitGroup pattern)
- Add new fields to framework outputs (update model structs)
- Extend wizard mode functionality (preserve versioning and forward-only progression)
- Add more detailed timing logs (help identify bottlenecks)
- Implement dependency graph optimization (advanced - requires careful testing)
