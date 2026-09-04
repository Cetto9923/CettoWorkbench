# workbench — common build, quality, and run commands (execute at repository root)
BINARY := workbench
DIST := dist/workbench
GOCACHE_DIR := $(CURDIR)/tmp/gocache

.PHONY: build check check-gofmt check-test check-vet check-whitespace check-file-length check-patterns check-secrets check-architecture

check: check-gofmt check-test check-vet check-whitespace check-file-length check-patterns check-secrets check-architecture
	@echo "required regression gates passed; inspect the printed existing-debt counts"

check-gofmt:
	@./scripts/check-gofmt.sh

check-test:
	@mkdir -p $(GOCACHE_DIR)
	@GOCACHE=$(GOCACHE_DIR) go test ./...

check-vet:
	@./scripts/check-go-vet.sh

check-whitespace:
	@git diff --check
	@git diff --cached --check
	@if git rev-parse --verify HEAD^ >/dev/null 2>&1; then git diff --check HEAD^ HEAD; else git show --check --format= HEAD; fi
	@echo "whitespace gate passed"

check-file-length:
	@./scripts/check-file-length.sh

check-patterns:
	@./scripts/check-patterns.sh

check-secrets:
	@./scripts/check-secrets.sh

check-architecture:
	@./scripts/check-architecture.sh

build: ## Build a self-contained deployment directory under $(DIST)
	@rm -rf $(DIST)
	@mkdir -p $(DIST)/configs $(DIST)/db
	@cp -a configs/config.yaml $(DIST)/configs/
	@cp -a web db $(DIST)/
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(DIST)/$(BINARY) ./cmd/server
	@echo ""
	@echo "deployment directory generated: $(DIST)"
