# flicker Project Makefile

# Build configuration
GO_SRC_DIR ?= .
GOLANGCILINT_CONFIG_PATH ?= $(PWD)/.golangci.yml

# Include modular makefiles
include makefiles/shared.mk
include makefiles/go.mk
include makefiles/git.mk

.PHONY: build flicker pr-ready verify install

# Add flicker to main build target
build: flicker

flicker: ## Build the flicker CLI binary
	$(info $(_bullet) Building <flicker>)
	@cd $(GO_SRC_DIR) && \
	go build -o ./bin/flicker ./cmd/flicker

pr-ready: tidy-go generate format build lint test git-dirty ## Run comprehensive pre-commit checks

install: ## Install the flicker binary
	$(info $(_bullet) Installing <flicker>)
	go install .

# Verify staged changes and record tree SHA for pre-commit hook
verify: pr-ready ## Verify staged changes pass all checks and record for commit
	$(info $(_bullet) Recording verified tree SHA)
	@git write-tree > "$$(git rev-parse --git-dir)/verified-tree"
	@echo "Verified tree: $$(cat "$$(git rev-parse --git-dir)/verified-tree")"
	@echo "You can now commit your changes."
