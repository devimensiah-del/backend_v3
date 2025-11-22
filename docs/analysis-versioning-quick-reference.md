# Analysis Versioning - Quick Reference

## Quick Start

### 1. Create a New Version
```go
edits := map[string]interface{}{
    "swot": map[string]interface{}{
        "strengths": []string{"Updated strength"},
    },
}
newVersion, err := service.CreateVersion(ctx, analysisID, edits)
```

### 2. Get Latest Version
```go
latest, err := service.GetLatestVersion(ctx, submissionID)
```

### 3. Get All Versions
```go
versions, err := service.GetAllVersions(ctx, submissionID)
```

### 4. Update Status
```go
analysis.Approve()  // approved
analysis.Send()     // sent
service.Update(ctx, analysis)
```

## Status Flow

```
pending → processing → completed → approved → sent
                          ↓
                       failed
```

## Edit Structure

### PESTEL
```json
{
  "pestel": {
    "political": ["item1"],
    "economic": ["item2"],
    "social": ["item3"],
    "technological": ["item4"],
    "environmental": ["item5"],
    "legal": ["item6"],
    "summary": "text"
  }
}
```

### Porter
```json
{
  "porter": {
    "competitive_rivalry": "text",
    "supplier_power": "text",
    "buyer_power": "text",
    "threat_new_entrants": "text",
    "threat_substitutes": "text",
    "overall_attractiveness": "text",
    "summary": "text"
  }
}
```

### SWOT
```json
{
  "swot": {
    "strengths": ["item1"],
    "weaknesses": ["item2"],
    "opportunities": ["item3"],
    "threats": ["item4"],
    "summary": "text"
  }
}
```

## Database Fields

| Field | Type | Description |
|-------|------|-------------|
| `version` | int | Version number (starts at 1) |
| `parent_analysis_id` | uuid | Reference to parent version |
| `status` | string | Current status |

## Repository Methods

| Method | Description |
|--------|-------------|
| `GetLatestVersionBySubmissionID()` | Get newest version |
| `GetAllVersionsBySubmissionID()` | Get version history |
| `Create()` | Create new version |
| `Update()` | Update existing version |

## Model Methods

| Method | Description |
|--------|-------------|
| `CreateNewVersion()` | Create version object |
| `Approve()` | Set status to approved |
| `Send()` | Set status to sent |
| `Complete()` | Set status to completed |
| `Fail()` | Set status to failed |

## Common Patterns

### Admin Review Workflow
```go
// 1. Get current analysis
current, _ := service.GetByID(ctx, analysisID)

// 2. Create new version with edits
edits := map[string]interface{}{...}
v2, _ := service.CreateVersion(ctx, current.ID, edits)

// 3. Process and complete
// ... processing logic ...
v2.Complete()
service.Update(ctx, v2)

// 4. Admin approves
v2.Approve()
service.Update(ctx, v2)

// 5. Send to user
v2.Send()
service.Update(ctx, v2)
```

### Get Version History
```go
versions, _ := service.GetAllVersions(ctx, submissionID)
for _, v := range versions {
    fmt.Printf("v%d: %s (%s)\n",
        v.Version,
        v.Status,
        v.CreatedAt.Format("2006-01-02"))
}
```

## SQL Queries

### Get Latest
```sql
SELECT * FROM analyses
WHERE submission_id = $1
ORDER BY version DESC
LIMIT 1;
```

### Get All Versions
```sql
SELECT * FROM analyses
WHERE submission_id = $1
ORDER BY version DESC;
```

### Get Version Chain
```sql
WITH RECURSIVE chain AS (
    SELECT * FROM analyses WHERE id = $1
    UNION ALL
    SELECT a.* FROM analyses a
    JOIN chain c ON a.id = c.parent_analysis_id
)
SELECT * FROM chain ORDER BY version;
```

## Migration

### Apply
```bash
psql -d dbname -f migrations/010_add_analysis_versioning.sql
```

### Verify
```sql
\d analyses
-- Check for version and parent_analysis_id columns
```

## Files Modified

- `domain/analysis/model.go` - Version fields and methods
- `domain/analysis/service.go` - Version creation logic
- `domain/analysis/repository.go` - Version queries
- `migrations/010_add_analysis_versioning.sql` - Schema changes

## Documentation

- Full Guide: `docs/analysis-versioning.md`
- Implementation: `docs/implementation-summary-analysis-versioning.md`
- Quick Ref: This file

## Tips

💡 Always use `GetLatestVersion()` for current data
💡 Only admins should create versions
💡 Never modify existing versions
💡 Version number auto-increments
💡 Edits are optional (can pass nil)
