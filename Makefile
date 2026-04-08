VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X github.com/stahnma/slack-emoji-tools/cmd.version=$(VERSION)"
BINARY  := slack-emoji-tools

.PHONY: build install test lint tidy clean

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY) .

install:
	CGO_ENABLED=0 go install $(LDFLAGS) .

test:
	go test ./... -v

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)
