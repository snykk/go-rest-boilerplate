# Development tasks. `make` or `make help` lists them.

.DEFAULT_GOAL := help

COMPOSE := docker compose -f deployment/compose.yml

.PHONY: help run test test-integration lint generate generate-check openapi openapi-check \
	migrate-up migrate-down migrate-status migrate-create docker-up docker-down tidy

help: ## List the targets
	@grep -hE '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  %-18s %s\n", $$1, $$2}'

run: ## Run the API; it reads .env when present
	go run ./cmd/server

test: ## Run the unit and architecture tests
	go test ./...

test-integration: ## Run every test, including those against Postgres in Docker, one package at a time
	go test -p 1 -tags=integration ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

generate: ## Regenerate the gomock mocks
	go generate ./...

generate-check: generate ## Fail when fresh mocks differ from the committed ones, a new file included
	@! git status --porcelain --untracked-files=all -- '*_mock_test.go' | grep .

openapi: ## Regenerate api/openapi from the handler annotations
	go tool swag init -g cmd/server/main.go -o api/openapi --parseInternal --outputTypes go,json,yaml

openapi-check: openapi ## Fail when fresh OpenAPI files differ from the committed ones, a new file included
	@! git status --porcelain --untracked-files=all -- api/openapi | grep .

migrate-up: ## Apply every pending migration
	go run ./cmd/migrate up

migrate-down: ## Roll back the latest migration
	go run ./cmd/migrate down

migrate-status: ## List the migrations and whether each is applied
	go run ./cmd/migrate status

migrate-create: ## Create a migration: make migrate-create name=<module>_<action>
	@test -n "$(name)" || { echo "usage: make migrate-create name=<module>_<action>"; exit 1; }
	go run ./cmd/migrate create $(name)

docker-up: ## Build and start Postgres, the migrations and the API
	$(COMPOSE) up --build -d

docker-down: ## Stop the stack and delete its database volume
	$(COMPOSE) down -v

tidy: ## Tidy go.mod and go.sum
	go mod tidy
