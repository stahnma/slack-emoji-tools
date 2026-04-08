export CGO_ENABLED := 0
BINARY  := slack-emoji-tools
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/stahnma/slack-emoji-tools/cmd.version=$(VERSION)"

.PHONY: build clean test test-short vet fmt install tidy help

build: tidy fmt ## Build the binary
	go build $(LDFLAGS) -o $(BINARY) .

install: ## Install to GOPATH/bin
	go install $(LDFLAGS) .

test: fmt vet ## Run tests
	go test ./... -v

test-short: ## Run tests (short mode)
	go test ./... -short

vet: ## Run go vet
	go vet ./...

fmt: ## Run gofmt
	gofmt -s -w .

tidy: ## Run go mod tidy
	go mod tidy

clean: ## Remove build artifacts
	rm -f $(BINARY)
	go clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
