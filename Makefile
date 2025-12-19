# uos-openldap-exporter Makefile
#
# This Makefile provides common build, test, and run targets for the project.

# Project variables
BINARY_NAME=uos-openldap-exporter
MAIN_PACKAGE=./main.go
VERSION ?= $(shell cat VERSION)

# Go variables
GO ?= go
GOFMT ?= gofmt -s
GOFILES := $(shell find . -name "*.go" -type f)

# Default target
.PHONY: all
all: build ## Build the project (default)

# Build binary
.PHONY: build
build:
	$(GO) build -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Build release binaries for different platforms
.PHONY: release
release: ## Build release binaries for Linux, macOS, and Windows
	GOOS=linux GOARCH=amd64 $(GO) build -o dist/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=amd64 $(GO) build -o dist/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	GOOS=windows GOARCH=amd64 $(GO) build -o dist/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)

# Install dependencies
.PHONY: deps
deps: ## Download dependencies
	$(GO) mod download

# Clean build artifacts
.PHONY: clean
clean: ## Remove build artifacts
	rm -f $(BINARY_NAME)
	rm -rf dist/

# Run tests
.PHONY: test
test: ## Run tests
	$(GO) test -v ./...

# Run tests with coverage
.PHONY: test-cover
test-cover: ## Run tests with coverage report
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out

# Format source code
.PHONY: fmt
fmt: ## Format source code
	$(GOFMT) -w $(GOFILES)

# Check for formatting issues
.PHONY: fmt-check
fmt-check: ## Check for formatting issues
	@diff=$$($(GOFMT) -d $(GOFILES)); \
	if [ -n "$$diff" ]; then \
		echo "Please run 'make fmt' to format the code"; \
		echo "$${diff}"; \
		exit 1; \
	fi

# Vet the source code
.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

# Run security checks
.PHONY: sec
sec: ## Run security checks with gosec
	@which gosec > /dev/null || { \
		echo "Installing gosec..."; \
		$(GO) install github.com/securego/gosec/v2/cmd/gosec@latest; \
	}
	gosec ./...

# Build Docker image
.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t $(BINARY_NAME):$(VERSION) .

# Push Docker image
.PHONY: docker-push
docker-push: ## Push Docker image
	docker push $(BINARY_NAME):$(VERSION)

# Help documentation
.PHONY: help
help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'