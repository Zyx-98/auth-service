.PHONY: help build run test lint clean migrate-up migrate-down docker-up docker-down

help:
	@echo "Available commands:"
	@echo "  make build         - Build the server"
	@echo "  make run           - Run the server"
	@echo "  make test          - Run tests"
	@echo "  make lint          - Run linter"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make migrate-up    - Run migrations up"
	@echo "  make migrate-down  - Run migrations down"
	@echo "  make docker-up     - Start Docker containers"
	@echo "  make docker-down   - Stop Docker containers"

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

test:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
	rm -f coverage.out

migrate-up:
	migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable" up

migrate-down:
	migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable" down

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

.PHONY: $(MAKECMDGOALS)
