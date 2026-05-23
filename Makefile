.PHONY: run build db-up db-down db-reset seed clean

# ── Config ────────────────────────────────────────────────────
BINARY   := bei
CMD_PATH := .
DB_DSN   := host=localhost port=5432 user=postgres password=postgres dbname=exchange sslmode=disable

# ── Development ───────────────────────────────────────────────

## Start the TUI application
run: db-up
	@sleep 1
	go run $(CMD_PATH)

## Build binary
build:
	go build -o $(BINARY) $(CMD_PATH)

## Run compiled binary
start: build
	./$(BINARY)

# ── Database ──────────────────────────────────────────────────

## Start PostgreSQL via Docker Compose
db-up:
	docker compose up -d postgres
	@echo "⏳ Waiting for PostgreSQL to be ready..."
	@docker compose exec postgres sh -c 'until pg_isready -U postgres -d exchange; do sleep 1; done' 2>/dev/null || true

## Stop PostgreSQL
db-down:
	docker compose down

## Reset database (drop + recreate)
db-reset:
	docker compose down -v
	docker compose up -d postgres
	@echo "⏳ Waiting for PostgreSQL..."
	@sleep 3

## Connect to psql shell
db-shell:
	docker compose exec postgres psql -U postgres -d exchange

# ── Misc ──────────────────────────────────────────────────────

## Install Go dependencies
deps:
	go mod download
	go mod tidy

## Remove built binary
clean:
	rm -f $(BINARY)

## Show help
help:
	@echo "BEI Exchange Simulator — Available targets:"
	@grep -E '^##' Makefile | sed 's/## /  /'
