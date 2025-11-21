# Railway Deployment Guide - Backend

## Prerequisites
- Railway account: https://railway.app
- OpenRouter API key: https://openrouter.ai/keys
- Supabase project with migrations run
- GitHub repository connected

## Environment Variables Required

Add these in Railway Backend Service > Variables:

```bash
# Server Configuration
SERVER_PORT=8080
ENV=production

# CORS - Add your frontend Railway URL after deployment
ALLOWED_ORIGINS=https://your-frontend.up.railway.app,https://yourdomain.com

# Database (from Supabase)
DATABASE_URL=postgres://postgres.xxxxx:[YOUR_PASSWORD]@aws-0-us-west-1.pooler.supabase.com:5432/postgres?sslmode=require

# Redis (from Railway Redis service - auto-populated)
REDIS_ADDR=${{Redis.REDIS_URL}}
REDIS_PASSWORD=${{Redis.REDIS_PASSWORD}}

# Supabase Auth (from Supabase Settings > API > JWT Settings)
SUPABASE_URL=https://xxxxx.supabase.co
SUPABASE_JWT_SECRET=your-jwt-secret-from-supabase

# AI Processing (from OpenRouter)
OPENROUTER_API_KEY=sk-or-v1-xxxxx
AI_ENRICHMENT_MODEL=google/gemini-2.0-flash-001
AI_ANALYSIS_MODEL=google/gemini-2.0-pro-exp-02-05
AI_SYNTHESIS_MODEL=anthropic/claude-3.5-sonnet

# Background Jobs
ASYNQ_CONCURRENCY=10
RATE_LIMIT_REQUESTS_PER_MINUTE=100

# PDF Generation (Optional - can skip initially)
GOTENBERG_URL=http://gotenberg:3000
```

## Deployment Steps

### 1. Create Railway Project
```bash
# Go to Railway Dashboard
# Click "New Project" > "Empty Project"
# Name: "IMENSIAH Backend"
```

### 2. Add Redis Service
```bash
# In Railway project:
# Click "+ New"
# Select "Database" > "Redis"
# Wait for deployment
```

### 3. Deploy Backend Service
```bash
# Click "+ New"
# Select "GitHub Repo"
# Connect your GitHub account
# Select: devimensiah-del/backend_v3
# Railway will auto-detect Dockerfile
```

### 4. Configure Environment Variables
```bash
# Click on Backend service
# Go to "Variables" tab
# Click "+ New Variable" for each variable above
# Use ${{Redis.REDIS_URL}} syntax for Redis variables
```

### 5. Deploy
```bash
# Click "Deploy"
# Watch deployment logs
# Wait for "Deployed" status (5-10 minutes)
```

### 6. Get Backend URL
```bash
# Go to "Settings" tab
# Under "Domains" > "Generate Domain"
# Copy URL: https://backend-production-xxxx.up.railway.app
# Save this for frontend configuration
```

### 7. Test Deployment
```bash
# Health check
curl https://your-backend-url.up.railway.app/health

# Expected response:
{
  "status": "healthy",
  "services": {
    "database": "healthy",
    "redis": "healthy"
  }
}
```

## API Endpoints

All endpoints are now versioned under `/api/v1`:

### Public Endpoints
- `POST /api/v1/submit` - Create submission

### Protected Endpoints (Require JWT)
- `GET /api/v1/submissions/:id` - Get submission
- `GET /api/v1/submissions/:id/report/preview` - Preview report
- `POST /api/v1/submissions/:id/report/publish` - Publish report

### Admin Endpoints (Require admin role)
- `GET /api/v1/admin/submissions` - List all submissions
- `POST /api/v1/admin/submissions/:id/retry-enrichment` - Retry enrichment
- `POST /api/v1/admin/submissions/:id/retry-analysis` - Retry analysis

## Troubleshooting

### Issue: Database connection failed
**Solution**: Check `DATABASE_URL` format includes `?sslmode=require`

### Issue: Redis connection failed
**Solution**: Ensure Railway Redis service is running and variables are set correctly

### Issue: CORS errors
**Solution**: Update `ALLOWED_ORIGINS` to include your frontend Railway URL

### Issue: 502 Bad Gateway
**Solution**: Check logs for errors, ensure port 8080 is exposed in Dockerfile

### Issue: JWT validation fails
**Solution**: Verify `SUPABASE_JWT_SECRET` matches your Supabase project's JWT secret

## Monitoring

### View Logs
```bash
# In Railway Dashboard
# Click on Backend service
# Go to "Deployments" tab
# Click latest deployment
# View logs in real-time
```

### Check Resource Usage
```bash
# Go to "Metrics" tab
# Monitor CPU, Memory, Network
# Set alerts if needed
```

## Scaling

### Increase Resources
```bash
# Go to "Settings" tab
# Scroll to "Resources"
# Adjust CPU/Memory as needed
```

### Add More Instances
```bash
# Go to "Settings" tab
# Scroll to "Instances"
# Increase replica count
```

## Cost Optimization

- Start with 0.5 vCPU, 512MB RAM (~$3/month)
- Enable autoscaling if traffic increases
- Use Redis for caching to reduce database calls
- Monitor OpenRouter API costs

## Support

- Railway Docs: https://docs.railway.app
- Railway Discord: https://discord.gg/railway
- Backend GitHub: https://github.com/devimensiah-del/backend_v3

## Post-Deployment Checklist

- [ ] Health check returns "healthy"
- [ ] Database connection works
- [ ] Redis connection works
- [ ] Test submission endpoint
- [ ] CORS configured for frontend
- [ ] Logs show no errors
- [ ] Update frontend with backend URL
