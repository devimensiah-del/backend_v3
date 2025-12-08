# Enrichment Domain

## Purpose

Stateless service that gathers company information using Perplexity AI.
Called once during company creation, enriched data fills NULL fields on the company record.

## File Structure

```
enrichment/
├── model.go    # EnrichedCompanyData response struct
└── service.go  # Stateless Perplexity client
```

**Note:** No repository - enrichment is stateless. Data is stored on the `companies` table.

## Data Flow

```
Company Creation                     Enrichment Service              Perplexity API
      │                                    │                              │
      ├── runEnrichment() async ──────────►│                              │
      │   (fire-and-forget goroutine)      │                              │
      │                                    │                              │
      │                                    ├── SetEnrichmentProcessing() ►│
      │                                    │   (status = processing)      │
      │                                    │                              │
      │                                    ├── EnrichCompany() ───────────►│
      │                                    │   (Perplexity sonar-pro)      │
      │                                    │◄──────────────────────────────┤
      │                                    │                              │
      │                                    ├── SetEnrichmentCompleted() ──┤
      │                                    │   (fills NULL fields only)    │
      │                                    │   (status = completed)        │
      │                                    │                              │
      │   ON FAILURE:                      │                              │
      │                                    ├── SetEnrichmentFailed() ─────┤
      │                                    │   (status = failed)           │
      │                                    │   (error message stored)      │
```

## Business Rules

### Core Principles
- **One-time only**: Runs once at company creation, never again
- **Fills gaps only**: Only populates NULL fields, never overwrites user input
- **Perplexity only**: Uses sonar-pro model (with sonar fallback)
- **Stateless**: No repository, no database - just calls AI and returns data
- **Fire-and-forget**: Runs async, company creation doesn't wait for it

### Model Configuration
```
AI_PRESEARCH_MODEL=perplexity/sonar-pro (default)
AI_PRESEARCH_FALLBACK=perplexity/sonar
```

## Key Types

| Type | File | Purpose |
|------|------|---------|
| `Service` | service.go | Stateless Perplexity client |
| `CompanyInput` | service.go | Input with company name, CNPJ, website, etc. |
| `EnrichedCompanyData` | model.go | Response struct with enriched fields |

## EnrichedCompanyData Fields

```go
type EnrichedCompanyData struct {
    // Core identifiers
    CNPJ    *string
    Website *string

    // Business context
    Industry         *string
    CompanySize      *string
    Location         *string
    TargetMarket     *string
    FundingStage     *string
    AnnualRevenueMin *float64
    AnnualRevenueMax *float64

    // Enriched data
    FoundationYear    *string
    LegalName         *string
    Headquarters      *string
    Sector            *string
    TargetAudience    *string
    ValueProposition  *string
    EmployeesRange    *string
    RevenueEstimate   *string
    BusinessModel     *string
    MarketShareStatus *string
    DigitalMaturity   *int     // 1-10

    // Arrays
    Competitors []string
    Strengths   []string
    Weaknesses  []string

    // Social
    LinkedInURL   *string
    TwitterHandle *string

    // Meta
    ConfidenceScore float64  // 0-100
    Sources         []string // URLs used
}
```

## Service Methods

| Method | Visibility | Purpose |
|--------|------------|---------|
| `NewService()` | Public | Constructor with LLM client and config |
| `EnrichCompany()` | Public | Main method - calls Perplexity, returns data |
| `identifyMissingFields()` | Private | Determines what fields to ask Perplexity for |
| `buildEnrichmentPrompt()` | Private | Builds PT-BR prompt for Perplexity |
| `parseEnrichmentResponse()` | Private | Parses JSON response, validates confidence |

## Prompt Details

The enrichment prompt (built in `buildEnrichmentPrompt()`) is in PT-BR and:
- Lists known company data (name, CNPJ, website, etc.)
- Identifies missing fields
- Asks Perplexity to fill only high-confidence data (>70%)
- Requests JSON output with specific field structure
- Asks for source URLs

## Related Domains

- **Company**: Calls `EnrichCompany()` in async goroutine, stores results via `SetEnrichmentCompleted()`
- **Submission**: Creates company which triggers enrichment

## API Endpoints

**None** - Enrichment is internal to company creation flow.
No re-enrichment endpoint exists (enrichment is one-time only).

## Enrichment Status (on Company)

| Status | Meaning |
|--------|---------|
| `pending` | Company created, enrichment not started |
| `processing` | Enrichment in progress |
| `completed` | Enrichment finished successfully |
| `failed` | Enrichment failed (error stored in `enrichment_error`) |

## What Was Removed

| Removed | Reason |
|---------|--------|
| `enrichments` table | Data stored on company directly |
| `enrichment_id` field | No separate entity |
| Re-enrich endpoint | Enrichment is one-time only |
| Background jobs | Enrichment runs async but inline |
| Gemini integration | Perplexity only |
| Macro data integration | Moved to macroeconomics domain |

## AI Agent Warnings

### DO NOT
- Add re-enrichment capability (one-time only)
- Create enrichments table (data lives on company)
- Integrate Gemini or other AI (Perplexity only for enrichment)
- Make enrichment overwrite existing data (fills NULLs only)
- Make enrichment synchronous (must be async/fire-and-forget)

### SAFE TO MODIFY
- Add new fields to EnrichedCompanyData (update company model too)
- Improve Perplexity prompt for better data gathering
- Add retry logic for API failures
- Adjust confidence threshold (currently >70%)
- Change prompt language or structure
