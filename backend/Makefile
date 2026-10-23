# ==========================================
# NENAV Backend Makefile
# ==========================================

.PHONY: help run build test lint fmt vet deps \
        migrate-up migrate-down migrate-validate migrate-create \
        docker-build clean check

BINARY := nenav-api
COMPOSE := docker compose -f compose.yaml

# --- Help ---
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

# --- Development ---
run: ## Run the API server
	go run ./cmd/api

build: ## Build the binary
	CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/$(BINARY) ./cmd/api

test: ## Run tests with race detection
	go test ./... -v -race -count=1 -coverprofile=coverage.out

test-short: ## Run tests without verbose
	go test ./... -race -count=1

lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	gofmt -w .
	goimports -w . 2>/dev/null || true

vet: ## Vet code
	go vet ./...

deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

# --- Database ---
migrate-up: ## Run all pending migrations
	go run ./cmd/api -migrate-up

migrate-down: ## Rollback last migration
	go run ./cmd/api -migrate-down

migrate-validate: ## Validate migration files
	go run ./cmd/api -validate-migrations

migrate-create: ## Create new migration (usage: make migrate-create NAME=add_bookings)
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	@TS=$$(date +%Y%m%d%H%M%S); \
	touch migrations/$${TS}_$(NAME).up.sql; \
	touch migrations/$${TS}_$(NAME).down.sql; \
	echo "Created: migrations/$${TS}_$(NAME).{up,down}.sql"

# --- Docker ---
dev-up: ## Start local PostgreSQL + Redis
	$(COMPOSE) up -d
	@echo "Waiting for services..."
	@sleep 3
	@$(COMPOSE) ps

dev-down: ## Stop local infrastructure
	$(COMPOSE) down

dev-clean: ## Stop and remove all local data (DESTRUCTIVE)
	$(COMPOSE) down -v

dev-logs: ## Tail infrastructure logs
	$(COMPOSE) logs -f

docker-build: ## Build API Docker image
	docker build -f Dockerfile -t $(BINARY):latest ..

# --- Quality ---
check: fmt vet lint test ## Run all quality checks
	@echo "All checks passed."

# --- Clean ---
clean: ## Clean build artifacts
	rm -rf bin/ coverage.out
	go clean -cache -testcache
