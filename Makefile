.PHONY: help up down logs restart clean proto build-all test

# Color output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

help: ## Show this help message
	@echo "$(BLUE)GreenLane - Distributed EV Charging Orchestration$(NC)"
	@echo ""
	@echo "$(GREEN)Available commands:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'

up: ## Start all infrastructure services (Redis, Redpanda, TimescaleDB)
	@echo "$(GREEN)🚀 Starting GreenLane infrastructure...$(NC)"
	docker-compose -f deploy/docker-compose.yml up -d --remove-orphans
	@echo "$(GREEN)✅ Services started!$(NC)"
	@echo "$(BLUE)📊 Redpanda Console: http://localhost:8080$(NC)"
	@echo "$(BLUE)📊 TimescaleDB: postgresql://greenlane:greenlane_password@localhost:5432/greenlane$(NC)"

down: ## Stop all services
	@echo "$(YELLOW)🛑 Stopping GreenLane infrastructure...$(NC)"
	docker-compose -f deploy/docker-compose.yml down

logs: ## Show logs from all services
	docker-compose -f deploy/docker-compose.yml logs -f

logs-redis: ## Show Redis logs
	docker-compose -f deploy/docker-compose.yml logs -f redis

logs-redpanda: ## Show Redpanda logs
	docker-compose -f deploy/docker-compose.yml logs -f redpanda

logs-timescale: ## Show TimescaleDB logs
	docker-compose -f deploy/docker-compose.yml logs -f timescaledb

restart: down up ## Restart all services

clean: down ## Stop services and remove volumes (⚠️  deletes all data)
	@echo "$(YELLOW)⚠️  WARNING: This will delete all data!$(NC)"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1
	docker-compose -f deploy/docker-compose.yml down -v
	@echo "$(GREEN)✅ All data cleaned!$(NC)"

proto: ## Generate code from Protocol Buffers
	@echo "$(GREEN)📝 Generating protobuf code...$(NC)"
	./scripts/generate-proto.sh

build-ingestion: ## Build the Ingestion Service
	@echo "$(GREEN)🔨 Building Ingestion Service...$(NC)"
	cd services/ingestion && go build -o ../../bin/ingestion .

build-mock-grid: ## Build the Mock Grid Service
	@echo "$(GREEN)🔨 Building Mock Grid Service...$(NC)"
	cd services/mock-grid && go build -o ../../bin/mock-grid .

build-cli: ## Build the Live Ops CLI
	@echo "$(GREEN)🔨 Building Live Ops CLI...$(NC)"
	cd cli && go build -o ../bin/greenlane-cli .

build-pricing-worker: ## Build the Pricing Worker (Rust)
	@echo "$(GREEN)🔨 Building Pricing Worker...$(NC)"
	cd services/pricing-worker && cargo build --release
	cp services/pricing-worker/target/release/pricing-worker bin/

build-all: proto build-ingestion build-mock-grid build-cli build-pricing-worker ## Build all services

test: ## Run all tests
	@echo "$(GREEN)🧪 Running tests...$(NC)"
	cd services/ingestion && go test -v ./...
	cd services/mock-grid && go test -v ./...
	cd cli && go test -v ./...
	cd services/pricing-worker && cargo test

dev-ingestion: ## Run Ingestion Service in development mode
	cd services/ingestion && go run .

dev-mock-grid: ## Run Mock Grid Service in development mode
	cd services/mock-grid && go run .

dev-cli: ## Run Live Ops CLI in development mode
	cd cli && go run .

dev-simulator: ## Run Fleet Simulator
	cd simulator && python3 simulator.py

status: ## Show status of all containers
	docker-compose -f deploy/docker-compose.yml ps
