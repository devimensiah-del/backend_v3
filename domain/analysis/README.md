# Analysis Domain

## Purpose
Executes 11 strategic analysis frameworks using enrichment data, producing comprehensive business intelligence. Supports both batch execution and step-by-step wizard mode for human-in-the-loop validation.

## Data Flow (Strategic Cascade)
```
Background Worker                    Domain                         LLM
      │                                │                              │
      ├── Process job ────────────────►│                              │
      │                                │                              │
      │    ┌───────────────────────────┴──────────────────────────┐   │
      │    │              LAYER 1: ENVIRONMENT (parallel)         │   │
      │    │  ┌─────────┐  ┌────────┐  ┌─────────────┐           │   │
      │    │  │ PESTEL  │  │ Porter │  │ TAM-SAM-SOM │ ─────────────►│
      │    │  └─────────┘  └────────┘  └─────────────┘           │   │
      │    └──────────────────────────────────────────────────────┘   │
      │                                │                              │
      │    ┌───────────────────────────┴──────────────────────────┐   │
      │    │              LAYER 2: POSITIONING (parallel)         │   │
      │    │  ┌──────┐  ┌──────────────┐                          │   │
      │    │  │ SWOT │  │ Benchmarking │ ────────────────────────────►│
      │    │  └──────┘  └──────────────┘                          │   │
      │    └──────────────────────────────────────────────────────┘   │
      │                                │                              │
      │    ┌───────────────────────────┴──────────────────────────┐   │
      │    │              LAYER 3: STRATEGY (parallel)            │   │
      │    │  ┌────────────┐  ┌───────────────┐  ┌───────────┐   │   │
      │    │  │ Blue Ocean │  │ Growth Hacking│  │ Scenarios │ ─────►│
      │    │  └────────────┘  └───────────────┘  └───────────┘   │   │
      │    └──────────────────────────────────────────────────────┘   │
      │                                │                              │
      │    ┌───────────────────────────┴──────────────────────────┐   │
      │    │              LAYER 3.5: DECISION (sequential)        │   │
      │    │  ┌─────────────────┐                                 │   │
      │    │  │ Decision Matrix │ ──────────────────────────────────►│
      │    │  └─────────────────┘                                 │   │
      │    └──────────────────────────────────────────────────────┘   │
      │                                │                              │
      │    ┌───────────────────────────┴──────────────────────────┐   │
      │    │              LAYER 4: EXECUTION (parallel)           │   │
      │    │  ┌──────┐  ┌─────┐                                   │   │
      │    │  │ OKRs │  │ BSC │ ────────────────────────────────────►│
      │    │  └──────┘  └─────┘                                   │   │
      │    └──────────────────────────────────────────────────────┘   │
      │                                │                              │
      │    ┌───────────────────────────┴──────────────────────────┐   │
      │    │              SYNTHESIS (premium model)               │   │
      │    │  ┌───────────────────────────────────────────────┐   │   │
      │    │  │ Executive Summary + Cross-Framework Validation │──────►│
      │    │  └───────────────────────────────────────────────┘   │   │
      │    └──────────────────────────────────────────────────────┘   │
      │                                │                              │
      │◄── Analysis complete ──────────┤                              │
```

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

## Related Domains
- **Enrichment**: Source data for all frameworks
- **Submission**: Parent entity (via enrichment)
- **Company**: Linked for company-centric views
- **Framework**: Dynamic framework configuration (v2+)

## AI Agent Warnings

### DO NOT
- Change layer execution order
- Remove Decision Matrix → OKRs dependency
- Skip validation at the end (critical frameworks check)
- Remove checkpoint saves (needed for recovery)
- Change the 4-layer architecture
- Use individual framework columns (deprecated)

### SAFE TO MODIFY
- Add new frameworks (follow layer pattern)
- Modify individual framework prompts
- Adjust parallel execution within layers
- Add new fields to framework outputs
- Extend wizard mode functionality
