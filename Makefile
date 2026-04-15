# TelemetryFlow Makefile
# Run `make help` to see all available targets.

.DEFAULT_GOAL := help
SHELL := /bin/bash

# Load .env if present (for POSTGRES_DSN, TIMESCALE_DSN etc.)
-include .env
export

BIN_DIR := bin

# ── Build ──────────────────────────────────────────────────────────────────────

.PHONY: build
build: build-simulator build-ingestor build-processor build-sink ## Build all service binaries into bin/

.PHONY: build-simulator
build-simulator: ## Build the simulator binary
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/simulator ./services/simulator

.PHONY: build-ingestor
build-ingestor: ## Build the ingestor binary
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/ingestor ./services/ingestor

.PHONY: build-processor
build-processor: ## Build the processor binary
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/processor ./services/processor

.PHONY: build-sink
build-sink: ## Build the sink binary
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/sink ./services/sink

# ── Run (requires .env to be configured) ──────────────────────────────────────

.PHONY: run-simulator
run-simulator: ## Run the simulator service
	go run ./services/simulator

.PHONY: run-ingestor
run-ingestor: ## Run the ingestor service
	go run ./services/ingestor

.PHONY: run-processor
run-processor: ## Run the processor service
	go run ./services/processor

.PHONY: run-sink
run-sink: ## Run the sink service
	go run ./services/sink

# ── Database ───────────────────────────────────────────────────────────────────

.PHONY: migrate-postgres
migrate-postgres: ## Run PostgreSQL migrations (requires POSTGRES_DSN in .env)
	@test -n "$(POSTGRES_DSN)" || (echo "Error: POSTGRES_DSN not set. Copy .env.example to .env and fill in values." && exit 1)
	goose -dir migrations/postgres postgres "$(POSTGRES_DSN)" up

.PHONY: migrate-postgres-down
migrate-postgres-down: ## Roll back the last PostgreSQL migration
	goose -dir migrations/postgres postgres "$(POSTGRES_DSN)" down

.PHONY: migrate-postgres-status
migrate-postgres-status: ## Show PostgreSQL migration status
	goose -dir migrations/postgres postgres "$(POSTGRES_DSN)" status

.PHONY: migrate-timescale
migrate-timescale: ## Run TimescaleDB migrations (requires TIMESCALE_DSN in .env)
	@test -n "$(TIMESCALE_DSN)" || (echo "Error: TIMESCALE_DSN not set. Copy .env.example to .env and fill in values." && exit 1)
	goose -dir migrations/timescale postgres "$(TIMESCALE_DSN)" up

.PHONY: migrate-timescale-down
migrate-timescale-down: ## Roll back the last TimescaleDB migration
	goose -dir migrations/timescale postgres "$(TIMESCALE_DSN)" down

.PHONY: migrate-timescale-status
migrate-timescale-status: ## Show TimescaleDB migration status
	goose -dir migrations/timescale postgres "$(TIMESCALE_DSN)" status

.PHONY: create-db
create-db: ## Create the local telemetryflow database (PostgreSQL must already be running)
	createdb telemetryflow

.PHONY: drop-db
drop-db: ## Drop the local telemetryflow database (destructive!)
	dropdb --if-exists telemetryflow

.PHONY: seed
seed: ## Seed 1000 test devices into PostgreSQL
	psql "$(POSTGRES_DSN)" -f scripts/seed-devices.sql

# ── Kafka topics (Confluent Cloud) ────────────────────────────────────────────

.PHONY: create-topics
create-topics: ## Create all Kafka topics on Confluent Cloud (requires CONFLUENT_CLUSTER_ID)
	./scripts/create-topics.sh

# ── Testing ───────────────────────────────────────────────────────────────────

.PHONY: test
test: ## Run all tests across all modules
	go test ./...

.PHONY: test-verbose
test-verbose: ## Run all tests with verbose output
	go test -v ./...

.PHONY: test-race
test-race: ## Run tests with race detector enabled
	go test -race ./...

# ── Code quality ──────────────────────────────────────────────────────────────

.PHONY: vet
vet: ## Run go vet across all modules
	go vet ./...

.PHONY: tidy
tidy: ## Run go mod tidy across all modules and sync workspace
	go work sync
	cd config            && go mod tidy
	cd pkg               && go mod tidy
	cd services/simulator && go mod tidy
	cd services/ingestor  && go mod tidy
	cd services/processor && go mod tidy
	cd services/sink      && go mod tidy

# ── Cleanup ───────────────────────────────────────────────────────────────────

.PHONY: clean
clean: ## Remove compiled binaries
	rm -rf $(BIN_DIR)/

# ── Help ──────────────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help message
	@echo "TelemetryFlow — available make targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Quick start:"
	@echo "  cp .env.example .env    # fill in your credentials"
	@echo "  make create-db          # create the local database"
	@echo "  make migrate-postgres   # apply schema migrations"
	@echo "  make migrate-timescale  # apply TimescaleDB migrations"
	@echo "  make create-topics      # create Confluent Cloud topics"
	@echo "  make seed               # seed 1000 test devices"
