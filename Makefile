.PHONY: build test run clean install help

# Binary name
BINARY_NAME=wordcount

# Build variables
BUILD_DIR=.
CMD_DIR=./cmd/wordcount

help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the word counter binary
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: ./$(BINARY_NAME)"

test: ## Run unit tests
	@echo "Running tests..."
	@go test ./pkg/counter -v

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test ./pkg/counter -bench=. -benchmem

run: build ## Build and run with sample file
	@echo "Running with sample file..."
	@./$(BINARY_NAME) -file testdata/sample.txt -segments 4 -verbose

run-simple: build ## Build and run with simple output
	@./$(BINARY_NAME) -file testdata/sample.txt -segments 4

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@echo "Clean complete"

install: ## Install dependencies
	@echo "Installing dependencies..."
	@go mod download
	@echo "Dependencies installed"

all: clean test build ## Clean, test, and build

.DEFAULT_GOAL := help
