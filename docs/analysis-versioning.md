# Analysis Versioning System

## Overview

The analysis versioning system allows tracking multiple versions of strategic analyses for a single submission. This is useful when admin reviews require modifications or when iterative improvements are needed.

## Architecture

### Database Schema

The `analyses` table includes two versioning fields:

- `version` (INTEGER): Sequential version number, starts at 1
- `parent_analysis_id` (UUID): Reference to the previous version (NULL for v1)

### Status Flow

```
pending → processing → completed → approved → sent
                           ↓
                        failed
```

**New Statuses:**
- `approved`: Admin has reviewed and approved the analysis
- `sent`: Analysis has been made available to the user

## API Usage

### 1. Create a New Version

Creates a new version of an existing analysis with optional edits:

```go
// Example: Create version 2 with edits to SWOT analysis
edits := map[string]interface{}{
    "swot": map[string]interface{}{
        "strengths": []string{"Updated strength 1", "Updated strength 2"},
        "summary": "Updated SWOT summary",
    },
}

newVersion, err := analysisService.CreateVersion(ctx, originalAnalysisID, edits)
```

**Response:**
```json
{
    "id": "new-uuid",
    "version": 2,
    "parent_analysis_id": "original-uuid",
    "status": "pending",
    "submission_id": "submission-uuid",
    // ... all framework data copied from parent with edits applied
}
```

### 2. Get Latest Version

Retrieves the most recent version for a submission:

```go
latestAnalysis, err := analysisService.GetLatestVersion(ctx, submissionID)
```

### 3. Get All Versions

Retrieves version history for a submission:

```go
allVersions, err := analysisService.GetAllVersions(ctx, submissionID)
// Returns slice ordered by version DESC
```

### 4. Update Analysis Status

Use the model methods to transition between statuses:

```go
analysis.Complete()   // Sets status to "completed"
analysis.Approve()    // Sets status to "approved"
analysis.Send()       // Sets status to "sent"
analysis.Fail()       // Sets status to "failed"
```

## Edit Structure

Edits are applied as nested maps matching the framework structure. Supported frameworks:

### PESTEL Edits
```json
{
    "pestel": {
        "political": ["item1", "item2"],
        "economic": ["item1", "item2"],
        "social": ["item1", "item2"],
        "technological": ["item1", "item2"],
        "environmental": ["item1", "item2"],
        "legal": ["item1", "item2"],
        "summary": "Updated summary"
    }
}
```

### Porter's Five Forces Edits
```json
{
    "porter": {
        "competitive_rivalry": "Updated text",
        "supplier_power": "Updated text",
        "buyer_power": "Updated text",
        "threat_new_entrants": "Updated text",
        "threat_substitutes": "Updated text",
        "overall_attractiveness": "Updated text",
        "summary": "Updated summary"
    }
}
```

### SWOT Edits
```json
{
    "swot": {
        "strengths": ["item1", "item2"],
        "weaknesses": ["item1", "item2"],
        "opportunities": ["item1", "item2"],
        "threats": ["item1", "item2"],
        "summary": "Updated summary"
    }
}
```

## Workflow Example

### Admin Review Flow

1. **Initial Analysis Generated**
   ```
   Version 1: status = "completed"
   ```

2. **Admin Reviews and Requests Changes**
   ```go
   edits := map[string]interface{}{
       "swot": map[string]interface{}{
           "strengths": updatedStrengths,
       },
   }
   v2, _ := service.CreateVersion(ctx, v1.ID, edits)
   // Version 2: status = "pending", parent_analysis_id = v1.ID
   ```

3. **Version 2 Processed**
   ```go
   v2.Complete()
   service.Update(ctx, v2)
   // Version 2: status = "completed"
   ```

4. **Admin Approves**
   ```go
   v2.Approve()
   service.Update(ctx, v2)
   // Version 2: status = "approved"
   ```

5. **Sent to User**
   ```go
   v2.Send()
   service.Update(ctx, v2)
   // Version 2: status = "sent"
   ```

## Repository Methods

### Required Interface Methods

```go
type Repository interface {
    Create(ctx, analysis) error
    Update(ctx, analysis) error
    GetByID(ctx, id) (*Analysis, error)
    GetBySubmissionID(ctx, submissionID) (*Analysis, error)
    GetLatestVersionBySubmissionID(ctx, submissionID) (*Analysis, error)
    GetAllVersionsBySubmissionID(ctx, submissionID) ([]*Analysis, error)
    List(ctx, limit, offset) ([]*Analysis, error)
    Delete(ctx, id) error
}
```

## Model Methods

### CreateNewVersion()

Creates a new version of the analysis:

```go
newAnalysis := currentAnalysis.CreateNewVersion()
// Returns new Analysis with:
// - Version incremented by 1
// - ParentAnalysisID set to current ID
// - All framework data copied
// - Status reset to "pending"
// - Timestamps reset
```

## Database Queries

### Get Latest Version
```sql
SELECT * FROM analyses
WHERE submission_id = $1
ORDER BY version DESC, created_at DESC
LIMIT 1
```

### Get All Versions
```sql
SELECT * FROM analyses
WHERE submission_id = $1
ORDER BY version DESC, created_at DESC
```

### Get Version Chain
```sql
WITH RECURSIVE version_chain AS (
    -- Start with the latest version
    SELECT * FROM analyses WHERE id = $1
    UNION ALL
    -- Get parent versions
    SELECT a.* FROM analyses a
    INNER JOIN version_chain vc ON a.id = vc.parent_analysis_id
)
SELECT * FROM version_chain ORDER BY version ASC;
```

## Migration

Run the migration to add versioning support:

```bash
# Apply migration
psql -d your_database -f migrations/010_add_analysis_versioning.sql
```

The migration:
- Adds `version` column (default 1)
- Adds `parent_analysis_id` column
- Adds foreign key constraint
- Updates status constraint
- Creates indexes for performance

## Performance Considerations

### Indexes
- `idx_analyses_version`: Optimizes version queries by submission_id
- `idx_analyses_parent_id`: Optimizes parent lookups

### Best Practices
1. Always use `GetLatestVersion()` for current data
2. Use `GetAllVersions()` only when version history is needed
3. Consider archiving old versions after a certain period
4. Monitor version chain depth

## Testing

Example test cases:

```go
func TestCreateVersion(t *testing.T) {
    // Create initial analysis (v1)
    v1 := createTestAnalysis()

    // Create version 2 with edits
    edits := map[string]interface{}{
        "swot": map[string]interface{}{
            "strengths": []string{"New strength"},
        },
    }
    v2, err := service.CreateVersion(ctx, v1.ID, edits)

    // Assertions
    assert.NoError(t, err)
    assert.Equal(t, 2, v2.Version)
    assert.Equal(t, v1.ID, *v2.ParentAnalysisID)
    assert.Equal(t, "New strength", v2.SWOT.Strengths[0])
}
```

## Future Enhancements

1. **Diff Tracking**: Track what changed between versions
2. **Version Labels**: Add custom labels/tags to versions
3. **Rollback**: Ability to rollback to previous version
4. **Branching**: Support multiple version branches
5. **Merge**: Merge changes from different branches
6. **Audit Log**: Detailed change history with user attribution

## Frontend Integration

### Get Latest Analysis
```typescript
const analysis = await fetch(`/api/v3/analyses/latest/${submissionId}`)
```

### Get Version History
```typescript
const versions = await fetch(`/api/v3/analyses/versions/${submissionId}`)
```

### Create New Version
```typescript
const newVersion = await fetch(`/api/v3/analyses/${analysisId}/version`, {
    method: 'POST',
    body: JSON.stringify({ edits })
})
```

## Security Considerations

1. **Authorization**: Only admins can create new versions
2. **Validation**: Validate all edits before applying
3. **Immutability**: Never modify existing versions, always create new ones
4. **Audit Trail**: Log all version creation events
