# Documentation

This directory contains comprehensive documentation for setting up, deploying, and troubleshooting the Auth Service.

## Quick Links

### For First-Time Setup
- **[SETUP.md](SETUP.md)** — Complete step-by-step setup guide for GCP, Neon PostgreSQL, Redis Cloud, and local development

### For Development
- **[ENVIRONMENT.md](ENVIRONMENT.md)** — All environment variables, secrets management, and configuration options

### For Deployment
- **[DEPLOYMENT.md](DEPLOYMENT.md)** — How to deploy to Google Cloud Run, GitHub Actions CI/CD, monitoring, and rollback procedures

### For Troubleshooting
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)** — Common issues and solutions for GCP, databases, Redis, Docker, and deployment

---

## Documentation Overview

### [SETUP.md](SETUP.md)

Complete setup instructions covering:
- **GCP Project Setup**: Create project, enable APIs, create service accounts, set up IAM roles
- **Artifact Registry**: Create repository for Docker images
- **Secret Manager**: Store sensitive configuration (database URLs, API keys, etc.)
- **Workload Identity Federation**: Set up GitHub Actions authentication with GCP
- **Neon PostgreSQL**: Database setup, migrations, and connection configuration
- **Redis Cloud**: Create Redis database and configure connection
- **Local Development**: Create `.env` file with local configuration
- **GitHub Secrets**: Configure secrets for CI/CD pipeline

**Start here if**:
- Setting up the project for the first time
- Need to create a new GCP project
- Need to set up database and Redis
- Need to configure GitHub Actions

### [ENVIRONMENT.md](ENVIRONMENT.md)

Detailed reference for all environment variables:
- **Server Configuration**: Port, environment mode
- **Database**: PostgreSQL connection string
- **Redis**: Redis connection and credentials
- **JWT**: Access/refresh secrets and expiry times
- **OAuth**: Google credentials
- **TOTP**: Encryption key for two-factor authentication
- **CORS**: Allowed origins
- **Rate Limiting**: Request limits per endpoint
- **GCP**: Project ID and secret manager integration

Also covers:
- How variables are loaded in different environments
- GCP Secret Manager integration
- Secret rotation procedures
- Security best practices

**Start here if**:
- Need to add or modify a configuration variable
- Need to create or rotate secrets
- Need to understand how local vs production config works
- Deploying to a new environment

### [DEPLOYMENT.md](DEPLOYMENT.md)

Step-by-step deployment instructions:
- **Quick Start**: Automatic deployment via GitHub Actions
- **Manual Deployment**: Build, push, and deploy commands
- **CI/CD Workflow**: How GitHub Actions workflow works
- **Troubleshooting**: Common deployment issues
- **Post-Deployment Verification**: Testing the deployed service
- **Rollback**: How to revert to previous version
- **Scaling**: Adjust memory, CPU, instance limits
- **Monitoring**: View logs and metrics
- **Cost Optimization**: Understand pricing and reduce costs

**Start here if**:
- Deploying the application to production
- Need to manually deploy for testing
- Service is having issues after deployment
- Need to understand the deployment process

### [TROUBLESHOOTING.md](TROUBLESHOOTING.md)

Solutions for common issues organized by component:
- **GCP Setup Issues**: Project/API/authentication errors
- **Secret Manager**: Secret not found, access denied
- **Neon Database**: Connection, migration, and performance issues
- **Redis Cloud**: Connection, authentication errors
- **GitHub Actions**: CI/CD workflow failures
- **Docker**: Build failures and context issues
- **Cloud Run**: Deployment, scaling, and domain issues
- **Local Development**: Port conflicts, migrations, environment variables
- **Google OAuth**: Credential and redirect URI issues
- **Rate Limiting**: Configuration and functionality
- **Performance**: High latency, memory issues

**Start here if**:
- Something isn't working and you need a fix
- Getting an error message and need to solve it
- Experiencing performance or reliability issues

---

## Quick Setup Checklist

Use this checklist for a complete setup:

- [ ] GCP Project Created
  - [ ] Project ID noted
  - [ ] Required APIs enabled
  - [ ] Artifact Registry repository created

- [ ] Service Accounts Created
  - [ ] `auth-service-runner` for Cloud Run
  - [ ] `github-deployer` for GitHub Actions
  - [ ] Correct IAM roles assigned

- [ ] Neon PostgreSQL
  - [ ] Project created
  - [ ] Database connection string obtained
  - [ ] Added to GCP Secret Manager as `db-url`

- [ ] Redis Cloud
  - [ ] Database created
  - [ ] Connection string obtained
  - [ ] Added to GCP Secret Manager as `redis-url`

- [ ] GCP Secrets Created
  - [ ] `db-url` — database connection
  - [ ] `redis-url` — Redis connection
  - [ ] `jwt-access-secret` — JWT signing key
  - [ ] `jwt-refresh-secret` — JWT refresh key
  - [ ] `google-client-id` — OAuth client ID
  - [ ] `google-client-secret` — OAuth client secret

- [ ] Workload Identity Federation
  - [ ] WIF pool created (`github-pool`)
  - [ ] WIF provider created (`github-provider`)
  - [ ] Service account bindings configured
  - [ ] Provider resource name noted

- [ ] GitHub Secrets Configured
  - [ ] `GCP_PROJECT_ID`
  - [ ] `WIF_PROVIDER`
  - [ ] `WIF_SERVICE_ACCOUNT`
  - [ ] `GCP_SERVICE_ACCOUNT`
  - [ ] `TOTP_ENCRYPTION_KEY`

- [ ] Local Development Setup
  - [ ] `.env` file created
  - [ ] All variables populated
  - [ ] Database migrations run
  - [ ] Server starts successfully

- [ ] Deployment Verified
  - [ ] GitHub Actions workflow succeeds
  - [ ] Service deployed to Cloud Run
  - [ ] Health checks pass
  - [ ] Logs show no errors

---

## Key Concepts

### Environment Management

The application loads configuration differently based on the environment:

```
Local Development
├── .env file (plaintext)
└── Environment variables

Production (Cloud Run)
├── GCP Secret Manager (encrypted secrets)
└── Cloud Run environment variables
```

When `GCP_PROJECT_ID` is set, the application automatically fetches secrets from GCP Secret Manager.

### Secret Management

Sensitive data is stored in GCP Secret Manager:
- Database credentials
- Redis password
- JWT signing keys
- OAuth credentials

Non-sensitive configuration uses environment variables:
- Port numbers
- Environment mode (dev/prod)
- CORS settings
- Rate limits

### CI/CD Pipeline

Push to `master` branch triggers:
1. Docker image build
2. Push to Artifact Registry
3. Deploy to Cloud Run
4. Run database migrations (if needed)

The workflow uses Workload Identity Federation for secure, keyless authentication.

---

## Architecture

### Components

```
┌─────────────────┐
│  GitHub Actions │  (CI/CD)
└────────┬────────┘
         │
    ┌────▼─────────────────────┐
    │  GCP Artifact Registry    │  (Docker images)
    └────┬─────────────────────┘
         │
    ┌────▼──────────────────┐
    │  Google Cloud Run      │  (Application)
    ├──────────────────────┤
    │  auth-service:latest  │
    └────┬──────┬──────┬───┘
         │      │      │
    ┌────▼──┐┌──▼──┐┌──▼──────────────┐
    │  Neon │ │Redis│ │GCP Secret Mgr  │
    │  Pgsql│ │Cloud│ │(secrets)       │
    └───────┘└─────┘└────────────────┘
```

### Data Flow

1. **Configuration Loading**
   - Application checks for `GCP_PROJECT_ID` environment variable
   - If set, fetches secrets from GCP Secret Manager
   - Otherwise, reads from environment variables and `.env` file

2. **Deployment Process**
   - Push to `master` → GitHub Actions trigger
   - Build Docker image → Push to Artifact Registry
   - Deploy to Cloud Run with secrets and environment variables
   - Cloud Run injects secrets at runtime

3. **Secret Rotation**
   - Update secret in GCP Secret Manager
   - Redeploy Cloud Run service
   - New revision uses updated secret

---

## Common Workflows

### Deploy New Feature

```bash
# 1. Create feature branch
git checkout -b feature/my-feature

# 2. Make changes and test locally
# ... code changes ...
go test ./...

# 3. Commit and push
git add .
git commit -m "feat: add my feature"
git push origin feature/my-feature

# 4. Create PR and merge to master
# ... review and merge ...

# 5. Automatic deployment happens
# Check GitHub Actions → Deployments tab
```

### Rotate Secrets

```bash
# 1. Generate new secret
NEW_SECRET=$(openssl rand -base64 32)

# 2. Update in GCP Secret Manager
echo -n "$NEW_SECRET" | \
  gcloud secrets versions add jwt-access-secret --data-file=-

# 3. Redeploy to pick up new secret
gcloud run deploy auth-service \
  --region=us-central1 \
  --image=us-central1-docker.pkg.dev/$GCP_PROJECT_ID/auth-service/api:latest
```

### Debug Deployment Issues

```bash
# 1. Check latest logs
gcloud run services logs read auth-service --region=us-central1 --limit=100

# 2. Verify secrets are accessible
gcloud secrets list --project=$GCP_PROJECT_ID

# 3. Check service configuration
gcloud run services describe auth-service --region=us-central1

# 4. Redeploy if needed
git push origin master  # Triggers automatic redeploy
```

### Test Locally with Production Secrets

```bash
# Download secrets from GCP
export $(gcloud secrets versions access latest --secret=jwt-access-secret | xargs -I {} echo "JWT_ACCESS_SECRET={}")

# Start server
go run cmd/main.go
```

---

## Important Files

### Configuration
- `.env` — Local development environment variables (git-ignored)
- `config/config.go` — Configuration loading and validation

### Deployment
- `.github/workflows/deploy.yml` — CI/CD pipeline
- `Dockerfile` — Docker image definition
- `docker-compose.yml` — Local development with Docker

### Database
- `migrations/` — Database schema migrations
- `pkg/database/` — Database connection and queries

### Application
- `cmd/main.go` — Application entry point
- `internal/` — Core business logic
- `pkg/` — Reusable packages
- `web/` — Frontend (if applicable)

---

## Support

For issues not covered in the documentation:

1. Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common solutions
2. Review application logs and error messages
3. Check the relevant official documentation:
   - [Google Cloud Run](https://cloud.google.com/run/docs)
   - [Neon PostgreSQL](https://neon.tech/docs)
   - [Redis Cloud](https://docs.redis.com/latest/)
   - [GitHub Actions](https://docs.github.com/en/actions)

---

## Document Updates

Last updated: April 2026

To keep documentation current:
- Update when adding new configuration variables
- Update when changing deployment process
- Add troubleshooting entries as new issues are discovered
- Review quarterly for accuracy

---

## Related Resources

- [Project Repository](https://github.com/yourusername/auth-service)
- [GCP Console](https://console.cloud.google.com)
- [Neon Console](https://console.neon.tech)
- [Redis Cloud Console](https://app.redislabs.com)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
