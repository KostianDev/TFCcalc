# Makefile for TFCcalc
# Usage:
#   make all     — start the DB if needed, run tests, build and run the binary
#   make db-up   — start MySQL via docker-compose if not already running, wait until ready
#   make db-down — stop MySQL container
#   make test    — run unit tests
#   make build   — build the tfccalc binary
#   make run     — run the tfccalc binary (after make build)

BINARY := tfccalc

.PHONY: all db-up db-down test build run

all: db-up test build run

# Start MySQL only if not already running, then wait until ready
db-up:
	@running=$$(docker-compose ps -q mysql | xargs docker inspect -f '{{.State.Running}}'); \
	if [ "$$running" != "true" ]; then \
		echo "Starting MySQL container..."; \
		docker-compose up -d; \
	else \
		echo "MySQL container already running"; \
	fi
	@echo "Waiting for MySQL TCP port to open..."
	@bash -c 'while ! </dev/tcp/127.0.0.1/3306; do sleep 1; done'
	@echo "TCP port 3306 is open; pausing 5 more seconds for full initialization..."
	@sleep 5
	@echo "MySQL is ready."

# Stop MySQL container
db-down:
	docker-compose down

# Run unit tests in calculator and data packages
test:
	@echo "=== Running unit tests ==="
	@go test ./calculator ./data

# Build the Go binary
build:
	@echo "=== Building $(BINARY) ==="
	@go build -o $(BINARY) main.go

# Run the compiled binary
run:
	@echo "=== Running $(BINARY) ==="
	@./$(BINARY)