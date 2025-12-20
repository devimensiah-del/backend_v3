# Backend V3 - Business Intelligence Platform

**Production-Ready Go Backend** with parallel processing, background jobs, and comprehensive business analysis.

## 🚀 Features

- **4-Stage Workflow**: Submission → Enrichment → Analysis → Report
- **Parallel Analysis**: 7 frameworks processed simultaneously (2.8-4.4x faster)
- **Background Jobs**: Asynq-powered asynchronous processing
- **13-Page Reports**: Comprehensive HTML reports with PDF generation
- **RESTful API**: Clean API with Gin framework
- **PostgreSQL + Redis**: Robust data persistence and job queue
- **Structured Logging**: Production-ready logging with zerolog
- **Health Checks**: Monitoring endpoints for DB and Redis

## 📋 Architecture

```
backend_v3/
├── api/              # HTTP handlers, middleware, routing
├── domain/           # Business logic organized by entity
│   ├── submission/   # Submission management
│   ├── enrichment/   # External data enrichment
│   ├── analysis/     # 7-framework business analysis
│   └── report/       # 13-page report generation
├── jobs/             # Background worker with Asynq
├── migrations/       # Database schema
└── main.go           # Application bootstrap
```

## 🎯 Workflow

1. **Submission** (POST /api/submit)
   - User submits business information
   - Creates submission record
   - Enqueues enrichment job

2. **Enrichment** (Background Job)
   - Gathers data from 15+ external sources
   - Enriches company, market, competitor data
   - Takes ~30-60 seconds

3. **Analysis** (Background Job)
   - **Runs 7 frameworks in PARALLEL**:
     - SWOT Analysis
     - PESTEL Analysis
     - Porter's Five Forces
     - OKRs
     - BCG Matrix
     - Value Proposition Canvas
     - Business Model Canvas
   - Generates synthesis with actionable recommendations
   - Takes ~5-10 seconds (vs ~15-20 seconds sequential)

4. **Report** (Background Job)
   - Generates 13 HTML pages:
     1. Cover Page
     2. Executive Summary
     3. Table of Contents
     4. SWOT Page
     5. PESTEL Page
     6. Porter Page
     7. OKR Page
     8. BCG Matrix Page
     9. Value Proposition Page
     10. Business Model Page
     11. Strategic Priorities
     12. Risks & Mitigation
     13. Appendix
   - Creates PDF from HTML pages
   - Uploads to cloud storage

## 🛠️ Setup Instructions

### Prerequisites

- **Go 1.21+**
- **PostgreSQL 15+**
- **Redis 7+**
- **OpenAI API Key** (or Claude API)

### 1. Clone and Install Dependencies

```bash
cd backend_v3
go mod download
```

### 2. Configure Environment

```bash
cp .env.example .env
# Edit .env with your configuration
```

**Required Environment Variables:**
- `DATABASE_URL`: PostgreSQL connection string
- `REDIS_ADDR`: Redis address
- `OPENAI_API_KEY`: OpenAI API key for analysis

### 3. Setup Database

```bash
# Create database
createdb imensiah

# Run migrations
psql -d imensiah -f migrations/001_initial_schema.sql
```

### 4. Start Services

**Option A: Run Locally**
```bash
# Terminal 1: Start Redis
redis-server

# Terminal 2: Start PostgreSQL (if not running)
postgres -D /usr/local/var/postgres

# Terminal 3: Start backend
go run main.go
```

### 5. Verify Installation

```bash
# Health check
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","services":{"database":"healthy","redis":"healthy"}}
```

## 📡 API Endpoints

### Public Endpoints

#### Create Submission
```bash
POST /api/submit
Content-Type: application/json

{
  "company_name": "Acme Corp",
  "industry_name": "Technology",
  "email": "founder@acme.com",
  "website_url": "https://acme.com",
  "annual_revenue": 5000000,
  "employee_count": 50,
  "location": "San Francisco, CA",
  "description": "B2B SaaS platform for team collaboration"
}

# Response:
{
  "id": "uuid",
  "company_name": "Acme Corp",
  "status": "pending",
  "status_message": "Submission received. Enrichment will start shortly.",
  "created_at": "2024-01-15T10:30:00Z"
}
```

#### Get Submission Status
```bash
GET /api/submissions/{id}

# Response:
{
  "id": "uuid",
  "company_name": "Acme Corp",
  "status": "analyzing",
  "status_message": "Running 7 strategic frameworks in parallel",
  "enrichment_id": "uuid",
  "enrichment_status": "completed",
  "analysis_id": "uuid",
  "analysis_status": "processing",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:32:00Z"
}
```

#### Get Report
```bash
GET /api/submissions/{id}/report

# Response:
{
  "id": "uuid",
  "status": "completed",
  "cover_page": "<html>...</html>",
  "executive_summary": "<html>...</html>",
  "table_of_contents": "<html>...</html>",
  "swot_page": "<html>...</html>",
  "pestel_page": "<html>...</html>",
  "porter_page": "<html>...</html>",
  "okr_page": "<html>...</html>",
  "bcg_matrix_page": "<html>...</html>",
  "value_proposition_page": "<html>...</html>",
  "business_model_page": "<html>...</html>",
  "strategic_priorities_page": "<html>...</html>",
  "risks_and_mitigation_page": "<html>...</html>",
  "appendix_page": "<html>...</html>",
  "pdf_url": "https://s3.amazonaws.com/reports/uuid.pdf",
  "total_pages": 13,
  "created_at": "2024-01-15T10:35:00Z",
  "completed_at": "2024-01-15T10:35:30Z"
}
```

### Admin Endpoints (Require Auth)

#### List All Submissions
```bash
GET /api/admin/submissions?page=1&limit=20
Authorization: Bearer <jwt-token>

# Response:
{
  "submissions": [...],
  "page": 1,
  "limit": 20,
  "total": 145
}
```

#### Retry Enrichment
```bash
POST /api/admin/submissions/{id}/retry-enrichment
Authorization: Bearer <jwt-token>

# Response:
{
  "message": "Enrichment retry enqueued for Acme Corp",
  "data": {
    "task_id": "uuid",
    "submission_id": "uuid"
  }
}
```

#### Retry Analysis
```bash
POST /api/admin/submissions/{id}/retry-analysis
Authorization: Bearer <jwt-token>

# Response:
{
  "message": "Analysis retry enqueued",
  "data": {
    "task_id": "uuid",
    "submission_id": "uuid"
  }
}
```

### System Endpoints

#### Health Check
```bash
GET /health

# Response:
{
  "status": "healthy",
  "services": {
    "database": "healthy",
    "redis": "healthy"
  }
}
```

## 🤝 Contributing

This is the production-ready v3 implementation. All core features are complete and well-documented.

## 📄 License

Proprietary - IMENSIAH Platform

## 🆘 Support

For issues or questions, contact the development team.

---

**Built with ❤️ using Go, PostgreSQL, Redis, and parallel goroutines**
