# Makefile for mcpx-cli

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary name
BINARY_NAME=mcpx-cli
BINARY_UNIX=$(BINARY_NAME)_unix

# Build the project
all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)

deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./...

# Install for local use
install:
	$(GOCMD) install -v ./...

# Format code
fmt:
	$(GOCMD) fmt ./...

# Vet code
vet:
	$(GOCMD) vet ./...

# Help commands
help-health:
	./$(BINARY_NAME) health

help-servers:
	./$(BINARY_NAME) servers --limit 5

help-server:
	@echo "To get server details, run:"
	@echo "./$(BINARY_NAME) server <server-id>"

help-publish:
	@echo "To publish a server, run:"
	@echo "./$(BINARY_NAME) publish example-server.json --token <your-token>"

# Demo commands (requires mcpx server running)
demo-health: build
	@echo "=== Testing Health Endpoint ==="
	./$(BINARY_NAME) health

demo-servers: build
	@echo "=== Testing Servers List Endpoint ==="
	./$(BINARY_NAME) servers --limit 10

demo-all: demo-health demo-servers

.PHONY: all build test clean run deps build-linux install fmt vet help-health help-servers help-server help-publish demo-health demo-servers demo-all
