.PHONY: build test clean install tidy run vet build-all help fmt lint coverage test-race

#################################################################################
# GLOBALS                                                                       #
#################################################################################

PROJECT_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

BINARY=lfsnag
BUILD_DIR=bin

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet
GOFMT=$(GOCMD) fmt

all: help

## System:

sys/changelog: ## Generate changelog from git tags
	@echo "Generating CHANGELOG.md..."
	@echo "" > CHANGELOG.md;
	@previous_tag=0; \
	for current_tag in $$(git tag --sort=-creatordate | grep '^v'); do \
		if [ "$$previous_tag" != 0 ]; then \
			tag_date=$$(git log -1 --pretty=format:'%ad' --date=short $$previous_tag); \
			printf "\n## $$previous_tag ($$tag_date)\n\n" >> CHANGELOG.md; \
			git log $$current_tag...$$previous_tag --pretty=format:'*  %s [%an]' --reverse | grep -v Merge >> CHANGELOG.md; \
			printf "\n" >> CHANGELOG.md; \
		fi; \
		previous_tag=$$current_tag; \
	done
	@echo "CHANGELOG.md generated successfully."

sys/tag: ## Create and push version tag
	@read -p "Enter tag version (e.g., 1.0.0): " TAG; \
	if echo "$$TAG" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		git tag -a v$$TAG -m v$$TAG; \
		git push origin v$$TAG; \
		echo "Tag v$$TAG created and pushed successfully."; \
	else \
		echo "Invalid tag format. Please use X.Y.Z (e.g., 1.0.0)"; \
		exit 1; \
	fi

## Development:

build: tidy ## Build the binary
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY) ./cmd/lfsnag

run: build ## Build and run
	@./$(BUILD_DIR)/$(BINARY)

run/quick: ## Run without rebuild
	@./$(BUILD_DIR)/$(BINARY)

test: ## Run tests
	$(GOTEST) -v ./...

test-race: ## Run tests with race detection
	$(GOTEST) -race -v ./...

coverage: ## Run tests with coverage report
	$(GOTEST) -cover ./...

coverage-html: ## Generate HTML coverage report
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

fmt: ## Format code
	$(GOFMT) ./...

vet: ## Run go vet
	$(GOVET) ./...

lint: fmt vet ## Run fmt and vet

tidy: ## Tidy dependencies
	$(GOMOD) tidy

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY)
	rm -f coverage.out coverage.html

## Installation:

install: tidy ## Install to GOPATH/bin
	$(GOCMD) install ./cmd/lfsnag

build-all: tidy ## Build for linux/darwin/windows amd64/arm64
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/lfsnag
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 ./cmd/lfsnag
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/lfsnag
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/lfsnag
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/lfsnag

## Help:

help: ## Show this help
	@echo "lfsnag - Logfire Trace Query CLI"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; section=""} \
		/^##/ { section=substr($$0, 4); next } \
		/^[a-zA-Z_\/-]+:.*##/ { \
			if (section != "") { printf "\n\033[1m%s\033[0m\n", section; section="" } \
			printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 \
		}' $(MAKEFILE_LIST)
	@echo ""
