MOCKERY_BIN := $(GOPATH)/bin/mockery

.PHONY: serve dev build tidy test test-cover mock mig-up mig-down seed lint fmt docker-up docker-down help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

serve: ## Run the API server
	go run cmd/api/main.go

dev: ## Run with hot-reload (requires: go install github.com/air-verse/air@latest)
	air

build: ## Build the API binary
	go build -o bin/api cmd/api/main.go

tidy: ## Tidy and vendor dependencies
	go mod tidy && go mod vendor

test: ## Run unit tests (mocks only — fast, no Docker)
	go test -v ./...

test-integration: ## Run integration tests (requires Docker; spins up Postgres + Redis)
	go test -tags=integration -v ./...

swag: ## Regenerate OpenAPI spec (docs/) from godoc annotations (requires: go install github.com/swaggo/swag/cmd/swag@latest)
	$(GOPATH)/bin/swag init -g cmd/api/main.go --output docs --parseDependency --parseInternal

test-cover: ## Run tests with coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

mock: ## Generate mock for an interface (usage: make mock interface=Name dir=path filename=mock.name.go)
	@echo "Generating mocks for interface $(interface) in directory $(dir)..."
	@$(MOCKERY_BIN) --name=$(interface) --dir=$(dir) --output=./internal/mocks
	cd ./internal/mocks && \
	mv $(interface).go $(filename).go

mig-up: ## Run database migrations (up)
	go run cmd/migration/main.go -up

mig-down: ## Run database migrations (down)
	go run cmd/migration/main.go -down

seed: ## Seed the database
	go run cmd/seed/main.go

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format all Go files
	gofmt -w .

docker-up: ## Start all services with Docker Compose
	docker compose -f deploy/docker-compose.yml up --build -d

docker-down: ## Stop all Docker Compose services
	docker compose -f deploy/docker-compose.yml down
