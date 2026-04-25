# Stage 1: Vue build
FROM node:20-alpine AS vue-builder

WORKDIR /app/web

COPY web/package*.json ./
RUN npm ci

COPY web/ .

RUN npm run build

# Stage 2: Go build
FROM golang:1.26-alpine AS go-builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=vue-builder /app/web/dist ./web/dist

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o auth-service ./cmd/server

# Stage 3: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=go-builder /app/auth-service .
COPY --from=go-builder /app/migrations ./migrations
COPY --from=go-builder /app/web/dist ./web/dist

USER appuser

EXPOSE 8080

CMD ["./auth-service"]
