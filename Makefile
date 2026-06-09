# ====================================================================================
#  Makefile for Go Projects
# ====================================================================================

# ------------------------------------------------------------------------------------
#  Configuration
# ------------------------------------------------------------------------------------

# Binary name for the compiled application
BINARY_NAME=main

# The path to the main package to build/run  
CMD_PATH=./cmd

# Go command
GO=go

# Linker flags for building a smaller binary in production
LDFLAGS=-ldflags="-s -w"

# ====================================================================================
#  Commands
# ====================================================================================

.DEFAULT_GOAL := help

.PHONY: help build dev prod clean wire docker-build docker-run docker-prod docker-clean

help: ## ✨ Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## 📦 Build the application binary
	@$(MAKE) clean
	@$(MAKE) wire
	@echo "Building binary..."
	@$(GO) build $(LDFLAGS) -o ./cmd/$(BINARY_NAME) $(CMD_PATH)
	@echo "Binary created at cmd/$(BINARY_NAME)"

dev: ## 🚀 Run the application in development mode (with hot-reload)
	@echo "Starting dev server with hot-reload using 'go run'..."
	@go run github.com/air-verse/air@latest -c .air.toml

test: ## 🧪 Run all tests
	@echo "Running tests..."
	@$(GO) test ./... -v -coverprofile=coverage.out

coverage: ## 🧪 coverage report
	@echo "Running tests..."
	@$(GO) tool cover -html=coverage.out

wire: ## ⚡ Generate wire dependency injection code
	@echo "Generating wire dependency injection code..."
	@cd internal/apps && wire
	@echo "Wire code generation completed"

local: ## 🚀 Run the application in local mode (with hot-reload)
	@echo "Starting dev server with hot-reload using 'go run'..."
	@go run github.com/air-verse/air@latest -c .air.local.toml

prod: ## ⚙️  Run the application in production mode
	@$(MAKE) build
	@echo "Starting application in production mode..."
	@./cmd/$(BINARY_NAME) --stage prod

clean: ## 🧹 Remove build artifacts
	@echo "Cleaning up..."
	@rm -rf ./cmd/main
	@echo "Done."

migrate-create: ## 🛠️  Create a new database migration file (usage: make migrate-create name=my_migration)
	@if [ -z "$(name)" ]; then \
		echo "Error: name is required. Usage: make migrate-create name=my_migration"; \
		exit 1; \
	fi
	@echo "Creating migration files for $(name)..."
	@go run -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest create -ext sql -dir internal/dbs/migrations -seq $(name)
	@echo "Migration created successfully."