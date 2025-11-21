# Production Deployment Guide - IMENSIAH Backend V3

## Pre-Deployment Checklist

Before deploying to Railway/Render/Fly.io, verify these critical configurations:

### 1. Environment Variables (CRITICAL)

Set these **exact** values in your deployment platform dashboard:

```bash
# === SERVER CONFIGURATION ===
ENVIRONMENT=production
SERVER_PORT=8080
ALLOWED_ORIGINS=https://your-frontend.vercel.app  # ← EXACT match, NO trailing slash

# === DATABASE ===
DATABASE_URL=postgres://user:pass@host:5432/db?sslmode=require

# === AUTHENTICATION (Supabase) ===
SUPABASE_URL=https://your-project-id.supabase.co
SUPABASE_ANON_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...  # From API settings
SUPABASE_JWT_SECRET=your-jwt-secret                        # From API -> JWT Settings

# === AI CONFIGURATION ===
OPENROUTER_API_KEY=sk-or-v1-...
AI_ENRICHMENT_MODEL=google/gemini-2.0-flash-001
AI_ANALYSIS_MODEL=google/gemini-2.0-pro-exp-02-05
AI_SYNTHESIS_MODEL=anthropic/claude-3.5-sonnet

# === INFRASTRUCTURE ===
REDIS_URL=redis://default:password@host:port  # From Railway Redis addon
GOTENBERG_URL=http://gotenberg-service:3000   # Internal Railway URL

# === WORKER ===
WORKER_ENABLED=true
ASYNQ_CONCURRENCY=10
```

### 2. Code Fixes Applied

✅ **CORS Security Fix** (`api/middleware.go`)
- Removed insecure fallback logic
- Unauthorized origins now explicitly rejected with 403 Forbidden
- **Action Required**: Ensure `ALLOWED_ORIGINS` matches your frontend URL exactly

✅ **Worker Crash Protection** (`main.go`)
- Worker failures no longer crash the entire API
- Changed `log.Fatal` to `log.Error` for graceful degradation
- API stays alive even if Redis is unreachable at startup

✅ **Worker Enable/Disable** (`config/config.go`)
- Added `WORKER_ENABLED` environment variable support
- Can now disable background processing in specific environments
- Defaults to `true` if not set

✅ **Supabase Key Separation** (`config/config.go`, `api/handlers.go`)
- Split `SUPABASE_ANON_KEY` (for auth API calls) from `SUPABASE_JWT_SECRET` (for token validation)
- Follows Supabase best practices
- **Action Required**: Add both keys to environment variables

✅ **Dockerfile Template Fix** (`Dockerfile`)
- Added `COPY --from=builder /app/templates ./templates`
- Prevents "template not found" errors during PDF generation
- Critical for report publishing feature

### 3. Railway Deployment Steps

#### Step 1: Add Required Services

In your Railway project dashboard:

1. **Add Redis Service**:
   - Click "+ New" → "Database" → "Add Redis"
   - Railway automatically sets `REDIS_URL` environment variable
   - Your Go app will parse this automatically

2. **Add Gotenberg Service**:
   - Click "+ New" → "Empty Service"
   - Name: `gotenberg`
   - Deploy from Docker Image: `gotenberg/gotenberg:8`
   - Expose port `3000`
   - Copy the internal URL (e.g., `http://gotenberg.railway.internal:3000`)
   - Set `GOTENBERG_URL` in your Go service to this URL

3. **Configure Go Backend**:
   - Already configured via `railway.json`
   - Ensure all environment variables from checklist above are set
   - Railway will build using your `Dockerfile`

#### Step 2: Verify Build Logs

After deployment, check Railway logs for:

```
✅ "Starting Background Worker"
✅ "HTTP Server listening on 8080"
✅ "Connecting to Database..."
❌ "Worker failed to start" (acceptable - API will still run)
```

#### Step 3: Test Deployment

1. **Health Check**:
   ```bash
   curl https://your-backend.railway.app/health
   ```
   Expected response:
   ```json
   {
     "status": "healthy",
     "services": {
       "database": "healthy",
       "redis": "healthy"
     }
   }
   ```

2. **CORS Verification**:
   - Open browser console on `https://your-frontend.vercel.app`
   - Attempt login
   - Check Network tab for CORS errors
   - If you see "Origin not allowed", verify `ALLOWED_ORIGINS` matches exactly

3. **Worker Verification**:
   - Submit a test form
   - Check Railway logs for:
     ```
     Job Started: Enrichment Agent
     Job Started: Strategic Cascade
     Workflow Paused. Waiting for Admin Review.
     ```

### 4. Admin Emergency Toolkit

If jobs get stuck, use these endpoints to manually trigger processing:

#### Retry Enrichment
```bash
POST https://your-backend.railway.app/api/v1/admin/submissions/{id}/retry-enrichment
Authorization: Bearer {admin-jwt-token}
```

#### Retry Analysis
```bash
POST https://your-backend.railway.app/api/v1/admin/submissions/{id}/retry-analysis
Authorization: Bearer {admin-jwt-token}
```

**Recommended**: Create a Postman collection with these endpoints pre-configured.

### 5. Monitoring (Week 1)

Check these daily for the first week:

1. **Railway Logs**:
   - Look for "Worker failed to start" (indicates Redis connection issues)
   - Monitor "Panic recovered" errors
   - Track request latency

2. **Redis Queue Status**:
   - Option A: Use Railway Redis CLI:
     ```bash
     LLEN asynq:queues:default
     LLEN asynq:queues:critical
     ```
   - Option B: Install Asynq Web UI (see below)

3. **Error Patterns**:
   - CORS errors → Check `ALLOWED_ORIGINS`
   - 401 Unauthorized → Check `SUPABASE_JWT_SECRET`
   - "Template not found" → Verify Dockerfile includes templates

### 6. Optional: Asynq Web Dashboard

To monitor background jobs visually:

1. Add to your Railway project:
   ```bash
   # New service from Docker image
   hibiken/asynqmon:latest
   ```

2. Environment variables:
   ```bash
   REDIS_ADDR=redis:6379
   PORT=8081
   ```

3. Access at `https://asynqmon.railway.app` (or internal URL)

---

## Common Production Issues

### Issue: "Origin not allowed by CORS policy"
**Cause**: `ALLOWED_ORIGINS` doesn't match frontend URL
**Fix**: Ensure exact match (no trailing slash)
```bash
✅ ALLOWED_ORIGINS=https://app.example.com
❌ ALLOWED_ORIGINS=https://app.example.com/
```

### Issue: Jobs never process
**Cause**: Worker not running or Redis disconnected
**Fix**: Check Railway logs for "Starting Background Worker"
**Workaround**: Use admin retry endpoints manually

### Issue: "template not found" during report generation
**Cause**: Dockerfile missing templates folder
**Fix**: Already applied - redeploy with updated Dockerfile

### Issue: All auth requests fail with 401
**Cause**: Using wrong Supabase key
**Fix**:
- `SUPABASE_ANON_KEY` = From API settings → anon public key
- `SUPABASE_JWT_SECRET` = From API settings → JWT Secret (not the anon key)

---

## Rollback Plan

If deployment fails:

1. **Immediate**: Railway → "Deployments" → "Redeploy" previous version
2. **Verify**: Check health endpoint returns 200 OK
3. **Investigate**: Review deployment logs for root cause
4. **Fix**: Apply fix locally, test with `docker-compose up`, then redeploy

---

## Local Testing with Full Stack

Before deploying to production, test the complete stack locally:

```bash
# 1. Ensure .env file has correct values
cp .env.example .env
# Edit .env with your Supabase credentials

# 2. Start entire stack
docker-compose up --build

# 3. Test endpoints
curl http://localhost:8080/health
```

This starts:
- Go Backend on `localhost:8080`
- Redis on `localhost:6379`
- Gotenberg on `localhost:3000`

---

## Post-Deployment Success Criteria

✅ Health endpoint returns 200 OK
✅ Frontend can login without CORS errors
✅ Test submission reaches "ready_for_review" status
✅ Railway logs show "Job Started: Enrichment Agent"
✅ Admin retry endpoints respond successfully
✅ Report preview generates HTML
✅ Report publish generates PDF and uploads to Supabase

---

## Support Checklist

If users report issues, verify:

1. **User can login**: Auth working → Supabase keys correct
2. **Form submits**: API accepting requests → CORS configured
3. **Analysis runs**: Worker processing → Redis connected
4. **Report generates**: PDF service working → Gotenberg accessible

---

**Last Updated**: 2025-11-21
**Configuration Version**: V3 (Post-Stability Audit)
