# Setup Guide

This guide covers the complete setup for developing and deploying the Auth Service, including GCP, Redis Cloud, and Neon Postgres.

## Prerequisites

- Google Cloud Account
- Redis Cloud Account
- Neon (PostgreSQL) Account
- Docker installed
- Go 1.21+
- gcloud CLI installed and configured

---

## 1. Google Cloud Project Setup

### Create GCP Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click the project selector dropdown at the top
3. Click "New Project"
4. Enter project name (e.g., `auth-service`)
5. Click "Create"

### Enable Required APIs

Enable these APIs in your GCP project:

```bash
gcloud services enable \
  run.googleapis.com \
  artifactregistry.googleapis.com \
  secretmanager.googleapis.com \
  cloudbuild.googleapis.com \
  cloudresourcemanager.googleapis.com
```

### Create Service Account

Create a service account for Cloud Run deployments:

```bash
# Set your project ID
export GCP_PROJECT_ID="your-project-id"

# Create service account
gcloud iam service-accounts create auth-service-runner \
  --display-name="Auth Service Cloud Run Runner" \
  --project=$GCP_PROJECT_ID

# Grant necessary roles
gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
  --member="serviceAccount:auth-service-runner@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/run.admin"

gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
  --member="serviceAccount:auth-service-runner@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

### Create Artifact Registry Repository

```bash
gcloud artifacts repositories create auth-service \
  --repository-format=docker \
  --location=us-central1 \
  --description="Auth Service Docker images" \
  --project=$GCP_PROJECT_ID
```

### Create Secrets in Secret Manager

Create the following secrets in GCP Secret Manager:

```bash
# Database URL (Neon PostgreSQL connection string)
echo -n "postgresql://user:password@host/database" | \
  gcloud secrets create db-url --data-file=- --project=$GCP_PROJECT_ID

# Redis URL (Redis Cloud connection string)
echo -n "redis://default:password@host:port" | \
  gcloud secrets create redis-url --data-file=- --project=$GCP_PROJECT_ID

# JWT Access Secret
echo -n "your-jwt-access-secret" | \
  gcloud secrets create jwt-access-secret --data-file=- --project=$GCP_PROJECT_ID

# JWT Refresh Secret
echo -n "your-jwt-refresh-secret" | \
  gcloud secrets create jwt-refresh-secret --data-file=- --project=$GCP_PROJECT_ID

# Google OAuth Client ID
echo -n "your-google-client-id" | \
  gcloud secrets create google-client-id --data-file=- --project=$GCP_PROJECT_ID

# Google OAuth Client Secret
echo -n "your-google-client-secret" | \
  gcloud secrets create google-client-secret --data-file=- --project=$GCP_PROJECT_ID
```

### Configure Workload Identity Federation (for GitHub Actions)

1. Create a Workload Identity Pool:
```bash
gcloud iam workload-identity-pools create "github-pool" \
  --project=$GCP_PROJECT_ID \
  --location=global \
  --display-name="GitHub Actions"
```

2. Create a Workload Identity Provider:
```bash
gcloud iam workload-identity-pools providers create-oidc "github-provider" \
  --project=$GCP_PROJECT_ID \
  --location=global \
  --workload-identity-pool="github-pool" \
  --display-name="GitHub" \
  --attribute-mapping="google.subject=assertion.sub,assertion.aud=assertion.aud" \
  --issuer-uri="https://token.actions.githubusercontent.com"
```

3. Create a service account for GitHub Actions:
```bash
gcloud iam service-accounts create github-deployer \
  --display-name="GitHub Actions Deployer" \
  --project=$GCP_PROJECT_ID

# Grant Cloud Run deployment permissions
gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
  --member="serviceAccount:github-deployer@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/run.admin"

# Grant Artifact Registry push permissions
gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
  --member="serviceAccount:github-deployer@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/artifactregistry.writer"

# Grant Secret Manager access
gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
  --member="serviceAccount:github-deployer@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

4. Create Workload Identity binding:
```bash
gcloud iam service-accounts add-iam-policy-binding \
  github-deployer@$GCP_PROJECT_ID.iam.gserviceaccount.com \
  --project=$GCP_PROJECT_ID \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/github-pool/attribute.repository/YOUR_GITHUB_ORG/YOUR_GITHUB_REPO"
```

5. Get the Workload Identity Provider resource name:
```bash
gcloud iam workload-identity-pools providers describe github-provider \
  --project=$GCP_PROJECT_ID \
  --location=global \
  --workload-identity-pool=github-pool
```

Store the output in GitHub Secrets as `WIF_PROVIDER`.

---

## 2. Neon PostgreSQL Setup

### Create Neon Project and Database

1. Go to [Neon Console](https://console.neon.tech/)
2. Click "Create project"
3. Enter project name (e.g., `auth-service`)
4. Select region closest to your GCP region
5. Click "Create project"

### Get Connection String

1. In the Neon console, select your project
2. Click "Connection string" or find the database connection details
3. Copy the connection string (format: `postgresql://user:password@host/database`)

### Create Database User and Database

If you want a separate user for the application:

```bash
# Connect to the default database first
psql "postgresql://postgres:password@host/postgres"

# Create database
CREATE DATABASE auth_service;

# Create user
CREATE USER auth_app WITH PASSWORD 'secure-password';

# Grant privileges
GRANT ALL PRIVILEGES ON DATABASE auth_service TO auth_app;
ALTER DATABASE auth_service OWNER TO auth_app;
```

The connection string will be: `postgresql://auth_app:secure-password@host/auth_service`

### Store in GCP Secret Manager

```bash
echo -n "postgresql://user:password@host/database" | \
  gcloud secrets create db-url --data-file=- --project=$GCP_PROJECT_ID
```

### Run Migrations

Migrations are managed via the `migrations/` directory. They're run automatically during application startup.

---

## 3. Redis Cloud Setup

### Create Redis Cloud Account and Database

1. Go to [Redis Cloud Console](https://app.redislabs.com/)
2. Create a new database
3. Choose a deployment plan (Free tier available)
4. Select region closest to your GCP region
5. Create the database

### Get Connection Details

1. In Redis Cloud console, select your database
2. Copy the "Redis CLI address" (format: `redis-host:port`)
3. Copy the password from "Access Control" or "Security"

### Connection String Format

The connection string format is:
```
redis://default:password@host:port
```

Example:
```
redis://default:my-secure-password@redis-12345.c123.us-east-1-2.ec2.cloud.redislabs.com:12345
```

### Store in GCP Secret Manager

```bash
echo -n "redis://default:password@host:port" | \
  gcloud secrets create redis-url --data-file=- --project=$GCP_PROJECT_ID
```

### Test Connection

```bash
redis-cli -u "redis://default:password@host:port" ping
```

---

## 4. Local Development Setup

### Environment Variables

Create a `.env` file in the project root:

```env
# Server
PORT=8080
ENV=development

# Database (Neon)
DATABASE_URL=postgresql://user:password@host/database

# Redis (Redis Cloud)
REDIS_ADDR=redis://default:password@host:port
REDIS_PASSWORD=password

# JWT
JWT_ACCESS_SECRET=your-access-secret-min-32-chars
JWT_REFRESH_SECRET=your-refresh-secret-min-32-chars
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# OAuth (Google)
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback

# TOTP
TOTP_ISSUER=AuthService
TOTP_ENCRYPTION_KEY=your-totp-key-min-32-chars

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

# Rate Limiting
RATE_LIMIT_LOGIN=5-M
RATE_LIMIT_GLOBAL=100-M

# GCP (optional for local dev)
GCP_PROJECT_ID=your-project-id
```

### Generate Secrets

For development, you can generate secure secrets:

```bash
# Generate JWT secrets (use base64 of random data)
openssl rand -base64 32

# Generate TOTP encryption key
openssl rand -base64 32
```

### Install Dependencies

```bash
go mod download
```

### Run Database Migrations

```bash
# Using the CLI (if available)
go run cmd/main.go migrate

# Or use sql-migrate or similar
```

### Start Development Server

```bash
go run cmd/main.go
```

The server will start on `http://localhost:8080`

---

## 5. GitHub Secrets Configuration

Add these secrets to your GitHub repository settings (`Settings > Secrets and variables > Actions`):

| Secret Name | Value |
|---|---|
| `GCP_PROJECT_ID` | Your GCP project ID |
| `WIF_PROVIDER` | Workload Identity Provider resource name |
| `WIF_SERVICE_ACCOUNT` | `github-deployer@PROJECT_ID.iam.gserviceaccount.com` |
| `GCP_SERVICE_ACCOUNT` | `auth-service-runner@PROJECT_ID.iam.gserviceaccount.com` |
| `TOTP_ENCRYPTION_KEY` | TOTP encryption key (base64 or plain) |

---

## 6. Deployment

### Manual Deployment to Cloud Run

```bash
# Set project
gcloud config set project $GCP_PROJECT_ID

# Build and push image
docker build -t us-central1-docker.pkg.dev/$GCP_PROJECT_ID/auth-service/api:latest .
docker push us-central1-docker.pkg.dev/$GCP_PROJECT_ID/auth-service/api:latest

# Deploy to Cloud Run
gcloud run deploy auth-service \
  --image=us-central1-docker.pkg.dev/$GCP_PROJECT_ID/auth-service/api:latest \
  --region=us-central1 \
  --platform=managed \
  --service-account=auth-service-runner@$GCP_PROJECT_ID.iam.gserviceaccount.com \
  --set-secrets=DATABASE_URL=db-url:latest,REDIS_ADDR=redis-url:latest,JWT_ACCESS_SECRET=jwt-access-secret:latest,JWT_REFRESH_SECRET=jwt-refresh-secret:latest,GOOGLE_CLIENT_ID=google-client-id:latest,GOOGLE_CLIENT_SECRET=google-client-secret:latest \
  --allow-unauthenticated
```

### Automatic Deployment via GitHub Actions

Push to the `master` branch triggers automatic deployment via `.github/workflows/deploy.yml`.

The workflow:
1. Builds the Docker image
2. Pushes to GCP Artifact Registry
3. Deploys to Cloud Run with secrets from Secret Manager

---

## Troubleshooting

### GCP Authentication Issues

```bash
# Re-authenticate
gcloud auth application-default login

# Verify authentication
gcloud auth list
```

### Redis Connection Issues

```bash
# Test Redis connection
redis-cli -u "redis://default:password@host:port" ping
```

### Database Connection Issues

```bash
# Test PostgreSQL connection
psql "postgresql://user:password@host/database"
```

### Secret Manager Access

```bash
# List all secrets
gcloud secrets list --project=$GCP_PROJECT_ID

# View specific secret version
gcloud secrets versions access latest --secret=db-url --project=$GCP_PROJECT_ID
```

---

## References

- [Google Cloud Run Documentation](https://cloud.google.com/run/docs)
- [GCP Secret Manager](https://cloud.google.com/secret-manager)
- [Neon PostgreSQL](https://neon.tech/)
- [Redis Cloud](https://redis.com/cloud/)
- [Workload Identity Federation Setup](https://cloud.google.com/docs/authentication/workload-identity-federation)
