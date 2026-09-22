.PHONY: run build test lint migrate swagger docker-up docker-down help

APP_NAME=api
MAIN_PATH=cmd/api/main.go
BIN_DIR=bin

help: ## Display available make targets
	@echo "Available commands:"
	@echo "  make run         - Run the API application locally"
	@echo "  make build       - Build the binary executable"
	@echo "  make test        - Run all tests with race detector"
	@echo "  make lint        - Run static code analysis"
	@echo "  make migrate     - Apply database migrations"
	@echo "  make swagger     - Re-generate OpenAPI / Swagger documentation"
	@echo "  make docker-up   - Start docker containers (PostgreSQL & Redis)"
	@echo "  make docker-down - Stop docker containers"

run: ## Run the application
	go run $(MAIN_PATH)

build: ## Build binary
	go build -ldflags="-w -s" -o $(BIN_DIR)/$(APP_NAME).exe $(MAIN_PATH)

test: ## Run tests
	go test -v -race ./...

lint: ## Run linter
	golangci-lint run ./...

migrate: ## Run migrations via Go
	go run $(MAIN_PATH)

swagger: ## Generate Swagger docs
	go run github.com/swaggo/swag/cmd/swag init -g $(MAIN_PATH) -o docs

docker-up: ## Start database & redis
	docker compose up -d

docker-down: ## Stop containers
	docker compose down
