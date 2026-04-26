# Deployment Guide

This document covers the deployment process for Auth Service to Google Cloud Run.

## Quick Start

### Prerequisites

1. GCP project set up (see [SETUP.md](SETUP.md))
2. GitHub repository with secrets configured
3. Docker image built and pushed to Artifact Registry

### Automatic Deployment

Deployment is **automatic** when you push to the `master` branch. The GitHub Actions workflow in `.github/workflows/deploy.yml` will:

1. Build the Docker image
2. Push to GCP Artifact Registry
3. Deploy to Cloud Run

Monitor progress in GitHub Actions tab.

---

## Manual Deployment

### Prerequisites

```bash
# Authenticate with GCP
gcloud auth login

# Set project
export GCP_PROJECT_ID="your-project-id"
gcloud config set project $GCP_PROJECT_ID

# Authenticate Docker
gcloud auth configure-docker us-central1-docker.pkg.dev
```

### Build and Push Image

```bash
# Build
docker build \
  -t us-central1-docker.pkg.dev/$GCP_PROJECT_ID/auth-service/api:latest \
  -t us-central1-docker.pkg.dev/$GCP_PROJECT_ID/auth-service/api:$(git rev-parse --short HEAD) \
  .

# Push
docker push us-central1-docker.pkg.dev/$GCP_PROJECT_ID/auth-service/api:latest
docker push us-central1-docker.pkg.dev/$GCP_PROJECT_ID/auth-service/api:$(git rev-parse --short HEAD)
```

### Deploy to Cloud Run

```bash
gcloud run deploy auth-service \
  --image=us-central1-docker.pkg.dev/$GCP_PROJECT_ID/auth-service/api:latest \
  --region=us-central1 \
  --platform=managed \
  --service-account=auth-service-runner@$GCP_PROJECT_ID.iam.gserviceaccount.com \
  --set-secrets=\
DATABASE_URL=db-url:latest,\
REDIS_ADDR=redis-url:latest,\
JWT_ACCESS_SECRET=jwt-access-secret:latest,\
JWT_REFRESH_SECRET=jwt-refresh-secret:latest,\
GOOGLE_CLIENT_ID=google-client-id:latest,\
GOOGLE_CLIENT_SECRET=google-client-secret:latest \
  --set-env-vars=\
GCP_PROJECT_ID=$GCP_PROJECT_ID,\
ENV=production,\
TOTP_ISSUER=AuthService,\
TOTP_ENCRYPTION_KEY=$(echo -n 'YOUR_TOTP_KEY' | base64),\
CORS_ALLOWED_ORIGINS=https://yourdomain.com \
  --min-instances=0 \
  --max-instances=1 \
  --memory=256Mi \
  --cpu=1 \
  --concurrency=80 \
  --timeout=60 \
  --allow-unauthenticated \
  --quiet
```

### View Deployment Status

```bash
# Get service URL
gcloud run services describe auth-service \
  --region=us-central1 \
  --format='value(status.url)'

# View logs
gcloud run services logs read auth-service \
  --region=us-central1 \
  --limit=50

# View in real-time
gcloud run services logs read auth-service \
  --region=us-central1 \
  --follow
```

---

## Deployment Workflow

### GitHub Actions Workflow

The automatic deployment is triggered by `.github/workflows/deploy.yml`:

1. **Checkout Code** — Gets the latest code from master branch
2. **Authenticate to GCP** — Uses Workload Identity Federation
3. **Set up Cloud SDK** — Configures gcloud CLI
4. **Configure Docker Auth** — Authenticates Docker with Artifact Registry
5. **Build Docker Image** — Builds with `latest` and git SHA tags
6. **Push to Artifact Registry** — Pushes both tags
7. **Deploy to Cloud Run** — Updates service with new image and secrets
8. **Get Service URL** — Outputs the deployed service URL

### Secrets Used in CI/CD

These GitHub Secrets are required for automatic deployment:

| Secret | Usage | Example |
|---|---|---|
| `GCP_PROJECT_ID` | Identifies GCP project | `my-auth-service` |
| `WIF_PROVIDER` | Workload Identity Provider resource name | `projects/123/locations/global/workloadIdentityPools/github-pool/providers/github-provider` |
| `WIF_SERVICE_ACCOUNT` | Service account email for GitHub Actions | `github-deployer@my-auth-service.iam.gserviceaccount.com` |
| `GCP_SERVICE_ACCOUNT` | Service account for Cloud Run | `auth-service-runner@my-auth-service.iam.gserviceaccount.com` |
| `TOTP_ENCRYPTION_KEY` | TOTP encryption key | Base64-encoded 32+ char string |

---

## Troubleshooting

### Deployment Fails in GitHub Actions

1. Check GitHub Actions logs (Actions tab in GitHub)
2. Common issues:
   - Workload Identity Federation not configured
   - Missing GitHub Secrets
   - Insufficient IAM permissions
   - Artifact Registry not created

### Docker Build Fails

```bash
# Check if Dockerfile exists
ls -la Dockerfile

# Test build locally
docker build -t test-image:latest .

# If build succeeds locally but fails in CI, check:
# - Base image availability
# - All required files are committed
# - Secrets not hardcoded in Dockerfile
```

### Service Fails to Start on Cloud Run

```bash
# View recent logs
gcloud run services logs read auth-service --region=us-central1 --limit=100

# Common issues:
# 1. Missing secrets in Secret Manager
gcloud secrets list --project=$GCP_PROJECT_ID

# 2. Service account missing permissions
gcloud projects get-iam-policy $GCP_PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:auth-service-runner*"

# 3. Invalid environment variable format
# Check TOTP_ENCRYPTION_KEY is properly encoded
```

### Database Connection Issues

```bash
# Verify database is accessible from Cloud Run
# Cloud Run can access Neon (hosted database) directly

# Test from local machine
psql "$DATABASE_URL"

# If connection times out:
# - Check Neon allows connections from GCP IP ranges
# - Verify DATABASE_URL format is correct
```

### Redis Connection Issues

```bash
# Test Redis connection from Cloud Run logs
# Verify Redis Cloud allows connections from GCP

# Test locally
redis-cli -u "$REDIS_ADDR" ping

# Common issues:
# - Incorrect password
# - Firewall blocking connections
# - Redis URL format incorrect
```

---

## Post-Deployment Verification

After deployment, verify the service is working:

```bash
# Get service URL
SERVICE_URL=$(gcloud run services describe auth-service \
  --region=us-central1 \
  --format='value(status.url)')

# Test health endpoint (if available)
curl $SERVICE_URL/health

# Test login endpoint
curl -X POST $SERVICE_URL/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}'

# Check logs for errors
gcloud run services logs read auth-service --region=us-central1 --limit=50
```

---

## Rollback

If deployment causes issues:

### Option 1: Redeploy Previous Version

```bash
# Get list of recent revisions
gcloud run services describe auth-service --region=us-central1

# Revert to previous image tag
export PREVIOUS_SHA="abc123def456"  # Commit SHA of previous working version
docker pull us-central1-docker.pkg.dev/$GCP_PROJECT_ID/auth-service/api:$PREVIOUS_SHA

gcloud run deploy auth-service \
  --image=us-central1-docker.pkg.dev/$GCP_PROJECT_ID/auth-service/api:$PREVIOUS_SHA \
  --region=us-central1 \
  --no-traffic  # Creates new revision without routing traffic
```

### Option 2: Promote Previous Revision

```bash
# List revisions
gcloud run revisions list --service=auth-service --region=us-central1

# Promote specific revision
gcloud run services update-traffic auth-service \
  --to-revisions=REVISION_ID=100 \
  --region=us-central1
```

---

## Scaling Configuration

Current production configuration:

- **Min Instances**: 0 (scales to zero when idle, saves cost)
- **Max Instances**: 1 (prevents auto-scaling for cost control)
- **Memory**: 256Mi
- **CPU**: 1 vCPU
- **Concurrency**: 80 requests per container
- **Timeout**: 60 seconds

To adjust:

```bash
gcloud run services update auth-service \
  --region=us-central1 \
  --min-instances=1 \
  --max-instances=5 \
  --memory=512Mi \
  --concurrency=100
```

---

## Monitoring and Logging

### View Logs

```bash
# Last 50 lines
gcloud run services logs read auth-service --region=us-central1 --limit=50

# Real-time logs
gcloud run services logs read auth-service --region=us-central1 --follow

# Logs from specific time
gcloud run services logs read auth-service --region=us-central1 \
  --limit=100 \
  --start-time='2024-04-25T10:00:00Z'
```

### View Metrics

```bash
# In Cloud Console:
# 1. Go to Cloud Run > Services > auth-service
# 2. Click "Metrics" tab
# 3. View request count, latency, errors, memory usage
```

### Set Up Alerts

In Cloud Console:
1. Go to Cloud Run > auth-service
2. Click "Monitoring" tab
3. Create alert policies for:
   - High error rate
   - High latency
   - Out of memory

---

## Cost Optimization

Current setup optimizes for free tier:

- **Min instances = 0**: Scales to zero when idle (only pay when receiving requests)
- **Max instances = 1**: Prevents expensive auto-scaling
- **Memory = 256Mi**: Minimum memory for typical applications
- **Free tier includes**: 2M requests/month, 600K GB-seconds compute

To estimate costs:
```bash
# Cloud Run pricing calculator
# https://cloud.google.com/run/pricing

# Rough calculation:
# - 1M requests/month at 100ms avg = 100K GB-seconds = $0.40
# - 256Mi = 0.25 GB, so (100K * 0.25) / 2.5M = $0.01
```

---

## References

- [Cloud Run Deployment Documentation](https://cloud.google.com/run/docs/deploying-overview)
- [Workload Identity Federation](https://cloud.google.com/docs/authentication/workload-identity-federation)
- [Cloud Run Pricing](https://cloud.google.com/run/pricing)
- [gcloud run deploy Reference](https://cloud.google.com/sdk/gcloud/reference/run/deploy)
