# Troubleshooting Guide

Common issues and solutions for Auth Service setup, development, and deployment.

---

## GCP Setup Issues

### "Project not found" or Authorization errors

**Problem**: Commands fail with "Project not found" or "Permission denied"

**Solution**:
```bash
# List your projects
gcloud projects list

# Set the correct project
export GCP_PROJECT_ID="your-correct-project-id"
gcloud config set project $GCP_PROJECT_ID

# Verify authentication
gcloud auth list
gcloud auth application-default login
```

### "API not enabled" errors

**Problem**: Error like "Cloud Run API has not been used in project"

**Solution**:
```bash
# Enable required APIs
gcloud services enable \
  run.googleapis.com \
  artifactregistry.googleapis.com \
  secretmanager.googleapis.com \
  cloudbuild.googleapis.com

# Or enable specific API
gcloud services enable run.googleapis.com
```

### "Access Denied" when accessing Secret Manager

**Problem**: "User does not have permission to access secret"

**Solution**:
```bash
export SERVICE_ACCOUNT="auth-service-runner@$GCP_PROJECT_ID.iam.gserviceaccount.com"

# Grant Secret Manager access
gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
  --member="serviceAccount:$SERVICE_ACCOUNT" \
  --role="roles/secretmanager.secretAccessor"

# Verify the binding
gcloud projects get-iam-policy $GCP_PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:$SERVICE_ACCOUNT"
```

---

## Secret Manager Issues

### Secret not found errors

**Problem**: Application fails with "failed to access secret: not found"

**Solution**:
```bash
# List all secrets
gcloud secrets list --project=$GCP_PROJECT_ID

# Create missing secret
echo -n "secret-value" | \
  gcloud secrets create secret-name --data-file=- --project=$GCP_PROJECT_ID

# Verify secret exists and has a version
gcloud secrets versions list secret-name --project=$GCP_PROJECT_ID
```

### Old secret values being used after update

**Problem**: Deployed service still uses old secret value after update

**Solution**:
```bash
# Cloud Run caches secrets for a short time
# Redeploy the service to pick up new secrets

gcloud run deploy auth-service \
  --region=us-central1 \
  --no-traffic  # Optional: test without routing traffic first
```

### Secret value too large

**Problem**: "Invalid argument: The secret size exceeds the maximum allowed size"

**Solution**: Maximum secret size is 64KB. If your secret is larger:
- Store in Cloud Storage instead
- Split into multiple secrets
- Compress the content

```bash
# Example: split database credentials
echo -n "postgresql://..." | gcloud secrets create db-url --data-file=-
echo -n "extra-config" | gcloud secrets create db-config --data-file=-
```

---

## Database (Neon) Issues

### Cannot connect to Neon database

**Problem**: `psql` fails with connection timeout or "host not found"

**Solution**:
```bash
# Verify connection string format
# Should be: postgresql://user:password@host:port/database

# Test connection
psql "postgresql://user:password@host:5432/database"

# If connection times out:
# 1. Check Neon dashboard for database status
# 2. Verify IP allowlist if using Neon's IP restrictions
# 3. Ensure password doesn't contain special characters
#    (URL encode if needed: @ -> %40, : -> %3A)
```

### Migration failures

**Problem**: Database migrations fail on startup

**Solution**:
```bash
# Check migration files are in migrations/ directory
ls -la migrations/

# Verify database is empty or has correct schema
psql "$DATABASE_URL" -c "\dt"  # List tables

# Check migration logs in application startup
# Look for SQL errors in application logs
```

### "too many connections" error

**Problem**: Database connection limit exceeded

**Solution**:
```bash
# Check current connection count in Neon
psql "$DATABASE_URL" -c \
  "SELECT count(*) FROM pg_stat_activity;"

# If using connection pooling:
# - Reduce max connections in application
# - Enable Neon's connection pooling
```

---

## Redis (Redis Cloud) Issues

### Redis connection refused

**Problem**: `redis-cli: ECONNREFUSED` or connection timeout

**Solution**:
```bash
# Verify Redis URL format
# Should be: redis://default:password@host:port

# Test Redis connection
redis-cli -u "redis://default:password@host:port" ping
# Should return: PONG

# If it fails:
# 1. Verify password is correct (no special chars without URL encoding)
# 2. Check host:port is correct (in Redis Cloud console)
# 3. Verify Redis Cloud allows your IP
# 4. Test with redis-cli directly:
redis-cli -h hostname -p 12345 -a password ping
```

### "WRONGPASS" or authentication failed

**Problem**: Redis returns "WRONGPASS invalid username-password pair"

**Solution**:
```bash
# Double-check password in Redis Cloud console
# Password may contain special characters - must be URL encoded

# Example: password "my@password!" becomes "my%40password%21"
# URL: redis://default:my%40password%21@host:port

# Test password directly
redis-cli -h hostname -p port -a password ping
```

### Redis URL parsing errors in Cloud Run

**Problem**: Application fails with "failed to parse Redis URL"

**Solution**: The application supports both formats:
```
# Format 1: Host and port
redis-host.example.com:12345

# Format 2: Full Redis URL
redis://default:password@redis-host.example.com:12345
```

Use full Redis URL format for Cloud Run:
```bash
echo -n "redis://default:password@host:port" | \
  gcloud secrets create redis-url --data-file=-
```

---

## GitHub Actions / CI/CD Issues

### Workflow fails on "Authenticate to Google Cloud"

**Problem**: GitHub Actions fails with authentication error

**Solution**: Verify Workload Identity Federation setup:

```bash
# Check WIF pool exists
gcloud iam workload-identity-pools list \
  --location=global --project=$GCP_PROJECT_ID

# Check WIF provider exists
gcloud iam workload-identity-pools providers list \
  --workload-identity-pool=github-pool \
  --location=global --project=$GCP_PROJECT_ID

# Check service account has bindings
gcloud iam service-accounts get-iam-policy \
  github-deployer@$GCP_PROJECT_ID.iam.gserviceaccount.com \
  --project=$GCP_PROJECT_ID
```

### Missing GitHub Secrets

**Problem**: Error like "GCP_PROJECT_ID secret not found"

**Solution**:
```bash
# Go to GitHub repo Settings > Secrets and variables > Actions
# Add all required secrets:
# - GCP_PROJECT_ID
# - WIF_PROVIDER
# - WIF_SERVICE_ACCOUNT
# - GCP_SERVICE_ACCOUNT
# - TOTP_ENCRYPTION_KEY

# Get values for missing secrets:

# WIF_PROVIDER resource name
gcloud iam workload-identity-pools providers describe github-provider \
  --workload-identity-pool=github-pool \
  --location=global --project=$GCP_PROJECT_ID \
  --format='value(name)'

# Service account emails
gcloud iam service-accounts list --project=$GCP_PROJECT_ID
```

### Artifact Registry push fails

**Problem**: "denied: denied" when pushing to Artifact Registry

**Solution**:
```bash
# Verify Artifact Registry repository exists
gcloud artifacts repositories list --project=$GCP_PROJECT_ID

# Verify service account has artifact registry permissions
gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
  --member="serviceAccount:github-deployer@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/artifactregistry.writer"

# Re-authenticate Docker
gcloud auth configure-docker us-central1-docker.pkg.dev
```

---

## Docker Build Issues

### "Build context too large"

**Problem**: Docker build fails with context size error

**Solution**: Create `.dockerignore`:
```
.git
.github
node_modules
dist
vendor
coverage
.env
.env.*
Dockerfile
```

### Build fails with "stage not found"

**Problem**: Multi-stage Dockerfile references non-existent stage

**Solution**: Verify Dockerfile stage names match references:
```dockerfile
FROM golang:1.21 AS builder
# ... build steps ...

FROM alpine:latest AS runtime
COPY --from=builder /app/binary ./
```

### Go dependencies not found

**Problem**: "go: cannot find module" during build

**Solution**:
```bash
# Update go.mod
go mod tidy

# Verify go.mod and go.sum are committed
git add go.mod go.sum
git commit -m "update dependencies"

# Test build locally
docker build -t test:latest .
```

---

## Cloud Run Deployment Issues

### Service fails to start

**Problem**: Cloud Run revision stays in "Deploying" state or crashes

**Solution**:
```bash
# Check logs immediately after deployment
gcloud run services logs read auth-service \
  --region=us-central1 \
  --limit=100 \
  --start-time='2024-04-25T10:00:00Z'

# Common causes:
# 1. Missing secrets
gcloud secrets list --project=$GCP_PROJECT_ID

# 2. Invalid environment variables
# Check for special characters, spaces, etc.

# 3. Port mismatch
# Application should listen on PORT env var (default 8080)
```

### "CPU throttled" or "Memory limit exceeded"

**Problem**: Service times out or returns 503 errors

**Solution**:
```bash
# Increase resources
gcloud run services update auth-service \
  --region=us-central1 \
  --memory=512Mi \
  --cpu=2

# Check current metrics
gcloud run services describe auth-service \
  --region=us-central1 \
  --format='value(spec.template.spec.resources.limits)'
```

### Domain mapping not working

**Problem**: Custom domain shows 404 or SSL certificate errors

**Solution**:
```bash
# Verify domain mapping exists
gcloud run domain-mappings list --region=us-central1

# Create domain mapping
gcloud run domain-mappings create \
  --service=auth-service \
  --domain=api.example.com \
  --region=us-central1

# Update DNS records to point to Cloud Run
# Check status
gcloud run domain-mappings describe \
  api.example.com \
  --region=us-central1
```

---

## Local Development Issues

### Port already in use

**Problem**: `bind: address already in use`

**Solution**:
```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 PID

# Or use different port
PORT=8081 go run cmd/main.go
```

### Database migrations not running

**Problem**: Application starts but database tables don't exist

**Solution**:
```bash
# Check migrations directory
ls -la migrations/

# Verify DATABASE_URL is correct
echo $DATABASE_URL

# Test connection
psql "$DATABASE_URL" -c "\dt"

# Check application logs for migration errors
# Add debug logging if needed
```

### Redis not connecting locally

**Problem**: "localhost:6379: connection refused"

**Solution**:
```bash
# Start Redis if not running
# Option 1: Docker
docker run -d -p 6379:6379 redis:latest

# Option 2: Redis CLI (if installed)
redis-server

# Option 3: Disable Redis locally (if optional)
# Set REDIS_ADDR to empty string in .env
REDIS_ADDR=

# Verify connection
redis-cli ping  # Should return: PONG
```

### Environment variables not loaded

**Problem**: Application uses default values, .env not being read

**Solution**:
```bash
# Verify .env exists in current directory
cat .env

# Verify .env format (no spaces around =)
# Correct: KEY=value
# Wrong: KEY = value

# Load explicitly
export $(cat .env | xargs)
go run cmd/main.go

# Or use env command
env -i bash -c 'export $(cat .env | xargs); go run cmd/main.go'
```

---

## Google OAuth Issues

### OAuth redirect URI mismatch

**Problem**: "Redirect URI mismatch" error when testing Google login

**Solution**:
```bash
# Verify GOOGLE_REDIRECT_URL in configuration
echo $GOOGLE_REDIRECT_URL
# Should be: https://your-domain/auth/google/callback

# Add to Google Cloud Console:
# 1. Go to OAuth 2.0 Credentials
# 2. Edit the OAuth application
# 3. Add authorized redirect URIs:
#    - http://localhost:8080/auth/google/callback (dev)
#    - https://yourdomain.com/auth/google/callback (prod)
```

### Invalid OAuth credentials

**Problem**: OAuth fails with invalid credentials error

**Solution**:
```bash
# Verify credentials are correct
echo $GOOGLE_CLIENT_ID
echo $GOOGLE_CLIENT_SECRET

# Re-create credentials in Google Cloud Console
# 1. Go to OAuth 2.0 Credentials
# 2. Create new OAuth 2.0 Client ID (Web application)
# 3. Copy new Client ID and Client Secret
# 4. Update secrets:
echo -n "new-client-id" | \
  gcloud secrets versions add google-client-id --data-file=-
```

---

## Rate Limiting Issues

### Legitimate requests being rate limited

**Problem**: Valid requests return 429 Too Many Requests

**Solution**:
```bash
# Check rate limit configuration
echo $RATE_LIMIT_LOGIN
echo $RATE_LIMIT_GLOBAL

# Increase limits if needed
# Format: {requests}-{duration} (M=minute, S=second, H=hour)
# Current: 5-M = 5 per minute
# Increase to: 10-M = 10 per minute

gcloud run services update auth-service \
  --region=us-central1 \
  --set-env-vars=RATE_LIMIT_LOGIN=10-M,RATE_LIMIT_GLOBAL=200-M
```

### Rate limiting not working

**Problem**: Requests exceed limit but are still served

**Solution**:
```bash
# Verify Redis is available (rate limiting uses Redis)
redis-cli -u "$REDIS_ADDR" ping

# If Redis is not configured, rate limiting is disabled
# Configure REDIS_ADDR to enable

# Check application logs for rate limit errors
gcloud run services logs read auth-service --region=us-central1 --limit=50
```

---

## Performance Issues

### High latency or timeouts

**Problem**: Requests take too long or timeout (>60s)

**Solution**:
```bash
# Check Cloud Run metrics
gcloud run services describe auth-service \
  --region=us-central1

# Increase timeout
gcloud run services update auth-service \
  --region=us-central1 \
  --timeout=120  # seconds

# Check database performance
# Slow queries are typically the bottleneck
# Enable slow query logging in Neon dashboard
```

### High memory usage

**Problem**: Out of memory errors, OOMKilled pods

**Solution**:
```bash
# Increase memory allocation
gcloud run services update auth-service \
  --region=us-central1 \
  --memory=512Mi

# Check for memory leaks in application
# Use profiling tools:
# - pprof: https://pkg.go.dev/net/http/pprof
# - heaptrack (local)
```

---

## Monitoring and Debugging

### Enable debug logging

**Problem**: Need more detailed logs to diagnose issues

**Solution**:
```bash
# Set ENV to development for more verbose logging
gcloud run services update auth-service \
  --region=us-central1 \
  --set-env-vars=ENV=development

# Or check application code for debug flags
# Usually requires code change and redeploy
```

### Stream real-time logs

**Problem**: Need to see logs as they happen

**Solution**:
```bash
# Using gcloud
gcloud run services logs read auth-service \
  --region=us-central1 \
  --follow

# Using Cloud Logging with filter
gcloud logging read \
  "resource.type=cloud_run_revision AND resource.labels.service_name=auth-service" \
  --region=us-central1 \
  --follow
```

---

## When All Else Fails

1. **Check official documentation**:
   - [Cloud Run Docs](https://cloud.google.com/run/docs)
   - [Neon Docs](https://neon.tech/docs)
   - [Redis Cloud Docs](https://docs.redis.com/latest/)

2. **Review application logs thoroughly**:
   ```bash
   gcloud run services logs read auth-service --limit=200
   ```

3. **Test components in isolation**:
   ```bash
   # Test database
   psql "$DATABASE_URL"
   
   # Test Redis
   redis-cli -u "$REDIS_ADDR" ping
   
   # Test Google OAuth (manually)
   curl "https://oauth2.googleapis.com/token" ...
   ```

4. **Review recent changes**:
   ```bash
   git log --oneline -20
   git diff HEAD~5
   ```

5. **Reset and redeploy from scratch**:
   - Create new Cloud Run service
   - Create new database
   - Check if issue reproduces

6. **Ask for help**:
   - Google Cloud Support (for GCP issues)
   - Neon Support (for database issues)
   - Redis Support (for Redis issues)
