# Railway Deployment Troubleshooting Guide

## Issue: Deployment Crashes After Successful Startup

### Symptoms
- ✅ All 13 frameworks load successfully
- ✅ Server starts listening on port 8080
- ✅ Background worker starts
- ❌ Container crashes shortly after

## Root Cause Analysis

### 1. Health Check Failure (Most Likely)

Railway hits `/health` endpoint which pings the database. If database connection fails, Railway restarts the container.

**Check Railway Logs for:**
```
GET /health HTTP/1.1 503 Service Unavailable
```

**Fix:** Ensure Railway's PostgreSQL connection pooling is configured correctly:
1. Check Railway → Database → Settings → Connection Pooling
2. Verify `DATABASE_URL` includes `?sslmode=require`
3. Consider increasing health check timeout in `railway.json`:
   ```json
   "healthcheckTimeout": 300
   ```

### 2. Missing Environment Variables

The application uses **36+ environment variables**. Missing variables cause defaults to be used, which may not work in production.

**Required Variables (Already Set):**
- ✅ `DATABASE_URL`
- ✅ `OPENROUTER_API_KEY` (maps to OPENAI_API_KEY internally)
- ✅ `SUPABASE_JWT_SECRET`
- ✅ `SUPABASE_ANON_KEY`
- ✅ `REDIS_URL` (Railway auto-sets this)

**Critical Missing Variables to Set:**

```bash
# Server Configuration
ENV=production
ALLOWED_ORIGINS=https://your-frontend-domain.com,https://your-admin-domain.com

# External Services
GOTENBERG_URL=https://gotenberg.railway.app  # Or deploy Gotenberg on Railway
SUPABASE_URL=https://your-project.supabase.co

# Worker Configuration
WORKER_ENABLED=true
ASYNQ_CONCURRENCY=10
```

**Framework-Specific Models (Optional but Recommended):**

If not set, these default to the optimized values from `.env.example`:

```bash
# Layer 1: Environment Scanning
AI_PESTEL_MODEL=openai/o3-mini
AI_PESTEL_TEMP=0.2
AI_PESTEL_MAX_TOKENS=1500

AI_PORTER_MODEL=openai/gpt-4o
AI_PORTER_TEMP=0.3
AI_PORTER_MAX_TOKENS=1500

AI_TAM_MODEL=openai/o3-mini
AI_TAM_TEMP=0.1
AI_TAM_MAX_TOKENS=1200

# Layer 2: Positioning
AI_SWOT_MODEL=openai/gpt-4o-mini
AI_SWOT_TEMP=0.4
AI_SWOT_MAX_TOKENS=1500

AI_BENCHMARKING_MODEL=openai/gpt-4o-mini
AI_BENCHMARKING_TEMP=0.35
AI_BENCHMARKING_MAX_TOKENS=1500

# Layer 3: Strategy
AI_BLUE_OCEAN_MODEL=openai/gpt-4o
AI_BLUE_OCEAN_TEMP=0.7
AI_BLUE_OCEAN_MAX_TOKENS=1500

AI_GROWTH_HACKING_MODEL=openai/gpt-4o-mini
AI_GROWTH_HACKING_TEMP=0.6
AI_GROWTH_HACKING_MAX_TOKENS=1500

AI_SCENARIOS_MODEL=openai/gpt-4o
AI_SCENARIOS_TEMP=0.6
AI_SCENARIOS_MAX_TOKENS=1800

# Layer 4: Execution
AI_OKRS_MODEL=openai/o3-mini
AI_OKRS_TEMP=0.25
AI_OKRS_MAX_TOKENS=1500

AI_BSC_MODEL=openai/gpt-4o-mini
AI_BSC_TEMP=0.35
AI_BSC_MAX_TOKENS=1500

AI_DECISION_MATRIX_MODEL=openai/o3-mini
AI_DECISION_MATRIX_TEMP=0.2
AI_DECISION_MATRIX_MAX_TOKENS=1500

# Synthesis Layer
AI_SYNTHESIS_MODEL=openai/gpt-4o
AI_SYNTHESIS_TEMP=0.4
AI_SYNTHESIS_MAX_TOKENS=3000

# Enrichment Layer
AI_ENRICHMENT_MODEL=google/gemini-2.0-flash-001
AI_ENRICHMENT_TEMP=0.5
AI_ENRICHMENT_MAX_TOKENS=8000
```

### 3. Memory Limit Exceeded

**Symptoms:**
- Container killed with exit code 137 (OOMKilled)
- Railway shows memory usage spiking to 100%

**Fix:**
1. Check Railway → Metrics → Memory Usage
2. If hitting limits, upgrade to Railway Pro ($5/month for 1GB)
3. Or optimize memory usage:
   ```go
   // main.go:54-56
   db.SetMaxOpenConns(10)  // Reduce from 25
   db.SetMaxIdleConns(2)   // Reduce from 5
   ```

### 4. CORS Configuration Issue

If Railway's health check is being rejected by CORS middleware:

**Fix:** Update `ALLOWED_ORIGINS` to include Railway's health checker:
```bash
ALLOWED_ORIGINS=*
# Or specifically:
ALLOWED_ORIGINS=https://your-frontend.com,https://railway.app
```

### 5. Gotenberg Service Missing

The application initializes a Gotenberg client even if it's not used immediately:

```go
// main.go:79
pdfGen := infrastructure.NewGotenbergClient(cfg.GotenbergURL)
```

**Current Default:** `http://localhost:3000` (doesn't exist on Railway)

**Fix Options:**

**Option A: Deploy Gotenberg on Railway (Recommended)**
1. Railway → New Service → Docker Image
2. Image: `gotenberg/gotenberg:8`
3. Port: 3000
4. Get the service URL: `https://gotenberg-production-xxxx.up.railway.app`
5. Set environment variable: `GOTENBERG_URL=https://gotenberg-production-xxxx.up.railway.app`

**Option B: Use External Gotenberg Service**
```bash
GOTENBERG_URL=https://your-gotenberg-service.com
```

**Option C: Lazy Load Gotenberg (Code Change Required)**
Modify to only create Gotenberg client when needed instead of at startup.

## Diagnostic Commands

### 1. View Full Railway Logs

```bash
# In Railway dashboard, click on deployment → View Logs
# Look for:
# - Panic messages
# - Fatal errors
# - HTTP 503 responses on /health
# - Database connection errors
# - Out of memory (OOMKilled)
```

### 2. Test Health Check Locally

```bash
# Ensure you have PostgreSQL and Redis running
docker-compose up -d postgres redis

# Start the backend
go run main.go

# Test health check
curl http://localhost:8080/health

# Should return:
{
  "status": "healthy",
  "services": {
    "database": "healthy",
    "redis": "healthy"
  }
}
```

### 3. Test Database Connection

```bash
# In Railway → Database → Connect
# Copy the DATABASE_URL
# Test connection:
psql "postgres://postgres:..."

# If connection fails, check:
# - SSL mode (should be ?sslmode=require)
# - Connection pooling enabled
# - IP whitelist (Railway should auto-allow)
```

### 4. Monitor Memory Usage

```bash
# Local test to estimate memory usage:
go run main.go &
PID=$!

# Monitor memory for 60 seconds
while true; do
  ps aux | grep $PID | grep -v grep
  sleep 5
done

# Kill when done
kill $PID
```

## Quick Fix Checklist

### Immediate Actions

- [ ] Check Railway → Deployment Logs for panic/fatal errors
- [ ] Verify `DATABASE_URL` has `?sslmode=require`
- [ ] Set `ENV=production`
- [ ] Set `ALLOWED_ORIGINS=*` (temporarily for testing)
- [ ] Set `SUPABASE_URL=https://your-project.supabase.co`
- [ ] Check Railway → Metrics → Memory Usage (upgrade if >512MB)
- [ ] Deploy Gotenberg service on Railway or set `GOTENBERG_URL`

### If Still Crashing

1. **Disable Worker Temporarily:**
   ```bash
   WORKER_ENABLED=false
   ```
   This isolates whether the worker is causing the crash.

2. **Increase Health Check Timeout:**
   ```json
   // railway.json
   "healthcheckTimeout": 300
   ```

3. **Reduce Database Connection Pool:**
   Set environment variables:
   ```bash
   DB_MAX_OPEN_CONNS=10
   DB_MAX_IDLE_CONNS=2
   ```
   (Requires code change to read these)

4. **Check Full Error Logs:**
   ```bash
   # Railway → Service → Deployments → [Latest] → Logs
   # Download full logs and search for:
   grep -i "panic\|fatal\|error\|crash" railway.log
   ```

## Expected Railway Log Sequence

**✅ Successful Deployment:**
```
INFO Loading framework-specific AI model configurations
INFO Framework AI configuration framework=enrichment model=google/gemini-2.0-flash-001
INFO Framework AI configuration framework=pestel model=openai/o3-mini
...
INFO Framework configurations loaded successfully frameworks_loaded=13
INFO Starting IMENSIAH Backend V3
INFO Connecting to Database...
INFO Using REDIS_URL addr=redis-xxx:6379 has_password=true
INFO Starting Background Worker
asynq: pid=1 INFO: Starting processing
INFO HTTP Server listening on 8080
[GIN-debug] POST   /api/v1/submit
[GIN-debug] POST   /api/v1/submissions
...
GET /health HTTP/1.1 200 OK
```

**❌ Failed Deployment (Health Check):**
```
...
INFO HTTP Server listening on 8080
GET /health HTTP/1.1 503 Service Unavailable
ERROR Failed to ping database: connection refused
Railway: Health check failed (timeout after 100s)
Railway: Restarting container...
```

**❌ Failed Deployment (OOM):**
```
...
INFO HTTP Server listening on 8080
[Killed]
Railway: Container killed (exit code 137: OOMKilled)
Railway: Restarting container...
```

## Production Deployment Best Practices

### Environment Variables Management

Create a `railway-env.txt` file (do NOT commit):
```bash
# Copy .env.example and fill in production values
cp .env.example railway-env.txt

# Then set in Railway dashboard:
# Settings → Variables → Raw Editor → Paste contents
```

### Monitoring Setup

1. **Enable Railway Metrics:**
   - CPU usage
   - Memory usage
   - Network traffic
   - Request latency

2. **Add Application Logging:**
   ```bash
   # Railway automatically captures stdout/stderr
   # No changes needed - zerolog outputs to stderr
   ```

3. **Set Up Alerts:**
   - Memory usage > 80%
   - Health check failures
   - Error rate > 1%

### Scaling Considerations

**Free Tier Limitations:**
- 512MB RAM
- $5 free credit/month
- Shared CPU

**Upgrade to Pro ($5/month) if:**
- Memory usage >400MB consistently
- Need dedicated CPU
- Require >5GB storage

### Database Connection Pooling

**Recommended Settings for Railway:**
```bash
# PostgreSQL connection pool (Supabase)
# Set in Supabase → Settings → Database → Connection Pooling
Pool Mode: Transaction
Pool Size: 15
Max Client Connections: 3
```

**Railway Redis (Upstash):**
- Default settings work well
- No pooling configuration needed

## Support & Debugging

### Railway Support
- Docs: https://docs.railway.app/
- Discord: https://discord.gg/railway
- Status: https://status.railway.app/

### Application Logs
If you need help debugging, share:
1. Full Railway deployment logs (Railway → Deployment → Logs → Download)
2. Environment variables list (remove sensitive values)
3. Railway service plan (Free/Pro)
4. Database provider (Supabase/Railway Postgres)

---

**Last Updated:** 2025-11-21
**Status:** Production Troubleshooting Guide
**Version:** backend_v3 with Heterogeneous Model Optimization
