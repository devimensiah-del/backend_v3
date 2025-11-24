# API Contract – Frontend Expectations (v1.0 Final)

> **Last Updated**: 2024-11-24
> **Base URL**: `/api/v1`

Detailed request/response shapes for all API endpoints under `/api/v1`, with the submission → enrichment → analysis → report workflow enforced as described. Types are expressed as JSON object fields with primitive types; arrays are `[]`; timestamps are ISO 8601 strings.

---

## Auth
- `POST /auth/login`
  - Request: `{ "email": string, "password": string }`
  - Response `200`:
    ```json
    {
      "user": { "id": string, "email": string, "role": string },
      "access_token": string,
      "token_type": "Bearer",
      "expires_in": number,
      "expires_at": number
    }
    ```
- `POST /auth/signup`
  - Request: `{ "email": string, "password": string, "fullName": string }`
  - Response `201`:
    ```json
    {
      "user": { "id": string, "email": string },
      "access_token": string,
      "token_type": "Bearer",
      "expires_in": number
    }
    ```
- `POST /auth/logout`
  - Request: Bearer token only
  - Response `200`: `{ "message": string }`
- `POST /auth/forgot-password`
  - Request: `{ "email": string }`
  - Response `200`: `{ "message": string }`
- `POST /auth/reset-password`
  - Request: `{ "token": string, "newPassword": string }`
  - Response `200`: `{ "message": string }`
- `PUT /auth/update-password`
  - Request: Bearer token; `{ "currentPassword": string, "newPassword": string }`
  - Response `200`: `{ "message": string }`
- `GET /auth/me`
  - Request: Bearer token
  - Response `200`: `{ "user": UserProfile }`

`UserProfile`:
```json
{
  "id": string,
  "email": string,
  "fullName": string,
  "role": "user" | "admin" | "super_admin" | "service_role",
  "isActive": boolean,
  "createdAt": string,
  "updatedAt": string
}
```

## Public
- `POST /submissions` (alias `/submit`)
  - Request (maps to `CreateSubmissionRequest` + `AdditionalInfoData`):
```json
{
  "companyName": string,          // required
  "cnpj": string,                 // optional, backend defaults to "00.000.000/0000-00" if empty
  "industry": string,             // optional, default "Não especificado"
  "companySize": string,          // optional, default "Não especificado"
  "strategicGoal": string,        // optional, default "Em definição"
  "currentChallenges": string,    // optional
  "competitivePosition": string,  // optional, default "Em análise"
  "website": string|null,         // optional
  "additionalInfo": string|null   // optional JSON string with the object below
}
```
`additionalInfo` JSON string parses to:
```json
{
  "contactName": string,          // required
  "contactEmail": string,         // required, email
  "contactPhone": string,         // optional
  "contactPosition": string,      // optional
  "companyLocation": string,      // optional
  "targetMarket": string,         // optional
  "annualRevenueMin": number|null,// optional
  "annualRevenueMax": number|null,// optional
  "fundingStage": string,         // optional
  "additionalNotes": string,      // optional
  "linkedinUrl": string,          // optional
  "twitterHandle": string         // optional
}
```
  - Response `201`: `SubmissionSummary`

`SubmissionSummary`:
```json
{
  "id": string,
  "status": "received",           // always received on create
  "createdAt": string,
  "workflow": {
    "enrichmentId"?: string,
    "enrichmentStatus"?: EnrichmentStatus,
    "analysisId"?: string,
    "analysisStatus"?: AnalysisStatus,
    "reportStatus"?: ReportStatus
  }
}
```

## User (auth required)
- `GET /submissions`
  - Response `200`: `{ "items": SubmissionListItem[], "page": number, "pageSize": number, "total": number }`

`SubmissionListItem` mirrors `SubmissionSummary`.

- `GET /submissions/:id`
  - Response `200`: `SubmissionDetail` (IDs only; fetch enrichment/analysis/report via their endpoints)

`SubmissionDetail` (as returned today):
```json
{
  "id": string,
  "status": "received",
  "createdAt": string,
  "updatedAt": string,
  "companyName": string,
  "cnpj": string|null,
  "companyWebsite": string|null,
  "companyIndustry": string|null,
  "companySize": string|null,
  "companyLocation": string|null,
  "contactName": string,
  "contactEmail": string,
  "contactPhone": string|null,
  "contactPosition": string|null,
  "targetMarket": string|null,
  "annualRevenueMin": number|null,
  "annualRevenueMax": number|null,
  "fundingStage": string|null,
  "businessChallenge": string,
  "additionalNotes": string|null,
  "linkedinUrl": string|null,
  "twitterHandle": string|null,
  "userId": string|null,
  "enrichmentId": string|null,
  "analysisId": string|null,
  "reportId": string|null,
  "pdfUrl": string|null
}
```

- `GET /submissions/:id/enrichment`
  - Response `200`:
    ```json
    {
      "enrichment": {
        "id": string,
        "submissionId": string,
        "status": "pending" | "completed" | "approved",
        "progress": number,           // 0-100
        "currentStep": string,        // e.g., "Enrichment complete - awaiting review"
        "data": EnrichmentContent,    // Note: backend field is "enrichedData", mapped to "data"
        "isLocked": boolean,
        "createdAt": string,
        "updatedAt": string
      }
    }
    ```
- `GET /submissions/:id/analysis`
  - Response `200`:
    ```json
    {
      "analysis": {
        "id": string,
        "submissionId": string,
        "status": "pending" | "completed" | "approved" | "sent",
        "analysis": AnalysisContent,  // Contains all 12 frameworks
        "createdAt": string,
        "updatedAt": string
      }
    }
    ```
- `GET /submissions/:id/report/preview`
  - Response `200`: `ReportPreview`
- `GET /submissions/:id/report/download`
  - Response `200`: binary PDF stream (content-disposition attachment)
- `PUT /user/profile`
  - Request: `{ "fullName"?: string }`
  - Response `200`: `{ "user": UserProfile }`
- `DELETE /user`
  - Request: Bearer token only
  - Response `200`: `{ "message": string }`

## Admin (auth + admin role)
- `GET /admin/submissions`
  - Query params: `status?`, `email?`, `page?`, `pageSize?`
  - Response `200`: `{ "items": SubmissionListItem[], "page": number, "pageSize": number, "total": number }`
- `GET /admin/submissions/:id`
  - Response `200`: `SubmissionDetail`
- `GET /admin/submissions/:id/enrichment`
  - Response `200`: `Enrichment`
- `POST /admin/submissions/:id/retry-enrichment`
  - Response `202`: `{ "enrichmentId": string, "status": "pending" }`
- `POST /admin/submissions/:id/retry-analysis`
  - Response `202`: `{ "analysisId": string, "status": "pending", "version": string }`
- `GET /admin/enrichment/:id`
  - Response `200`: `Enrichment`
- `PUT /admin/enrichment/:id`
  - **IMPORTANT**: Only allowed when `status === "completed"`. Returns 400 if `status === "approved"`.
  - Request: `Partial<EnrichmentContent>` (any fields from enrichedData)
  - Response `200`: `{ "enrichment": Enrichment }`
  - Error `400`: `{ "error": "Update failed", "message": "Cannot edit enrichment after approval" }`
- `POST /admin/enrichment/:id/approve`
  - Triggers analysis job. Changes status: `completed → approved`
  - Response `200`: `{ "enrichment": Enrichment, "message": "Enrichment approved, analysis job started" }`
- `POST /admin/enrichment/:id/unlock`
  - Response `200`: `{ "enrichmentId": string, "status": EnrichmentStatus }` (remains unchanged; unlock toggles editability)
- `GET /admin/analysis/:id`
  - Response `200`: `Analysis`
- `PUT /admin/analysis/:id`
  - **IMPORTANT**: Only allowed when `status === "completed"`.
  - Cannot edit when: `pending` (AI processing), `approved` (PDF generated), `sent` (delivered to user)
  - Request: `Partial<AnalysisContent>` (any framework fields)
    ```json
    {
      "pestel": { "political": ["Updated point 1", "Updated point 2"] },
      "swot": { "strengths": [{ "content": "New strength", "confidence": "Alta", "source": "Admin" }] }
    }
    ```
  - Response `200`: `{ "analysis": Analysis }`
  - Error `400`: `{ "error": "Update failed", "message": "Cannot edit analysis while AI is still processing" }`
- `POST /admin/analysis/:id/approve`
  - Triggers PDF generation job. Changes status: `completed → approved`
  - Response `200`: `{ "analysis": Analysis, "message": "Analysis approved successfully" }`
- `POST /admin/analysis/:id/send`
  - **Prerequisite**: PDF must exist (status `approved` + PDF generated)
  - Changes status: `approved → sent`. Triggers user notification.
  - Request: `{ "userEmail": string }` (required)
  - Response `200`: `{ "analysis": Analysis, "message": "Analysis sent to user successfully" }`
  - Error `400`: `{ "error": "Send failed", "message": "PDF não está disponível ainda" }`
- `GET /admin/analytics`
  - Response `200`: `{ "summary": object }` (implementation-defined)

## Domain Models

### Submission
```json
{
  "id": string,
  "status": "received",           // submission status does not change
  "createdAt": string,
  "updatedAt": string,
  "companyName": string,
  "cnpj": string|null,
  "companyWebsite": string|null,
  "companyIndustry": string|null,
  "companySize": string|null,
  "companyLocation": string|null,
  "contactName": string,
  "contactEmail": string,
  "contactPhone": string|null,
  "contactPosition": string|null,
  "targetMarket": string|null,
  "annualRevenueMin": number|null,
  "annualRevenueMax": number|null,
  "fundingStage": string|null,
  "businessChallenge": string,     // concatenated strategicGoal | currentChallenges | competitivePosition
  "additionalNotes": string|null,
  "linkedinUrl": string|null,
  "twitterHandle": string|null,
  "userId": string|null
}
```

### Enrichment
```json
{
  "id": string,
  "submissionId": string,
  "status": EnrichmentStatus,     // pending -> completed -> approved
  "currentStep"?: string,
  "content": EnrichmentContent,   // returned as "data" key in responses
  "notes"?: string,
  "createdAt": string,
  "updatedAt": string
}
```

`EnrichmentStatus = "pending" | "completed" | "approved"`  
`EnrichmentContent` is a structured map with the following keys (all strings unless noted):
```json
{
  "profile_overview": {
    "legal_name": string,
    "website": string,
    "foundation_year": string,
    "headquarters": string
  },
  "market_position": {
    "sector": string,
    "target_audience": string,
    "value_proposition": string
  },
  "financials": {
    "employees_range": string,
    "revenue_estimate": string,
    "business_model": string
  },
  "competitive_landscape": {
    "competitors": string[],
    "market_share_status": string
  },
  "strategic_assessment": {
    "digital_maturity": number,      // int
    "strengths": string[],
    "weaknesses": string[]
  },
  "macro_context": {                 // optional; present when available
    "economic_indicators": {
      "country": string,
      "gdp_growth": string,
      "inflation_rate": string,
      "interest_rate": string,
      "exchange_rate": string,
      "unemployment_rate": string,
      "political_stability": string,
      "economic_outlook": string,
      "recent_policy_changes": string[]
    },
    "industry_trends": {
      "industry_sector": string,
      "growth_rate": string,
      "key_trends": string[],
      "technology_adoption": string,
      "market_concentration": string,
      "barriers_to_entry": string,
      "mergers_acquisitions": string[]
    },
    "regulatory_landscape": {
      "recent_regulations": string[],
      "upcoming_changes": string[],
      "compliance_requirements": string,
      "industry_standards": string[]
    },
    "market_signals": {
      "commodity_prices": string[],
      "supply_chain_status": string,
      "consumer_sentiment": string,
      "competitor_activity": string[],
      "emerging_threats": string[]
    },
    "data_sources": string[],
    "last_updated": string            // timestamp
  },
  "sourcesStatus": object,           // map of source -> status strings
  "sourcesUsed": string|null,        // comma- or array-encoded in backend
  "enrichmentScore": number|null,
  "processingTimeMs": number|null
}
```
Frontend responses expose `content` under the `data` property.

### Analysis

**Note**: Analysis does NOT have versioning. Each submission has exactly one analysis record.

```json
{
  "id": string,
  "submissionId": string,
  "enrichmentId": string,
  "status": AnalysisStatus,       // pending -> completed -> approved -> sent
  "createdAt": string,
  "updatedAt": string,
  "completedAt": string|null,
  "sentAt": string|null,
  "sentTo": string|null,          // Email address when sent
  "errorMessage": string|null,    // Set if processing failed
  "processingTimeMs": number|null,
  // Framework data (PESTEL, Porter, SWOT, etc.)
  ...AnalysisContent
}
```

`AnalysisStatus = "pending" | "completed" | "approved" | "sent"`

**Status Transition Rules:**
- `pending`: AI is processing. **No edits allowed.**
- `completed`: AI finished. **Admin can edit.** Can approve.
- `approved`: PDF generating/generated. **No edits allowed.** Can send.
- `sent`: Delivered to user. **No edits allowed.** Terminal state.

`AnalysisContent` contains the 12 frameworks and synthesis, each with the fields below (strings unless specified):
```json
{
  "pestel": {
    "political": string[],
    "economic": string[],
    "social": string[],
    "technological": string[],
    "environmental": string[],
    "legal": string[],
    "summary": string
  },
  "porter": {
    "competitive_rivalry": string,
    "supplier_power": string,
    "buyer_power": string,
    "threat_new_entrants": string,
    "threat_substitutes": string,
    "power_partnerships_ecosystems": string,
    "disruption_ai_data": string,
    "competitive_rivalry_intensity": string,
    "supplier_power_intensity": string,
    "buyer_power_intensity": string,
    "threat_new_entrants_intensity": string,
    "threat_substitutes_intensity": string,
    "power_partnerships_ecosystems_intensity": string,
    "disruption_ai_data_intensity": string,
    "strategic_implications": string[],
    "overall_attractiveness": string,
    "summary": string
  },
  "tam_sam_som": {
    "tam": string,
    "sam": string,
    "som": string,
    "assumptions": string[],
    "cagr": string,
    "data_quality": string,       // "complete"|"partial"|"insufficient"
    "next_steps": string[],
    "proxy_indicators": string[],
    "expected_outputs": string[],
    "methodological_note": string,
    "summary": string
  },
  "swot": {
    "strengths": SWOTItem[],
    "weaknesses": SWOTItem[],
    "opportunities": SWOTItem[],
    "threats": SWOTItem[],
    "summary": string
  },
  "benchmarking": {
    "competitors_analyzed": string[],
    "performance_gaps": string[],
    "best_practices": string[],
    "summary": string
  },
  "blue_ocean": {
    "eliminate": string[],
    "reduce": string[],
    "raise": string[],
    "create": string[],
    "new_value_curve": string,
    "summary": string
  },
  "growth_hacking": {
    "leap_loop": GrowthLoop,
    "scale_loop": GrowthLoop,
    "summary": string
  },
  "scenarios": {
    "optimistic": Scenario,
    "realist": Scenario,
    "pessimistic": Scenario,
    "mitigation_tactics": string[],
    "early_warning_signals": string[],
    "summary": string
  },
  "okrs": {
    "quarters": QuarterlyOKR[],
    "summary": string
  },
  "bsc": {
    "financial": string[],
    "customer": string[],
    "internal_processes": string[],
    "learning_growth": string[],
    "summary": string
  },
  "decision_matrix": {
    "alternatives": string[],
    "criteria": string[],
    "final_recommendation": string,
    "recommended_option": string,
    "score": string,
    "score_comparison": string,
    "priority_recommendations": PriorityRecommendation[],
    "review_cycle": ReviewCycle,
    "monitoring_metrics": string[],
    "summary": string
  },
  "synthesis": {
    "executive_summary": string,
    "central_challenge": string,
    "main_findings": string[],
    "important_notes": string[],
    "key_findings": string[],
    "strategic_priorities": string[],
    "roadmap": string[],
    "overall_recommendation": string
  }
}
```
`SWOTItem { "content": string, "confidence": "Alta" | "Média" | "Baixa", "source": string }`
`GrowthLoop { "name": string, "type": string, "steps": string[], "metrics": string[], "bottleneck": string }`
`Scenario { "name": string, "probability": number, "description": string, "required_actions": string[] }`
`QuarterlyOKR { "quarter": string, "objective": string, "key_results": string[], "investment": string, "timeline": string }`
`PriorityRecommendation { "priority": number, "title": string, "description": string, "timeline": string, "budget": string }`
`ReviewCycle { "frequency": string, "extraordinary_triggers": string[] }`

### Report
```json
{
  "id": string,
  "submissionId": string,
  "analysisVersion": string,
  "status": ReportStatus,         // pending | processing | completed | failed
  "contentUrl"?: string,          // pdf_url
  "pdfGenerationStatus"?: string, // pending | processing | completed | failed
  "pages": ReportPages?,          // when previewing
  "createdAt": string,
  "updatedAt": string
}
```

`ReportStatus = "pending" | "processing" | "completed" | "failed"`  
`ReportPages` map each section to HTML strings; page keys align to the fields below.

`Report` entity contains HTML for each page:
```json
{
  "cover_page": string,
  "executive_summary": string,
  "table_of_contents": string,
  "divider_part1_page": string,
  "pestel_pes_page": string,
  "pestel_tel_page": string,
  "porter_page": string,
  "swot_page": string,
  "divider_part2_page": string,
  "tam_sam_som_page": string,
  "blue_ocean_page": string,
  "divider_part3_page": string,
  "okr_page": string,
  "growth_loops_page": string,
  "divider_part4_page": string,
  "scenarios_page": string,
  "recommendations_page": string,
  "bsc_page": string,
  "benchmarking_page": string,
  "financial_projections_page": string,
  "growth_hacking_page": string,
  "risk_assessment_page": string,
  "roadmap_page": string,
  "appendix_page": string,
  "pdf_url": string,                 // signed URL (time-limited) to Supabase Storage
  "pdf_generated_at": string|null,
  "pdf_generation_status": string,
  "generation_time_ms": number,
  "total_pages": number,
  "completed_at": string|null
}
```

### ReportPreview
```json
{
  "id": string,
  "analysisVersion": string,
  "html": string,
  "generatedAt": string
}
```

## Workflow & Status Logic (frontend expectations)

### Complete Workflow Diagram

```
[User Submits Form]
        ↓
[Submission Created] status: "received" (never changes)
        ↓
[Enrichment Job Queued] status: "pending", progress: 0-100
        ↓ (AI Worker)
[Enrichment Completed] status: "completed", progress: 100
        ↓ (Admin Review & Edit)
[Admin Approves Enrichment] status: "approved"
        ↓
[Analysis Job Triggered] status: "pending"
        ↓ (AI Worker - 12 frameworks)
[Analysis Completed] status: "completed"
        ↓ (Admin Review & Edit)
[Admin Approves Analysis] status: "approved"
        ↓
[Report/PDF Job Triggered]
        ↓ (Gotenberg PDF generation)
[PDF Ready]
        ↓
[Admin Sends to User] status: "sent"
        ↓
[User Downloads PDF]
```

### Status Rules Summary

| Entity | Statuses | Editable When | Actions |
|--------|----------|---------------|---------|
| Submission | `received` | Never changes | - |
| Enrichment | `pending` → `completed` → `approved` | `completed` only | Approve triggers Analysis |
| Analysis | `pending` → `completed` → `approved` → `sent` | `completed` only | Approve triggers PDF, Send notifies user |
| Report | `pending` → `processing` → `completed` | Never | Download when completed |

### Derived Status for UI Display

```typescript
function getDerivedStatus(submission: SubmissionDetail): string {
  // Check analysis status first (later in workflow)
  if (submission.analysisId) {
    const analysis = await getAnalysis(submission.id);
    switch (analysis.status) {
      case 'sent': return 'delivered';      // User can download
      case 'approved': return 'ready';      // PDF generating/ready
      case 'completed': return 'review';    // Awaiting admin review
      case 'pending': return 'analyzing';   // AI processing
    }
  }

  // Check enrichment status
  if (submission.enrichmentId) {
    const enrichment = await getEnrichment(submission.id);
    switch (enrichment.status) {
      case 'approved': return 'analyzing';  // Analysis should start
      case 'completed': return 'enriched';  // Awaiting admin approval
      case 'pending': return 'enriching';   // AI processing
    }
  }

  return 'received';  // Just submitted
}
```

### Progress Indicators

**Enrichment Progress (0-100):**
- `0-30`: Gathering company data
- `30-60`: Market & competitor analysis
- `60-90`: Finalizing insights
- `100`: Complete

**Analysis Progress (derived from frameworks completed):**
- Each of 12 frameworks = ~8% progress
- Layer 1 (PESTEL, Porter, TAM): 0-25%
- Layer 2 (SWOT, Benchmarking): 25-42%
- Layer 3 (Blue Ocean, Growth, Scenarios): 42-67%
- Layer 4 (OKRs, BSC, Decision Matrix): 67-92%
- Synthesis: 92-100%

## Error Shape
All error responses: HTTP status code with body:
```json
{
  "error": string,      // Error type (e.g., "Not found", "Forbidden")
  "message": string     // Human-readable description
}
```

### Common Error Codes

| Status | When |
|--------|------|
| 400 | Validation failed, invalid status transition |
| 401 | Missing or invalid auth token |
| 403 | User doesn't own resource / not admin |
| 404 | Resource not found |
| 429 | Rate limited (auth: 5/15min, general: 100/min) |
| 500 | Server error |

---

## Changelog

- **v1.0** (2024-11-24): Finalized contract
  - Removed analysis versioning (single record per submission)
  - Added edit protection rules (only editable when `completed`)
  - Fixed auth response to include `access_token`
  - Added derived status logic for UI
  - Added progress indicators
  - Clarified enrichment `data` field mapping
