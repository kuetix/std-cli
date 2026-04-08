APP_NAME := cli
BUILD_DIR := runtime/bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)'

#-ldflags "-X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(date -u +"%Y-%m-%dT%H:%M:%SZ")'"
.PHONY: all pkg clean test

.DEFAULT_GOAL := help

help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the kue PKG tool
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/pkg/

all: pkg help

pkg:  ## Build the kue PKG tool
	@echo "Building package runner..."
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/pkg

test: ## Run all tests
	kue test --dir tests

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)
