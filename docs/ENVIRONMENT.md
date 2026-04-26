# Environment Variables and Secrets

This document describes all environment variables used by the Auth Service and how they're configured across different environments.

## Overview

The application uses different methods to load configuration depending on the environment:

- **Local Development**: `.env` file with plaintext values
- **Cloud Run**: GCP Secret Manager for sensitive data, environment variables for non-sensitive config
- **Docker**: Environment variables or Secret Manager integration

---

## Configuration Variables

### Server Configuration

| Variable | Type | Required | Default | Description |
|---|---|---|---|---|
| `PORT` | int | No | 8080 | HTTP server port |
| `SERVER_PORT` | int | No | 8080 | Alternative port variable (fallback if PORT not set) |
| `ENV` | string | No | development | Environment mode: `development`, `staging`, `production` |

### Database Configuration

| Variable | Type | Required | Default | Description |
| `DATABASE_URL` | string | Yes | - | PostgreSQL connection string. Format: `postgresql://user:password@host:port/database` |

**Source**:
- Local: `.env` file or environment variable
- Cloud Run: GCP Secret Manager (`db-url` secret)

**Example**:
```
postgresql://auth_user:secure-password@neon-db.neon.tech:5432/auth_service
```

### Redis Configuration

| Variable | Type | Required | Default | Description |
|---|---|---|---|---|
| `REDIS_ADDR` | string | Yes* | - | Redis connection string. Supports both `host:port` and full Redis URL format |
| `REDIS_PASSWORD` | string | No | - | Redis password (extracted from URL if using Redis URL format) |

**Source**:
- Local: `.env` file or environment variable
- Cloud Run: GCP Secret Manager (`redis-url` secret)

**Examples**:
```
# Host and port format
redis-db.c123.us-east-1-2.ec2.cloud.redislabs.com:12345

# Full Redis URL format (preferred for Cloud Run)
redis://default:my-password@redis-db.c123.us-east-1-2.ec2.cloud.redislabs.com:12345
```

### JWT Configuration

| Variable | Type | Required | Default | Description |
|---|---|---|---|---|
| `JWT_ACCESS_SECRET` | string | Yes | - | Secret key for signing access tokens. Minimum 32 characters recommended |
| `JWT_REFRESH_SECRET` | string | Yes | - | Secret key for signing refresh tokens. Minimum 32 characters recommended |
| `JWT_ACCESS_EXPIRY` | duration | No | 15m | Access token expiration time (Go duration format: `15m`, `1h`, etc.) |
| `JWT_REFRESH_EXPIRY` | duration | No | 168h | Refresh token expiration time (Go duration format) |

**Source**:
- Local: `.env` file or environment variable
- Cloud Run: GCP Secret Manager (`jwt-access-secret`, `jwt-refresh-secret`)

**Generating Secrets**:
```bash
# Generate 32+ character secret
openssl rand -base64 32
```

### OAuth Configuration (Google)

| Variable | Type | Required | Default | Description |
|---|---|---|---|---|
| `GOOGLE_CLIENT_ID` | string | Yes | - | OAuth client ID from Google Cloud Console |
| `GOOGLE_CLIENT_SECRET` | string | Yes | - | OAuth client secret from Google Cloud Console |
| `GOOGLE_REDIRECT_URL` | string | No | - | OAuth redirect URI (e.g., `https://example.com/auth/google/callback`) |

**Source**:
- Local: `.env` file or environment variable
- Cloud Run: GCP Secret Manager (`google-client-id`, `google-client-secret`)

**Setup**:
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create OAuth 2.0 credentials (Web application)
3. Add authorized redirect URIs
4. Copy Client ID and Client Secret

### TOTP Configuration

| Variable | Type | Required | Default | Description |
|---|---|---|---|---|
| `TOTP_ENCRYPTION_KEY` | string | Yes | - | Encryption key for TOTP secrets. Minimum 32 characters |
| `TOTP_ISSUER` | string | No | AuthService | TOTP issuer name shown in authenticator apps |

**Source**:
- Local: `.env` file or environment variable
- Cloud Run: Environment variable (not stored in Secret Manager for performance)

**Generating Key**:
```bash
openssl rand -base64 32
```

### CORS Configuration

| Variable | Type | Required | Default | Description |
|---|---|---|---|---|
| `CORS_ALLOWED_ORIGINS` | string | No | empty | Comma-separated list of allowed origins. Use `*` for all origins (development only) |

**Examples**:
```
# Development
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

# Production
CORS_ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com

# Allow all (development only)
CORS_ALLOWED_ORIGINS=*
```

### Rate Limiting Configuration

| Variable | Type | Required | Default | Description |
|---|---|---|---|---|
| `RATE_LIMIT_LOGIN` | string | No | 5-M | Login endpoint rate limit (format: `requests-duration`, e.g., `5-M` = 5 requests per minute) |
| `RATE_LIMIT_GLOBAL` | string | No | 100-M | Global rate limit for all endpoints |

**Format**: `{requests}-{duration}` where duration is `S` (second), `M` (minute), `H` (hour)

**Examples**:
```
5-M   # 5 requests per minute
10-S  # 10 requests per second
100-H # 100 requests per hour
```

### GCP Configuration

| Variable | Type | Required | Default | Description |
|---|---|---|---|---|
| `GCP_PROJECT_ID` | string | No | - | GCP project ID. When set, enables Secret Manager integration for loading secrets |

**Source**:
- Local: `.env` file or environment variable
- Cloud Run: Automatically provided by Cloud Run environment

---

## Environment Profiles

### Development Environment

**.env file example**:
```env
PORT=8080
ENV=development

DATABASE_URL=postgresql://postgres:password@localhost:5432/auth_service
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=

JWT_ACCESS_SECRET=dev-secret-min-32-chars-for-testing
JWT_REFRESH_SECRET=dev-refresh-secret-min-32-chars
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

GOOGLE_CLIENT_ID=xxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=xxx
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback

TOTP_ISSUER=AuthService
TOTP_ENCRYPTION_KEY=dev-totp-key-min-32-chars-for-testing

CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

RATE_LIMIT_LOGIN=5-M
RATE_LIMIT_GLOBAL=100-M
```

### Cloud Run Deployment

Secrets are loaded from GCP Secret Manager, environment variables are set in the Cloud Run service:

```bash
gcloud run deploy auth-service \
  --image=us-central1-docker.pkg.dev/$PROJECT_ID/auth-service/api:latest \
  --region=us-central1 \
  --set-secrets=\
DATABASE_URL=db-url:latest,\
REDIS_ADDR=redis-url:latest,\
JWT_ACCESS_SECRET=jwt-access-secret:latest,\
JWT_REFRESH_SECRET=jwt-refresh-secret:latest,\
GOOGLE_CLIENT_ID=google-client-id:latest,\
GOOGLE_CLIENT_SECRET=google-client-secret:latest \
  --set-env-vars=\
ENV=production,\
TOTP_ISSUER=AuthService,\
TOTP_ENCRYPTION_KEY=$TOTP_ENCRYPTION_KEY,\
CORS_ALLOWED_ORIGINS=https://example.com \
  --service-account=auth-service-runner@$PROJECT_ID.iam.gserviceaccount.com
```

---

## GCP Secret Manager Integration

When `GCP_PROJECT_ID` is set, the application automatically loads secrets from GCP Secret Manager:

### Secret Names

The application expects secrets with these names in Secret Manager:

| Secret Name | Maps To |
|---|---|
| `db-url` | `DATABASE_URL` |
| `redis-url` | `REDIS_ADDR` |
| `jwt-access-secret` | `JWT_ACCESS_SECRET` |
| `jwt-refresh-secret` | `JWT_REFRESH_SECRET` |
| `google-client-id` | `GOOGLE_CLIENT_ID` |
| `google-client-secret` | `GOOGLE_CLIENT_SECRET` |

### Creating Secrets

```bash
export GCP_PROJECT_ID="your-project-id"

# Database URL
echo -n "postgresql://user:password@host/database" | \
  gcloud secrets create db-url --data-file=- --project=$GCP_PROJECT_ID

# Redis URL
echo -n "redis://default:password@host:port" | \
  gcloud secrets create redis-url --data-file=- --project=$GCP_PROJECT_ID

# JWT Secrets
echo -n "your-jwt-access-secret" | \
  gcloud secrets create jwt-access-secret --data-file=- --project=$GCP_PROJECT_ID

echo -n "your-jwt-refresh-secret" | \
  gcloud secrets create jwt-refresh-secret --data-file=- --project=$GCP_PROJECT_ID

# OAuth Secrets
echo -n "your-client-id" | \
  gcloud secrets create google-client-id --data-file=- --project=$GCP_PROJECT_ID

echo -n "your-client-secret" | \
  gcloud secrets create google-client-secret --data-file=- --project=$GCP_PROJECT_ID
```

### Updating Secrets

```bash
# Add a new version to a secret
echo -n "new-secret-value" | \
  gcloud secrets versions add db-url --data-file=- --project=$GCP_PROJECT_ID

# Verify it was added
gcloud secrets versions list db-url --project=$GCP_PROJECT_ID
```

---

## Secret Rotation

### Manual Rotation

1. Generate new secret value
2. Add new version to GCP Secret Manager
3. Update Cloud Run service to use the new version (if needed)

### Automatic Rotation (Recommended)

For critical secrets like JWT and OAuth credentials, implement key rotation:

```bash
# Create a new version for automatic rollover
echo -n "new-secret-value" | \
  gcloud secrets versions add jwt-access-secret --data-file=- --project=$GCP_PROJECT_ID

# Cloud Run will use the latest version automatically
```

---

## Local Testing with Cloud Run Secrets

For local testing with Cloud Run's secret values:

```bash
# Download secrets from Cloud Run
export GCP_PROJECT_ID="your-project-id"

export DATABASE_URL=$(gcloud secrets versions access latest --secret=db-url --project=$GCP_PROJECT_ID)
export REDIS_ADDR=$(gcloud secrets versions access latest --secret=redis-url --project=$GCP_PROJECT_ID)
export JWT_ACCESS_SECRET=$(gcloud secrets versions access latest --secret=jwt-access-secret --project=$GCP_PROJECT_ID)
export JWT_REFRESH_SECRET=$(gcloud secrets versions access latest --secret=jwt-refresh-secret --project=$GCP_PROJECT_ID)
export GOOGLE_CLIENT_ID=$(gcloud secrets versions access latest --secret=google-client-id --project=$GCP_PROJECT_ID)
export GOOGLE_CLIENT_SECRET=$(gcloud secrets versions access latest --secret=google-client-secret --project=$GCP_PROJECT_ID)

# Now run the application
go run cmd/main.go
```

---

## Validation

The application validates all required environment variables on startup. If any required variable is missing:

1. The application will log an error
2. Configuration loading will fail
3. The application will exit with a non-zero status

Check the startup logs to identify missing variables.

---

## Security Best Practices

1. **Never commit `.env` to version control** — add to `.gitignore`
2. **Use strong secrets** — minimum 32 characters, random generation recommended
3. **Rotate secrets regularly** — especially OAuth and JWT secrets
4. **Use GCP Secret Manager** — for production environments instead of environment variables
5. **Limit secret access** — configure IAM roles for Secret Manager access
6. **Audit secret access** — review Cloud Audit Logs for secret access patterns

---

## References

- [GCP Secret Manager](https://cloud.google.com/secret-manager)
- [Cloud Run Configuration](https://cloud.google.com/run/docs/configuring/services)
- [Go Duration Format](https://golang.org/pkg/time/#ParseDuration)
