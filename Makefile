# TelemetryFlow Makefile
# Run `make help` to see all available targets.

.DEFAULT_GOAL := help
SHELL := /bin/bash

BIN_DIR := bin

# Ensure tools installed via `go install` (e.g. goose) are on PATH.
# `go install` puts binaries in $GOPATH/bin which Make doesn't inherit.
export PATH := $(shell go env GOPATH)/bin:$(PATH)

# Load .env as shell variables (not Make variables) to avoid Makefile
# syntax conflicts with special characters in DSN strings.
# Usage: $(call load_env) at the start of any recipe that needs .env vars.
define load_env
	set -a; source .env; set +a
endef

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
	@$(call load_env); go run ./services/simulator

.PHONY: run-ingestor
run-ingestor: ## Run the ingestor service
	@$(call load_env); go run ./services/ingestor

.PHONY: run-processor
run-processor: ## Run the processor service
	@$(call load_env); go run ./services/processor

.PHONY: run-sink
run-sink: ## Run the sink service
	@$(call load_env); go run ./services/sink

# ── Database ───────────────────────────────────────────────────────────────────

.PHONY: migrate-postgres
migrate-postgres: ## Run PostgreSQL migrations (requires POSTGRES_DSN in .env)
	@$(call load_env); \
	test -n "$$POSTGRES_DSN" || (echo "Error: POSTGRES_DSN not set. Copy .env.example to .env and fill in values." && exit 1); \
	goose -dir migrations/postgres postgres "$$POSTGRES_DSN" up

.PHONY: migrate-postgres-down
migrate-postgres-down: ## Roll back the last PostgreSQL migration
	@$(call load_env); goose -dir migrations/postgres postgres "$$POSTGRES_DSN" down

.PHONY: migrate-postgres-status
migrate-postgres-status: ## Show PostgreSQL migration status
	@$(call load_env); goose -dir migrations/postgres postgres "$$POSTGRES_DSN" status

.PHONY: migrate-timescale
migrate-timescale: ## Run TimescaleDB migrations (requires TIMESCALE_DSN in .env)
	@$(call load_env); \
	test -n "$$TIMESCALE_DSN" || (echo "Error: TIMESCALE_DSN not set. Copy .env.example to .env and fill in values." && exit 1); \
	goose -dir migrations/timescale postgres "$$TIMESCALE_DSN" up

.PHONY: migrate-timescale-down
migrate-timescale-down: ## Roll back the last TimescaleDB migration
	@$(call load_env); goose -dir migrations/timescale postgres "$$TIMESCALE_DSN" down

.PHONY: migrate-timescale-status
migrate-timescale-status: ## Show TimescaleDB migration status
	@$(call load_env); goose -dir migrations/timescale postgres "$$TIMESCALE_DSN" status

.PHONY: create-db
create-db: ## Create the local telemetryflow database (PostgreSQL must already be running)
	createdb telemetryflow

.PHONY: drop-db
drop-db: ## Drop the local telemetryflow database (destructive!)
	dropdb --if-exists telemetryflow

.PHONY: seed
seed: ## Seed 1000 test devices into PostgreSQL
	@$(call load_env); psql "$$POSTGRES_DSN" -f scripts/seed-devices.sql

# ── Kafka topics (Confluent Cloud) ────────────────────────────────────────────

.PHONY: create-topics
create-topics: ## Create all Kafka topics on Confluent Cloud (requires CONFLUENT_CLUSTER_ID)
	@$(call load_env); ./scripts/create-topics.sh

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

# ── Tools ─────────────────────────────────────────────────────────────────────

.PHONY: install-tools
install-tools: ## Install CLI tools required by this Makefile (goose, confluent CLI)
	go install github.com/pressly/goose/v3/cmd/goose@latest
	@echo "goose installed: $$(goose --version)"
	brew install confluentinc/tap/cli
	@echo "confluent installed: $$(confluent version)"
	@echo ""
	@echo "Next: run 'confluent login' to authenticate with Confluent Cloud"

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
	@echo "  make install-tools      # install goose and other CLI tools"
	@echo "  make create-db          # create the local database"
	@echo "  make migrate-postgres   # apply schema migrations"
	@echo "  make migrate-timescale  # apply TimescaleDB migrations"
	@echo "  make create-topics      # create Confluent Cloud topics"
	@echo "  make seed               # seed 1000 test devices"
