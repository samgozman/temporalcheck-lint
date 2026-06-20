BINARY := ./bin/custom-gcl

COVERAGE_MIN := 90

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the custom golangci-lint binary (reads .custom-gcl.yml)
	golangci-lint custom

.PHONY: run
run: ## Lint this repo with the custom binary
	$(BINARY) run

.PHONY: test
test: ## Run the Go test suite
	go test -race -v ./...

.PHONY: test-cov
test-cov: ## Run the Go test suite with coverage
	go test -race -coverprofile=coverage.out ./...

.PHONY: cover-check
cover-check: test-cov ## Run tests with coverage and fail below COVERAGE_MIN
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ {print $$3}' | tr -d '%'); \
	echo "total coverage: $$total% (minimum $(COVERAGE_MIN)%)"; \
	awk "BEGIN { exit !($$total >= $(COVERAGE_MIN)) }" || \
		{ echo "FAIL: coverage $$total% is below minimum $(COVERAGE_MIN)%"; exit 1; }

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: tidy
tidy: ## Tidy module dependencies
	go mod tidy
