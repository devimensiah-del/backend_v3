# Integration Tests Setup Guide

## Quick Start (Recommended)

### Option 1: Run with Test Skips (No Setup Needed)

```bash
cd backend_v3/integration_tests
go test -v
```

**Result**: Tests that require a database will be **automatically skipped** with a clear message.

### Option 2: Set Up Local Test Database (Full Testing)

#### Step 1: Install PostgreSQL

**Windows:**
```powershell
# Download installer from https://www.postgresql.org/download/windows/
# Or use Chocolatey
choco install postgresql
```

**Mac:**
```bash
brew install postgresql
brew services start postgresql
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get update
sudo apt-get install postgresql postgresql-contrib
sudo systemctl start postgresql
```

#### Step 2: Create Test Database

```bash
# Connect to PostgreSQL
psql -U postgres

# Create test database
CREATE DATABASE imensiah_test;

# Exit
\q
```

#### Step 3: Run Migrations

```bash
cd backend_v3

# Run all migration files in order
psql -U postgres -d imensiah_test -f migrations/001_initial_schema.sql
psql -U postgres -d imensiah_test -f migrations/002_add_enrichment.sql
# ... run all migration files
```

**Or use a migration script:**

```bash
# Create a simple migration runner
for file in migrations/*.sql; do
    echo "Running $file..."
    psql -U postgres -d imensiah_test -f "$file"
done
```

#### Step 4: Configure Environment

**Option A: Create `.env` in `backend_v3/` directory:**

```bash
cd backend_v3
cat > .env << EOF
DATABASE_URL=postgres://postgres:postgres@localhost:5432/imensiah_test?sslmode=disable
EOF
```

**Option B: Copy example file:**

```bash
cd backend_v3/integration_tests
cp .env.test.example ../.env
# Edit ../.env with your database credentials
```

#### Step 5: Run Tests

```bash
cd backend_v3/integration_tests
go test -v
```

## Understanding Test Modes

### 1. Database Required Tests (Skipped Without DB)

**Tests that need a database:**
- `TestEndToEndWorkflow_SubmissionToEnrichment`
- `TestEnrichmentStatusTransitions`
- `TestDatabaseOperations`

**When skipped:**
```
--- SKIP: TestEndToEndWorkflow_SubmissionToEnrichment (0.00s)
    helpers.go:38: Skipping test: No valid DATABASE_URL configured
```

### 2. Mock-Only Tests (Always Run)

**Tests that don't need a database:**
- `TestEndToEndWorkflow_WithMocks`
- `TestSupabaseJSONMock_*`
- All tests in `mocks/`

**These always run**, even without database configuration.

## Troubleshooting

### Problem: Tests skip with "No valid DATABASE_URL"

**Solution 1: Use mock-only tests**
```bash
go test -v -run TestEndToEndWorkflow_WithMocks
```

**Solution 2: Set up local database** (see Step 2-4 above)

### Problem: "Connection refused" or "dial tcp ... connection refused"

**PostgreSQL is not running.**

**Fix:**
```bash
# Mac
brew services start postgresql

# Linux
sudo systemctl start postgresql

# Windows
# Start PostgreSQL from Services app or:
net start postgresql-x64-14
```

### Problem: "password authentication failed"

**Wrong credentials in DATABASE_URL.**

**Fix:**

1. Check PostgreSQL user password:
```bash
psql -U postgres
\password postgres
# Enter new password
\q
```

2. Update `.env`:
```bash
DATABASE_URL=postgres://postgres:YOUR_PASSWORD@localhost:5432/imensiah_test?sslmode=disable
```

### Problem: "database 'imensiah_test' does not exist"

**Database not created.**

**Fix:**
```bash
psql -U postgres -c "CREATE DATABASE imensiah_test;"
```

### Problem: "relation 'submissions' does not exist"

**Migrations not run.**

**Fix:**
```bash
cd backend_v3
psql -U postgres -d imensiah_test -f migrations/001_initial_schema.sql
# Run all migration files
```

## Environment Configuration

### Minimal (Mock Mode)

**No `.env` file needed!** Tests use mocks automatically.

```bash
# Just run tests
go test -v
```

**Result**:
- Database tests: SKIPPED
- Mock tests: RUN

### With Database (Partial Real Mode)

**Create `backend_v3/.env`:**

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5432/imensiah_test?sslmode=disable
```

**Result**:
- Database tests: RUN
- LLM tests: MOCK (no API key)
- Storage tests: MOCK (no Supabase URL)

### Full Real Mode (All Services)

**Create `backend_v3/.env`:**

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5432/imensiah_test?sslmode=disable
OPENROUTER_API_KEY=sk-or-your-actual-key
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key
```

**Result**:
- Database tests: RUN (real database)
- LLM tests: RUN (real OpenRouter API)
- Storage tests: RUN (real Supabase)

## CI/CD Setup

### GitHub Actions

```yaml
name: Integration Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: imensiah_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

    steps:
      - uses: actions/checkout@v3

      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run Migrations
        run: |
          for file in backend_v3/migrations/*.sql; do
            PGPASSWORD=postgres psql -h localhost -U postgres -d imensiah_test -f "$file"
          done

      - name: Run Integration Tests
        env:
          DATABASE_URL: postgres://postgres:postgres@localhost:5432/imensiah_test?sslmode=disable
        run: |
          cd backend_v3/integration_tests
          go test -v -timeout 5m
```

## Docker Setup (Alternative)

### Using Docker Compose

**Create `docker-compose.test.yml`:**

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: imensiah_test
    ports:
      - "5432:5432"
    volumes:
      - ./migrations:/docker-entrypoint-initdb.d
```

**Run tests:**

```bash
# Start PostgreSQL
docker-compose -f docker-compose.test.yml up -d

# Wait for PostgreSQL to be ready
sleep 5

# Run tests
DATABASE_URL=postgres://postgres:postgres@localhost:5432/imensiah_test?sslmode=disable \
  go test -v ./backend_v3/integration_tests/...

# Stop PostgreSQL
docker-compose -f docker-compose.test.yml down
```

## Test Database Cleanup

### Between Test Runs

Tests automatically clean up data created in the last hour:

```go
defer helper.Cleanup(t)  // Removes test data
```

### Manual Cleanup

```bash
# Reset entire test database
psql -U postgres -d imensiah_test -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# Re-run migrations
cd backend_v3
for file in migrations/*.sql; do
    psql -U postgres -d imensiah_test -f "$file"
done
```

### Separate Test Database

**Best practice**: Use a dedicated test database, not your development database!

```bash
# Development
DATABASE_URL=postgres://postgres:postgres@localhost:5432/imensiah_dev

# Testing
DATABASE_URL=postgres://postgres:postgres@localhost:5432/imensiah_test
```

## Summary

### No Setup (Mock Mode)
```bash
cd backend_v3/integration_tests
go test -v
# Database tests skipped, mock tests run
```

### With Local Database (Recommended)
```bash
# 1. Create database
psql -U postgres -c "CREATE DATABASE imensiah_test;"

# 2. Run migrations
cd backend_v3
for file in migrations/*.sql; do
    psql -U postgres -d imensiah_test -f "$file"
done

# 3. Configure
echo "DATABASE_URL=postgres://postgres:postgres@localhost:5432/imensiah_test?sslmode=disable" > .env

# 4. Run tests
cd integration_tests
go test -v
```

### Quick Reference

| Environment | Database Tests | Mock Tests | Setup Required |
|-------------|---------------|------------|----------------|
| None | SKIP | RUN | ❌ No |
| With DB | RUN | RUN | ✅ Yes |
| With DB + APIs | RUN (real) | RUN (real) | ✅ Yes + API keys |

**Recommendation**: Start with no setup (mock mode), add database when needed.
