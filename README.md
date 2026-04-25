# Auth Service

A standalone, production-ready authentication microservice built with Go, Vue 3, and deployed on GCP Cloud Run. Provides JWT token authentication, Google OAuth2, TOTP 2FA, and RBAC for modern web applications.

## Features

✅ **Dual-token JWT** — Access tokens (15min) + Refresh tokens (7d) with separate secrets  
✅ **Google OAuth2** — One-click OAuth flow with user upsert  
✅ **TOTP 2FA** — RFC 6238 compatible, works with Google Authenticator & iOS Passwords  
✅ **RBAC** — Roles, permissions, and user role assignment  
✅ **Token Introspection** — Validate tokens from other services (`/auth/introspect`)  
✅ **Redis Session Management** — Instant logout, distributed cache  
✅ **Rate Limiting** — Redis-backed per-IP and per-endpoint limits  
✅ **Vue 3 UI** — Embedded login/register/2FA frontend, Vite + TypeScript  
✅ **Multi-stage Docker** — Single 47MB container for API + frontend  
✅ **CI/CD** — GitHub Actions + Cloud Build, Workload Identity Federation  

## Tech Stack

| Component | Technology |
|-----------|------------|
| **Backend** | Go 1.26, Gin, GORM |
| **Database** | PostgreSQL + pgcrypto (UUID, hashing) |
| **Cache** | Redis (refresh tokens, rate limiting) |
| **Frontend** | Vue 3, Vite, TypeScript |
| **Auth** | JWT (HS256), Google OAuth2, TOTP |
| **Deployment** | GCP Cloud Run, Artifact Registry |
| **CI/CD** | GitHub Actions, Cloud Build |

## Quick Start

### Local Development

**Prerequisites:**
- Docker & Docker Compose
- Go 1.26+ (for building without Docker)
- Node 20+ (for frontend development)

**Start services:**
```bash
make docker-up
```

This starts:
- PostgreSQL (port 5432, user: `postgres`, password: `postgres`)
- Redis (port 6379)
- API server (port 8080)
- Frontend dev server (port 3000)

**Verify:**
```bash
curl http://localhost:8080/health
open http://localhost:3000
```

**Stop:**
```bash
make docker-down
```

### API Endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/auth/register` | — | Register with email + password |
| POST | `/auth/login` | — | Login, returns tokens or 2FA flag |
| POST | `/auth/refresh` | — | Refresh access token |
| POST | `/auth/logout` | JWT | Revoke refresh token |
| GET | `/auth/login/google` | — | OAuth2 redirect URL |
| GET | `/auth/callback/google` | — | OAuth2 callback |
| POST | `/auth/2fa/setup` | JWT | Generate TOTP secret + QR |
| POST | `/auth/2fa/verify` | JWT/temp | Verify TOTP code |
| POST | `/auth/2fa/disable` | JWT | Disable 2FA |
| GET | `/auth/me` | JWT | Current user profile |
| POST | `/auth/introspect` | — | Validate token (for other services) |

### Example: Login & Token Refresh

```bash
# Register
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123","password_confirm":"password123"}'

# Login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# Response includes: access_token, refresh_token, expires_in
# Use access_token in Authorization header: Bearer <token>

# Refresh access token
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token>"}'
```

### Example: Token Introspection (for other services)

```bash
curl -X POST http://localhost:8080/auth/introspect \
  -H "Content-Type: application/json" \
  -d '{"token":"<access_token>"}'

# Response:
# {
#   "data": {
#     "valid": true,
#     "user_id": "550e8400-e29b-41d4-a716-446655440000",
#     "email": "user@example.com",
#     "roles": ["user"],
#     "permissions": ["profile:read", "profile:write"],
#     "expires_at": 1234567890
#   }
# }
```

## Project Structure

```
auth-service/
├── cmd/server/                 # Entry point, dependency wiring
├── internal/
│   ├── domain/                 # Entity, Repository interfaces
│   ├── service/                # Business logic (Auth, OAuth, TOTP, RBAC)
│   ├── handler/                # HTTP handlers
│   ├── middleware/             # JWT auth, rate limit, security headers
│   ├── repository/             # GORM & Redis implementations
│   ├── dto/                    # Request/response types
│   └── app/                    # Router & server setup
├── pkg/
│   ├── jwt/                    # Token generation/verification
│   ├── oauth/                  # Google OAuth2 client
│   ├── totp/                   # TOTP setup, verify, AES encrypt
│   ├── hash/                   # bcrypt password hashing
│   ├── apperror/               # Error handling
│   ├── response/               # HTTP response helpers
│   └── validator/              # Request validation
├── config/                     # Configuration, secret manager
├── migrations/                 # SQL migration files
├── web/                        # Vue 3 frontend (src/, dist/)
├── .github/workflows/          # CI/CD pipelines
│   ├── ci.yml                  # Tests, linter, Vue build
│   └── deploy.yml              # Build & push to Cloud Run
├── .gcp/
│   └── cloudbuild.yaml         # Cloud Build configuration
├── Dockerfile                  # Multi-stage build
├── docker-compose.yml          # Local dev services
├── Makefile                    # Common commands
├── GCP_SETUP.md                # GCP infrastructure guide
├── TESTING.md                  # Test strategy & running tests
└── README.md                   # This file
```

## Development

### Build locally (without Docker)
```bash
make build
./bin/server
```

### Run tests
```bash
make test              # All tests with coverage
go test -v ./...      # Verbose
go test -race ./...   # Race detector
```

### Linting & formatting
```bash
make lint
go fmt ./...
```

### Database migrations
```bash
make migrate-up        # Apply pending migrations
make migrate-down      # Rollback last migration
```

### Frontend development
```bash
cd web
npm install
npm run dev
```

## Configuration

Environment variables (see `.env.example`):

```bash
SERVER_PORT=8080
ENV=development
DATABASE_URL=postgres://user:password@localhost:5432/auth_db?sslmode=disable
REDIS_ADDR=localhost:6379
JWT_ACCESS_SECRET=change-me-in-production
JWT_REFRESH_SECRET=change-me-in-production
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/callback/google
TOTP_ENCRYPTION_KEY=00000000000000000000000000000000
TOTP_ISSUER=AuthService
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000
```

In production, secrets are loaded from **GCP Secret Manager** (see `GCP_SETUP.md`).

## Deployment

### GCP Cloud Run (recommended)

1. **Set up GCP infrastructure** (one-time):
   ```bash
   # Follow GCP_SETUP.md for APIs, secrets, IAM, databases
   ```

2. **Push to main branch**:
   ```bash
   git push origin main
   ```

3. **GitHub Actions + Cloud Build handles**:
   - CI checks (tests, lint)
   - Docker build (with caching)
   - Push to Artifact Registry
   - Deploy to Cloud Run

4. **Verify deployment**:
   ```bash
   gcloud run services describe auth-service --region=us-central1
   curl https://auth-service-HASH.run.app/health
   ```

## Security

- ✅ **Password hashing**: bcrypt cost=12
- ✅ **Token storage**: Redis with TTL (instant revocation)
- ✅ **2FA**: AES-256-GCM encrypted TOTP secrets
- ✅ **JWT secrets**: Rotate every 90 days (see GCP_SETUP.md)
- ✅ **CORS**: Configurable allowed origins
- ✅ **Rate limiting**: Redis-backed per-IP limits
- ✅ **Security headers**: HSTS, X-Content-Type-Options, etc.
- ✅ **HTTPS**: Cloud Run auto-provides TLS

## Testing

See `TESTING.md` for detailed test strategy.

```bash
make test                      # All tests with coverage
go test -v ./internal/service  # Service layer tests
go test -v ./internal/handler  # E2E handler tests
go tool cover -html=coverage.out  # View coverage
```

Test coverage:
- `pkg/jwt`: 90%+
- `pkg/totp`: 85%+
- `pkg/hash`: 95%+
- `internal/service`: 80%+

## Troubleshooting

### Docker build fails
- Ensure Go version matches `Dockerfile`: `golang:1.26-alpine`
- Run `docker system prune` to free space

### Tests fail locally but pass in CI
- Use `-race` flag: `go test -race ./...`
- Ensure database is accessible: `make docker-up`

### TOTP code not working
- Verify time is synced: `date`
- QR code should be scanned with Google Authenticator or iOS Passwords

### Cloud Run deployment fails
- Check Secret Manager secrets exist: `gcloud secrets list`
- Verify IAM roles assigned to Cloud Build SA
- View logs: `gcloud logging read "resource.service_name=auth-service" --limit 50`

## Contributing

1. Create feature branch: `git checkout -b feature/my-feature`
2. Commit changes with clear messages
3. Push and open PR
4. GitHub Actions runs tests automatically
5. Merge after approval

## License

MIT

## Support

For issues, open a GitHub issue or email support@example.com.
